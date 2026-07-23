package localrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

type fixtureIntentVerifier struct {
	calls atomic.Int32
	err   error
}

func (verifier *fixtureIntentVerifier) Verify(
	_ string,
	_ domain.WorkSpecIntentReference,
) error {
	verifier.calls.Add(1)
	return verifier.err
}

type fixtureObserver struct {
	root         string
	revision     string
	observations []WorktreeObservation
	calls        atomic.Int32
}

func (observer *fixtureObserver) ResolveRepository(
	_ context.Context,
	_ string,
	_ string,
) (string, string, error) {
	return observer.root, observer.revision, nil
}

func (observer *fixtureObserver) Observe(
	_ context.Context,
	_ string,
) (WorktreeObservation, error) {
	index := int(observer.calls.Add(1) - 1)
	if index >= len(observer.observations) {
		index = len(observer.observations) - 1
	}
	return observer.observations[index], nil
}

type countingFixtureAdapter struct {
	result ProviderObservation
	calls  atomic.Int32
}

func (*countingFixtureAdapter) Name() string    { return "fixture" }
func (*countingFixtureAdapter) Version() string { return "v0" }

func (adapter *countingFixtureAdapter) Launch(
	_ context.Context,
	_ LaunchRequest,
) ProviderObservation {
	adapter.calls.Add(1)
	return adapter.result
}

func TestPrepareIsInspectableAndActivationDenialCreatesNoRun(t *testing.T) {
	service, spec, store, verifier := productionServiceFixture(t)
	prepared := prepareFixture(t, service, spec)
	if verifier.calls.Load() != 1 {
		t.Fatalf("intent verifier calls = %d, want 1", verifier.calls.Load())
	}
	inspected, err := service.InspectPrepared(prepared.WorkSpec.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inspected.WorkSpec.CanonicalJSON(), prepared.WorkSpec.CanonicalJSON()) {
		t.Fatal("inspect changed canonical WorkSpec")
	}

	var idCalls atomic.Int32
	service.newRunID = func() (domain.RunID, error) {
		idCalls.Add(1)
		return "run-should-not-exist", nil
	}
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "codex",
	}); !errors.Is(err, ErrActivationRequired) {
		t.Fatalf("expected activation denial, got %v", err)
	}
	if idCalls.Load() != 0 {
		t.Fatal("activation denial generated a run ID")
	}
	exists, err := store.Exists(preparedPath(prepared.WorkSpec.Digest(), "launch-claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("activation denial created a launch claim")
	}
}

func TestFixtureLifecycleIsOneShotAndProducesBoundedReceipt(t *testing.T) {
	adapter := &countingFixtureAdapter{result: ProviderObservation{
		Outcome:        ProviderCompleted,
		IdentityStatus: IdentityVerified,
		RootSessionRef: "session-root",
	}}
	service, spec, _, _ := fixtureService(t, adapter, nil)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-one", nil }

	started, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
		Arguments:      []string{"--provider-only-argument"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.State != StateAwaitingVerification || started.Receipt != nil {
		t.Fatalf("start result = %+v", started)
	}
	if adapter.calls.Load() != 1 {
		t.Fatalf("adapter calls = %d, want 1", adapter.calls.Load())
	}
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); !errors.Is(err, ErrLaunchAlreadyClaimed) {
		t.Fatalf("expected duplicate start rejection, got %v", err)
	}
	if adapter.calls.Load() != 1 {
		t.Fatal("duplicate start invoked the adapter")
	}

	receipt, err := service.Finish(context.Background(), FinishInput{
		RunID: "run-one",
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
	if receipt.Status != StatePassed || receipt.RootSessionRef != "session-root" {
		t.Fatalf("receipt = %+v", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{"provider-only-argument", "prompt", "transcript", "source_body"} {
		if bytes.Contains(encoded, []byte(prohibited)) {
			t.Fatalf("receipt retained prohibited value %q", prohibited)
		}
	}
	if _, err := service.Finish(context.Background(), FinishInput{RunID: "run-one"}); !errors.Is(err, ErrTerminalReceiptExists) {
		t.Fatalf("expected immutable terminal receipt, got %v", err)
	}
}

func TestConcurrentStartInvokesFixtureAtMostOnce(t *testing.T) {
	adapter := &countingFixtureAdapter{result: ProviderObservation{
		Outcome:        ProviderCompleted,
		IdentityStatus: IdentityVerified,
		RootSessionRef: "session-root",
	}}
	service, spec, _, _ := fixtureService(t, adapter, nil)
	prepared := prepareFixture(t, service, spec)
	var sequence atomic.Int32
	service.newRunID = func() (domain.RunID, error) {
		return domain.RunID(fmt.Sprintf("run-%d", sequence.Add(1))), nil
	}

	const attempts = 12
	var wait sync.WaitGroup
	wait.Add(attempts)
	results := make(chan error, attempts)
	for range attempts {
		go func() {
			defer wait.Done()
			_, err := service.Start(context.Background(), StartInput{
				WorkSpecDigest: prepared.WorkSpec.Digest(),
				Adapter:        "fixture",
			})
			results <- err
		}()
	}
	wait.Wait()
	close(results)

	successes := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrLaunchAlreadyClaimed):
		default:
			t.Fatalf("unexpected start error: %v", err)
		}
	}
	if successes != 1 || adapter.calls.Load() != 1 {
		t.Fatalf("successes=%d adapter_calls=%d, want 1 and 1", successes, adapter.calls.Load())
	}
}

func TestFailureUnlinkedAndUnknownStatesNeverRetry(t *testing.T) {
	adapter := &countingFixtureAdapter{result: ProviderObservation{
		Outcome:        ProviderLaunchFailed,
		IdentityStatus: IdentityUnlinked,
		Reason:         "MISSING_ROOT_SESSION",
	}}
	service, spec, store, _ := fixtureService(t, adapter, nil)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-failed", nil }
	started, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.State != StateUnlinked || started.Receipt == nil {
		t.Fatalf("failed start = %+v", started)
	}
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); !errors.Is(err, ErrLaunchAlreadyClaimed) {
		t.Fatalf("expected no retry, got %v", err)
	}
	if adapter.calls.Load() != 1 {
		t.Fatal("failure path retried the adapter")
	}

	orphanClaim := LaunchClaim{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		RunID:          "run-orphan",
		Adapter:        "fixture",
		AdapterVersion: "v0",
		AttemptedAt:    time.Now().UTC(),
	}
	if err := store.WriteJSONOnce(runPath("run-orphan", "launch-claim.json"), orphanClaim, false); err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectRun("run-orphan")
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != StateLaunchAttemptedUnknown {
		t.Fatalf("orphan state = %s", inspection.State)
	}

	orphanPrepared := prepared
	orphanPreparedDigest := orphanPrepared.WorkSpec.Digest()
	secondStore, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := secondStore.WriteBytesOnce(
		preparedPath(orphanPreparedDigest, "work-spec.json"),
		orphanPrepared.WorkSpec.CanonicalJSON(),
		false,
	); err != nil {
		t.Fatal(err)
	}
	if err := secondStore.WriteJSONOnce(
		preparedPath(orphanPreparedDigest, "preparation.json"),
		orphanPrepared.Preparation,
		false,
	); err != nil {
		t.Fatal(err)
	}
	if err := secondStore.WriteJSONOnce(
		preparedPath(orphanPreparedDigest, "launch-claim.json"),
		orphanClaim,
		false,
	); err != nil {
		t.Fatal(err)
	}
	orphanService := NewService(secondStore, &fixtureObserver{}, &fixtureIntentVerifier{})
	preparedInspection, err := orphanService.InspectPrepared(orphanPreparedDigest)
	if err != nil {
		t.Fatal(err)
	}
	if preparedInspection.State != StateLaunchAttemptedUnknown ||
		preparedInspection.Claim == nil ||
		preparedInspection.Claim.RunID != "run-orphan" {
		t.Fatalf("prepared orphan inspection = %+v", preparedInspection)
	}
}

func TestProviderDenialAndLaunchFailureAreTerminal(t *testing.T) {
	tests := map[string]struct {
		outcome ProviderOutcome
		want    RunState
	}{
		"denied":        {outcome: ProviderDenied, want: StateFailed},
		"launch failed": {outcome: ProviderLaunchFailed, want: StateLaunchFailed},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			adapter := &countingFixtureAdapter{result: ProviderObservation{
				Outcome:        test.outcome,
				IdentityStatus: IdentityVerified,
				RootSessionRef: "session-root",
			}}
			service, spec, _, _ := fixtureService(t, adapter, nil)
			prepared := prepareFixture(t, service, spec)
			service.newRunID = func() (domain.RunID, error) { return "run-terminal", nil }
			result, err := service.Start(context.Background(), StartInput{
				WorkSpecDigest: prepared.WorkSpec.Digest(),
				Adapter:        "fixture",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.State != test.want || result.Receipt == nil || result.Receipt.Status != test.want {
				t.Fatalf("result = %+v, want terminal %s", result, test.want)
			}
			if adapter.calls.Load() != 1 {
				t.Fatalf("adapter calls = %d, want 1", adapter.calls.Load())
			}
		})
	}
}

func TestFinishCannotPassMissingChecksOrScopeViolation(t *testing.T) {
	t.Run("missing check", func(t *testing.T) {
		adapter := &countingFixtureAdapter{result: ProviderObservation{
			Outcome:        ProviderCompleted,
			IdentityStatus: IdentityVerified,
			RootSessionRef: "session-root",
		}}
		service, spec, _, _ := fixtureService(t, adapter, nil)
		prepared := prepareFixture(t, service, spec)
		service.newRunID = func() (domain.RunID, error) { return "run-missing", nil }
		if _, err := service.Start(context.Background(), StartInput{WorkSpecDigest: prepared.WorkSpec.Digest(), Adapter: "fixture"}); err != nil {
			t.Fatal(err)
		}
		receipt, err := service.Finish(context.Background(), FinishInput{RunID: "run-missing"})
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != StateVerificationIncomplete ||
			receipt.CheckResults[0].State != domain.CheckResultMissing {
			t.Fatalf("receipt = %+v", receipt)
		}
	})

	t.Run("scope violation", func(t *testing.T) {
		adapter := &countingFixtureAdapter{result: ProviderObservation{
			Outcome:        ProviderCompleted,
			IdentityStatus: IdentityVerified,
			RootSessionRef: "session-root",
		}}
		baseline := fixtureObservation("base")
		terminal := fixtureObservation("terminal")
		terminal.Entries = []WorktreeEntry{{
			Path:   "outside.txt",
			Status: "??",
			Mode:   uint32(0o100600),
			Digest: "sha256:" + strings.Repeat("c", 64),
		}}
		service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{baseline, terminal})
		prepared := prepareFixture(t, service, spec)
		service.newRunID = func() (domain.RunID, error) { return "run-scope", nil }
		if _, err := service.Start(context.Background(), StartInput{WorkSpecDigest: prepared.WorkSpec.Digest(), Adapter: "fixture"}); err != nil {
			t.Fatal(err)
		}
		receipt, err := service.Finish(context.Background(), FinishInput{
			RunID:   "run-scope",
			Results: []CheckResult{{ID: "test", State: domain.CheckResultPass}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != StateFailed ||
			strings.Join(receipt.WorktreeDelta.ScopeViolations, ",") != "outside.txt" {
			t.Fatalf("receipt = %+v", receipt)
		}
	})
}

func TestPrepareRejectsScopedSymlinkEscapeAndMissingIntent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escaped")); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fixtureObserver{
		root:         resolvedRoot,
		revision:     revision,
		observations: []WorktreeObservation{fixtureObservation("base")},
	}
	verifier := &fixtureIntentVerifier{err: os.ErrNotExist}
	service := NewService(store, observer, verifier)
	spec := fixtureWorkSpec(root, revision)
	spec.Paths = []string{"escaped"}
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(context.Background(), bytes.NewReader(raw)); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink scope rejection, got %v", err)
	}
	if verifier.calls.Load() != 0 {
		t.Fatal("intent resolver ran after scope rejection")
	}

	spec.Paths = []string{"inside"}
	raw, err = json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(context.Background(), bytes.NewReader(raw)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing intent rejection, got %v", err)
	}
}

func productionServiceFixture(
	t *testing.T,
) (*Service, domain.WorkSpec, *Store, *fixtureIntentVerifier) {
	t.Helper()
	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	observer := &fixtureObserver{
		root:         resolvedRoot,
		revision:     revision,
		observations: []WorktreeObservation{fixtureObservation("base")},
	}
	verifier := &fixtureIntentVerifier{}
	service := NewService(store, observer, verifier)
	service.now = fixedClock()
	return service, fixtureWorkSpec(resolvedRoot, revision), store, verifier
}

func fixtureService(
	t *testing.T,
	adapter Adapter,
	observations []WorktreeObservation,
) (*Service, domain.WorkSpec, *Store, *fixtureIntentVerifier) {
	t.Helper()
	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	if len(observations) == 0 {
		observations = []WorktreeObservation{
			fixtureObservation("base"),
			fixtureObservation("terminal"),
		}
	}
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	observer := &fixtureObserver{
		root:         resolvedRoot,
		revision:     revision,
		observations: observations,
	}
	verifier := &fixtureIntentVerifier{}
	service, err := NewFixtureService(store, observer, verifier, adapter)
	if err != nil {
		t.Fatal(err)
	}
	service.now = fixedClock()
	return service, fixtureWorkSpec(resolvedRoot, revision), store, verifier
}

func prepareFixture(t *testing.T, service *Service, spec domain.WorkSpec) PreparedRun {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := service.Prepare(context.Background(), bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func fixtureWorkSpec(root, revision string) domain.WorkSpec {
	return domain.WorkSpec{
		Schema:  domain.WorkSpecSchemaV0,
		ID:      "work-dogfood",
		Version: 1,
		Repository: domain.WorkSpecRepository{
			Root:         root,
			BaseRevision: revision,
		},
		Intent: domain.WorkSpecIntentReference{
			ID:          "intent-dogfood",
			Version:     3,
			ArtifactRef: "intent.md",
			Digest:      "sha256:" + strings.Repeat("b", 64),
		},
		Task:  "Implement one bounded local change.",
		Paths: []string{"inside"},
		Checks: []domain.WorkSpecCheck{{
			ID:   "test",
			Argv: []string{"go", "test", "./..."},
		}},
		StopConditions: []domain.WorkSpecStopCondition{{
			ID:          "receipt",
			Description: "Stop after the terminal receipt.",
		}},
		Posture: domain.PostureTrustedLocalProviderEnforcedV0,
	}
}

func fixtureObservation(label string) WorktreeObservation {
	fill := "a"
	if label == "terminal" {
		fill = "b"
	}
	return WorktreeObservation{
		Head:    strings.Repeat("a", 40),
		Entries: nil,
		Digest:  "sha256:" + strings.Repeat(fill, 64),
	}
}

func fixedClock() func() time.Time {
	var sequence atomic.Int64
	base := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	return func() time.Time {
		return base.Add(time.Duration(sequence.Add(1)) * time.Second)
	}
}
