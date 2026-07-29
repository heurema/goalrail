package localrun

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

var (
	ErrActivationRequired    = errors.New("ACTIVATION_REQUIRED")
	ErrLaunchAlreadyClaimed  = errors.New("WorkSpec launch was already attempted")
	ErrInvalidProviderResult = errors.New("invalid provider observation")
	ErrTerminalReceiptExists = errors.New("terminal receipt already exists")
)

type PreparedRun struct {
	WorkSpec    domain.FrozenWorkSpec `json:"-"`
	Preparation Preparation           `json:"preparation"`
	Claim       *LaunchClaim          `json:"claim,omitempty"`
	State       RunState              `json:"state"`
}

type StartInput struct {
	WorkSpecDigest domain.WorkSpecDigest
	Adapter        string
	Arguments      []string
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
}

type StartResult struct {
	Claim       LaunchClaim         `json:"claim"`
	Observation ProviderObservation `json:"observation"`
	State       RunState            `json:"state"`
	Receipt     *TerminalReceipt    `json:"receipt,omitempty"`
}

type FinishInput struct {
	RunID   domain.RunID
	Results []CheckResult
}

type RunInspection struct {
	Claim       LaunchClaim          `json:"claim"`
	Observation *ProviderObservation `json:"observation,omitempty"`
	Receipt     *TerminalReceipt     `json:"receipt,omitempty"`
	State       RunState             `json:"state"`
}

type Service struct {
	store     *Store
	observer  WorktreeObserver
	intent    IntentVerifier
	adapter   Adapter
	admission *dogfoodAdmission
	now       func() time.Time
	newRunID  func() (domain.RunID, error)
}

func NewService(store *Store, observer WorktreeObserver, intent IntentVerifier) *Service {
	return &Service{
		store:    store,
		observer: observer,
		intent:   intent,
		now:      func() time.Time { return time.Now().UTC() },
		newRunID: randomRunID,
	}
}

// NewFixtureService enables only the in-memory fixture adapter used by local
// tests. Production command wiring never calls this constructor.
func NewFixtureService(
	store *Store,
	observer WorktreeObserver,
	intent IntentVerifier,
	adapter Adapter,
) (*Service, error) {
	if adapter == nil || adapter.Name() != "fixture" {
		return nil, fmt.Errorf("fixture service requires the fixture adapter")
	}
	service := NewService(store, observer, intent)
	service.adapter = adapter
	return service, nil
}

// NewDogfoodService enables only the Codex adapter behind the fixed,
// one-time admission record for activate-dogfood-run-v0. It does not create
// the admission record or expose a reusable activation control.
func NewDogfoodService(
	store *Store,
	observer WorktreeObserver,
	intent IntentVerifier,
	adapter Adapter,
) (*Service, error) {
	if adapter == nil || adapter.Name() != "codex" {
		return nil, fmt.Errorf("dogfood service requires the Codex adapter")
	}
	admission, err := newDogfoodAdmission(store)
	if err != nil {
		return nil, err
	}
	service := NewService(store, observer, intent)
	service.adapter = adapter
	service.admission = admission
	return service, nil
}

func (service *Service) Prepare(ctx context.Context, reader io.Reader) (PreparedRun, error) {
	if service.store == nil || service.observer == nil || service.intent == nil {
		return PreparedRun{}, fmt.Errorf("local-run service dependencies are incomplete")
	}
	spec, err := domain.DecodeWorkSpec(reader)
	if err != nil {
		return PreparedRun{}, err
	}
	root, revision, err := service.observer.ResolveRepository(
		ctx,
		spec.Repository.Root,
		spec.Repository.BaseRevision,
	)
	if err != nil {
		return PreparedRun{}, err
	}
	if pathWithin(root, service.store.Root()) {
		return PreparedRun{}, fmt.Errorf("Goalrail state root must remain outside the target repository")
	}
	spec.Repository.Root = root
	spec.Repository.BaseRevision = revision
	if err := validateScopedPathBoundaries(root, spec.Paths); err != nil {
		return PreparedRun{}, err
	}
	// The reserved escalation path gets the same boundary treatment as a
	// declared scoped path. Without it, a `.goalrail` symlink pointing outside
	// the repository would let a provider write the question into an external
	// directory that observation never sees, silently killing the channel.
	if err := validateScopedPathBoundaries(root, []string{ReservedEscalationPath}); err != nil {
		return PreparedRun{}, err
	}
	frozen, err := domain.FreezeWorkSpec(spec)
	if err != nil {
		return PreparedRun{}, err
	}
	if err := service.intent.Verify(root, spec.Intent); err != nil {
		return PreparedRun{}, err
	}

	if prepared, err := service.InspectPrepared(frozen.Digest()); err == nil {
		if !bytes.Equal(prepared.WorkSpec.CanonicalJSON(), frozen.CanonicalJSON()) {
			return PreparedRun{}, fmt.Errorf("prepared WorkSpec digest collision")
		}
		return prepared, nil
	} else if !errors.Is(err, ErrStateNotFound) {
		return PreparedRun{}, err
	}

	baseline, err := service.observer.Observe(ctx, root)
	if err != nil {
		return PreparedRun{}, err
	}
	if observationHasEscalation(baseline) {
		return PreparedRun{}, fmt.Errorf(
			"%w: %s is already populated at the frozen baseline",
			ErrEscalationArtifactPresent,
			ReservedEscalationPath,
		)
	}
	preparation := Preparation{
		WorkSpecDigest: frozen.Digest(),
		PreparedAt:     service.now(),
		Baseline:       baseline,
		State:          StatePrepared,
	}
	if err := service.store.WriteBytesOnce(
		preparedPath(frozen.Digest(), "work-spec.json"),
		frozen.CanonicalJSON(),
		true,
	); err != nil {
		return PreparedRun{}, err
	}
	if err := service.store.WriteJSONOnce(
		preparedPath(frozen.Digest(), "preparation.json"),
		preparation,
		false,
	); err != nil {
		return PreparedRun{}, err
	}
	return PreparedRun{
		WorkSpec:    frozen,
		Preparation: preparation,
		State:       StatePrepared,
	}, nil
}

func (service *Service) InspectPrepared(digest domain.WorkSpecDigest) (PreparedRun, error) {
	if !validDigest(string(digest)) {
		return PreparedRun{}, fmt.Errorf("invalid WorkSpec digest")
	}
	canonical, err := service.store.ReadBytes(preparedPath(digest, "work-spec.json"))
	if err != nil {
		return PreparedRun{}, err
	}
	frozen, err := domain.OpenFrozenWorkSpec(canonical, digest)
	if err != nil {
		return PreparedRun{}, err
	}
	var preparation Preparation
	if err := service.store.ReadJSON(preparedPath(digest, "preparation.json"), &preparation); err != nil {
		return PreparedRun{}, err
	}
	if preparation.WorkSpecDigest != digest || preparation.State != StatePrepared {
		return PreparedRun{}, fmt.Errorf("invalid preparation state")
	}
	prepared := PreparedRun{
		WorkSpec:    frozen,
		Preparation: preparation,
		State:       StatePrepared,
	}
	var claim LaunchClaim
	if err := service.store.ReadJSON(preparedPath(digest, "launch-claim.json"), &claim); err == nil {
		if err := validateLaunchClaim(claim); err != nil || claim.WorkSpecDigest != digest {
			return PreparedRun{}, fmt.Errorf("invalid stored launch claim")
		}
		prepared.Claim = &claim
		prepared.State = StateLaunchAttemptedUnknown
		if inspection, inspectErr := service.InspectRun(claim.RunID); inspectErr == nil {
			prepared.State = inspection.State
		} else if !errors.Is(inspectErr, ErrStateNotFound) {
			return PreparedRun{}, inspectErr
		}
	} else if !errors.Is(err, ErrStateNotFound) {
		return PreparedRun{}, err
	}
	return prepared, nil
}

func (service *Service) Start(ctx context.Context, input StartInput) (StartResult, error) {
	if service.adapter == nil {
		return StartResult{}, ErrActivationRequired
	}
	if input.Adapter != service.adapter.Name() {
		return StartResult{}, fmt.Errorf("prepared service has no adapter %q", input.Adapter)
	}
	prepared, err := service.InspectPrepared(input.WorkSpecDigest)
	if err != nil {
		return StartResult{}, err
	}
	if prepared.Claim != nil {
		return StartResult{}, ErrLaunchAlreadyClaimed
	}
	if service.adapter.Name() != "fixture" {
		if service.observer == nil || service.admission == nil {
			return StartResult{}, ErrActivationRequired
		}
		spec := prepared.WorkSpec.Spec()
		root, revision, err := service.observer.ResolveRepository(
			ctx,
			spec.Repository.Root,
			spec.Repository.BaseRevision,
		)
		if err != nil ||
			root != spec.Repository.Root ||
			revision != spec.Repository.BaseRevision {
			return StartResult{}, ErrActivationRequired
		}
		if err := service.admission.consume(prepared.WorkSpec); err != nil {
			return StartResult{}, err
		}
	}
	// A prepared run is reused rather than re-observed, and it can be started
	// long after preparation froze its baseline. An artifact that appears in
	// that window belongs to no provider, so it must not reach a launch.
	present, err := escalationArtifactPresent(prepared.WorkSpec.Spec().Repository.Root)
	if err != nil {
		return StartResult{}, err
	}
	if present {
		return StartResult{}, fmt.Errorf(
			"%w: %s was populated after preparation and before launch",
			ErrEscalationArtifactPresent,
			ReservedEscalationPath,
		)
	}
	runID, err := service.newRunID()
	if err != nil {
		return StartResult{}, err
	}
	if !domain.IsCanonicalID(string(runID)) {
		return StartResult{}, fmt.Errorf("generated run ID is not canonical")
	}
	claim := LaunchClaim{
		WorkSpecDigest: input.WorkSpecDigest,
		RunID:          runID,
		Adapter:        service.adapter.Name(),
		AdapterVersion: service.adapter.Version(),
		AttemptedAt:    service.now(),
	}
	if err := validateLaunchClaim(claim); err != nil {
		return StartResult{}, err
	}
	if err := service.store.WriteJSONOnce(
		preparedPath(input.WorkSpecDigest, "launch-claim.json"),
		claim,
		false,
	); err != nil {
		if errors.Is(err, ErrStateAlreadyExists) {
			return StartResult{}, ErrLaunchAlreadyClaimed
		}
		return StartResult{}, err
	}
	if err := service.store.WriteJSONOnce(runPath(runID, "launch-claim.json"), claim, false); err != nil {
		return StartResult{}, err
	}

	observation := service.adapter.Launch(ctx, LaunchRequest{
		RunID:     runID,
		WorkSpec:  prepared.WorkSpec,
		Arguments: append([]string(nil), input.Arguments...),
		Stdin:     input.Stdin,
		Stdout:    input.Stdout,
		Stderr:    input.Stderr,
	})
	observation.WorkSpecDigest = input.WorkSpecDigest
	observation.RunID = runID
	observation.Adapter = service.adapter.Name()
	observation.AdapterVersion = service.adapter.Version()
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = service.now()
	}
	if err := validateProviderObservation(observation); err != nil {
		observation = ProviderObservation{
			WorkSpecDigest: input.WorkSpecDigest,
			RunID:          runID,
			Adapter:        service.adapter.Name(),
			AdapterVersion: service.adapter.Version(),
			Outcome:        ProviderLaunchFailed,
			IdentityStatus: IdentityUnlinked,
			Reason:         "INVALID_PROVIDER_OBSERVATION",
			ObservedAt:     service.now(),
		}
	}
	if err := service.store.WriteJSONOnce(
		runPath(runID, "provider-observation.json"),
		observation,
		false,
	); err != nil {
		return StartResult{}, err
	}

	// One observation decides both the escalation artifact and the run's delta.
	// Observing again at finish would leave a window in which the artifact could
	// be substituted between the read that classifies it and the read that
	// computes the delta.
	terminal, err := service.observer.Observe(ctx, prepared.WorkSpec.Spec().Repository.Root)
	if err != nil {
		return StartResult{}, err
	}
	// Retention happens before the observation is persisted. A persisted
	// observation naming the reserved path without a retained record would let a
	// later Finish treat the question as absent.
	escalation, err := service.captureEscalation(
		prepared.WorkSpec.Spec().Repository.Root,
		runID,
		terminal,
	)
	if err != nil {
		return StartResult{}, err
	}
	if escalation != nil {
		if err := service.store.WriteJSONOnce(
			runPath(runID, "escalation.json"),
			escalation,
			false,
		); err != nil {
			return StartResult{}, err
		}
	}
	if err := service.store.WriteJSONOnce(
		runPath(runID, "terminal-observation.json"),
		terminal,
		false,
	); err != nil {
		return StartResult{}, err
	}

	result := StartResult{
		Claim:       claim,
		Observation: observation,
		State:       StateAwaitingVerification,
	}
	if observation.Outcome == ProviderCompleted && observation.IdentityStatus == IdentityVerified {
		return result, nil
	}
	receipt, err := service.failureReceipt(prepared, claim, observation, terminal, escalation)
	if err != nil {
		return StartResult{}, err
	}
	result.State = receipt.Status
	result.Receipt = &receipt
	return result, nil
}

func (service *Service) Finish(ctx context.Context, input FinishInput) (TerminalReceipt, error) {
	inspection, err := service.InspectRun(input.RunID)
	if err != nil {
		return TerminalReceipt{}, err
	}
	if inspection.Receipt != nil {
		return TerminalReceipt{}, ErrTerminalReceiptExists
	}
	if inspection.Observation == nil {
		return TerminalReceipt{}, fmt.Errorf("run is %s and has no provider observation", inspection.State)
	}
	observation := *inspection.Observation
	if observation.Outcome != ProviderCompleted || observation.IdentityStatus != IdentityVerified {
		return TerminalReceipt{}, fmt.Errorf("run is not awaiting verification")
	}
	prepared, err := service.InspectPrepared(inspection.Claim.WorkSpecDigest)
	if err != nil {
		return TerminalReceipt{}, err
	}
	results, status, reasons, err := normalizeCheckResults(prepared.WorkSpec.Spec().Checks, input.Results)
	if err != nil {
		return TerminalReceipt{}, err
	}
	terminal, err := service.terminalObservation(ctx, input.RunID, prepared.WorkSpec.Spec().Repository.Root)
	if err != nil {
		return TerminalReceipt{}, err
	}
	escalation, err := service.retainedEscalation(input.RunID)
	if err != nil {
		return TerminalReceipt{}, err
	}
	// Fail closed rather than treat an unretained question as no question. This
	// is reachable when retention failed after the observation was taken, and
	// without it the run could still be finished as passed.
	if escalation == nil && observationHasEscalation(terminal) {
		return TerminalReceipt{}, fmt.Errorf(
			"run observed %s but retained no escalation record",
			ReservedEscalationPath,
		)
	}
	delta := CompareWorktrees(prepared.Preparation.Baseline, terminal, prepared.WorkSpec.Spec().Paths)
	if delta.HeadChanged {
		status = StateFailed
		reasons = appendReason(reasons, "HEAD_CHANGED")
	}
	if len(delta.ScopeViolations) != 0 {
		status = StateFailed
		reasons = appendReason(reasons, "SCOPE_VIOLATION")
	}
	status, reasons = applyEscalationRules(
		status,
		reasons,
		escalation,
		delta,
		prepared.WorkSpec.Spec().Paths,
	)
	receipt := buildReceipt(
		prepared,
		inspection.Claim,
		observation,
		results,
		status,
		reasons,
		terminal,
		delta,
		escalation,
		service.now(),
	)
	if err := validateTerminalReceipt(receipt); err != nil {
		return TerminalReceipt{}, err
	}
	if err := service.store.WriteJSONOnce(runPath(input.RunID, "terminal-receipt.json"), receipt, false); err != nil {
		if errors.Is(err, ErrStateAlreadyExists) {
			return TerminalReceipt{}, ErrTerminalReceiptExists
		}
		return TerminalReceipt{}, err
	}
	return receipt, nil
}

func (service *Service) InspectRun(runID domain.RunID) (RunInspection, error) {
	if !domain.IsCanonicalID(string(runID)) {
		return RunInspection{}, fmt.Errorf("invalid run ID")
	}
	var claim LaunchClaim
	if err := service.store.ReadJSON(runPath(runID, "launch-claim.json"), &claim); err != nil {
		return RunInspection{}, err
	}
	if err := validateLaunchClaim(claim); err != nil || claim.RunID != runID {
		return RunInspection{}, fmt.Errorf("invalid stored launch claim")
	}
	inspection := RunInspection{Claim: claim, State: StateLaunchAttemptedUnknown}
	var observation ProviderObservation
	if err := service.store.ReadJSON(runPath(runID, "provider-observation.json"), &observation); err == nil {
		if err := validateProviderObservation(observation); err != nil ||
			observation.RunID != claim.RunID ||
			observation.WorkSpecDigest != claim.WorkSpecDigest ||
			observation.Adapter != claim.Adapter ||
			observation.AdapterVersion != claim.AdapterVersion {
			return RunInspection{}, fmt.Errorf("invalid stored provider observation")
		}
		inspection.Observation = &observation
		inspection.State = stateForObservation(observation)
	} else if !errors.Is(err, ErrStateNotFound) {
		return RunInspection{}, err
	}
	var receipt TerminalReceipt
	if err := service.store.ReadJSON(runPath(runID, "terminal-receipt.json"), &receipt); err == nil {
		if err := validateTerminalReceipt(receipt); err != nil ||
			receipt.RunID != claim.RunID ||
			receipt.WorkSpecDigest != claim.WorkSpecDigest ||
			receipt.Adapter != claim.Adapter ||
			receipt.AdapterVersion != claim.AdapterVersion {
			return RunInspection{}, fmt.Errorf("invalid stored terminal receipt")
		}
		inspection.Receipt = &receipt
		inspection.State = receipt.Status
	} else if !errors.Is(err, ErrStateNotFound) {
		return RunInspection{}, err
	}
	return inspection, nil
}

// failureReceipt records a terminal outcome that Start already decided. It uses
// the observation Start took rather than taking its own, so the artifact and the
// delta come from one snapshot.
//
// An unlinked, denied, or launch-failed outcome takes precedence over blocked:
// an unattributable session must not mint an authoritative question. The
// escalation is still recorded as evidence.
func (service *Service) failureReceipt(
	prepared PreparedRun,
	claim LaunchClaim,
	observation ProviderObservation,
	terminal WorktreeObservation,
	escalation *EscalationRecord,
) (TerminalReceipt, error) {
	spec := prepared.WorkSpec.Spec()
	delta := CompareWorktrees(prepared.Preparation.Baseline, terminal, spec.Paths)
	results := make([]CheckResult, 0, len(spec.Checks))
	for _, check := range spec.Checks {
		results = append(results, CheckResult{ID: check.ID, State: domain.CheckResultUnavailable})
	}
	status := stateForObservation(observation)
	reasons := make([]domain.EvidenceReasonCode, 0, 2)
	if observation.Reason != "" {
		reasons = append(reasons, observation.Reason)
	}
	if delta.HeadChanged {
		reasons = appendReason(reasons, "HEAD_CHANGED")
	}
	if len(delta.ScopeViolations) != 0 {
		reasons = appendReason(reasons, "SCOPE_VIOLATION")
	}
	if escalation != nil {
		reasons = appendReason(reasons, "ESCALATION_RECORDED")
	}
	receipt := buildReceipt(
		prepared,
		claim,
		observation,
		results,
		status,
		reasons,
		terminal,
		delta,
		escalation,
		service.now(),
	)
	if err := validateTerminalReceipt(receipt); err != nil {
		return TerminalReceipt{}, err
	}
	if err := service.store.WriteJSONOnce(runPath(claim.RunID, "terminal-receipt.json"), receipt, false); err != nil {
		return TerminalReceipt{}, err
	}
	return receipt, nil
}

func buildReceipt(
	prepared PreparedRun,
	claim LaunchClaim,
	observation ProviderObservation,
	results []CheckResult,
	status RunState,
	reasons []domain.EvidenceReasonCode,
	terminal WorktreeObservation,
	delta WorktreeDelta,
	escalation *EscalationRecord,
	terminalAt time.Time,
) TerminalReceipt {
	spec := prepared.WorkSpec.Spec()
	receipt := TerminalReceipt{
		Schema:          TerminalReceiptSchemaV1,
		WorkSpecID:      spec.ID,
		WorkSpecVersion: spec.Version,
		WorkSpecDigest:  prepared.WorkSpec.Digest(),
		Intent: &ReceiptIntentReference{
			ID:      spec.Intent.ID,
			Version: spec.Intent.Version,
			Digest:  spec.Intent.Digest,
		},
		Escalation:         escalation,
		RunID:              claim.RunID,
		Adapter:            claim.Adapter,
		AdapterVersion:     claim.AdapterVersion,
		BaseRevision:       spec.Repository.BaseRevision,
		TerminalHead:       terminal.Head,
		EffectivePaths:     append([]string(nil), spec.Paths...),
		Checks:             append([]domain.WorkSpecCheck(nil), spec.Checks...),
		CheckResults:       append([]CheckResult(nil), results...),
		Status:             status,
		Reasons:            sortedReasons(reasons),
		WorktreeDelta:      delta,
		PreparedAt:         prepared.Preparation.PreparedAt,
		LaunchAttemptedAt:  claim.AttemptedAt,
		ProviderObservedAt: observation.ObservedAt,
		TerminalAt:         terminalAt,
	}
	if observation.IdentityStatus == IdentityVerified {
		receipt.RootSessionRef = observation.RootSessionRef
	} else {
		receipt.UnlinkedReason = observation.Reason
	}
	return receipt
}

func normalizeCheckResults(
	checks []domain.WorkSpecCheck,
	supplied []CheckResult,
) ([]CheckResult, RunState, []domain.EvidenceReasonCode, error) {
	known := make(map[domain.WorkSpecCheckID]struct{}, len(checks))
	for _, check := range checks {
		known[check.ID] = struct{}{}
	}
	byID := make(map[domain.WorkSpecCheckID]CheckResult, len(supplied))
	for _, result := range supplied {
		if _, exists := known[result.ID]; !exists {
			return nil, "", nil, fmt.Errorf("unknown check result %q", result.ID)
		}
		if _, exists := byID[result.ID]; exists {
			return nil, "", nil, fmt.Errorf("duplicate check result %q", result.ID)
		}
		if err := validateCheckResult(result); err != nil {
			return nil, "", nil, err
		}
		byID[result.ID] = result
	}

	status := StatePassed
	reasons := make([]domain.EvidenceReasonCode, 0)
	results := make([]CheckResult, 0, len(checks))
	for _, check := range checks {
		result, exists := byID[check.ID]
		if !exists {
			result = CheckResult{ID: check.ID, State: domain.CheckResultMissing}
		}
		switch result.State {
		case domain.CheckResultPass:
		case domain.CheckResultFail:
			status = StateFailed
			reasons = appendReason(reasons, "CHECK_FAILED")
		case domain.CheckResultUnavailable, domain.CheckResultMissing:
			if status != StateFailed {
				status = StateVerificationIncomplete
			}
			reasons = appendReason(reasons, "CHECK_RESULT_INCOMPLETE")
		}
		results = append(results, result)
	}
	return results, status, reasons, nil
}

func validateCheckResult(result CheckResult) error {
	if !domain.IsCanonicalID(string(result.ID)) {
		return fmt.Errorf("check result ID must be canonical")
	}
	switch result.State {
	case domain.CheckResultPass, domain.CheckResultFail, domain.CheckResultUnavailable:
	default:
		return fmt.Errorf("unsupported check result state %q", result.State)
	}
	if result.EvidenceRef != "" {
		if len(result.EvidenceRef) > domain.MaxEvidenceResultBytes || !domain.IsEvidenceReference(result.EvidenceRef) {
			return fmt.Errorf("check evidence reference is invalid")
		}
	}
	if result.EvidenceDigest != "" && !validDigest(result.EvidenceDigest) {
		return fmt.Errorf("check evidence digest is invalid")
	}
	return nil
}

func validateLaunchClaim(claim LaunchClaim) error {
	if !validDigest(string(claim.WorkSpecDigest)) ||
		!domain.IsCanonicalID(string(claim.RunID)) ||
		!domain.IsCanonicalID(claim.Adapter) ||
		strings.TrimSpace(claim.AdapterVersion) == "" ||
		len(claim.AdapterVersion) > 128 ||
		claim.AttemptedAt.IsZero() {
		return fmt.Errorf("invalid launch claim")
	}
	return nil
}

func validateProviderObservation(observation ProviderObservation) error {
	if !validDigest(string(observation.WorkSpecDigest)) ||
		!domain.IsCanonicalID(string(observation.RunID)) ||
		!domain.IsCanonicalID(observation.Adapter) ||
		strings.TrimSpace(observation.AdapterVersion) == "" ||
		len(observation.AdapterVersion) > 128 ||
		observation.ObservedAt.IsZero() {
		return ErrInvalidProviderResult
	}
	switch observation.Outcome {
	case ProviderCompleted, ProviderDenied, ProviderLaunchFailed:
	default:
		return ErrInvalidProviderResult
	}
	switch observation.IdentityStatus {
	case IdentityVerified:
		if !domain.IsCanonicalID(observation.RootSessionRef) || observation.Reason != "" {
			return ErrInvalidProviderResult
		}
	case IdentityUnlinked:
		if observation.RootSessionRef != "" || !domain.IsCanonicalID(string(observation.Reason)) {
			return ErrInvalidProviderResult
		}
	default:
		return ErrInvalidProviderResult
	}
	return nil
}

func validateTerminalReceipt(receipt TerminalReceipt) error {
	if !domain.IsCanonicalID(string(receipt.WorkSpecID)) ||
		receipt.WorkSpecVersion == 0 ||
		!validDigest(string(receipt.WorkSpecDigest)) ||
		!domain.IsCanonicalID(string(receipt.RunID)) ||
		!domain.IsCanonicalID(receipt.Adapter) ||
		receipt.BaseRevision == "" ||
		receipt.TerminalHead == "" ||
		receipt.PreparedAt.IsZero() ||
		receipt.LaunchAttemptedAt.IsZero() ||
		receipt.ProviderObservedAt.IsZero() ||
		receipt.TerminalAt.IsZero() {
		return fmt.Errorf("terminal receipt has invalid required metadata")
	}
	// An empty schema identifies a receipt written before the identifier
	// existed. Such a receipt stays readable and is never rewritten, so the
	// intent reference is required only from v1 onwards.
	switch receipt.Schema {
	case "":
	case TerminalReceiptSchemaV1:
		if receipt.Intent == nil ||
			!domain.IsCanonicalID(string(receipt.Intent.ID)) ||
			receipt.Intent.Version == 0 ||
			!validDigest(receipt.Intent.Digest) {
			return fmt.Errorf("terminal receipt lacks a usable frozen intent reference")
		}
	default:
		return fmt.Errorf("terminal receipt has unsupported schema %q", receipt.Schema)
	}
	switch receipt.Status {
	case StatePassed, StateFailed, StateVerificationIncomplete, StateLaunchFailed, StateUnlinked:
	case StateBlocked:
		if receipt.Escalation == nil || !receipt.Escalation.Valid {
			return fmt.Errorf("blocked receipt requires a valid retained escalation")
		}
	default:
		return fmt.Errorf("terminal receipt has non-terminal status %q", receipt.Status)
	}
	if receipt.Escalation != nil {
		if receipt.Escalation.Path != ReservedEscalationPath {
			return fmt.Errorf("terminal receipt escalation names an unexpected path")
		}
		if receipt.Escalation.Valid &&
			(!validDigest(receipt.Escalation.Digest) || receipt.Escalation.RetainedRef == "") {
			return fmt.Errorf("terminal receipt escalation lacks retained evidence")
		}
		if receipt.Status == StatePassed {
			return fmt.Errorf("terminal receipt cannot pass while an escalation is retained")
		}
	}
	if len(receipt.Checks) != len(receipt.CheckResults) {
		return fmt.Errorf("terminal receipt check results do not match the frozen check set")
	}
	for index, check := range receipt.Checks {
		if receipt.CheckResults[index].ID != check.ID {
			return fmt.Errorf("terminal receipt check result order differs from the frozen check set")
		}
	}
	if receipt.WorktreeDelta.HeadChanged && receipt.Status == StatePassed {
		return fmt.Errorf("terminal receipt cannot pass after repository HEAD changed")
	}
	return nil
}

func stateForObservation(observation ProviderObservation) RunState {
	if observation.IdentityStatus == IdentityUnlinked {
		return StateUnlinked
	}
	switch observation.Outcome {
	case ProviderCompleted:
		return StateAwaitingVerification
	case ProviderDenied:
		return StateFailed
	default:
		return StateLaunchFailed
	}
}

func validateScopedPathBoundaries(root string, paths []string) error {
	for _, relative := range paths {
		candidate := filepath.Join(root, filepath.FromSlash(relative))
		existing := candidate
		for {
			_, err := os.Lstat(existing)
			if err == nil {
				break
			}
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("inspect scoped path %q: %w", relative, err)
			}
			parent := filepath.Dir(existing)
			if parent == existing {
				return fmt.Errorf("resolve scoped path %q", relative)
			}
			existing = parent
		}
		resolved, err := filepath.EvalSymlinks(existing)
		if err != nil {
			return fmt.Errorf("resolve scoped path %q: %w", relative, err)
		}
		if !pathWithin(root, resolved) {
			return fmt.Errorf("scoped path %q escapes the repository through a symlink", relative)
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil &&
		relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(relative)
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && value == strings.ToLower(value)
}

func preparedPath(digest domain.WorkSpecDigest, name string) string {
	key := strings.TrimPrefix(string(digest), "sha256:")
	return filepath.Join("prepared", key, name)
}

func runPath(runID domain.RunID, name string) string {
	return filepath.Join("runs", string(runID), name)
}

func appendReason(
	reasons []domain.EvidenceReasonCode,
	reason domain.EvidenceReasonCode,
) []domain.EvidenceReasonCode {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func randomRunID() (domain.RunID, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate run ID: %w", err)
	}
	return domain.RunID("run-" + hex.EncodeToString(raw)), nil
}

// FixtureAdapter is deterministic and performs no provider or external call.
type FixtureAdapter struct {
	Result ProviderObservation
}

func (FixtureAdapter) Name() string    { return "fixture" }
func (FixtureAdapter) Version() string { return "v0" }

func (adapter FixtureAdapter) Launch(
	_ context.Context,
	_ LaunchRequest,
) ProviderObservation {
	return adapter.Result
}

func sortedReasons(reasons []domain.EvidenceReasonCode) []domain.EvidenceReasonCode {
	result := append([]domain.EvidenceReasonCode(nil), reasons...)
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}
