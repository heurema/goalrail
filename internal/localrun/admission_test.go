package localrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

type recordingCodexAdapter struct {
	result    ProviderObservation
	calls     atomic.Int32
	mu        sync.Mutex
	arguments []string
	digest    domain.WorkSpecDigest
}

func (*recordingCodexAdapter) Name() string    { return "codex" }
func (*recordingCodexAdapter) Version() string { return "local-test-v0" }

func (adapter *recordingCodexAdapter) Launch(
	_ context.Context,
	request LaunchRequest,
) ProviderObservation {
	adapter.calls.Add(1)
	adapter.mu.Lock()
	adapter.arguments = append([]string(nil), request.Arguments...)
	adapter.digest = request.WorkSpec.Digest()
	adapter.mu.Unlock()
	return adapter.result
}

func (adapter *recordingCodexAdapter) recorded() ([]string, domain.WorkSpecDigest) {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	return append([]string(nil), adapter.arguments...), adapter.digest
}

func TestNewDogfoodServiceRequiresCodexAdapter(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		adapter Adapter
	}{
		{name: "missing adapter"},
		{name: "fixture adapter", adapter: FixtureAdapter{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewDogfoodService(store, nil, nil, test.adapter); err == nil {
				t.Fatal("expected dogfood constructor to reject a non-Codex adapter")
			}
		})
	}
}

func TestDogfoodAdmissionAllowsExactWorkSpecOnceAndPreservesProviderOutcome(t *testing.T) {
	adapter := &recordingCodexAdapter{result: ProviderObservation{
		Outcome:        ProviderDenied,
		IdentityStatus: IdentityVerified,
		RootSessionRef: "session-denied",
	}}
	service, spec, store := dogfoodServiceFixture(t, adapter, nil)
	prepared := prepareFixture(t, service, spec)
	writeDogfoodAdmission(t, store, dogfoodAdmissionRecord{
		Schema:         dogfoodAdmissionSchema,
		Change:         dogfoodAdmissionChange,
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		BaseRevision:   prepared.WorkSpec.Spec().Repository.BaseRevision,
	})

	var idCalls atomic.Int32
	service.newRunID = func() (domain.RunID, error) {
		idCalls.Add(1)
		return "run-dogfood", nil
	}
	arguments := []string{"--sandbox", "workspace-write", "--ask-for-approval", "on-request"}
	result, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "codex",
		Arguments:      arguments,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != StateFailed ||
		result.Receipt == nil ||
		result.Observation.Outcome != ProviderDenied {
		t.Fatalf("provider denial was not preserved: %+v", result)
	}
	recordedArguments, recordedDigest := adapter.recorded()
	if !reflect.DeepEqual(recordedArguments, arguments) ||
		recordedDigest != prepared.WorkSpec.Digest() {
		t.Fatalf(
			"adapter request = arguments %v digest %s",
			recordedArguments,
			recordedDigest,
		)
	}
	if idCalls.Load() != 1 || adapter.calls.Load() != 1 {
		t.Fatalf(
			"run ID calls=%d adapter calls=%d, want 1 and 1",
			idCalls.Load(),
			adapter.calls.Load(),
		)
	}
	sourceInfo, err := os.Stat(filepath.Join(store.Root(), dogfoodAdmissionRecordName))
	if err != nil {
		t.Fatal(err)
	}
	consumedInfo, err := os.Stat(filepath.Join(store.Root(), dogfoodAdmissionConsumedName))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(sourceInfo, consumedInfo) {
		t.Fatal("consumed marker is not bound to the exact admission record")
	}

	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "codex",
	}); !errors.Is(err, ErrLaunchAlreadyClaimed) {
		t.Fatalf("second start error = %v, want launch already claimed", err)
	}
	if idCalls.Load() != 1 || adapter.calls.Load() != 1 {
		t.Fatal("duplicate start created another run ID or provider invocation")
	}
}

func TestDogfoodAdmissionMismatchFailsBeforeRunArtifacts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *Store, PreparedRun)
	}{
		{name: "missing record"},
		{
			name: "malformed record",
			setup: func(t *testing.T, store *Store, _ PreparedRun) {
				writeDogfoodAdmissionRaw(t, store, []byte("{"))
			},
		},
		{
			name: "unknown field",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				raw := fmt.Sprintf(
					`{"schema":%q,"change":%q,"work_spec_digest":%q,"base_revision":%q,"extra":true}`,
					dogfoodAdmissionSchema,
					dogfoodAdmissionChange,
					prepared.WorkSpec.Digest(),
					prepared.WorkSpec.Spec().Repository.BaseRevision,
				)
				writeDogfoodAdmissionRaw(t, store, []byte(raw))
			},
		},
		{
			name: "wrong change",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				writeDogfoodAdmission(t, store, validDogfoodAdmission(prepared, "other-change"))
			},
		},
		{
			name: "different WorkSpec digest",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				record := validDogfoodAdmission(prepared, dogfoodAdmissionChange)
				record.WorkSpecDigest = domain.WorkSpecDigest("sha256:" + strings.Repeat("b", 64))
				writeDogfoodAdmission(t, store, record)
			},
		},
		{
			name: "stale base revision",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				record := validDogfoodAdmission(prepared, dogfoodAdmissionChange)
				record.BaseRevision = strings.Repeat("b", 40)
				writeDogfoodAdmission(t, store, record)
			},
		},
		{
			name: "already consumed record",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				writeDogfoodAdmission(
					t,
					store,
					validDogfoodAdmission(prepared, dogfoodAdmissionChange),
				)
				if err := os.Link(
					filepath.Join(store.Root(), dogfoodAdmissionRecordName),
					filepath.Join(store.Root(), dogfoodAdmissionConsumedName),
				); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "group-readable record",
			setup: func(t *testing.T, store *Store, prepared PreparedRun) {
				writeDogfoodAdmission(
					t,
					store,
					validDogfoodAdmission(prepared, dogfoodAdmissionChange),
				)
				if err := os.Chmod(
					filepath.Join(store.Root(), dogfoodAdmissionRecordName),
					0o640,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := &recordingCodexAdapter{result: ProviderObservation{
				Outcome:        ProviderCompleted,
				IdentityStatus: IdentityVerified,
				RootSessionRef: "session-should-not-exist",
			}}
			service, spec, store := dogfoodServiceFixture(t, adapter, nil)
			prepared := prepareFixture(t, service, spec)
			if test.setup != nil {
				test.setup(t, store, prepared)
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
				t.Fatalf("start error = %v, want ACTIVATION_REQUIRED", err)
			}
			if idCalls.Load() != 0 || adapter.calls.Load() != 0 {
				t.Fatal("denied admission generated a run ID or invoked the adapter")
			}
			assertNoDogfoodRunArtifacts(t, store, prepared.WorkSpec.Digest())
			if test.name != "already consumed record" {
				if _, err := os.Stat(
					filepath.Join(store.Root(), dogfoodAdmissionConsumedName),
				); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("denied admission created a consumed marker: %v", err)
				}
			}
		})
	}
}

func TestDogfoodAdmissionConcurrentStartsInvokeAdapterAtMostOnce(t *testing.T) {
	adapter := &recordingCodexAdapter{result: ProviderObservation{
		Outcome:        ProviderCompleted,
		IdentityStatus: IdentityVerified,
		RootSessionRef: "session-one",
	}}
	service, spec, store := dogfoodServiceFixture(t, adapter, nil)
	prepared := prepareFixture(t, service, spec)
	writeDogfoodAdmission(
		t,
		store,
		validDogfoodAdmission(prepared, dogfoodAdmissionChange),
	)
	var idCalls atomic.Int32
	service.newRunID = func() (domain.RunID, error) {
		return domain.RunID(fmt.Sprintf("run-%d", idCalls.Add(1))), nil
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
				Adapter:        "codex",
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
		case errors.Is(err, ErrActivationRequired):
		case errors.Is(err, ErrLaunchAlreadyClaimed):
		default:
			t.Fatalf("unexpected concurrent start error: %v", err)
		}
	}
	if successes != 1 || idCalls.Load() != 1 || adapter.calls.Load() != 1 {
		t.Fatalf(
			"successes=%d run IDs=%d adapter calls=%d, want 1, 1, 1",
			successes,
			idCalls.Load(),
			adapter.calls.Load(),
		)
	}
}

func dogfoodServiceFixture(
	t *testing.T,
	adapter Adapter,
	observations []WorktreeObservation,
) (*Service, domain.WorkSpec, *Store) {
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
	service, err := NewDogfoodService(
		store,
		&fixtureObserver{
			root:         resolvedRoot,
			revision:     revision,
			observations: observations,
		},
		&fixtureIntentVerifier{},
		adapter,
	)
	if err != nil {
		t.Fatal(err)
	}
	service.now = fixedClock()
	return service, fixtureWorkSpec(resolvedRoot, revision), store
}

func validDogfoodAdmission(
	prepared PreparedRun,
	change string,
) dogfoodAdmissionRecord {
	return dogfoodAdmissionRecord{
		Schema:         dogfoodAdmissionSchema,
		Change:         change,
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		BaseRevision:   prepared.WorkSpec.Spec().Repository.BaseRevision,
	}
}

func writeDogfoodAdmission(
	t *testing.T,
	store *Store,
	record dogfoodAdmissionRecord,
) {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeDogfoodAdmissionRaw(t, store, raw)
}

func writeDogfoodAdmissionRaw(t *testing.T, store *Store, raw []byte) {
	t.Helper()
	if err := os.WriteFile(
		filepath.Join(store.Root(), dogfoodAdmissionRecordName),
		raw,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
}

func assertNoDogfoodRunArtifacts(
	t *testing.T,
	store *Store,
	digest domain.WorkSpecDigest,
) {
	t.Helper()
	exists, err := store.Exists(preparedPath(digest, "launch-claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("denied admission created a launch claim")
	}
	entries, err := os.ReadDir(filepath.Join(store.Root(), "runs"))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("denied admission created run artifacts: %v", entries)
	}
}
