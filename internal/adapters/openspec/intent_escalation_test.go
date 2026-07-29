package openspec

import (
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

const fixtureEscalationDigest = "sha256:" +
	"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

func TestReadIntentParsesTheAnsweringEscalationRecord(t *testing.T) {
	raw := intentWithMetadata(
		"- **Resolves:** `run-blocked-one` escalation `" + fixtureEscalationDigest + "`\n" +
			"- **Disposition:** answered",
	)
	snapshot, err := ReadIntent(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	resolution := snapshot.ResolvedEscalation
	if resolution == nil {
		t.Fatal("the answering record was dropped")
	}
	if resolution.RunID != "run-blocked-one" {
		t.Fatalf("run ID = %q", resolution.RunID)
	}
	if resolution.EscalationDigest != fixtureEscalationDigest {
		t.Fatalf("escalation digest = %q", resolution.EscalationDigest)
	}
	if resolution.Disposition != domain.DispositionAnswered {
		t.Fatalf("disposition = %q", resolution.Disposition)
	}
}

func TestReadIntentAcceptsSpuriousAndWithdrawnDispositions(t *testing.T) {
	// A low-value escalation must stay visible and countable rather than
	// disappearing, so both directions are recordable.
	for _, disposition := range []string{"spurious", "withdrawn"} {
		t.Run(disposition, func(t *testing.T) {
			raw := intentWithMetadata(
				"- **Resolves:** `run-blocked-one` escalation `" + fixtureEscalationDigest + "`\n" +
					"- **Disposition:** " + disposition,
			)
			snapshot, err := ReadIntent(strings.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			if string(snapshot.ResolvedEscalation.Disposition) != disposition {
				t.Fatalf("disposition = %q, want %q", snapshot.ResolvedEscalation.Disposition, disposition)
			}
		})
	}
}

func TestReadIntentWithoutAnAnsweringRecordInfersNothing(t *testing.T) {
	snapshot, err := ReadIntent(strings.NewReader(minimalIntent("confirmed", "None.", true)))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ResolvedEscalation != nil {
		t.Fatalf("absent record produced %+v", snapshot.ResolvedEscalation)
	}
}

func TestReadIntentRejectsMalformedAnsweringRecords(t *testing.T) {
	for name, metadata := range map[string]string{
		"resolution without disposition": "- **Resolves:** `run-blocked-one` escalation `" +
			fixtureEscalationDigest + "`",
		"disposition without resolution": "- **Disposition:** answered",
		"missing escalation keyword": "- **Resolves:** `run-blocked-one` `" +
			fixtureEscalationDigest + "`\n- **Disposition:** answered",
		"too few fields": "- **Resolves:** `run-blocked-one`\n- **Disposition:** answered",
		"unknown disposition": "- **Resolves:** `run-blocked-one` escalation `" +
			fixtureEscalationDigest + "`\n- **Disposition:** acknowledged",
		"malformed digest": "- **Resolves:** `run-blocked-one` escalation `sha256:short`\n" +
			"- **Disposition:** answered",
		"non-canonical run id": "- **Resolves:** `Run One` escalation `" +
			fixtureEscalationDigest + "`\n- **Disposition:** answered",
		"duplicate record": "- **Resolves:** `run-blocked-one` escalation `" +
			fixtureEscalationDigest + "`\n" +
			"- **Resolves:** `run-blocked-two` escalation `" + fixtureEscalationDigest + "`\n" +
			"- **Disposition:** answered",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadIntent(strings.NewReader(intentWithMetadata(metadata))); err == nil {
				t.Fatal("a malformed answering record was silently accepted")
			}
		})
	}
}

func intentWithMetadata(metadata string) string {
	return strings.Replace(
		minimalIntent("confirmed", "None.", true),
		"- **Owner:** owner",
		"- **Owner:** owner\n"+metadata,
		1,
	)
}
