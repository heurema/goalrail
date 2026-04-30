package pilotlead

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"strings"
)

var ErrMailUnavailable = errors.New("mail unavailable")

type Mailer interface {
	SendText(ctx context.Context, to, subject, text, replyTo string) (string, error)
}

type TransportMailer struct {
	Config Config
}

func NewTransportMailer(config Config) *TransportMailer {
	return &TransportMailer{Config: config.withDefaults()}
}

func (m *TransportMailer) SendText(ctx context.Context, to, subject, text, replyTo string) (string, error) {
	config := m.Config.withDefaults()
	if !ValidateEmail(to) {
		return "", fmt.Errorf("%w: recipient_unavailable", ErrMailUnavailable)
	}

	apiKey, err := readResendAPIKey(config.ResendAPIKeyPath)
	if err != nil {
		return "", err
	}
	if apiKey != "" {
		if err := sendResendEmail(ctx, config, apiKey, to, subject, text, replyTo); err != nil {
			return "", err
		}
		return "resend", nil
	}

	if err := sendSendmailEmail(ctx, config, to, subject, text, replyTo); err != nil {
		return "", err
	}
	return "sendmail", nil
}

func MailRecipient(config Config) (string, error) {
	config = config.withDefaults()
	content, err := os.ReadFile(config.RecipientOverridePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return config.RecipientDefault, nil
		}
		return "", fmt.Errorf("%w: recipient_unavailable", ErrMailUnavailable)
	}

	recipient := strings.TrimSpace(string(content))
	if !ValidateEmail(recipient) {
		return "", fmt.Errorf("%w: recipient_unavailable", ErrMailUnavailable)
	}
	return recipient, nil
}

func ValidateEmail(email string) bool {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || trimmed != email || len(trimmed) > 254 {
		return false
	}
	if strings.ContainsAny(trimmed, "\r\n") || !strings.Contains(trimmed, "@") {
		return false
	}
	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed == nil {
		return false
	}
	if parsed.Name != "" || parsed.Address != trimmed {
		return false
	}
	return true
}

func readResendAPIKey(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return "", nil
		}
		return "", fmt.Errorf("%w: resend_key_unavailable", ErrMailUnavailable)
	}
	apiKey := strings.TrimSpace(string(content))
	if apiKey == "" || len(apiKey) > 512 || strings.ContainsAny(apiKey, "\r\n") || !strings.HasPrefix(apiKey, "re_") {
		return "", fmt.Errorf("%w: resend_key_unavailable", ErrMailUnavailable)
	}
	return apiKey, nil
}

func sendResendEmail(ctx context.Context, config Config, apiKey, to, subject, text, replyTo string) error {
	payload := map[string]any{
		"from":    config.ResendFrom,
		"to":      []string{to},
		"subject": subject,
		"text":    text,
	}
	if ValidateEmail(replyTo) {
		payload["reply_to"] = replyTo
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: resend_payload_unavailable", ErrMailUnavailable)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, config.ResendAPIURL, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("%w: resend_unavailable", ErrMailUnavailable)
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := config.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: resend_unavailable", ErrMailUnavailable)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: resend_unavailable", ErrMailUnavailable)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: resend_rejected", ErrMailUnavailable)
	}

	var decoded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.ID == "" {
		return fmt.Errorf("%w: resend_unexpected_response", ErrMailUnavailable)
	}
	return nil
}

func sendSendmailEmail(ctx context.Context, config Config, to, subject, text, replyTo string) error {
	sendmailPath, err := resolveSendmailPath(config.SendmailPath)
	if err != nil {
		return err
	}

	message := buildSendmailMessage(config, to, subject, text, replyTo)
	command := exec.CommandContext(ctx, sendmailPath, "-f", config.SendmailEnvelopeFrom, "-t")
	command.Stdin = strings.NewReader(message)
	output, err := command.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: sendmail_unavailable", ErrMailUnavailable)
		}
		return fmt.Errorf("%w: sendmail_unavailable", ErrMailUnavailable)
	}
	return nil
}

func resolveSendmailPath(configured string) (string, error) {
	candidates := make([]string, 0, 4)
	if configured != "" {
		candidates = append(candidates, configured)
	}
	if found, err := exec.LookPath("sendmail"); err == nil {
		candidates = append(candidates, found)
	}
	candidates = append(candidates, "/usr/sbin/sendmail", "/usr/lib/sendmail")

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: sendmail_unavailable", ErrMailUnavailable)
}

func buildSendmailMessage(config Config, to, subject, text, replyTo string) string {
	headers := []string{
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"From: " + config.SendmailFrom,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
		"X-Mailer: GoalRail pilot fallback mail transport",
	}
	if ValidateEmail(replyTo) {
		headers = append(headers, "Reply-To: "+replyTo)
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + text + "\r\n"
}
