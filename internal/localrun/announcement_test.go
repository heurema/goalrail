package localrun

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

// silentFixtureAdapter has no way to tell a run that the escalation channel
// exists. It stands in for any future adapter whose provider contract offers no
// launch-time context.
type silentFixtureAdapter struct {
	calls atomic.Int32
}

func (*silentFixtureAdapter) Name() string    { return "fixture" }
func (*silentFixtureAdapter) Version() string { return "v0" }

func (*silentFixtureAdapter) VerifyAnnouncementDelivery() error {
	return errors.New("this adapter has no launch-time context")
}

func (adapter *silentFixtureAdapter) Launch(
	_ context.Context,
	_ LaunchRequest,
) ProviderObservation {
	adapter.calls.Add(1)
	return completedObservation()
}

func TestAnnouncementNamesTheChannelAndItsConditions(t *testing.T) {
	text := EscalationAnnouncement
	if !strings.Contains(text, ReservedEscalationPath) {
		t.Fatal("the announcement does not name the reserved path")
	}
	for _, required := range []string{
		"cannot be completed as specified",
		"blocked",
		"goalrail.escalation/v0",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("the announcement does not state %q", required)
		}
	}
	// The channel only stands on an untouched scope; a run that is not told
	// this will hedge, and a hedge is recorded as a failure.
	if !strings.Contains(text, "change nothing else") &&
		!strings.Contains(text, "untouched scope") {
		t.Fatal("the announcement does not state the clean-scope condition")
	}
	if !strings.Contains(text, "does not resume") {
		t.Fatal("the announcement does not state that the run is terminal")
	}
}

func TestAnnouncementCarriesNoProviderAndNoTaskHint(t *testing.T) {
	lowered := strings.ToLower(EscalationAnnouncement)

	// Content is provider-neutral; only transport may be provider-specific.
	for _, provider := range []string{
		"codex", "claude", "openai", "anthropic", "gpt", "model", "agent", "harness",
	} {
		if strings.Contains(lowered, provider) {
			t.Fatalf("the announcement names %q; content must stay provider-neutral", provider)
		}
	}

	// The measurement that motivates this channel recorded its treatment branch
	// as loud and flagged its own 6/6 as possibly inflated. A hint about the
	// work would measure the hint rather than the environment.
	for _, hint := range []string{
		"conflict", "contradict", "ambiguous", "ambiguity", "disagree",
		"inconsistent", "requirement", "document", "policy", "check whether",
		"look for", "verify that the",
	} {
		if strings.Contains(lowered, hint) {
			t.Fatalf("the announcement hints at the work with %q", hint)
		}
	}
}

func TestStartRefusesToLaunchWhenTheChannelCannotBeAnnounced(t *testing.T) {
	adapter := &silentFixtureAdapter{}
	service, spec, store, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal"),
	})
	prepared := prepareFixture(t, service, spec)
	var idCalls atomic.Int32
	service.newRunID = func() (domain.RunID, error) {
		idCalls.Add(1)
		return "run-unannounced", nil
	}

	_, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	})
	if !errors.Is(err, ErrAnnouncementUndeliverable) {
		t.Fatalf("expected the launch to fail, got %v", err)
	}
	// Silent degradation is the dangerous option: the run would guess and still
	// produce an ordinary-looking receipt.
	if adapter.calls.Load() != 0 {
		t.Fatal("a run was launched whose escalation channel was unreachable")
	}
	if idCalls.Load() != 0 {
		t.Fatal("a run ID was generated for a launch that must not happen")
	}
	exists, err := store.Exists(preparedPath(prepared.WorkSpec.Digest(), "launch-claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("an unannounceable launch created a launch claim")
	}
}

func TestLaunchCarriesTheAnnouncementExactlyOnce(t *testing.T) {
	adapter := &recordingAnnouncementAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal"),
	})
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-announced", nil }

	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if got := adapter.announcements.Load(); got != 1 {
		t.Fatalf("announcements = %d, want exactly 1", got)
	}
	if adapter.received != EscalationAnnouncement {
		t.Fatal("the run received altered announcement text")
	}

	// Finishing must not announce again: one statement, no reply path.
	if _, err := service.Finish(context.Background(), FinishInput{
		RunID: "run-announced",
		Results: []CheckResult{{
			ID:             "test",
			State:          domain.CheckResultPass,
			EvidenceRef:    "local:test-log",
			EvidenceDigest: "sha256:" + strings.Repeat("d", 64),
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := adapter.announcements.Load(); got != 1 {
		t.Fatalf("announcements after finish = %d, want 1", got)
	}
}

func TestAnnouncementReachesNoCanonicalArtifact(t *testing.T) {
	adapter := &recordingAnnouncementAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal"),
	})
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-clean", nil }
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Finish(context.Background(), FinishInput{
		RunID: "run-clean",
		Results: []CheckResult{{
			ID:             "test",
			State:          domain.CheckResultPass,
			EvidenceRef:    "local:test-log",
			EvidenceDigest: "sha256:" + strings.Repeat("d", 64),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The announcement is launch-time transport, not recorded state.
	canonical := string(prepared.WorkSpec.CanonicalJSON())
	if strings.Contains(canonical, ReservedEscalationPath) ||
		strings.Contains(canonical, "escalation") {
		t.Fatal("the canonical WorkSpec gained an escalation field")
	}
	encoded, err := receiptJSON(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded, "This run has one escalation channel") {
		t.Fatal("the terminal receipt embedded the announcement")
	}
}

type recordingAnnouncementAdapter struct {
	result        ProviderObservation
	received      string
	announcements atomic.Int32
}

func (*recordingAnnouncementAdapter) Name() string    { return "fixture" }
func (*recordingAnnouncementAdapter) Version() string { return "v0" }

func (*recordingAnnouncementAdapter) VerifyAnnouncementDelivery() error { return nil }

func (adapter *recordingAnnouncementAdapter) Launch(
	_ context.Context,
	request LaunchRequest,
) ProviderObservation {
	if request.EscalationAnnouncement != "" {
		adapter.announcements.Add(1)
		adapter.received = request.EscalationAnnouncement
	}
	return adapter.result
}

func receiptJSON(receipt TerminalReceipt) (string, error) {
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
