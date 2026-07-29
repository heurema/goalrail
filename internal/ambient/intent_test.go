package ambient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func intentArtifact(id, status string, version string) string {
	confirmation := `- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending`
	if status == "confirmed" {
		confirmation = `- **Confirmed by:** owner
- **Confirmed at:** 2026-07-29
- **Verification action:** owner-reviewed-three-groups`
	}
	predecessor := ""
	if version != "1" {
		predecessor = "\n- **Previous version:** 1"
	}
	return `# Intent Snapshot

- **Intent ID:** ` + id + `
- **Version:** ` + version + predecessor + `
- **Status:** ` + status + `
- **Owner:** owner

## Source Evidence

- **SE-1 — Owner statement:** The owner asked for a bounded result.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded result. | Inspect it. | SE-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local artifact exists. | SE-1 |

## Ambiguities and Unknowns

None.

## Confirmation

` + confirmation + `
`
}

func writeChange(t *testing.T, root, change, id, status, version string) {
	t.Helper()
	directory := filepath.Join(root, "openspec", "changes", change)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "intent.md"),
		[]byte(intentArtifact(id, status, version)),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
}

func TestBindsExactlyOneConfirmedIntent(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "current-change", "INTENT-CURRENT", "confirmed", "2")

	reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if reference == nil {
		t.Fatalf("no binding, reason %q", reason)
	}
	if reference.ID != "INTENT-CURRENT" || reference.Version != 2 {
		t.Fatalf("reference = %+v", reference)
	}
	if !strings.HasPrefix(reference.Digest, "sha256:") || reference.Change != "current-change" {
		t.Fatalf("reference lacks resolvable identity: %+v", reference)
	}
}

func TestArchivedChangesDoNotBind(t *testing.T) {
	// An archived change is finished work; binding a live question to it
	// would point the chain at a decision already made.
	root := t.TempDir()
	directory := filepath.Join(root, "openspec", "changes", "archive", "2026-07-01-old")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "intent.md"),
		[]byte(intentArtifact("INTENT-OLD", "confirmed", "1")),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if reference != nil {
		t.Fatalf("an archived change bound the question: %+v", reference)
	}
	if reason != reasonNoChanges {
		t.Fatalf("reason = %q", reason)
	}
}

func TestUnconfirmedAndAmbiguousCasesReportWhy(t *testing.T) {
	t.Run("candidate only", func(t *testing.T) {
		root := t.TempDir()
		writeChange(t, root, "current-change", "INTENT-CURRENT", "candidate", "1")
		reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(root)
		if reference != nil || reason != reasonNoConfirmed {
			t.Fatalf("reference = %+v reason = %q", reference, reason)
		}
	})

	t.Run("two confirmed", func(t *testing.T) {
		root := t.TempDir()
		writeChange(t, root, "change-one", "INTENT-ONE", "confirmed", "1")
		writeChange(t, root, "change-two", "INTENT-TWO", "confirmed", "1")
		reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(root)
		if reference != nil || reason != reasonAmbiguous {
			t.Fatalf("reference = %+v reason = %q", reference, reason)
		}
	})

	t.Run("no openspec at all", func(t *testing.T) {
		reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(t.TempDir())
		if reference != nil || reason != reasonNoChanges {
			t.Fatalf("reference = %+v reason = %q", reference, reason)
		}
	})
}

func TestAMalformedIntentDoesNotBlockAValidOne(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "good-change", "INTENT-GOOD", "confirmed", "1")
	broken := filepath.Join(root, "openspec", "changes", "broken-change")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "intent.md"), []byte("# not an intent"), 0o644); err != nil {
		t.Fatal(err)
	}

	reference, reason := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if reference == nil || reference.ID != "INTENT-GOOD" {
		t.Fatalf("reference = %+v reason = %q", reference, reason)
	}
}
