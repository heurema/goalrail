package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var manifestTestTime = time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

func TestIntentCanaryV0ManifestFreezesEveryRequiredRule(t *testing.T) {
	manifest, err := NewIntentCanaryV0Manifest()
	if err != nil {
		t.Fatalf("create manifest: %v", err)
	}
	if err := ValidateCanaryManifest(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if !manifest.FrozenAt.Equal(IntentCanaryV0FrozenAt) {
		t.Fatalf("frozen_at = %s, want %s", manifest.FrozenAt, IntentCanaryV0FrozenAt)
	}

	var flow, baseline int
	for index, variant := range manifest.VariantRotation {
		expected, err := CanaryVariantForOrdinal(uint32(index + 1))
		if err != nil {
			t.Fatalf("assign ordinal %d: %v", index+1, err)
		}
		if variant != expected {
			t.Fatalf("rotation[%d] = %q, want %q", index, variant, expected)
		}
		if variant == VariantFlow {
			flow++
		} else {
			baseline++
		}
	}
	if flow != 10 || baseline != 5 {
		t.Fatalf("rotation counts = flow %d baseline %d, want 10 and 5", flow, baseline)
	}
}

func TestValidateCanaryManifestRejectsIncompleteOrChangedRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CanaryManifest)
		code   string
	}{
		{
			name: "manifest ID changed",
			mutate: func(manifest *CanaryManifest) {
				manifest.ID = "another-canary"
			},
			code: "canary.manifest.id_changed",
		},
		{
			name: "version changed",
			mutate: func(manifest *CanaryManifest) {
				manifest.Version = 2
			},
			code: "canary.manifest.frozen_at_changed",
		},
		{
			name: "not frozen",
			mutate: func(manifest *CanaryManifest) {
				manifest.FrozenAt = time.Time{}
			},
			code: "canary.manifest.frozen_at_invalid",
		},
		{
			name: "assignment count changed",
			mutate: func(manifest *CanaryManifest) {
				manifest.AssignmentCount = 14
			},
			code: "canary.manifest.assignment_count_changed",
		},
		{
			name: "rotation changed",
			mutate: func(manifest *CanaryManifest) {
				manifest.VariantRotation[0] = VariantBaseline
			},
			code: "canary.manifest.rotation_changed",
		},
		{
			name: "rotation incomplete",
			mutate: func(manifest *CanaryManifest) {
				manifest.VariantRotation = manifest.VariantRotation[:14]
			},
			code: "canary.manifest.rotation_length_changed",
		},
		{
			name: "eligibility missing",
			mutate: func(manifest *CanaryManifest) {
				manifest.EligibilityRuleRef = ""
			},
			code: "canary.manifest.rule_ref_invalid",
		},
		{
			name: "check rule changed",
			mutate: func(manifest *CanaryManifest) {
				manifest.CheckRecordingRuleRef = "canary-rule:check-recording-v2"
			},
			code: "canary.manifest.rule_ref_changed",
		},
		{
			name: "assessment timing missing",
			mutate: func(manifest *CanaryManifest) {
				manifest.AssessmentTimingRuleRef = ""
			},
			code: "canary.manifest.rule_ref_invalid",
		},
		{
			name: "missing-data rule missing",
			mutate: func(manifest *CanaryManifest) {
				manifest.MissingDataRuleRef = ""
			},
			code: "canary.manifest.rule_ref_invalid",
		},
		{
			name: "overhead rule missing",
			mutate: func(manifest *CanaryManifest) {
				manifest.OverheadRuleRef = ""
			},
			code: "canary.manifest.rule_ref_invalid",
		},
		{
			name: "rate formula missing",
			mutate: func(manifest *CanaryManifest) {
				manifest.RateFormulaRuleRef = ""
			},
			code: "canary.manifest.rule_ref_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := NewIntentCanaryV0Manifest()
			if err != nil {
				t.Fatalf("create manifest: %v", err)
			}
			test.mutate(&manifest)
			err = ValidateCanaryManifest(manifest)
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("validate changed manifest error = %v, want code %s", err, test.code)
			}
		})
	}
}

func TestIntentCanaryV0ManifestV2FreezesContextAndTelemetryRules(t *testing.T) {
	manifest, err := NewIntentCanaryV0ManifestV2()
	if err != nil {
		t.Fatalf("create v2 manifest: %v", err)
	}
	if manifest.Version != IntentCanaryV0ManifestVersion2 ||
		manifest.ContextRuleRef != CanaryContextRuleV2 ||
		manifest.TelemetryRuleRef != CanaryTelemetryRuleV2 ||
		manifest.AssessmentBasisRuleRef != CanaryAssessmentBasisRuleV2 {
		t.Fatalf("unexpected v2 manifest: %#v", manifest)
	}
	if err := ValidateCanaryManifest(manifest); err != nil {
		t.Fatalf("validate v2 manifest: %v", err)
	}

	manifest.ContextRuleRef = ""
	if err := ValidateCanaryManifest(manifest); err == nil ||
		!strings.Contains(err.Error(), "canary.manifest.rule_ref_invalid") {
		t.Fatalf("missing v2 context rule error = %v", err)
	}
}

func TestManifestVersionSelectionRejectsUnknownVersion(t *testing.T) {
	for _, version := range []uint32{1, 2} {
		manifest, err := NewIntentCanaryV0ManifestForVersion(version)
		if err != nil || manifest.Version != version {
			t.Fatalf("select manifest %d: manifest=%#v err=%v", version, manifest, err)
		}
	}
	if _, err := NewIntentCanaryV0ManifestForVersion(3); err == nil {
		t.Fatal("unknown manifest version was accepted")
	}
}

func TestManifestArtifactMatchesCanonicalV2ReferencesAndRotation(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	contents, err := os.ReadFile(filepath.Join(repoRoot, "canary", "intent-canary-v0", "manifest-v2.md"))
	if err != nil {
		t.Fatalf("read v2 manifest artifact: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"**Version:** 2",
		"**State:** `frozen_not_activated`",
		string(CanaryContextRuleV2),
		string(CanaryTelemetryRuleV2),
		string(CanaryAssessmentBasisRuleV2),
		"`canary/intent-canary-v0/events-v2.jsonl`",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("v2 manifest artifact is missing %q", required)
		}
	}
	for ordinal, variant := range intentCanaryV0Rotation() {
		row := fmt.Sprintf("| %d | `%s` |", ordinal+1, variant)
		if !strings.Contains(text, row) {
			t.Fatalf("v2 manifest artifact is missing rotation row %q", row)
		}
	}
}

func TestIntentCanaryV0ManifestReturnsIndependentRotation(t *testing.T) {
	first, err := NewIntentCanaryV0Manifest()
	if err != nil {
		t.Fatalf("create first manifest: %v", err)
	}
	first.VariantRotation[0] = VariantBaseline

	second, err := NewIntentCanaryV0Manifest()
	if err != nil {
		t.Fatalf("create second manifest: %v", err)
	}
	if second.VariantRotation[0] != VariantFlow {
		t.Fatal("caller mutation changed a later manifest rotation")
	}
}

func TestManifestArtifactMatchesCanonicalV1ReferencesAndRotation(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	manifestPath := filepath.Join(
		repoRoot,
		"canary",
		"intent-canary-v0",
		"manifest-v1.md",
	)
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest artifact: %v", err)
	}
	archivedPath := filepath.Join(
		repoRoot,
		"openspec",
		"changes",
		"archive",
		"2026-07-22-intent-canary-v0",
		"canary",
		"manifest-v1.md",
	)
	archived, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("read archived manifest artifact: %v", err)
	}
	if string(contents) != string(archived) {
		t.Fatal("runtime manifest differs from archived frozen manifest")
	}
	text := string(contents)
	for _, required := range []string{
		"**State:** `frozen_not_activated`",
		string(CanaryEligibilityRuleV1),
		string(CanaryCheckRecordingRuleV1),
		string(CanaryAssessmentTimingRuleV1),
		string(CanaryMissingDataRuleV1),
		string(CanaryOverheadRuleV1),
		string(CanaryRateFormulaRuleV1),
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("manifest artifact is missing %q", required)
		}
	}
	for ordinal, variant := range intentCanaryV0Rotation() {
		row := fmt.Sprintf("| %d | `%s` |", ordinal+1, variant)
		if !strings.Contains(text, row) {
			t.Fatalf("manifest artifact is missing rotation row %q", row)
		}
	}
}

func TestCalculateCanaryFlowOverheadUsesDistinctTurnEnvelopesAndOwnerReview(t *testing.T) {
	measurement, err := CalculateCanaryFlowOverhead(CanaryFlowOverheadInput{
		AgentTurns: []CanaryTimingInterval{
			{
				Reference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				StartedAt: manifestTestTime,
				EndedAt:   manifestTestTime.Add(2 * time.Minute),
			},
			{
				Reference: "langfuse-trace:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				StartedAt: manifestTestTime.Add(3 * time.Minute),
				EndedAt:   manifestTestTime.Add(6 * time.Minute),
			},
		},
		OwnerReview: &CanaryTimingInterval{
			Reference: "owner-review:change-1",
			StartedAt: manifestTestTime.Add(7 * time.Minute),
			EndedAt:   manifestTestTime.Add(12 * time.Minute),
		},
		OwnerReviewRequired: true,
	})
	if err != nil {
		t.Fatalf("calculate overhead: %v", err)
	}
	if !measurement.Available || measurement.AgentTurnCount != 2 ||
		measurement.AgentSeconds != 300 || measurement.OwnerReviewSeconds != 300 ||
		measurement.TotalMinutes != 10 {
		t.Fatalf("unexpected overhead measurement: %#v", measurement)
	}
}

func TestCalculateCanaryFlowOverheadDoesNotImputeMissingEvidence(t *testing.T) {
	agentTurn := CanaryTimingInterval{
		Reference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StartedAt: manifestTestTime,
		EndedAt:   manifestTestTime.Add(2 * time.Minute),
	}

	withoutTrace, err := CalculateCanaryFlowOverhead(CanaryFlowOverheadInput{
		OwnerReviewRequired: true,
	})
	if err != nil {
		t.Fatalf("calculate missing trace: %v", err)
	}
	if withoutTrace.Available || withoutTrace.TotalMinutes != 0 {
		t.Fatalf("missing trace was imputed: %#v", withoutTrace)
	}

	withoutRequiredReview, err := CalculateCanaryFlowOverhead(CanaryFlowOverheadInput{
		AgentTurns:          []CanaryTimingInterval{agentTurn},
		OwnerReviewRequired: true,
	})
	if err != nil {
		t.Fatalf("calculate missing review: %v", err)
	}
	if withoutRequiredReview.Available || withoutRequiredReview.AgentSeconds != 120 ||
		withoutRequiredReview.TotalMinutes != 0 {
		t.Fatalf("missing review was imputed: %#v", withoutRequiredReview)
	}

	abandonedBeforeReview, err := CalculateCanaryFlowOverhead(CanaryFlowOverheadInput{
		AgentTurns: []CanaryTimingInterval{agentTurn},
	})
	if err != nil {
		t.Fatalf("calculate pre-review abandonment: %v", err)
	}
	if !abandonedBeforeReview.Available || abandonedBeforeReview.TotalMinutes != 2 {
		t.Fatalf("valid pre-review overhead unavailable: %#v", abandonedBeforeReview)
	}
}

func TestCalculateCanaryFlowOverheadRejectsDuplicateOrInvalidIntervals(t *testing.T) {
	valid := CanaryTimingInterval{
		Reference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StartedAt: manifestTestTime,
		EndedAt:   manifestTestTime.Add(time.Minute),
	}
	tests := []struct {
		name      string
		intervals []CanaryTimingInterval
		code      string
	}{
		{
			name:      "duplicate trace",
			intervals: []CanaryTimingInterval{valid, valid},
			code:      "canary.overhead.trace_duplicate",
		},
		{
			name: "reversed interval",
			intervals: []CanaryTimingInterval{
				{
					Reference: valid.Reference,
					StartedAt: valid.EndedAt,
					EndedAt:   valid.StartedAt,
				},
			},
			code: "canary.overhead.interval_reversed",
		},
		{
			name: "raw reference",
			intervals: []CanaryTimingInterval{
				{
					Reference: "raw trace content",
					StartedAt: valid.StartedAt,
					EndedAt:   valid.EndedAt,
				},
			},
			code: "canary.overhead.reference_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CalculateCanaryFlowOverhead(CanaryFlowOverheadInput{AgentTurns: test.intervals})
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("calculate invalid overhead error = %v, want code %s", err, test.code)
			}
		})
	}
}
