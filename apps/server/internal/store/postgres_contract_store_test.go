package store

import (
	"context"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresContractStoreListBuildsScopedFilteredQuery(t *testing.T) {
	ctx := context.Background()
	rows := &recordingProjectContextRowsQuerier{
		rows: &fakeProjectContextRows{
			rows: []fakeProjectContextRow{
				{values: validContractRowValues()},
			},
		},
	}
	store := NewPostgresContractStoreWithExecutorQuerierAndRows(&recordingProjectContextExecer{}, nil, rows)

	contracts, err := store.List(ctx, spine.ContractListFilter{
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		GoalID:         "018f0000-0000-7000-8000-000000000201",
		State:          spine.ContractStateDraft,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("contracts len = %d, want 1", len(contracts))
	}
	if contracts[0].ID != "018f0000-0000-7000-8000-000000000301" {
		t.Fatalf("contract id = %q, want persisted contract", contracts[0].ID)
	}

	call := rows.calls[0]
	for _, want := range []string{
		"FROM contracts",
		"organization_id =",
		"project_id =",
		"repo_binding_id =",
		"goal_id =",
		"state =",
		"ORDER BY updated_at DESC, created_at DESC, id DESC",
		"LIMIT 50",
	} {
		if !strings.Contains(call.sql, want) {
			t.Fatalf("SQL = %q, want %q", call.sql, want)
		}
	}
	if got, want := len(call.args), 5; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func validContractRowValues() []any {
	now := testStoreTime()
	return []any{
		"018f0000-0000-7000-8000-000000000301",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000201",
		string(spine.ContractStateDraft),
		"018f0000-0000-7000-8000-000000000401",
		"018f0000-0000-7000-8000-000000000501",
		nil,
		now,
		now,
	}
}
