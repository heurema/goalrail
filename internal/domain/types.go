package domain

import "time"

type (
	IntentID           string
	IntentItemID       string
	SourceEvidenceID   string
	AmbiguityID        string
	CanaryID           string
	ChangeID           string
	RunID              string
	SessionID          string
	EvidenceEventID    string
	ActorID            string
	EvidenceReference  string
	EvidenceReasonCode string
	ProposalChangeID   string
)

type IntentStatus string

const (
	IntentCandidate IntentStatus = "candidate"
	IntentConfirmed IntentStatus = "confirmed"
)

type SourceEvidenceKind string

const (
	EvidenceOwnerStatement SourceEvidenceKind = "owner_statement"
	EvidenceRepositoryFact SourceEvidenceKind = "repository_fact"
)

// SourceEvidence preserves what was observed separately from interpretation.
type SourceEvidence struct {
	ID        SourceEvidenceID
	Kind      SourceEvidenceKind
	Statement string
	Reference string
}

// IntentItem is one owner-reviewable semantic statement with stable identity.
type IntentItem struct {
	ID           IntentItemID
	Statement    string
	EvidenceRefs []SourceEvidenceID
}

type IntentAmbiguity struct {
	ID           AmbiguityID
	Question     string
	EvidenceRefs []SourceEvidenceID
}

type IntentConfirmation struct {
	Owner              string
	ConfirmedAt        time.Time
	VerificationAction string
}

// IntentSnapshot contains exactly three semantic groups. Provenance and
// lifecycle data are kept in separate fields and grant no effect authority.
type IntentSnapshot struct {
	ID              IntentID
	Version         uint32
	PreviousVersion uint32
	Status          IntentStatus
	SourceEvidence  []SourceEvidence
	DesiredOutcomes []IntentItem
	NonGoals        []IntentItem
	SuccessSignals  []IntentItem
	Ambiguities     []IntentAmbiguity
	Confirmation    *IntentConfirmation
}

// GrantsEffectAuthority is deliberately invariant: confirmed intent describes
// a desired result but never authorizes tools, credentials, writes, or other
// effects. Effect execution belongs to a separate task-grant boundary.
func (IntentSnapshot) GrantsEffectAuthority() bool {
	return false
}

type AmendmentKind string

const (
	AmendmentWordingOnly AmendmentKind = "wording_only"
	AmendmentMaterial    AmendmentKind = "material"
)

// ProposalChange is provider-neutral compiled work traced to confirmed intent.
type ProposalChange struct {
	ID                     ProposalChangeID
	Summary                string
	IntentRefs             []IntentItemID
	ConflictingNonGoalRefs []IntentItemID
}

// Proposal declares the non-goals preserved by its compiled changes. V0 treats
// every confirmed non-goal as applicable and requires explicit coverage.
type Proposal struct {
	Changes              []ProposalChange
	PreservedNonGoalRefs []IntentItemID
}

type LineageStatus string

const (
	LineageVerified LineageStatus = "verified"
	LineageUnlinked LineageStatus = "unlinked"
)

type SessionIdentitySource string

const (
	SessionIdentityLifecycleHook SessionIdentitySource = "lifecycle_hook"
	SessionIdentityLaunchReceipt SessionIdentitySource = "launch_receipt"
)

// ExecutionLineage joins immutable Goalrail run context to provider-authoritative
// root session identity. Missing or conflicting joins remain explicitly unlinked.
type ExecutionLineage struct {
	Status             LineageStatus         `json:"status"`
	ChangeID           ChangeID              `json:"change_id"`
	RunID              RunID                 `json:"run_id"`
	RootSessionID      SessionID             `json:"root_session_id,omitempty"`
	IdentitySource     SessionIdentitySource `json:"identity_source,omitempty"`
	ContextDigest      string                `json:"context_digest"`
	UnlinkedReasonCode EvidenceReasonCode    `json:"unlinked_reason_code,omitempty"`
}

type IntentOutcome string

const (
	IntentMatch   IntentOutcome = "match"
	IntentPartial IntentOutcome = "partial"
	IntentMiss    IntentOutcome = "miss"
)

// Assessment keeps owner intent judgment independent from recorded checks.
type Assessment struct {
	Outcome                          IntentOutcome `json:"outcome"`
	AssessedBy                       ActorID       `json:"assessed_by"`
	AssessedAt                       time.Time     `json:"assessed_at"`
	ChecksGreen                      bool          `json:"checks_green"`
	MaterialCorrectionBeforeDelivery bool          `json:"material_correction_before_delivery"`
	RepeatOptIn                      *bool         `json:"repeat_opt_in,omitempty"`
}

type EvidenceEventKind string

const (
	EventAdmissionDecided     EvidenceEventKind = "admission_decided"
	EventChangeStarted        EvidenceEventKind = "change_started"
	EventCheckSetFrozen       EvidenceEventKind = "check_set_frozen"
	EventLineageRecorded      EvidenceEventKind = "lineage_recorded"
	EventTerminalStateChanged EvidenceEventKind = "terminal_state_changed"
	EventAssessmentRecorded   EvidenceEventKind = "assessment_recorded"
	EventMaterialCorrection   EvidenceEventKind = "material_correction"
	EventEvidenceCorrected    EvidenceEventKind = "evidence_corrected"
	EventCanaryStopped        EvidenceEventKind = "canary_stopped"
)

type AdmissionDecision string

const (
	AdmissionEligible AdmissionDecision = "eligible"
	AdmissionExcluded AdmissionDecision = "excluded"
)

// CanaryAdmission records the operator decision before any variant is
// revealed. An eligible event carries its assignment separately in the same
// append-only envelope; an exclusion never does.
type CanaryAdmission struct {
	Decision  AdmissionDecision `json:"decision"`
	Synthetic bool              `json:"synthetic"`
}

// CanaryAssignment freezes ordinal and variant before execution outcome is
// known. RunID is generated by the launcher and carried in immutable context;
// no session identity is accepted here.
type CanaryAssignment struct {
	Ordinal         uint32        `json:"ordinal"`
	Variant         CanaryVariant `json:"variant"`
	ManifestVersion uint32        `json:"manifest_version"`
	IntentVersion   uint32        `json:"intent_version"`
	RunID           RunID         `json:"run_id"`
	Synthetic       bool          `json:"synthetic"`
}

// CanaryCheckSet freezes bounded verification references before terminal
// verification. Corrections append a new payload and retain every prior ref.
type CanaryCheckSet struct {
	CheckRefs []EvidenceReference `json:"check_refs"`
}

// CanaryTerminal records reviewable delivery or explicit abandonment. Check
// and source details remain bounded references rather than raw output.
type CanaryTerminal struct {
	State                    CanaryTerminalState `json:"state"`
	CheckRefs                []EvidenceReference `json:"check_refs,omitempty"`
	ChecksGreen              bool                `json:"checks_green"`
	FlowOverheadMinutes      *float64            `json:"flow_overhead_minutes,omitempty"`
	AbandonmentReason        EvidenceReasonCode  `json:"abandonment_reason,omitempty"`
	ProcessCausedAbandonment bool                `json:"process_caused_abandonment,omitempty"`
}

// EvidenceEvent is an append-only envelope. SupersedesEventID links a
// correction without replacing the earlier event.
type EvidenceEvent struct {
	ID                        EvidenceEventID     `json:"id"`
	CanaryID                  CanaryID            `json:"canary_id"`
	ChangeID                  ChangeID            `json:"change_id,omitempty"`
	Kind                      EvidenceEventKind   `json:"kind"`
	OccurredAt                time.Time           `json:"occurred_at"`
	Actor                     ActorID             `json:"actor"`
	SourceRef                 EvidenceReference   `json:"source_ref"`
	ObservationRefs           []EvidenceReference `json:"observation_refs,omitempty"`
	ReasonCode                EvidenceReasonCode  `json:"reason_code,omitempty"`
	SupersedesEventID         EvidenceEventID     `json:"supersedes_event_id,omitempty"`
	Admission                 *CanaryAdmission    `json:"admission,omitempty"`
	Assignment                *CanaryAssignment   `json:"assignment,omitempty"`
	CheckSet                  *CanaryCheckSet     `json:"check_set,omitempty"`
	Lineage                   *ExecutionLineage   `json:"lineage,omitempty"`
	LineageResolutionAttempts uint8               `json:"lineage_resolution_attempts,omitempty"`
	Terminal                  *CanaryTerminal     `json:"terminal,omitempty"`
	Assessment                *Assessment         `json:"assessment,omitempty"`
}

type CanaryVariant string

const (
	VariantFlow     CanaryVariant = "flow"
	VariantBaseline CanaryVariant = "baseline"
)

// CanaryManifest identifies the immutable measurement rules frozen before the
// first assignment. Changing any rule requires a new manifest version.
type CanaryManifest struct {
	ID                      CanaryID          `json:"id"`
	Version                 uint32            `json:"version"`
	FrozenAt                time.Time         `json:"frozen_at"`
	AssignmentCount         uint32            `json:"assignment_count"`
	VariantRotation         []CanaryVariant   `json:"variant_rotation"`
	EligibilityRuleRef      EvidenceReference `json:"eligibility_rule_ref"`
	CheckRecordingRuleRef   EvidenceReference `json:"check_recording_rule_ref"`
	AssessmentTimingRuleRef EvidenceReference `json:"assessment_timing_rule_ref"`
	MissingDataRuleRef      EvidenceReference `json:"missing_data_rule_ref"`
	OverheadRuleRef         EvidenceReference `json:"overhead_rule_ref"`
	RateFormulaRuleRef      EvidenceReference `json:"rate_formula_rule_ref"`
}
