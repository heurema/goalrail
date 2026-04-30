package pilotlead

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeMailer struct {
	err       error
	transport string
	calls     int
	lastTo    string
	lastBody  string
}

func (m *fakeMailer) SendText(_ context.Context, to, _ string, text, _ string) (string, error) {
	m.calls++
	m.lastTo = to
	m.lastBody = text
	if m.err != nil {
		return "", m.err
	}
	if m.transport != "" {
		return m.transport, nil
	}
	return "sendmail", nil
}

func TestHandlerNewLeadMailSuccessStoresNotified(t *testing.T) {
	server, mailer, logPath := testServer(t, nil)
	response := postLead(t, server, `{"email":"lead@example.com","source":"test","page":"pilot.goalrail.ru"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	assertJSONBool(t, response.Body.Bytes(), "duplicate", false)
	if mailer.calls != 1 {
		t.Fatalf("mailer calls = %d, want 1", mailer.calls)
	}

	records := readLeadRecords(t, logPath)
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	if got := stringValue(records[0], "notification_status"); got != StatusNotified {
		t.Fatalf("notification_status = %q, want %q", got, StatusNotified)
	}
	if got := stringValue(records[0], "notification_transport"); got != "sendmail" {
		t.Fatalf("notification_transport = %q, want sendmail", got)
	}
	if got := stringValue(records[0], "notification_error"); got != "" {
		t.Fatalf("notification_error = %q, want empty", got)
	}
	if _, ok := records[0]["user_agent"]; ok {
		t.Fatalf("new lead record contains user_agent: %#v", records[0]["user_agent"])
	}
}

func TestHandlerNewLeadMailFailureStoresGenericFailure(t *testing.T) {
	server, mailer, logPath := testServer(t, errors.New("transport down"))
	response := postLead(t, server, `{"email":"lead@example.com"}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	assertJSONError(t, response.Body.Bytes(), ErrorMailUnavailable)
	if mailer.calls != 1 {
		t.Fatalf("mailer calls = %d, want 1", mailer.calls)
	}

	records := readLeadRecords(t, logPath)
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	if got := stringValue(records[0], "notification_status"); got != StatusNotificationFailed {
		t.Fatalf("notification_status = %q, want %q", got, StatusNotificationFailed)
	}
	if got := stringValue(records[0], "notification_error"); got != ErrorMailUnavailable {
		t.Fatalf("notification_error = %q, want %q", got, ErrorMailUnavailable)
	}
}

func TestRetryAfterNotificationFailedCanSucceed(t *testing.T) {
	server, mailer, logPath := testServer(t, errors.New("transport down"))
	failed := postLead(t, server, `{"email":"lead@example.com"}`)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("first status = %d", failed.Code)
	}

	mailer.err = nil
	success := postLead(t, server, `{"email":"lead@example.com"}`)
	if success.Code != http.StatusOK {
		t.Fatalf("retry status = %d, body = %s", success.Code, success.Body.String())
	}
	assertJSONBool(t, success.Body.Bytes(), "duplicate", false)
	if mailer.calls != 2 {
		t.Fatalf("mailer calls = %d, want 2", mailer.calls)
	}

	duplicate := postLead(t, server, `{"email":"lead@example.com"}`)
	if duplicate.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d", duplicate.Code)
	}
	assertJSONBool(t, duplicate.Body.Bytes(), "duplicate", true)
	if mailer.calls != 2 {
		t.Fatalf("mailer calls after duplicate = %d, want 2", mailer.calls)
	}

	records := readLeadRecords(t, logPath)
	if len(records) != 1 {
		t.Fatalf("records = %d, want in-place single row", len(records))
	}
	if got := stringValue(records[0], "notification_status"); got != StatusNotified {
		t.Fatalf("notification_status = %q, want %q", got, StatusNotified)
	}
}

func TestLegacyRowWithoutStatusIsDuplicate(t *testing.T) {
	server, mailer, logPath := testServer(t, nil)
	appendRawRecord(t, logPath, LeadRecord{"email": "lead@example.com", "submitted_date_local": "2026-04-30"})

	response := postLead(t, server, `{"email":"lead@example.com"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	assertJSONBool(t, response.Body.Bytes(), "duplicate", true)
	if mailer.calls != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.calls)
	}
}

func TestReceivedRowIsInflightDuplicate(t *testing.T) {
	server, mailer, logPath := testServer(t, nil)
	appendRawRecord(t, logPath, LeadRecord{"email": "lead@example.com", "notification_status": StatusReceived, "submitted_date_local": "2026-04-30"})

	response := postLead(t, server, `{"email":"lead@example.com"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	assertJSONBool(t, response.Body.Bytes(), "duplicate", true)
	if mailer.calls != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.calls)
	}
}

func TestUnknownStatusIsConservativeDuplicate(t *testing.T) {
	server, mailer, logPath := testServer(t, nil)
	appendRawRecord(t, logPath, LeadRecord{"email": "lead@example.com", "notification_status": "mystery", "submitted_date_local": "2026-04-30"})

	response := postLead(t, server, `{"email":"lead@example.com"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	assertJSONBool(t, response.Body.Bytes(), "duplicate", true)
	if mailer.calls != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.calls)
	}
}

func testServer(t *testing.T, mailErr error) (*Server, *fakeMailer, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "leads.jsonl")
	now := time.Date(2026, 4, 30, 9, 10, 11, 0, time.UTC)
	config := DefaultConfig()
	config.LeadLogPath = logPath
	config.RecipientOverridePath = filepath.Join(t.TempDir(), "missing-recipient.local")
	config.Now = func() time.Time { return now }
	mailer := &fakeMailer{err: mailErr}
	server := NewServer(config, NewStore(logPath, config.Now), mailer)
	return server, mailer, logPath
}

func postLead(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/pilot-lead", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "unit-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func readLeadRecords(t *testing.T, path string) []LeadRecord {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var records []LeadRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record LeadRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}

func appendRawRecord(t *testing.T, path string, record LeadRecord) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o770); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		t.Fatal(err)
	}
}

func assertJSONError(t *testing.T, body []byte, want string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if got := payload["error"]; got != want {
		t.Fatalf("error = %v, want %q", got, want)
	}
}

func assertJSONBool(t *testing.T, body []byte, key string, want bool) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(body), &payload); err != nil {
		t.Fatal(err)
	}
	if got := payload[key]; got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}
