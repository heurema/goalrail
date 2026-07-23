package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

type commandService struct {
	prepared      localrun.PreparedRun
	runInspection localrun.RunInspection
	startResult   localrun.StartResult
	startErr      error
	finishReceipt localrun.TerminalReceipt
	finishErr     error
	startInput    localrun.StartInput
	finishInput   localrun.FinishInput
}

func (service *commandService) Prepare(
	_ context.Context,
	reader io.Reader,
) (localrun.PreparedRun, error) {
	spec, err := domain.DecodeWorkSpec(reader)
	if err != nil {
		return localrun.PreparedRun{}, err
	}
	frozen, err := domain.FreezeWorkSpec(spec)
	if err != nil {
		return localrun.PreparedRun{}, err
	}
	service.prepared = localrun.PreparedRun{
		WorkSpec: frozen,
		Preparation: localrun.Preparation{
			WorkSpecDigest: frozen.Digest(),
			PreparedAt:     time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC),
			Baseline: localrun.WorktreeObservation{
				Head:   strings.Repeat("a", 40),
				Digest: "sha256:" + strings.Repeat("b", 64),
			},
			State: localrun.StatePrepared,
		},
		State: localrun.StatePrepared,
	}
	return service.prepared, nil
}

func (service *commandService) InspectPrepared(
	_ domain.WorkSpecDigest,
) (localrun.PreparedRun, error) {
	return service.prepared, nil
}

func (service *commandService) Start(
	_ context.Context,
	input localrun.StartInput,
) (localrun.StartResult, error) {
	service.startInput = input
	return service.startResult, service.startErr
}

func (service *commandService) Finish(
	_ context.Context,
	input localrun.FinishInput,
) (localrun.TerminalReceipt, error) {
	service.finishInput = input
	return service.finishReceipt, service.finishErr
}

func (service *commandService) InspectRun(
	_ domain.RunID,
) (localrun.RunInspection, error) {
	return service.runInspection, nil
}

func TestPrepareAndInspectDisplayFrozenWorkSpec(t *testing.T) {
	service := &commandService{}
	spec := commandWorkSpec(t.TempDir())
	path := filepath.Join(t.TempDir(), "work-spec.json")
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	factory := func(string) (runService, error) { return service, nil }

	var prepared bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"prepare", "--file", path, "--state-dir", t.TempDir()},
		bytes.NewReader(nil),
		&prepared,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	var output preparedOutput
	if err := json.Unmarshal(prepared.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Digest != service.prepared.WorkSpec.Digest() ||
		!bytes.Equal(output.WorkSpec, service.prepared.WorkSpec.CanonicalJSON()) {
		t.Fatalf("prepare output = %s", prepared.String())
	}

	var inspected bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"inspect", "--digest", string(output.Digest)},
		bytes.NewReader(nil),
		&inspected,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(inspected.Bytes(), service.prepared.WorkSpec.CanonicalJSON()) {
		t.Fatalf("inspect output omitted frozen WorkSpec: %s", inspected.String())
	}
}

func TestProductionStartIsActivationDeniedBeforeState(t *testing.T) {
	stateRoot := t.TempDir()
	store, err := localrun.NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	service := localrun.NewService(store, localrun.GitObserver{}, nil)
	factory := func(string) (runService, error) { return service, nil }
	digest := "sha256:" + strings.Repeat("a", 64)
	err = run(
		context.Background(),
		[]string{
			"start",
			"--digest", digest,
			"--adapter", "codex",
			"--",
			"--sandbox", "workspace-write",
		},
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
		factory,
	)
	if !errors.Is(err, localrun.ErrActivationRequired) {
		t.Fatalf("expected activation denial, got %v", err)
	}
	entries, err := os.ReadDir(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("activation denial wrote state: %v", entries)
	}
}

func TestStartAndFinishKeepAdapterArgumentsOutsideResults(t *testing.T) {
	service := &commandService{
		startResult: localrun.StartResult{
			State: localrun.StateAwaitingVerification,
		},
		finishReceipt: localrun.TerminalReceipt{
			Status: localrun.StatePassed,
		},
	}
	factory := func(string) (runService, error) { return service, nil }
	digest := "sha256:" + strings.Repeat("a", 64)
	if err := run(
		context.Background(),
		[]string{
			"start",
			"--digest", digest,
			"--adapter", "codex",
			"--",
			"--sandbox", "workspace-write",
		},
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if service.startInput.Adapter != "codex" ||
		strings.Join(service.startInput.Arguments, " ") != "--sandbox workspace-write" {
		t.Fatalf("start input = %+v", service.startInput)
	}

	if err := run(
		context.Background(),
		[]string{
			"finish",
			"--run", "run-one",
			"--result", "test=pass,local:test-log,sha256:" + strings.Repeat("b", 64),
		},
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if service.finishInput.RunID != "run-one" ||
		len(service.finishInput.Results) != 1 ||
		service.finishInput.Results[0].EvidenceRef != "local:test-log" {
		t.Fatalf("finish input = %+v", service.finishInput)
	}
}

func TestCommandFixtureRunsPrepareStartFinishSequence(t *testing.T) {
	service := &commandService{}
	factory := func(string) (runService, error) { return service, nil }
	spec := commandWorkSpec(t.TempDir())
	path := filepath.Join(t.TempDir(), "work-spec.json")
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	var prepared bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"prepare", "--file", path},
		bytes.NewReader(nil),
		&prepared,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	var preparedResult preparedOutput
	if err := json.Unmarshal(prepared.Bytes(), &preparedResult); err != nil {
		t.Fatal(err)
	}

	service.startResult = localrun.StartResult{
		Claim: localrun.LaunchClaim{
			WorkSpecDigest: preparedResult.Digest,
			RunID:          "run-command",
			Adapter:        "codex",
			AdapterVersion: "fixture-v0",
		},
		State: localrun.StateAwaitingVerification,
	}
	var started bytes.Buffer
	if err := run(
		context.Background(),
		[]string{
			"start",
			"--digest", string(preparedResult.Digest),
			"--adapter", "codex",
		},
		bytes.NewReader(nil),
		&started,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(started.Bytes(), []byte(`"run_id":"run-command"`)) {
		t.Fatalf("start output = %s", started.String())
	}

	service.finishReceipt = localrun.TerminalReceipt{
		WorkSpecDigest: preparedResult.Digest,
		RunID:          "run-command",
		Status:         localrun.StatePassed,
	}
	var finished bytes.Buffer
	if err := run(
		context.Background(),
		[]string{
			"finish",
			"--run", "run-command",
			"--result", "test=pass",
		},
		bytes.NewReader(nil),
		&finished,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(finished.Bytes(), []byte(`"status":"passed"`)) {
		t.Fatalf("finish output = %s", finished.String())
	}
}

func TestCommandHasNoFixtureActivationSurface(t *testing.T) {
	service := &commandService{}
	factory := func(string) (runService, error) { return service, nil }
	err := run(
		context.Background(),
		[]string{
			"start",
			"--digest", "sha256:" + strings.Repeat("a", 64),
			"--adapter", "fixture",
		},
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
		factory,
	)
	if err == nil || !strings.Contains(err.Error(), "only the codex adapter") {
		t.Fatalf("expected fixture rejection, got %v", err)
	}
	var help bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"help"},
		bytes.NewReader(nil),
		&help,
		io.Discard,
		factory,
	); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(help.String()), "fixture") {
		t.Fatalf("help exposed fixture activation: %s", help.String())
	}
}

func commandWorkSpec(root string) domain.WorkSpec {
	return domain.WorkSpec{
		Schema:  domain.WorkSpecSchemaV0,
		ID:      "work-command",
		Version: 1,
		Repository: domain.WorkSpecRepository{
			Root:         root,
			BaseRevision: strings.Repeat("a", 40),
		},
		Intent: domain.WorkSpecIntentReference{
			ID:          "intent-command",
			Version:     1,
			ArtifactRef: "intent.md",
			Digest:      "sha256:" + strings.Repeat("b", 64),
		},
		Task:  "Prepare one bounded command fixture.",
		Paths: []string{"cmd/gr"},
		Checks: []domain.WorkSpecCheck{{
			ID:   "test",
			Argv: []string{"go", "test", "./cmd/gr"},
		}},
		StopConditions: []domain.WorkSpecStopCondition{{
			ID:          "receipt",
			Description: "Stop after receipt.",
		}},
		Posture: domain.PostureTrustedLocalProviderEnforcedV0,
	}
}
