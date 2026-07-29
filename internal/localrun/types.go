// Package localrun coordinates one prepared, one-shot, trusted local run.
// Non-fixture activation requires the exact one-time dogfood admission.
package localrun

import (
	"context"
	"io"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

type RunState string

const (
	StatePrepared               RunState = "prepared"
	StateLaunchAttempted        RunState = "launch_attempted"
	StateAwaitingVerification   RunState = "awaiting_verification"
	StateLaunchFailed           RunState = "launch_failed"
	StateUnlinked               RunState = "unlinked"
	StateLaunchAttemptedUnknown RunState = "launch_attempted_unknown"
	StatePassed                 RunState = "passed"
	StateFailed                 RunState = "failed"
	StateVerificationIncomplete RunState = "verification_incomplete"
	StateBlocked                RunState = "blocked"
)

// TerminalReceiptSchemaV1 makes the receipt contract explicit. Adding the
// blocked status breaks every reader that whitelists terminal statuses, so the
// break happens once, deliberately, together with the version that makes it
// legible. A stored receipt without a schema identifier remains readable as the
// unversioned predecessor and is never rewritten.
const TerminalReceiptSchemaV1 = "goalrail.terminal-receipt/v1"

type ProviderOutcome string

const (
	ProviderCompleted    ProviderOutcome = "completed"
	ProviderDenied       ProviderOutcome = "denied"
	ProviderLaunchFailed ProviderOutcome = "launch_failed"
)

type IdentityStatus string

const (
	IdentityVerified IdentityStatus = "verified"
	IdentityUnlinked IdentityStatus = "unlinked"
)

type WorktreeEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Mode   uint32 `json:"mode,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type WorktreeObservation struct {
	Head    string          `json:"head"`
	Entries []WorktreeEntry `json:"entries"`
	Digest  string          `json:"digest"`
}

type WorktreeDelta struct {
	BaselineDigest  string   `json:"baseline_digest"`
	TerminalDigest  string   `json:"terminal_digest"`
	HeadChanged     bool     `json:"head_changed,omitempty"`
	ChangedPaths    []string `json:"changed_paths"`
	ScopeViolations []string `json:"scope_violations,omitempty"`
}

type Preparation struct {
	WorkSpecDigest domain.WorkSpecDigest `json:"work_spec_digest"`
	PreparedAt     time.Time             `json:"prepared_at"`
	Baseline       WorktreeObservation   `json:"baseline"`
	State          RunState              `json:"state"`
}

type LaunchClaim struct {
	WorkSpecDigest domain.WorkSpecDigest `json:"work_spec_digest"`
	RunID          domain.RunID          `json:"run_id"`
	Adapter        string                `json:"adapter"`
	AdapterVersion string                `json:"adapter_version"`
	AttemptedAt    time.Time             `json:"attempted_at"`
}

type ProviderObservation struct {
	WorkSpecDigest domain.WorkSpecDigest     `json:"work_spec_digest"`
	RunID          domain.RunID              `json:"run_id"`
	Adapter        string                    `json:"adapter"`
	AdapterVersion string                    `json:"adapter_version"`
	Outcome        ProviderOutcome           `json:"outcome"`
	ExitCode       int                       `json:"exit_code,omitempty"`
	IdentityStatus IdentityStatus            `json:"identity_status"`
	RootSessionRef string                    `json:"root_session_ref,omitempty"`
	Reason         domain.EvidenceReasonCode `json:"reason,omitempty"`
	ObservedAt     time.Time                 `json:"observed_at"`
}

type CheckResult struct {
	ID             domain.WorkSpecCheckID  `json:"id"`
	State          domain.CheckResultState `json:"state"`
	EvidenceRef    string                  `json:"evidence_ref,omitempty"`
	EvidenceDigest string                  `json:"evidence_digest,omitempty"`
}

// ReceiptIntentReference records which confirmed intent a run was launched
// from. The receipt carried no intent reference before this schema version, so
// the question-to-decision chain could not be followed from recorded
// identifiers alone.
type ReceiptIntentReference struct {
	ID      domain.IntentID `json:"id"`
	Version uint32          `json:"version"`
	Digest  string          `json:"digest"`
}

type TerminalReceipt struct {
	Schema             string                      `json:"schema,omitempty"`
	WorkSpecID         domain.WorkSpecID           `json:"work_spec_id"`
	WorkSpecVersion    uint32                      `json:"work_spec_version"`
	WorkSpecDigest     domain.WorkSpecDigest       `json:"work_spec_digest"`
	Intent             *ReceiptIntentReference     `json:"intent,omitempty"`
	Escalation         *EscalationRecord           `json:"escalation,omitempty"`
	RunID              domain.RunID                `json:"run_id"`
	Adapter            string                      `json:"adapter"`
	AdapterVersion     string                      `json:"adapter_version"`
	RootSessionRef     string                      `json:"root_session_ref,omitempty"`
	UnlinkedReason     domain.EvidenceReasonCode   `json:"unlinked_reason,omitempty"`
	BaseRevision       string                      `json:"base_revision"`
	TerminalHead       string                      `json:"terminal_head"`
	EffectivePaths     []string                    `json:"effective_paths"`
	Checks             []domain.WorkSpecCheck      `json:"checks"`
	CheckResults       []CheckResult               `json:"check_results"`
	Status             RunState                    `json:"status"`
	Reasons            []domain.EvidenceReasonCode `json:"reasons,omitempty"`
	WorktreeDelta      WorktreeDelta               `json:"worktree_delta"`
	PreparedAt         time.Time                   `json:"prepared_at"`
	LaunchAttemptedAt  time.Time                   `json:"launch_attempted_at"`
	ProviderObservedAt time.Time                   `json:"provider_observed_at"`
	TerminalAt         time.Time                   `json:"terminal_at"`
}

type LaunchRequest struct {
	RunID     domain.RunID
	WorkSpec  domain.FrozenWorkSpec
	Arguments []string

	// EscalationAnnouncement is the provider-neutral text telling the run that
	// the escalation channel exists. The adapter delivers it in the terms of its
	// own provider contract; it may not alter the content.
	EscalationAnnouncement string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Adapter interface {
	Name() string
	Version() string

	// VerifyAnnouncementDelivery reports whether this adapter can tell a run
	// that the escalation channel exists. Start calls it before launching and
	// refuses to launch when it returns an error, because a run whose
	// escalation channel is unreachable guesses and still produces an
	// ordinary-looking receipt.
	VerifyAnnouncementDelivery() error

	Launch(context.Context, LaunchRequest) ProviderObservation
}

type IntentVerifier interface {
	Verify(string, domain.WorkSpecIntentReference) error
}

type WorktreeObserver interface {
	ResolveRepository(context.Context, string, string) (string, string, error)
	Observe(context.Context, string) (WorktreeObservation, error)
}
