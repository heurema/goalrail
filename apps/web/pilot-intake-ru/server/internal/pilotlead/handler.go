package pilotlead

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Server struct {
	Config Config
	Store  *Store
	Mailer Mailer
}

func NewServer(config Config, store *Store, mailer Mailer) *Server {
	config = config.withDefaults()
	if store == nil {
		store = NewStore(config.LeadLogPath, config.Now)
	}
	if mailer == nil {
		mailer = NewTransportMailer(config)
	}
	return &Server{Config: config, Store: store, Mailer: mailer}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.URL.Path != "/api/pilot-lead" {
		respondJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		return
	}

	data, status, errorCode := parseRequestData(r)
	if errorCode != "" {
		respondJSON(w, status, map[string]any{"ok": false, "error": errorCode})
		return
	}

	if field(data, "website") != "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_request"})
		return
	}

	email := field(data, "email")
	if !ValidateEmail(email) {
		respondJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_email"})
		return
	}

	now := s.now()
	loc := MoscowLocation()
	submittedAt := formatUTC(now)
	record := LeadRecord{
		"submitted_at":         submittedAt,
		"submitted_at_local":   formatLocal(now, loc),
		"submitted_date_local": formatLocalDate(now, loc),
		"email":                email,
		"source":               truncateDefault(field(data, "source"), "ru-pilot", 80),
		"page":                 truncateDefault(field(data, "page"), "pilot.goalrail.ru", 120),
	}

	stored, err := s.Store.PrepareAttempt(record, email, now)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "log_unavailable"})
		return
	}
	if !stored {
		respondJSON(w, http.StatusOK, map[string]any{"ok": true, "duplicate": true})
		return
	}

	body := leadMailBody(email, stringValue(record, "source"), stringValue(record, "page"), submittedAt)
	recipient, err := MailRecipient(s.Config)
	if err != nil {
		s.markFailed(r.Context(), email)
		respondJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": ErrorMailUnavailable})
		return
	}

	transport, err := s.Mailer.SendText(r.Context(), recipient, LeadSubject, body, email)
	if err != nil {
		s.markFailed(r.Context(), email)
		respondJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": ErrorMailUnavailable})
		return
	}
	if err := s.Store.MarkNotificationResult(email, StatusNotified, transport, "", s.now()); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "log_unavailable"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "duplicate": false})
}

func (s *Server) markFailed(_ context.Context, email string) {
	_ = s.Store.MarkNotificationResult(email, StatusNotificationFailed, "", ErrorMailUnavailable, s.now())
}

func (s *Server) now() time.Time {
	if s.Config.Now != nil {
		return s.Config.Now()
	}
	return time.Now()
}

func parseRequestData(r *http.Request) (map[string]string, int, string) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, http.StatusBadRequest, "invalid_request"
	}
	if len(body) > MaxBodyBytes {
		return nil, http.StatusRequestEntityTooLarge, "request_too_large"
	}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil || decoded == nil {
			return nil, http.StatusBadRequest, "invalid_json"
		}
		return stringMap(decoded), 0, ""
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, http.StatusBadRequest, "invalid_request"
		}
		return valuesMap(values), 0, ""
	}

	if len(bytes.TrimSpace(body)) > 0 {
		values, err := url.ParseQuery(string(body))
		if err == nil && len(values) > 0 {
			return valuesMap(values), 0, ""
		}
	}

	return map[string]string{}, 0, ""
}

func respondJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		_, _ = fmt.Fprint(w, `{"ok":false,"error":"response_unavailable"}`)
	}
}

func stringMap(values map[string]any) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			out[key] = strings.TrimSpace(text)
		}
	}
	return out
}

func valuesMap(values url.Values) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			out[key] = strings.TrimSpace(value[0])
		}
	}
	return out
}

func field(data map[string]string, key string) string {
	if data == nil {
		return ""
	}
	return strings.TrimSpace(data[key])
}

func truncateDefault(value, fallback string, max int) string {
	if value == "" {
		value = fallback
	}
	return truncate(value, max)
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func leadMailBody(email, source, page, submittedAt string) string {
	return strings.Join([]string{
		"Новая заявка с RU лендинга GoalRail.",
		"",
		"Email: " + email,
		"Source: " + source,
		"Page: " + page,
		"Submitted at: " + submittedAt,
		"",
		"Ответьте на это письмо, чтобы написать посетителю напрямую.",
	}, "\n")
}
