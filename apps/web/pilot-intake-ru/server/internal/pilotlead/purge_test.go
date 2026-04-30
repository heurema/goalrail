package pilotlead

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPurgeDryRunRemovesNothingAndReportsCounts(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "old@example.com",
		"submitted_date_local": "2026-01-29",
		"notification_status":  StatusNotified,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "recent@example.com",
		"submitted_date_local": "2026-04-01",
		"notification_status":  StatusNotified,
	})

	var output bytes.Buffer
	config := purgeTestConfig(logPath)
	if err := RunPurge(config, NewStore(logPath, config.Now), &output); err != nil {
		t.Fatal(err)
	}

	want := "would_purge retention_days=90 cutoff=2026-01-30 purged=1 kept=1 kept_unknown_date=0\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
	if got := len(readLeadRecords(t, logPath)); got != 2 {
		t.Fatalf("records after dry-run = %d, want 2", got)
	}
}

func TestPurgeConfirmedRemovesOldRowsAndKeepsRecentRows(t *testing.T) {
	t.Setenv("GOALRAIL_PURGE_CONFIRM", "yes")
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "old-date@example.com",
		"submitted_date_local": "2026-01-29",
		"notification_status":  StatusNotified,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":               "old-derived@example.com",
		"submitted_at":        "2026-01-28T21:30:00Z",
		"notification_status": StatusNotified,
	})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "recent@example.com",
		"submitted_date_local": "2026-04-01",
		"notification_status":  StatusNotified,
	})

	var output bytes.Buffer
	config := purgeTestConfig(logPath)
	if err := RunPurge(config, NewStore(logPath, config.Now), &output); err != nil {
		t.Fatal(err)
	}

	want := "purged retention_days=90 cutoff=2026-01-30 purged=2 kept=1 kept_unknown_date=0\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
	records := readLeadRecords(t, logPath)
	if len(records) != 1 {
		t.Fatalf("records after purge = %d, want 1", len(records))
	}
	if got := stringValue(records[0], "email"); got != "recent@example.com" {
		t.Fatalf("remaining email = %q, want recent@example.com", got)
	}
}

func TestPurgeKeepsMalformedAndUnknownDateRows(t *testing.T) {
	t.Setenv("GOALRAIL_PURGE_CONFIRM", "yes")
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	appendRawLine(t, logPath, "{not-json")
	appendRawRecord(t, logPath, LeadRecord{"email": "unknown@example.com"})
	appendRawRecord(t, logPath, LeadRecord{
		"email":                "old@example.com",
		"submitted_date_local": "2026-01-29",
		"notification_status":  StatusNotified,
	})

	var output bytes.Buffer
	config := purgeTestConfig(logPath)
	if err := RunPurge(config, NewStore(logPath, config.Now), &output); err != nil {
		t.Fatal(err)
	}

	want := "purged retention_days=90 cutoff=2026-01-30 purged=1 kept=2 kept_unknown_date=2\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
	lines := readRawLines(t, logPath)
	if len(lines) != 2 {
		t.Fatalf("lines after purge = %d, want 2: %#v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "{not-json") || !strings.Contains(lines[1], "unknown@example.com") {
		t.Fatalf("unexpected kept lines: %#v", lines)
	}
}

func TestInvalidRetentionEnvReturnsError(t *testing.T) {
	for _, value := range []string{"6", "366", "abc"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GOALRAIL_LEAD_RETENTION_DAYS", value)
			logPath := filepath.Join(t.TempDir(), "leads.jsonl")
			var output bytes.Buffer
			config := purgeTestConfig(logPath)
			err := RunPurge(config, NewStore(logPath, config.Now), &output)
			if err == nil {
				t.Fatal("RunPurge returned nil error, want invalid retention error")
			}
			if output.String() != "invalid_retention\n" {
				t.Fatalf("output = %q, want invalid_retention", output.String())
			}
		})
	}
}

func purgeTestConfig(logPath string) Config {
	config := DefaultConfig()
	config.LeadLogPath = logPath
	config.Now = func() time.Time {
		return time.Date(2026, 4, 30, 6, 0, 0, 0, time.UTC)
	}
	return config
}

func appendRawLine(t *testing.T, path, line string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o770); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}

func readRawLines(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	trimmed := strings.TrimSuffix(string(content), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
