package migrations

import (
	"strings"
	"testing"
)

func TestInitMigrationAllowsContractDraftReadyForApprovalState(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	if !strings.Contains(sql, "CONSTRAINT contract_drafts_state_check CHECK (state IN ('draft', 'ready_for_approval'))") {
		t.Fatalf("contract_drafts_state_check does not allow draft and ready_for_approval states")
	}
}
