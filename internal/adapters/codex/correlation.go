// Package codex translates provider-authoritative Codex lifecycle and launch
// receipts into provider-neutral execution lineage.
package codex

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	RunContextEnvironment = "GOALRAIL_RUN_CONTEXT"
	RunContextSchema      = uint32(1)
	MaxResolutionAttempts = uint8(1)

	ReasonMissingRunContext      domain.EvidenceReasonCode = "MISSING_RUN_CONTEXT"
	ReasonInvalidRunContext      domain.EvidenceReasonCode = "INVALID_RUN_CONTEXT"
	ReasonContextConflict        domain.EvidenceReasonCode = "CONTEXT_CONFLICT"
	ReasonInvalidHookInput       domain.EvidenceReasonCode = "INVALID_HOOK_INPUT"
	ReasonUnsupportedHookEvent   domain.EvidenceReasonCode = "UNSUPPORTED_HOOK_EVENT"
	ReasonSessionConflict        domain.EvidenceReasonCode = "SESSION_CONFLICT"
	ReasonIdentitySourceConflict domain.EvidenceReasonCode = "IDENTITY_SOURCE_CONFLICT"
	ReasonMissingLaunchReceipt   domain.EvidenceReasonCode = "MISSING_LAUNCH_RECEIPT"
	ReasonInvalidLaunchReceipt   domain.EvidenceReasonCode = "INVALID_LAUNCH_RECEIPT"
	ReasonResolutionExhausted    domain.EvidenceReasonCode = "RESOLUTION_ATTEMPT_EXHAUSTED"
)

var ErrInvalidExpectedContext = errors.New("invalid expected Codex run context")

var sessionStartSources = map[string]struct{}{
	"startup": {},
	"resume":  {},
	"clear":   {},
	"compact": {},
}

// RunContext is immutable outside this package. A new value is created per run
// and passed by value, so concurrent changes never consult shared current state.
type RunContext struct {
	changeID domain.ChangeID
	runID    domain.RunID
	repoRoot string
}

type runContextWire struct {
	SchemaVersion uint32          `json:"schema_version"`
	ChangeID      domain.ChangeID `json:"change_id"`
	RunID         domain.RunID    `json:"run_id"`
	RepoRoot      string          `json:"repo_root"`
}

// CorrelationResult contains only sanitized provider metadata and canonical
// lineage. It never retains hook input, task text, prompts, or transcript paths.
type CorrelationResult struct {
	Lineage            domain.ExecutionLineage `json:"lineage"`
	ResolutionAttempts uint8                   `json:"resolution_attempts"`
	ProviderEvent      string                  `json:"provider_event,omitempty"`
	ProviderMatcher    string                  `json:"provider_matcher,omitempty"`
}

type lifecycleHookInput struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Source        string `json:"source"`
	TurnID        string `json:"turn_id"`
	AgentID       string `json:"agent_id"`
	AgentType     string `json:"agent_type"`
}

type launchReceipt struct {
	ThreadID string `json:"threadId"`
}

func NewRunContext(changeID domain.ChangeID, runID domain.RunID, repoRoot string) (RunContext, error) {
	if !domain.IsCanonicalID(string(changeID)) || !domain.IsCanonicalID(string(runID)) {
		return RunContext{}, fmt.Errorf("%w: change and run IDs must be canonical", ErrInvalidExpectedContext)
	}
	if strings.ContainsRune(repoRoot, '\x00') || !filepath.IsAbs(repoRoot) {
		return RunContext{}, fmt.Errorf("%w: repository root must be an absolute path", ErrInvalidExpectedContext)
	}
	cleanRoot := filepath.Clean(repoRoot)
	if cleanRoot == string(filepath.Separator) {
		return RunContext{}, fmt.Errorf("%w: filesystem root cannot be a repository boundary", ErrInvalidExpectedContext)
	}
	return RunContext{changeID: changeID, runID: runID, repoRoot: cleanRoot}, nil
}

func DecodeRunContext(encoded []byte) (RunContext, error) {
	var wire runContextWire
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return RunContext{}, fmt.Errorf("decode run context: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return RunContext{}, fmt.Errorf("decode run context: %w", err)
	}
	if wire.SchemaVersion != RunContextSchema {
		return RunContext{}, fmt.Errorf("unsupported run context schema %d", wire.SchemaVersion)
	}
	return NewRunContext(wire.ChangeID, wire.RunID, wire.RepoRoot)
}

func (context RunContext) EnvironmentValue() (string, error) {
	wire, err := context.wire()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return "", fmt.Errorf("encode run context: %w", err)
	}
	return string(encoded), nil
}

func (context RunContext) ChangeID() domain.ChangeID { return context.changeID }
func (context RunContext) RunID() domain.RunID       { return context.runID }
func (context RunContext) RepoRoot() string          { return context.repoRoot }

func (context RunContext) Digest() (string, error) {
	wire, err := context.wire()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return "", fmt.Errorf("encode run context digest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func (context RunContext) wire() (runContextWire, error) {
	validated, err := NewRunContext(context.changeID, context.runID, context.repoRoot)
	if err != nil {
		return runContextWire{}, err
	}
	return runContextWire{
		SchemaVersion: RunContextSchema,
		ChangeID:      validated.changeID,
		RunID:         validated.runID,
		RepoRoot:      validated.repoRoot,
	}, nil
}

func (result CorrelationResult) Verified() bool {
	return result.Lineage.Status == domain.LineageVerified
}

// BindLifecycleHook verifies that the process-scoped environment value is the
// exact expected run context, then binds the provider hook session identity.
// Provider failures return UNLINKED; invalid caller-owned expected context is
// the only programming error.
func BindLifecycleHook(
	expected RunContext,
	encodedContext string,
	rawHook []byte,
	previous *CorrelationResult,
) (CorrelationResult, error) {
	digest, err := expected.Digest()
	if err != nil {
		return CorrelationResult{}, err
	}
	attempts, reason := prepareResolutionAttempt(expected, digest, previous)
	failure := func(reason domain.EvidenceReasonCode) CorrelationResult {
		return unlinked(expected, digest, reason, failureAttemptCount(previous, attempts))
	}
	if reason != "" {
		return failure(reason), nil
	}
	if strings.TrimSpace(encodedContext) == "" {
		return failure(ReasonMissingRunContext), nil
	}
	observedContext, err := DecodeRunContext([]byte(encodedContext))
	if err != nil {
		return failure(ReasonInvalidRunContext), nil
	}
	observedDigest, err := observedContext.Digest()
	if err != nil || observedDigest != digest {
		return failure(ReasonContextConflict), nil
	}

	hook, err := decodeLifecycleHook(rawHook)
	if err != nil {
		return failure(ReasonInvalidHookInput), nil
	}
	event, matcher, reason := validateLifecycleHook(expected, hook)
	if reason != "" {
		return failure(reason), nil
	}
	sessionID := domain.SessionID(hook.SessionID)
	if reason := validatePrevious(expected, digest, sessionID, domain.SessionIdentityLifecycleHook, previous); reason != "" {
		return failure(reason), nil
	}
	return verified(expected, digest, sessionID, domain.SessionIdentityLifecycleHook, event, matcher, attempts), nil
}

// BindLaunchReceipt accepts only the immutable threadId returned by the
// provider-authoritative create_thread response. A manual task or response
// without threadId is explicitly UNLINKED.
func BindLaunchReceipt(
	expected RunContext,
	rawReceipt []byte,
	previous *CorrelationResult,
) (CorrelationResult, error) {
	digest, err := expected.Digest()
	if err != nil {
		return CorrelationResult{}, err
	}
	attempts, reason := prepareResolutionAttempt(expected, digest, previous)
	failure := func(reason domain.EvidenceReasonCode) CorrelationResult {
		return unlinked(expected, digest, reason, failureAttemptCount(previous, attempts))
	}
	if reason != "" {
		return failure(reason), nil
	}
	if len(bytes.TrimSpace(rawReceipt)) == 0 {
		return failure(ReasonMissingLaunchReceipt), nil
	}
	receipt, reason := decodeLaunchReceipt(rawReceipt)
	if reason != "" {
		return failure(reason), nil
	}
	sessionID := domain.SessionID(receipt.ThreadID)
	if reason := validatePrevious(expected, digest, sessionID, domain.SessionIdentityLaunchReceipt, previous); reason != "" {
		return failure(reason), nil
	}
	return verified(
		expected,
		digest,
		sessionID,
		domain.SessionIdentityLaunchReceipt,
		"create_thread",
		"threadId",
		attempts,
	), nil
}

func decodeLifecycleHook(raw []byte) (lifecycleHookInput, error) {
	var hook lifecycleHookInput
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&hook); err != nil {
		return lifecycleHookInput{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return lifecycleHookInput{}, err
	}
	return hook, nil
}

func validateLifecycleHook(expected RunContext, hook lifecycleHookInput) (event, matcher string, reason domain.EvidenceReasonCode) {
	if !domain.IsCanonicalID(hook.SessionID) || !filepath.IsAbs(hook.CWD) {
		return "", "", ReasonInvalidHookInput
	}
	if !pathWithin(filepath.Clean(hook.CWD), expected.repoRoot) {
		return "", "", ReasonContextConflict
	}
	switch hook.HookEventName {
	case "SessionStart":
		if _, allowed := sessionStartSources[hook.Source]; !allowed {
			return "", "", ReasonInvalidHookInput
		}
		return hook.HookEventName, hook.Source, ""
	case "SubagentStart":
		if !domain.IsCanonicalID(hook.TurnID) ||
			!domain.IsCanonicalID(hook.AgentID) ||
			!domain.IsCanonicalID(hook.AgentType) {
			return "", "", ReasonInvalidHookInput
		}
		return hook.HookEventName, hook.AgentType, ""
	default:
		return "", "", ReasonUnsupportedHookEvent
	}
}

func decodeLaunchReceipt(raw []byte) (launchReceipt, domain.EvidenceReasonCode) {
	var receipt *launchReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&receipt); err != nil || receipt == nil {
		return launchReceipt{}, ReasonInvalidLaunchReceipt
	}
	if err := requireJSONEOF(decoder); err != nil {
		return launchReceipt{}, ReasonInvalidLaunchReceipt
	}
	if strings.TrimSpace(receipt.ThreadID) == "" {
		return launchReceipt{}, ReasonMissingLaunchReceipt
	}
	if !domain.IsCanonicalID(receipt.ThreadID) {
		return launchReceipt{}, ReasonInvalidLaunchReceipt
	}
	return *receipt, ""
}

func validatePrevious(
	expected RunContext,
	digest string,
	sessionID domain.SessionID,
	source domain.SessionIdentitySource,
	previous *CorrelationResult,
) domain.EvidenceReasonCode {
	if previous == nil {
		return ""
	}
	lineage := previous.Lineage
	if lineage.ChangeID != expected.changeID ||
		lineage.RunID != expected.runID ||
		lineage.ContextDigest != digest {
		return ReasonContextConflict
	}
	switch lineage.Status {
	case domain.LineageUnlinked:
		return ""
	case domain.LineageVerified:
		if lineage.IdentitySource != source {
			return ReasonIdentitySourceConflict
		}
		if lineage.RootSessionID != sessionID {
			return ReasonSessionConflict
		}
		return ""
	default:
		return ReasonContextConflict
	}
}

func prepareResolutionAttempt(
	expected RunContext,
	digest string,
	previous *CorrelationResult,
) (uint8, domain.EvidenceReasonCode) {
	if previous == nil {
		return 0, ""
	}
	lineage := previous.Lineage
	if lineage.ChangeID != expected.changeID ||
		lineage.RunID != expected.runID ||
		lineage.ContextDigest != digest {
		return previous.ResolutionAttempts, ReasonContextConflict
	}
	if lineage.Status == domain.LineageVerified {
		return previous.ResolutionAttempts, ""
	}
	if lineage.Status != domain.LineageUnlinked {
		return previous.ResolutionAttempts, ReasonContextConflict
	}
	if previous.ResolutionAttempts >= MaxResolutionAttempts {
		return previous.ResolutionAttempts, ReasonResolutionExhausted
	}
	return previous.ResolutionAttempts + 1, ""
}

func failureAttemptCount(previous *CorrelationResult, attempts uint8) uint8 {
	if previous != nil && previous.Verified() && attempts < MaxResolutionAttempts {
		return MaxResolutionAttempts
	}
	return attempts
}

func verified(
	context RunContext,
	digest string,
	sessionID domain.SessionID,
	source domain.SessionIdentitySource,
	event string,
	matcher string,
	attempts uint8,
) CorrelationResult {
	return CorrelationResult{
		Lineage: domain.ExecutionLineage{
			Status:         domain.LineageVerified,
			ChangeID:       context.changeID,
			RunID:          context.runID,
			RootSessionID:  sessionID,
			IdentitySource: source,
			ContextDigest:  digest,
		},
		ResolutionAttempts: attempts,
		ProviderEvent:      event,
		ProviderMatcher:    matcher,
	}
}

func unlinked(context RunContext, digest string, reason domain.EvidenceReasonCode, attempts uint8) CorrelationResult {
	return CorrelationResult{
		Lineage: domain.ExecutionLineage{
			Status:             domain.LineageUnlinked,
			ChangeID:           context.changeID,
			RunID:              context.runID,
			ContextDigest:      digest,
			UnlinkedReasonCode: reason,
		},
		ResolutionAttempts: attempts,
	}
}

func pathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
