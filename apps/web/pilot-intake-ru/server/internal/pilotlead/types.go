package pilotlead

import "strings"

const (
	MaxBodyBytes = 8192

	LeadSubject        = "Пилот — заявка с RU лендинга"
	DigestSubjectStart = "Пилот"
	DigestTZ           = "Europe/Moscow"

	StatusReceived           = "received"
	StatusNotified           = "notified"
	StatusNotificationFailed = "notification_failed"
	StatusPending            = "pending"

	ErrorMailUnavailable = "mail_unavailable"
)

type LeadRecord map[string]any

type leadEntry struct {
	raw    string
	record LeadRecord
}

func stringValue(record LeadRecord, key string) string {
	value, ok := record[key].(string)
	if !ok {
		return ""
	}
	return value
}

func normalizedEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
