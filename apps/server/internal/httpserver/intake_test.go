package httpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

const validIntakeJSON = `{
  "repo_binding_id": "repo_demo_1",
  "source": {
    "kind": "codex_skill",
    "external_id": "local-session-1"
  },
  "title": "Refactor CSV export filters",
  "body": "Current code duplicates filter logic. Preserve current behavior.",
  "request_author": {
    "kind": "user",
    "id": "dev_1",
    "display_name": "Developer"
  }
}`

type testServerDeps struct {
	router http.Handler
	store  *store.IntakeStore
	events *eventlog.EventLog
}

func TestPostIntakeReturnsAccepted(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.code, http.StatusAccepted)
	}

	var body map[string]json.RawMessage
	decodeJSON(t, response.body, &body)

	var canonicalContractCreated bool
	decodeRawJSON(t, body["canonical_contract_created"], &canonicalContractCreated)
	if canonicalContractCreated {
		t.Fatal("canonical_contract_created = true, want false")
	}

	for _, forbiddenField := range []string{"goal_id", "contract_id", "work_item_id"} {
		if _, ok := body[forbiddenField]; ok {
			t.Fatalf("response includes forbidden field %q", forbiddenField)
		}
	}

	var state string
	decodeRawJSON(t, body["state"], &state)
	if state != "received" {
		t.Fatalf("state = %q, want %q", state, "received")
	}
}

func TestGetIntakeReturnsStoredRecord(t *testing.T) {
	server := testServer(t)

	postResponse := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if postResponse.code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", postResponse.code, http.StatusAccepted)
	}

	var accepted struct {
		IntakeID string `json:"intake_id"`
	}
	decodeJSON(t, postResponse.body, &accepted)

	getResponse := doJSON(t, server.router, http.MethodGet, "/v1/intake/"+accepted.IntakeID, "")
	if getResponse.code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResponse.code, http.StatusOK)
	}

	var record spine.IntakeRecord
	decodeJSON(t, getResponse.body, &record)

	if record.ID != spine.IntakeID(accepted.IntakeID) {
		t.Fatalf("record ID = %q, want %q", record.ID, accepted.IntakeID)
	}
	if record.State != spine.IntakeStateReceived {
		t.Fatalf("state = %q, want %q", record.State, spine.IntakeStateReceived)
	}
	if record.CanonicalContractCreated {
		t.Fatal("CanonicalContractCreated = true, want false")
	}
	if record.RepoBindingID != "repo_demo_1" {
		t.Fatalf("RepoBindingID = %q, want %q", record.RepoBindingID, "repo_demo_1")
	}
	if !reflect.DeepEqual(record.IntentOwner, record.RequestAuthor) {
		t.Fatalf("IntentOwner = %#v, want RequestAuthor %#v", record.IntentOwner, record.RequestAuthor)
	}
}

func TestGetUnknownIntakeReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodGet, "/v1/intake/missing", "")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostIntakeValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing repo_binding_id",
			body: `{"source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing source kind",
			body: `{"repo_binding_id":"repo_demo_1","source":{},"title":"Title","request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing title and body",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing request_author kind",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"id":"dev_1"}}`,
		},
		{
			name: "missing request_author id",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)

			response := doJSON(t, server.router, http.MethodPost, "/v1/intake", tt.body)
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
			}

			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response.body, &body)
			if body.Error.Code != "validation_failed" {
				t.Fatalf("error code = %q, want %q", body.Error.Code, "validation_failed")
			}
		})
	}
}

func TestPostIntakeRejectsUnknownJSONField(t *testing.T) {
	server := testServer(t)
	body := `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"dev_1"},"unexpected":true}`

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", body)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &responseBody)
	if responseBody.Error.Code != "invalid_json" {
		t.Fatalf("error code = %q, want %q", responseBody.Error.Code, "invalid_json")
	}
}

func TestPostIntakeDefaultsIntentOwnerToRequestAuthor(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.code, http.StatusAccepted)
	}

	record, ok, err := server.store.Get(context.Background(), "intake-1")
	if err != nil {
		t.Fatalf("store.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored intake not found")
	}
	if !reflect.DeepEqual(record.IntentOwner, record.RequestAuthor) {
		t.Fatalf("IntentOwner = %#v, want RequestAuthor %#v", record.IntentOwner, record.RequestAuthor)
	}
}

func testServer(t *testing.T) testServerDeps {
	t.Helper()

	intakeStore := store.NewIntakeStore()
	events := eventlog.NewEventLog()
	service := intake.NewService(intakeStore, events, fixedClock{now: testTime()}, &sequenceIDs{})
	intakeHandler := httpserver.NewIntakeHandler(service)

	return testServerDeps{
		router: baseHandlers(intakeHandler),
		store:  intakeStore,
		events: events,
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	intake int
	event  int
}

func (g *sequenceIDs) NewIntakeID() (spine.IntakeID, error) {
	g.intake++
	return spine.IntakeID(fmt.Sprintf("intake-%d", g.intake)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func decodeRawJSON(t *testing.T, input json.RawMessage, target any) {
	t.Helper()

	if len(input) == 0 {
		t.Fatal("missing JSON field")
	}
	if err := json.Unmarshal(input, target); err != nil {
		t.Fatalf("decode raw JSON %q: %v", string(input), err)
	}
}
