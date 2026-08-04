package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/harness"
)

const replacedSchemaFixture = `name: intent-driven
artifacts:
  - id: intent
    instruction: |
      Capture human intent.
    requires: []
  - id: proposal
    instruction: >
      Propose the confirmed work.
    requires:
      - intent
  - id: specs
    instruction: >
      Specify the behavior.
    requires:
      - proposal
  - id: design
    instruction: >
      Design the implementation.
    requires:
      - proposal
  - id: tasks
    instruction: >
      Break down the implementation.
    requires:
      - specs
      - design
`

func TestRunInitReportsAndRecordsSchemaAdoption(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	original := "schema: intent-driven\ncontext: keep exactly\nrules:\n  intent:\n    - ask exactly one question\n    - keep evidence distinct\n"
	writeAdoptionFixture(t, root, "openspec/config.yaml", original)
	writeAdoptionFixture(t, root, "openspec/schemas/intent-driven/schema.yaml", replacedSchemaFixture)
	writeAdoptionFixture(t, root, "openspec/changes/current/.openspec.yaml", "schema: intent-driven # active\n")
	writeAdoptionFixture(t, root, "openspec/changes/archive/finished/.openspec.yaml", "schema: 'intent-driven' # archived\n")

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root, "--confirm-schema-switch"}, &stdout, &stderr); err != nil {
		t.Fatalf("run init: %v\nstderr: %s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Adoption == nil {
		t.Fatal("adoption section is absent")
	}
	adoption := report.Adoption
	if adoption.ReplacedSchema != "intent-driven" || adoption.AdoptedAt.IsZero() {
		t.Fatalf("adoption identity = %#v", adoption)
	}
	if !adoption.SchemaDifference.Comparable {
		t.Fatalf("schema comparison = %#v", adoption.SchemaDifference)
	}
	if !reflect.DeepEqual(adoption.SchemaDifference.OnlyInAdopted, []string{"context"}) {
		t.Fatalf("added artifacts = %#v", adoption.SchemaDifference.OnlyInAdopted)
	}
	if !containsString(adoption.SchemaDifference.DependenciesChanged, "design") {
		t.Fatalf("dependency changes = %#v", adoption.SchemaDifference.DependenciesChanged)
	}
	if !adoption.Rules.Counted || adoption.Rules.Count != 2 || !strings.Contains(adoption.Rules.Text, "ask exactly one question") {
		t.Fatalf("rules = %#v", adoption.Rules)
	}
	if adoption.RulesDisclosure != rulesDisclosure || strings.Contains(strings.ToLower(adoption.RulesDisclosure), "stale") {
		t.Fatalf("rules disclosure = %q", adoption.RulesDisclosure)
	}
	encodedAdoption, err := json.Marshal(adoption)
	if err != nil {
		t.Fatal(err)
	}
	for _, verdict := range []string{"\"verdict\"", "valid rule", "invalid rule", "stale rule", "compatible rule", "incompatible rule"} {
		if strings.Contains(strings.ToLower(string(encodedAdoption)), verdict) {
			t.Fatalf("adoption section rendered a per-rule verdict %q: %s", verdict, encodedAdoption)
		}
	}
	if adoption.Pins == nil || adoption.Pins.Active != 1 || adoption.Pins.Archived != 1 ||
		!strings.Contains(adoption.SchemaDirectory, "must remain") {
		t.Fatalf("pins = %#v, directory = %q", adoption.Pins, adoption.SchemaDirectory)
	}

	wantConfig := strings.Replace(original, "schema: intent-driven", "schema: "+harness.SchemaName, 1)
	gotConfig, err := os.ReadFile(filepath.Join(root, "openspec", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotConfig) != wantConfig {
		t.Fatalf("configuration changed beyond the schema line:\n%s", gotConfig)
	}
	marker, err := ambient.ReadMarker(root)
	if err != nil {
		t.Fatal(err)
	}
	if marker.Adoption == nil || marker.Adoption.ReplacedSchema != "intent-driven" ||
		marker.Adoption.RulesDigest != adoption.Rules.Digest || !marker.Adoption.HadRules {
		t.Fatalf("marker adoption = %#v", marker.Adoption)
	}
}

func TestRunInitKeepsAdoptionDiagnosticsFailOpen(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeAdoptionFixture(t, root, "openspec/config.yaml", "schema: spec-driven\nrules: {intent: [\"a\", \"b\"]}\n")
	writeAdoptionFixture(t, root, "openspec/changes/broken/.openspec.yaml", "not-schema: x\n")

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root}, &stdout, &stderr); err != nil {
		t.Fatalf("reporting failure changed init status: %v\nstderr: %s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Adoption == nil {
		t.Fatal("stock adoption section is absent")
	}
	if report.Adoption.SchemaDifference.Comparable || !strings.Contains(report.Adoption.SchemaDifference.Reason, "not a file") {
		t.Fatalf("stock schema comparison = %#v", report.Adoption.SchemaDifference)
	}
	if report.Adoption.Rules.Counted || report.Adoption.Rules.Text == "" {
		t.Fatalf("flow rules were not disclosed as uncountable: %#v", report.Adoption.Rules)
	}
	if report.Adoption.Pins != nil || len(report.Adoption.Notices) < 3 {
		t.Fatalf("fail-open notices = %#v, pins = %#v", report.Adoption.Notices, report.Adoption.Pins)
	}
	if !strings.Contains(strings.Join(report.Adoption.Notices, "\n"), "schema difference") {
		t.Fatalf("schema comparison failure was not disclosed as a notice: %#v", report.Adoption.Notices)
	}
	if !strings.Contains(report.Adoption.SchemaDirectory, "no repository-local schema directory") ||
		strings.Contains(report.Adoption.SchemaDirectory, "may be removed") {
		t.Fatalf("stock schema received removal advice: %q", report.Adoption.SchemaDirectory)
	}
}

func TestRunInitKeepsExistingMarkerWhereAdoptionEvidenceCannotBeWritten(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce the marker mode used to force this write failure")
	}
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeAdoptionFixture(t, root, "openspec/config.yaml", "schema: intent-driven\nrules:\n  intent:\n    - keep evidence distinct\n")
	writeAdoptionFixture(t, root, "openspec/schemas/intent-driven/schema.yaml", replacedSchemaFixture)
	writeAdoptionFixture(t, root, ambient.MarkerPath,
		"{\n  \"schema\": \"goalrail.ambient-marker/v0\",\n  \"initialized_at\": \"2026-08-01T00:00:00Z\"\n}\n")
	markerPath := filepath.Join(root, filepath.FromSlash(ambient.MarkerPath))
	markerDirectory := filepath.Dir(markerPath)
	if err := os.Chmod(markerDirectory, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(markerDirectory, 0o700) })

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root, "--confirm-schema-switch"}, &stdout, &stderr); err != nil {
		t.Fatalf("adoption evidence failure changed init status: %v\nstderr: %s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Adoption == nil || report.Adoption.AdoptedAt.IsZero() ||
		!strings.Contains(strings.Join(report.Adoption.Notices, "\n"), "adoption record was not written") {
		t.Fatalf("degraded adoption report = %#v", report.Adoption)
	}
	marker, err := ambient.ReadMarker(root)
	if err != nil || marker.Adoption != nil {
		t.Fatalf("existing marker was not preserved: %#v, err = %v", marker, err)
	}
}

func TestRunInitDoesNotClaimAbsentRulesWerePresent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeAdoptionFixture(t, root, "openspec/config.yaml", "schema: intent-driven\ncontext: retained\n")
	writeAdoptionFixture(t, root, "openspec/schemas/intent-driven/schema.yaml", replacedSchemaFixture)

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root, "--confirm-schema-switch"}, &stdout, &stderr); err != nil {
		t.Fatalf("run init: %v\n%s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Adoption == nil || report.Adoption.Rules.HasRules || report.Adoption.RulesDisclosure != noRulesDisclosure {
		t.Fatalf("ruleless adoption = %#v", report.Adoption)
	}
	if !strings.Contains(report.Adoption.SchemaDirectory, "may be removed") {
		t.Fatalf("repository-local unpinned schema directory = %q", report.Adoption.SchemaDirectory)
	}
	marker, err := ambient.ReadMarker(root)
	if err != nil {
		t.Fatal(err)
	}
	if marker.Adoption == nil || marker.Adoption.HadRules {
		t.Fatalf("ruleless marker = %#v", marker.Adoption)
	}
}

func TestRunInitKeepsUncountableRulesAsUnreviewedEvidence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	rules := "rules:\n  intent:\n    [\n      \"ask exactly one question\"\n    ]\n"
	writeAdoptionFixture(t, root, "openspec/config.yaml", "schema: intent-driven\n"+rules)
	writeAdoptionFixture(t, root, "openspec/schemas/intent-driven/schema.yaml", replacedSchemaFixture)

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root, "--confirm-schema-switch"}, &stdout, &stderr); err != nil {
		t.Fatalf("run init: %v\n%s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Adoption == nil || report.Adoption.Rules.Counted || !report.Adoption.Rules.HasRules ||
		report.Adoption.Rules.Text != rules || report.Adoption.Rules.Digest == "" ||
		report.Adoption.RulesDisclosure != uncountableRulesDisclosure {
		t.Fatalf("uncountable adoption = %#v", report.Adoption)
	}
	marker, err := ambient.ReadMarker(root)
	if err != nil {
		t.Fatal(err)
	}
	if marker.Adoption == nil || !marker.Adoption.HadRules || marker.Adoption.RulesDigest == "" {
		t.Fatalf("uncountable marker = %#v", marker.Adoption)
	}
	if adoption := harnessAdoptionFromDiagnosis(t, root); adoption == nil {
		t.Fatal("uncountable rules produced no standing adoption advisory")
	}
}

func TestRunInitReproducesEachDuplicateRulesBlock(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	first := "rules:\n  intent:\n    - first rule\n"
	second := "rules:\n  proposal:\n    - second rule\n"
	config := "schema: intent-driven\n" + first + "context: retained\n" + second
	writeAdoptionFixture(t, root, "openspec/config.yaml", config)
	writeAdoptionFixture(t, root, "openspec/schemas/intent-driven/schema.yaml", replacedSchemaFixture)

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--repo", root, "--confirm-schema-switch"}, &stdout, &stderr); err != nil {
		t.Fatalf("run init: %v\n%s", err, stderr.String())
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Adoption == nil || report.Adoption.Rules.Counted || !report.Adoption.Rules.HasRules ||
		report.Adoption.Rules.Text != first+second || report.Adoption.Rules.Digest == "" ||
		report.Adoption.RulesDisclosure != uncountableRulesDisclosure {
		t.Fatalf("duplicate rules adoption = %#v", report.Adoption)
	}
	if strings.Contains(report.Adoption.Rules.Text, "context: retained") {
		t.Fatalf("unrelated configuration entered the rules span: %q", report.Adoption.Rules.Text)
	}
	marker, err := ambient.ReadMarker(root)
	if err != nil {
		t.Fatal(err)
	}
	if marker.Adoption == nil || marker.Adoption.RulesDigest != report.Adoption.Rules.Digest || !marker.Adoption.HadRules {
		t.Fatalf("duplicate rules marker = %#v", marker.Adoption)
	}
}

func harnessAdoptionFromDiagnosis(t *testing.T, root string) *harness.AdoptionAdvisory {
	t.Helper()
	diagnosis, err := harness.Diagnose(harness.DiagnoseInput{RepositoryRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	return diagnosis.Adoption
}

func TestLegacyMarkerAddsNoDoctorFault(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	if _, stderr, err := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code", "--fix-gitignore"); err != nil {
		t.Fatalf("init: %v\n%s", err, stderr)
	}
	writeAdoptionFixture(t, root, ambient.MarkerPath,
		"{\n  \"schema\": \"goalrail.ambient-marker/v0\",\n  \"initialized_at\": \"2026-08-01T00:00:00Z\"\n}\n")

	doctorOutput, doctorError, err := runCommand(t, "doctor", "--repo", root, "--scaffold", "claude-code", "--json")
	if err != nil {
		t.Fatalf("doctor rejected a legacy marker: %v\n%s\n%s", err, doctorError, doctorOutput)
	}
	var diagnosis struct {
		Working  bool            `json:"working"`
		Adoption json.RawMessage `json:"adoption"`
	}
	if err := json.Unmarshal([]byte(doctorOutput), &diagnosis); err != nil {
		t.Fatal(err)
	}
	if !diagnosis.Working || len(diagnosis.Adoption) != 0 {
		t.Fatalf("legacy diagnosis = %s", doctorOutput)
	}
}

func writeAdoptionFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
