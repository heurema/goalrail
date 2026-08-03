package domain

import "time"

type (
	IntentID           string
	IntentItemID       string
	SourceEvidenceID   string
	ContextPackID      string
	ContextItemID      string
	ContextUnknownID   string
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

type ContextItemKind string

const (
	ContextRepository ContextItemKind = "repository"
	ContextExternal   ContextItemKind = "external"
)

type ContextCollectionOutcome string

const (
	ContextSufficient      ContextCollectionOutcome = "sufficient"
	ContextMaterialUnknown ContextCollectionOutcome = "material_unknown"
	ContextBudgetExhausted ContextCollectionOutcome = "budget_exhausted"
)

// ContextItem retains one concise fact used to interpret a request. It never
// becomes owner intent without the owner's separate confirmation.
type ContextItem struct {
	ID                 ContextItemID
	Kind               ContextItemKind
	Claim              string
	SourceRef          EvidenceReference
	VerificationRecipe string
	ObservedAt         time.Time
	Relevance          string
}

// ContextUnknown makes an unresolved material fact explicit instead of
// allowing the collector to fill it by inference.
type ContextUnknown struct {
	ID         ContextUnknownID
	Question   string
	SourceRefs []EvidenceReference
}

// ContextPack is provider-neutral evidence gathered before flow intent. The
// outcome records why collection stopped; it is not another intent group.
type ContextPack struct {
	ID              ContextPackID
	Version         uint32
	PreviousVersion uint32
	StartedAt       time.Time
	CompletedAt     time.Time
	Outcome         ContextCollectionOutcome
	Items           []ContextItem
	Unknowns        []ContextUnknown
}

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
	ContextRefs  []ContextItemID
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
	ContextPack     *ContextPack

	// ResolvedEscalation records which blocked run this version answers. It is
	// lifecycle provenance rather than a fourth semantic intent group, it is
	// optional, and it grants no effect authority.
	ResolvedEscalation *IntentEscalationResolution
}

type IntentDisposition string

const (
	DispositionAnswered  IntentDisposition = "answered"
	DispositionSpurious  IntentDisposition = "spurious"
	DispositionWithdrawn IntentDisposition = "withdrawn"
)

// IntentEscalationResolution links an answering intent version back to the
// question it resolves — a blocked run's escalation or a background session's
// question record, both named by canonical identifier. `spurious` and
// `withdrawn` exist so a low-value escalation stays visible and countable
// instead of disappearing.
type IntentEscalationResolution struct {
	ResolvedID       string
	EscalationDigest string
	Disposition      IntentDisposition
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

type IntentItemCategory string

const (
	IntentCategoryDesiredOutcome IntentItemCategory = "desired_outcome"
	IntentCategoryNonGoal        IntentItemCategory = "non_goal"
	IntentCategorySuccessSignal  IntentItemCategory = "success_signal"
)

type IntentItemJudgmentValue string

const (
	JudgmentAchieved  IntentItemJudgmentValue = "achieved"
	JudgmentPartial   IntentItemJudgmentValue = "partial"
	JudgmentMissed    IntentItemJudgmentValue = "missed"
	JudgmentPreserved IntentItemJudgmentValue = "preserved"
	JudgmentViolated  IntentItemJudgmentValue = "violated"
	JudgmentObserved  IntentItemJudgmentValue = "observed"
	JudgmentMissing   IntentItemJudgmentValue = "missing"
)

type IntentItemJudgment struct {
	ItemID   IntentItemID            `json:"item_id"`
	Category IntentItemCategory      `json:"category"`
	Judgment IntentItemJudgmentValue `json:"judgment"`
}

// Assessment keeps owner intent judgment independent from recorded checks.
type Assessment struct {
	Outcome                          IntentOutcome        `json:"outcome"`
	AssessedBy                       ActorID              `json:"assessed_by"`
	AssessedAt                       time.Time            `json:"assessed_at"`
	ChecksGreen                      bool                 `json:"checks_green"`
	MaterialCorrectionBeforeDelivery bool                 `json:"material_correction_before_delivery"`
	RepeatOptIn                      *bool                `json:"repeat_opt_in,omitempty"`
	ItemJudgments                    []IntentItemJudgment `json:"item_judgments,omitempty"`
}

type EvidenceEventKind string

const (
	EventAdmissionDecided        EvidenceEventKind = "admission_decided"
	EventChangeStarted           EvidenceEventKind = "change_started"
	EventContextBound            EvidenceEventKind = "context_bound"
	EventAssessmentBasisRecorded EvidenceEventKind = "assessment_basis_recorded"
	EventFlowPhaseRecorded       EvidenceEventKind = "flow_phase_recorded"
	EventCheckSetFrozen          EvidenceEventKind = "check_set_frozen"
	EventLineageRecorded         EvidenceEventKind = "lineage_recorded"
	EventTelemetryRecorded       EvidenceEventKind = "telemetry_recorded"
	EventTerminalStateChanged    EvidenceEventKind = "terminal_state_changed"
	EventAssessmentRecorded      EvidenceEventKind = "assessment_recorded"
	EventMaterialCorrection      EvidenceEventKind = "material_correction"
	EventEvidenceCorrected       EvidenceEventKind = "evidence_corrected"
	EventCanaryStopped           EvidenceEventKind = "canary_stopped"
)

// CanaryContextBinding retains only Context Pack identity in runtime evidence.
type CanaryContextBinding struct {
	ContextPackID      ContextPackID `json:"context_pack_id"`
	ContextPackVersion uint32        `json:"context_pack_version"`
}

type AssessmentBasisTiming string

const (
	BasisPreExecution AssessmentBasisTiming = "pre_execution"
	BasisPostDelivery AssessmentBasisTiming = "post_delivery"
)

// CanaryAssessmentBasis freezes stable IDs without copying intent wording.
type CanaryAssessmentBasis struct {
	IntentRef         EvidenceReference     `json:"intent_ref"`
	IntentID          IntentID              `json:"intent_id"`
	IntentVersion     uint32                `json:"intent_version"`
	Timing            AssessmentBasisTiming `json:"timing"`
	DesiredOutcomeIDs []IntentItemID        `json:"desired_outcome_ids"`
	NonGoalIDs        []IntentItemID        `json:"non_goal_ids"`
	SuccessSignalIDs  []IntentItemID        `json:"success_signal_ids"`
}

// CanaryFlowPhase is the explicit pre-implementation window used for v2
// trace selection.
type CanaryFlowPhase struct {
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

type TelemetryStatus string

const (
	TelemetryAvailable   TelemetryStatus = "available"
	TelemetryUnavailable TelemetryStatus = "unavailable"
	TelemetryConflict    TelemetryStatus = "conflict"
)

// CanaryTelemetry retains only bounded lookup, trace-envelope, and component
// timing evidence. Detailed provider observations remain in Langfuse.
type CanaryTelemetry struct {
	Status         TelemetryStatus        `json:"status"`
	SessionLookup  EvidenceReference      `json:"session_lookup"`
	TraceIntervals []CanaryTimingInterval `json:"trace_intervals,omitempty"`
	OwnerReview    *CanaryTimingInterval  `json:"owner_review,omitempty"`
	FlowOverhead   *CanaryFlowOverhead    `json:"flow_overhead,omitempty"`
}

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
	ID                        EvidenceEventID        `json:"id"`
	CanaryID                  CanaryID               `json:"canary_id"`
	ManifestVersion           uint32                 `json:"manifest_version,omitempty"`
	ChangeID                  ChangeID               `json:"change_id,omitempty"`
	Kind                      EvidenceEventKind      `json:"kind"`
	OccurredAt                time.Time              `json:"occurred_at"`
	Actor                     ActorID                `json:"actor"`
	SourceRef                 EvidenceReference      `json:"source_ref"`
	ObservationRefs           []EvidenceReference    `json:"observation_refs,omitempty"`
	ReasonCode                EvidenceReasonCode     `json:"reason_code,omitempty"`
	SupersedesEventID         EvidenceEventID        `json:"supersedes_event_id,omitempty"`
	Admission                 *CanaryAdmission       `json:"admission,omitempty"`
	Assignment                *CanaryAssignment      `json:"assignment,omitempty"`
	Context                   *CanaryContextBinding  `json:"context,omitempty"`
	AssessmentBasis           *CanaryAssessmentBasis `json:"assessment_basis,omitempty"`
	FlowPhase                 *CanaryFlowPhase       `json:"flow_phase,omitempty"`
	CheckSet                  *CanaryCheckSet        `json:"check_set,omitempty"`
	Lineage                   *ExecutionLineage      `json:"lineage,omitempty"`
	LineageResolutionAttempts uint8                  `json:"lineage_resolution_attempts,omitempty"`
	Terminal                  *CanaryTerminal        `json:"terminal,omitempty"`
	Telemetry                 *CanaryTelemetry       `json:"telemetry,omitempty"`
	Assessment                *Assessment            `json:"assessment,omitempty"`
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
	ContextRuleRef          EvidenceReference `json:"context_rule_ref,omitempty"`
	TelemetryRuleRef        EvidenceReference `json:"telemetry_rule_ref,omitempty"`
	AssessmentBasisRuleRef  EvidenceReference `json:"assessment_basis_rule_ref,omitempty"`
}
