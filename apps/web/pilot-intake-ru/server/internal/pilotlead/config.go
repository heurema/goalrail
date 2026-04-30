package pilotlead

import (
	"net/http"
	"time"
)

const (
	DefaultListenAddr            = "127.0.0.1:8090"
	DefaultLeadLogPath           = "/srv/goalrail/pilot/leads/leads.jsonl"
	DefaultRecipient             = "hello@goalrail.dev"
	DefaultRecipientOverridePath = "/srv/goalrail/pilot/backend/lead-recipient.local"
	DefaultResendAPIKeyPath      = "/srv/goalrail/pilot/backend/resend-api-key.local"
	DefaultResendAPIURL          = "https://api.resend.com/emails"
	DefaultResendFrom            = "GoalRail Pilot <noreply@skill7.dev>"
	DefaultSendmailFrom          = "GoalRail Pilot <noreply@pilot.goalrail.ru>"
	DefaultSendmailEnvelopeFrom  = "noreply@pilot.goalrail.ru"
)

// Config contains the server-local operational settings for the narrow RU pilot
// lead sidecar. Defaults preserve the existing operator-managed paths.
type Config struct {
	ListenAddr            string
	LeadLogPath           string
	RecipientDefault      string
	RecipientOverridePath string
	ResendAPIKeyPath      string
	ResendAPIURL          string
	ResendFrom            string
	SendmailFrom          string
	SendmailEnvelopeFrom  string
	SendmailPath          string
	Now                   func() time.Time
	HTTPClient            *http.Client
}

// DefaultConfig returns production defaults without reading secrets.
func DefaultConfig() Config {
	return Config{
		ListenAddr:            DefaultListenAddr,
		LeadLogPath:           DefaultLeadLogPath,
		RecipientDefault:      DefaultRecipient,
		RecipientOverridePath: DefaultRecipientOverridePath,
		ResendAPIKeyPath:      DefaultResendAPIKeyPath,
		ResendAPIURL:          DefaultResendAPIURL,
		ResendFrom:            DefaultResendFrom,
		SendmailFrom:          DefaultSendmailFrom,
		SendmailEnvelopeFrom:  DefaultSendmailEnvelopeFrom,
		Now:                   time.Now,
		HTTPClient:            &http.Client{Timeout: 10 * time.Second},
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.ListenAddr == "" {
		c.ListenAddr = defaults.ListenAddr
	}
	if c.LeadLogPath == "" {
		c.LeadLogPath = defaults.LeadLogPath
	}
	if c.RecipientDefault == "" {
		c.RecipientDefault = defaults.RecipientDefault
	}
	if c.RecipientOverridePath == "" {
		c.RecipientOverridePath = defaults.RecipientOverridePath
	}
	if c.ResendAPIKeyPath == "" {
		c.ResendAPIKeyPath = defaults.ResendAPIKeyPath
	}
	if c.ResendAPIURL == "" {
		c.ResendAPIURL = defaults.ResendAPIURL
	}
	if c.ResendFrom == "" {
		c.ResendFrom = defaults.ResendFrom
	}
	if c.SendmailFrom == "" {
		c.SendmailFrom = defaults.SendmailFrom
	}
	if c.SendmailEnvelopeFrom == "" {
		c.SendmailEnvelopeFrom = defaults.SendmailEnvelopeFrom
	}
	if c.Now == nil {
		c.Now = defaults.Now
	}
	if c.HTTPClient == nil {
		c.HTTPClient = defaults.HTTPClient
	}
	return c
}
