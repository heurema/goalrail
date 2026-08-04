package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
)

func TestAdoptionAdvisoryFollowsOnlyTheRulesDigest(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, filepath.FromSlash(ConfigPath))
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "schema: goalrail-intent\ncontext: one\nrules:\n  intent:\n    - ask once\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := ReadRules(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 4, 14, 0, 0, 0, time.UTC)
	if _, _, err := ambient.InitializeWithAdoption(root, func() time.Time { return now }, &ambient.Adoption{
		ReplacedSchema: "intent-driven",
		RulesDigest:    rules.Digest,
		HadRules:       true,
	}); err != nil {
		t.Fatal(err)
	}

	advisory := adoptionAdvisory(root)
	if advisory == nil || !strings.Contains(advisory.Note, "intent-driven") || strings.Contains(advisory.Note, "ask once") {
		t.Fatalf("advisory = %#v", advisory)
	}

	unrelated := strings.Replace(config, "context: one", "context: two", 1)
	if err := os.WriteFile(configPath, []byte(unrelated), 0o644); err != nil {
		t.Fatal(err)
	}
	if adoptionAdvisory(root) == nil {
		t.Fatal("an unrelated edit removed the advisory")
	}

	editedRules := strings.Replace(unrelated, "ask once", "ask twice", 1)
	if err := os.WriteFile(configPath, []byte(editedRules), 0o644); err != nil {
		t.Fatal(err)
	}
	if adoptionAdvisory(root) != nil {
		t.Fatal("a rules edit left the advisory standing")
	}
}

func TestAdoptionAdvisoryIsAbsentForLegacyOrRulelessMarkers(t *testing.T) {
	root := t.TempDir()
	if _, _, err := ambient.Initialize(root, time.Now); err != nil {
		t.Fatal(err)
	}
	if adoptionAdvisory(root) != nil {
		t.Fatal("a legacy marker produced an adoption advisory")
	}

	ruleless := t.TempDir()
	configPath := filepath.Join(ruleless, filepath.FromSlash(ConfigPath))
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("schema: goalrail-intent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ambient.InitializeWithAdoption(ruleless, time.Now, &ambient.Adoption{
		ReplacedSchema: "intent-driven",
		RulesDigest:    digestBytes(nil),
		HadRules:       false,
	}); err != nil {
		t.Fatal(err)
	}
	if adoptionAdvisory(ruleless) != nil {
		t.Fatal("a ruleless adoption produced an advisory")
	}
}

func TestAdoptionAdvisoryChangesNoVerdict(t *testing.T) {
	base := Diagnosis{Initialized: true, Overlay: OverlayState{Present: true, Complete: true}, SchemaPresent: true, Schema: SchemaName}
	with := base
	with.Adoption = &AdoptionAdvisory{Note: "fact only"}
	baseProblems, baseActions := summarize(base)
	withProblems, withActions := summarize(with)
	if strings.Join(baseProblems, "\n") != strings.Join(withProblems, "\n") ||
		strings.Join(baseActions, "\n") != strings.Join(withActions, "\n") {
		t.Fatal("the advisory changed the diagnosis verdict inputs")
	}
}
