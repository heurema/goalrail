package store

import (
	"context"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresQualificationFeedStoreListBuildsScopedReadModelQuery(t *testing.T) {
	ctx := context.Background()
	rows := &recordingProjectContextRowsQuerier{
		rows: &fakeProjectContextRows{
			rows: []fakeProjectContextRow{
				{values: validQualificationFeedRowValues()},
			},
		},
	}
	store := NewPostgresQualificationFeedStoreWithRowsQuerier(rows)

	records, err := store.List(ctx, spine.QualificationFeedFilter{
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		GoalState:      spine.GoalStateNeedsClarification,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	record := records[0]
	if record.IntakeID != "018f0000-0000-7000-8000-000000000101" || record.GoalID != "018f0000-0000-7000-8000-000000000201" {
		t.Fatalf("record ids = %#v, want intake and goal ids", record)
	}
	if record.OpenClarificationRequest == nil || len(record.OpenClarificationRequest.Questions) != 1 {
		t.Fatalf("open clarification = %#v, want one question", record.OpenClarificationRequest)
	}
	if record.LinkedContract == nil || record.LinkedContract.State != spine.ContractStateDraft {
		t.Fatalf("linked contract = %#v, want draft contract", record.LinkedContract)
	}
	call := rows.calls[0]
	for _, want := range []string{
		"FROM goals g",
		"JOIN intake_records ir ON ir.id = g.intake_id",
		"JOIN repo_bindings rb ON rb.id = g.repo_binding_id",
		"LEFT JOIN clarification_requests cr ON cr.goal_id = g.id AND cr.state = 'open'",
		"LEFT JOIN contracts c ON c.goal_id = g.id",
		"g.organization_id =",
		"ir.organization_id =",
		"rb.organization_id =",
		"g.project_id =",
		"g.repo_binding_id =",
		"g.state =",
		"ORDER BY ir.created_at DESC",
		"LIMIT 50",
	} {
		if !strings.Contains(call.sql, want) {
			t.Fatalf("SQL = %q, want %q", call.sql, want)
		}
	}
	if got, want := len(call.args), 6; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func validQualificationFeedRowValues() []any {
	now := testStoreTime()
	return []any{
		"018f0000-0000-7000-8000-000000000101",
		"018f0000-0000-7000-8000-000000000201",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"heurema/goalrail",
		"Improve billing error handling",
		string(spine.IntakeStateReceived),
		string(spine.GoalStateNeedsClarification),
		[]byte(`["missing_scope_hint"]`),
		"018f0000-0000-7000-8000-000000000a01",
		string(spine.ClarificationRequestStateOpen),
		[]byte(`[{"id":"018f0000-0000-7000-8000-000000000a11","text":"What is the intended scope at a high level?","why_needed":"A scope hint is required before contract seed readiness.","answer_type":"text","maps_to":"goal.scope_hint"}]`),
		"018f0000-0000-7000-8000-000000000c01",
		string(spine.ContractStateDraft),
		now,
	}
}
