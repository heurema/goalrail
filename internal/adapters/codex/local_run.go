package codex

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

const (
	LocalRunContextEnvironment = "GOALRAIL_LOCAL_RUN_CONTEXT"
	LocalRunContextSchema      = uint32(1)
	LocalRunAdapterVersion     = "local-v0"
)

var ErrInvalidLocalRunContext = errors.New("invalid Codex local-run context")

type LocalRunContext struct {
	workSpecID     domain.WorkSpecID
	workSpecDigest domain.WorkSpecDigest
	runID          domain.RunID
	repoRoot       string
}

type localRunContextWire struct {
	SchemaVersion  uint32                `json:"schema_version"`
	WorkSpecID     domain.WorkSpecID     `json:"work_spec_id"`
	WorkSpecDigest domain.WorkSpecDigest `json:"work_spec_digest"`
	RunID          domain.RunID          `json:"run_id"`
	RepoRoot       string                `json:"repo_root"`
}

func NewLocalRunContext(
	workSpecID domain.WorkSpecID,
	workSpecDigest domain.WorkSpecDigest,
	runID domain.RunID,
	repoRoot string,
) (LocalRunContext, error) {
	if !domain.IsCanonicalID(string(workSpecID)) || !domain.IsCanonicalID(string(runID)) {
		return LocalRunContext{}, fmt.Errorf("%w: WorkSpec and run IDs must be canonical", ErrInvalidLocalRunContext)
	}
	if !validLocalRunDigest(string(workSpecDigest)) {
		return LocalRunContext{}, fmt.Errorf("%w: WorkSpec digest must be SHA-256", ErrInvalidLocalRunContext)
	}
	if !filepath.IsAbs(repoRoot) || strings.ContainsRune(repoRoot, '\x00') {
		return LocalRunContext{}, fmt.Errorf("%w: repository root must be absolute", ErrInvalidLocalRunContext)
	}
	repoRoot = filepath.Clean(repoRoot)
	if repoRoot == string(filepath.Separator) {
		return LocalRunContext{}, fmt.Errorf("%w: filesystem root is not a repository boundary", ErrInvalidLocalRunContext)
	}
	return LocalRunContext{
		workSpecID:     workSpecID,
		workSpecDigest: workSpecDigest,
		runID:          runID,
		repoRoot:       repoRoot,
	}, nil
}

func DecodeLocalRunContext(encoded []byte) (LocalRunContext, error) {
	var wire localRunContextWire
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return LocalRunContext{}, fmt.Errorf("decode local-run context: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return LocalRunContext{}, fmt.Errorf("decode local-run context: multiple JSON values")
		}
		return LocalRunContext{}, fmt.Errorf("decode local-run context: %w", err)
	}
	if wire.SchemaVersion != LocalRunContextSchema {
		return LocalRunContext{}, fmt.Errorf("unsupported local-run context schema %d", wire.SchemaVersion)
	}
	return NewLocalRunContext(wire.WorkSpecID, wire.WorkSpecDigest, wire.RunID, wire.RepoRoot)
}

func (runContext LocalRunContext) EnvironmentValue() (string, error) {
	wire, err := runContext.wire()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return "", fmt.Errorf("encode local-run context: %w", err)
	}
	return string(encoded), nil
}

func (runContext LocalRunContext) Digest() (string, error) {
	wire, err := runContext.wire()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return "", fmt.Errorf("encode local-run context digest: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (runContext LocalRunContext) WorkSpecID() domain.WorkSpecID {
	return runContext.workSpecID
}

func (runContext LocalRunContext) WorkSpecDigest() domain.WorkSpecDigest {
	return runContext.workSpecDigest
}

func (runContext LocalRunContext) RunID() domain.RunID {
	return runContext.runID
}

func (runContext LocalRunContext) RepoRoot() string {
	return runContext.repoRoot
}

func (runContext LocalRunContext) wire() (localRunContextWire, error) {
	validated, err := NewLocalRunContext(
		runContext.workSpecID,
		runContext.workSpecDigest,
		runContext.runID,
		runContext.repoRoot,
	)
	if err != nil {
		return localRunContextWire{}, err
	}
	return localRunContextWire{
		SchemaVersion:  LocalRunContextSchema,
		WorkSpecID:     validated.workSpecID,
		WorkSpecDigest: validated.workSpecDigest,
		RunID:          validated.runID,
		RepoRoot:       validated.repoRoot,
	}, nil
}

// BindLocalRunLifecycleHook validates the WorkSpec-bound adapter context, then
// delegates provider-hook and sticky-identity rules to the existing Codex
// correlation implementation.
func BindLocalRunLifecycleHook(
	expected LocalRunContext,
	encodedContext string,
	rawHook []byte,
	previous *CorrelationResult,
) (CorrelationResult, error) {
	if _, err := expected.wire(); err != nil {
		return CorrelationResult{}, err
	}
	legacy, err := NewRunContext(
		domain.ChangeID(expected.workSpecID),
		expected.runID,
		expected.repoRoot,
	)
	if err != nil {
		return CorrelationResult{}, err
	}
	if strings.TrimSpace(encodedContext) == "" {
		return BindLifecycleHook(legacy, "", rawHook, previous)
	}
	observed, err := DecodeLocalRunContext([]byte(encodedContext))
	if err != nil {
		return BindLifecycleHook(legacy, "{", rawHook, previous)
	}
	expectedDigest, err := expected.Digest()
	if err != nil {
		return CorrelationResult{}, err
	}
	observedDigest, err := observed.Digest()
	if err != nil || observedDigest != expectedDigest {
		conflicting, buildErr := NewRunContext(
			domain.ChangeID(expected.workSpecID),
			"run-context-conflict",
			expected.repoRoot,
		)
		if buildErr != nil {
			return CorrelationResult{}, buildErr
		}
		conflictingEncoded, buildErr := conflicting.EnvironmentValue()
		if buildErr != nil {
			return CorrelationResult{}, buildErr
		}
		return BindLifecycleHook(legacy, conflictingEncoded, rawHook, previous)
	}
	legacyEncoded, err := legacy.EnvironmentValue()
	if err != nil {
		return CorrelationResult{}, err
	}
	return BindLifecycleHook(legacy, legacyEncoded, rawHook, previous)
}

type ProcessRequest struct {
	Directory   string
	Arguments   []string
	Environment []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

type ProcessResult struct {
	Outcome       localrun.ProviderOutcome
	ExitCode      int
	LifecycleHook []byte
	Err           error
}

type ProcessLauncher interface {
	Launch(context.Context, ProcessRequest) ProcessResult
}

// LocalRunAdapter has no operating-system launcher. A later activation change
// must provide and deliberately wire one.
type LocalRunAdapter struct {
	launcher        ProcessLauncher
	baseEnvironment []string
}

func NewLocalRunAdapter(
	launcher ProcessLauncher,
	baseEnvironment []string,
) (*LocalRunAdapter, error) {
	if launcher == nil {
		return nil, fmt.Errorf("Codex process launcher is required")
	}
	for _, variable := range baseEnvironment {
		name, _, found := strings.Cut(variable, "=")
		if !found || strings.TrimSpace(name) == "" || prohibitedEnvironmentName(name) {
			return nil, fmt.Errorf("Codex base environment contains an invalid or credential-shaped variable")
		}
	}
	return &LocalRunAdapter{
		launcher:        launcher,
		baseEnvironment: append([]string(nil), baseEnvironment...),
	}, nil
}

func (*LocalRunAdapter) Name() string    { return "codex" }
func (*LocalRunAdapter) Version() string { return LocalRunAdapterVersion }

// VerifyAnnouncementDelivery reports that this adapter can tell a run the
// escalation channel exists. Delivery reuses the SessionStart hook Goalrail
// already renders for lineage: the announcement travels the invocation
// environment to the capsule, which returns it as the session's additional
// context.
func (*LocalRunAdapter) VerifyAnnouncementDelivery() error { return nil }

func (adapter *LocalRunAdapter) Launch(
	ctx context.Context,
	request localrun.LaunchRequest,
) localrun.ProviderObservation {
	spec := request.WorkSpec.Spec()
	runContext, err := NewLocalRunContext(
		spec.ID,
		request.WorkSpec.Digest(),
		request.RunID,
		spec.Repository.Root,
	)
	if err != nil {
		return invalidLaunchObservation("INVALID_RUN_CONTEXT")
	}
	encodedContext, err := runContext.EnvironmentValue()
	if err != nil {
		return invalidLaunchObservation("INVALID_RUN_CONTEXT")
	}
	if request.EscalationAnnouncement == "" {
		return invalidLaunchObservation("MISSING_ESCALATION_ANNOUNCEMENT")
	}
	environment := append([]string(nil), adapter.baseEnvironment...)
	environment = append(environment, LocalRunContextEnvironment+"="+encodedContext)
	environment = append(
		environment,
		EscalationAnnouncementEnvironment+"="+request.EscalationAnnouncement,
	)
	processResult := adapter.launcher.Launch(ctx, ProcessRequest{
		Directory:   spec.Repository.Root,
		Arguments:   append([]string(nil), request.Arguments...),
		Environment: environment,
		Stdin:       request.Stdin,
		Stdout:      request.Stdout,
		Stderr:      request.Stderr,
	})
	outcome := processResult.Outcome
	if processResult.Err != nil {
		outcome = localrun.ProviderLaunchFailed
	}
	switch outcome {
	case localrun.ProviderCompleted, localrun.ProviderDenied, localrun.ProviderLaunchFailed:
	default:
		outcome = localrun.ProviderLaunchFailed
	}

	correlation, bindErr := BindLocalRunLifecycleHook(
		runContext,
		encodedContext,
		processResult.LifecycleHook,
		nil,
	)
	if bindErr != nil {
		return invalidLaunchObservation("INVALID_RUN_CONTEXT")
	}
	observation := localrun.ProviderObservation{
		Outcome:  outcome,
		ExitCode: processResult.ExitCode,
	}
	if correlation.Verified() {
		observation.IdentityStatus = localrun.IdentityVerified
		observation.RootSessionRef = string(correlation.Lineage.RootSessionID)
	} else {
		observation.IdentityStatus = localrun.IdentityUnlinked
		observation.Reason = correlation.Lineage.UnlinkedReasonCode
	}
	return observation
}

func invalidLaunchObservation(reason domain.EvidenceReasonCode) localrun.ProviderObservation {
	return localrun.ProviderObservation{
		Outcome:        localrun.ProviderLaunchFailed,
		IdentityStatus: localrun.IdentityUnlinked,
		Reason:         reason,
	}
}

func validLocalRunDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && value == strings.ToLower(value)
}

func prohibitedEnvironmentName(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	for _, marker := range []string{"TOKEN", "SECRET", "PASSWORD", "API_KEY", "PRIVATE_KEY", "CREDENTIAL"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}
