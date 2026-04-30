package pilotlead

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDigestIncludesCorrectLocalDateRows(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "first@example.com",
		"source":               "ru-pilot",
		"page":                 "pilot.goalrail.ru",
		"submitted_at":         "2026-04-29T20:30:00Z",
		"submitted_at_local":   "2026-04-29T23:30:00+03:00",
		"submitted_date_local": "2026-04-29",
		"notification_status":  StatusNotified,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "failed@example.com",
		"source":               "ru-pilot",
		"page":                 "pilot.goalrail.ru",
		"submitted_at":         "2026-04-29T21:30:00Z",
		"submitted_at_local":   "2026-04-30T00:30:00+03:00",
		"submitted_date_local": "2026-04-29",
		"notification_status":  StatusNotificationFailed,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "wrong-day@example.com",
		"source":               "ru-pilot",
		"page":                 "pilot.goalrail.ru",
		"submitted_at":         "2026-04-30T01:00:00Z",
		"submitted_at_local":   "2026-04-30T04:00:00+03:00",
		"submitted_date_local": "2026-04-30",
		"notification_status":  StatusNotificationFailed,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "first@example.com",
		"source":               "retry",
		"page":                 "pilot.goalrail.ru",
		"submitted_at":         "2026-04-29T21:00:00Z",
		"submitted_at_local":   "2026-04-30T00:00:00+03:00",
		"submitted_date_local": "2026-04-29",
		"notification_status":  StatusNotified,
	})

	config := digestTestConfig(logPath)
	mailer := &fakeMailer{}
	var output bytes.Buffer
	if err := RunDigest(context.Background(), config, NewStore(logPath, config.Now), mailer, &output); err != nil {
		t.Fatal(err)
	}
	if mailer.calls != 1 {
		t.Fatalf("mailer calls = %d, want 1", mailer.calls)
	}
	if !strings.Contains(output.String(), "sent date=2026-04-29 count=2 transport=sendmail") {
		t.Fatalf("output = %q", output.String())
	}
	if strings.Contains(mailer.lastBody, "wrong-day@example.com") {
		t.Fatalf("digest body included wrong date row: %s", mailer.lastBody)
	}
	if !strings.Contains(mailer.lastBody, "first@example.com") || !strings.Contains(mailer.lastBody, "Source: retry") {
		t.Fatalf("digest body missed deduped target row: %s", mailer.lastBody)
	}
	if !strings.Contains(mailer.lastBody, "failed@example.com") {
		t.Fatalf("digest body missed failed target row: %s", mailer.lastBody)
	}
}

func TestDryRunDigestSendsNothing(t *testing.T) {
	t.Setenv("GOALRAIL_DIGEST_DRY_RUN", "yes")
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "lead@example.com",
		"source":               "ru-pilot",
		"page":                 "pilot.goalrail.ru",
		"submitted_at":         "2026-04-29T20:30:00Z",
		"submitted_at_local":   "2026-04-29T23:30:00+03:00",
		"submitted_date_local": "2026-04-29",
		"notification_status":  StatusNotificationFailed,
	})

	config := digestTestConfig(logPath)
	mailer := &fakeMailer{}
	var output bytes.Buffer
	if err := RunDigest(context.Background(), config, NewStore(logPath, config.Now), mailer, &output); err != nil {
		t.Fatal(err)
	}
	if mailer.calls != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.calls)
	}
	if !strings.Contains(output.String(), "would_send date=2026-04-29 count=1") {
		t.Fatalf("output = %q", output.String())
	}
}

func digestTestConfig(logPath string) Config {
	config := DefaultConfig()
	config.LeadLogPath = logPath
	config.RecipientOverridePath = filepath.Join(filepath.Dir(logPath), "missing-recipient.local")
	config.Now = func() time.Time {
		return time.Date(2026, 4, 30, 6, 0, 0, 0, time.UTC)
	}
	return config
}
