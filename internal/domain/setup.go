package domain

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type (
	SetupScope            string
	SetupPlanState        string
	SetupReceiptStatus    string
	SetupActionStatus     string
	SetupMutationKind     string
	SetupPrerequisiteKind string
)

const (
	SetupProfileSchemaV1      = "goalrail.setup-profile/v1"
	SetupPlanSchemaV1         = "goalrail.setup-plan/v1"
	SetupReceiptSchemaV1      = "goalrail.setup-receipt/v1"
	PlanAuthorizationSchemaV1 = "goalrail.plan-authorization/v1"
	MaxSetupProfileBytes      = 64 << 10
	MaxSetupPlanBytes         = 256 << 10
	MaxSetupReceiptBytes      = 256 << 10
	MaxPlanAuthorizationBytes = 32 << 10
	MaxSetupItems             = 512
	MaxSetupArguments         = 64
	MaxSetupStringBytes       = 2048

	SetupScopeRepository SetupScope = "repository"
	SetupScopeUserLocal  SetupScope = "user_local"

	SetupPlanComplete   SetupPlanState = "complete"
	SetupPlanIncomplete SetupPlanState = "incomplete"

	SetupReceiptSuccess SetupReceiptStatus = "success"
	SetupReceiptFailure SetupReceiptStatus = "failure"
	SetupReceiptRefused SetupReceiptStatus = "refused"
	SetupReceiptNoWrite SetupReceiptStatus = "no_write"

	SetupActionApplied    SetupActionStatus = "applied"
	SetupActionSkipped    SetupActionStatus = "skipped"
	SetupActionFailed     SetupActionStatus = "failed"
	SetupActionRolledBack SetupActionStatus = "rolled_back"

	SetupMutationInstallFile      SetupMutationKind = "install_file"
	SetupMutationEditConfig       SetupMutationKind = "edit_config"
	SetupMutationInstallHook      SetupMutationKind = "install_hook"
	SetupMutationInstallAdapter   SetupMutationKind = "install_adapter"
	SetupMutationSelectExecutable SetupMutationKind = "select_executable"

	SetupPrerequisiteRuntime    SetupPrerequisiteKind = "runtime"
	SetupPrerequisitePlatform   SetupPrerequisiteKind = "platform"
	SetupPrerequisiteFilesystem SetupPrerequisiteKind = "filesystem"
	SetupPrerequisiteTrust      SetupPrerequisiteKind = "trust"
)

type SetupAdapterPin struct {
	ID        string       `json:"id"`
	Version   string       `json:"version"`
	SourceRef string       `json:"source_ref"`
	Integrity SHA256Digest `json:"integrity"`
}

type SetupPlanningAdapter struct {
	Adapter         SetupAdapterPin `json:"adapter"`
	Runtime         string          `json:"runtime"`
	RuntimeVersion  string          `json:"runtime_version"`
	Compiler        string          `json:"compiler"`
	CompilerVersion string          `json:"compiler_version"`
}

type SetupProfile struct {
	Schema                   string               `json:"schema"`
	ProjectID                ProjectID            `json:"project_id"`
	CompatibleGoalrailBundle string               `json:"compatible_goalrail_bundle"`
	Planning                 SetupPlanningAdapter `json:"planning"`
	ScaffoldAdapters         []SetupAdapterPin    `json:"scaffold_adapters"`
	SharedAdmissionAdapter   *SetupAdapterPin     `json:"shared_admission_adapter"`
}

type SetupComponent struct {
	ID            string       `json:"id"`
	Version       string       `json:"version"`
	SourceRef     string       `json:"source_ref"`
	Integrity     SHA256Digest `json:"integrity"`
	SizeBytes     uint64       `json:"size_bytes"`
	Destination   string       `json:"destination"`
	Scope         SetupScope   `json:"scope"`
	LicenseRef    string       `json:"license_ref"`
	ProvenanceRef string       `json:"provenance_ref"`
}

type SetupMutation struct {
	ID                   string            `json:"id"`
	Kind                 SetupMutationKind `json:"kind"`
	Scope                SetupScope        `json:"scope"`
	Path                 string            `json:"path"`
	ExpectedBeforeDigest *SHA256Digest     `json:"expected_before_digest"`
	DesiredDigest        SHA256Digest      `json:"desired_digest"`
}

type SetupPrerequisite struct {
	ID                string                `json:"id"`
	Kind              SetupPrerequisiteKind `json:"kind"`
	VersionConstraint string                `json:"version_constraint"`
	Satisfied         bool                  `json:"satisfied"`
	EvidenceRef       string                `json:"evidence_ref"`
}

type SetupTrustStep struct {
	ID          string `json:"id"`
	AdapterID   string `json:"adapter_id"`
	ActionRef   string `json:"action_ref"`
	Interactive bool   `json:"interactive"`
}

type SetupNetworkAccess struct {
	ID        string `json:"id"`
	Method    string `json:"method"`
	URL       string `json:"url"`
	PurposeID string `json:"purpose_id"`
}

type SetupRollbackAction struct {
	MutationID       string        `json:"mutation_id"`
	Action           string        `json:"action"`
	Target           string        `json:"target"`
	PriorStateDigest *SHA256Digest `json:"prior_state_digest"`
}

type SetupVerification struct {
	ID   string   `json:"id"`
	Argv []string `json:"argv"`
}

type SetupPlan struct {
	Schema              string                `json:"schema"`
	ProjectID           ProjectID             `json:"project_id"`
	DeclarationDigest   SHA256Digest          `json:"declaration_digest"`
	SetupProfileDigest  SHA256Digest          `json:"setup_profile_digest"`
	Platform            string                `json:"platform"`
	State               SetupPlanState        `json:"state"`
	IncompleteReasonIDs []string              `json:"incomplete_reason_ids"`
	Components          []SetupComponent      `json:"components"`
	Mutations           []SetupMutation       `json:"mutations"`
	Prerequisites       []SetupPrerequisite   `json:"prerequisites"`
	TrustSteps          []SetupTrustStep      `json:"trust_steps"`
	NetworkAccess       []SetupNetworkAccess  `json:"network_access"`
	Rollback            []SetupRollbackAction `json:"rollback"`
	Verification        []SetupVerification   `json:"verification"`
	ProjectCodeWrites   uint32                `json:"project_code_writes"`
}

type PlanAuthorizationReference struct {
	Schema       string       `json:"schema"`
	ProjectID    ProjectID    `json:"project_id"`
	PlanDigest   SHA256Digest `json:"plan_digest"`
	DecisionRef  string       `json:"decision_ref"`
	ActorRef     string       `json:"actor_ref"`
	AuthorizedAt time.Time    `json:"authorized_at"`
}

type SetupComponentResult struct {
	ID        string            `json:"id"`
	Version   string            `json:"version"`
	Integrity SHA256Digest      `json:"integrity"`
	Status    SetupActionStatus `json:"status"`
}

type SetupMutationResult struct {
	ID           string            `json:"id"`
	Status       SetupActionStatus `json:"status"`
	BeforeDigest *SHA256Digest     `json:"before_digest"`
	AfterDigest  *SHA256Digest     `json:"after_digest"`
}

type SetupReceipt struct {
	Schema          string                      `json:"schema"`
	ProjectID       ProjectID                   `json:"project_id"`
	PlanDigest      SHA256Digest                `json:"plan_digest"`
	Authorization   *PlanAuthorizationReference `json:"authorization"`
	Components      []SetupComponentResult      `json:"components"`
	Mutations       []SetupMutationResult       `json:"mutations"`
	RollbackRef     string                      `json:"rollback_ref"`
	DiagnosisRef    string                      `json:"diagnosis_ref"`
	Status          SetupReceiptStatus          `json:"status"`
	BlockerRefs     []string                    `json:"blocker_refs"`
	ContinuationRef string                      `json:"continuation_ref"`
}

func DecodeSetupProfile(reader io.Reader) (SetupProfile, error) {
	value, err := decodeStrictBoundedJSON[SetupProfile](reader, MaxSetupProfileBytes, "setup profile")
	if err != nil {
		return SetupProfile{}, err
	}
	value = normalizeSetupProfile(value)
	if err := ValidateSetupProfile(value); err != nil {
		return SetupProfile{}, err
	}
	return value, nil
}

func FreezeSetupProfile(value SetupProfile) (CanonicalArtifact, error) {
	value = normalizeSetupProfile(value)
	if err := ValidateSetupProfile(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func DecodeSetupPlan(reader io.Reader) (SetupPlan, error) {
	value, err := decodeStrictBoundedJSON[SetupPlan](reader, MaxSetupPlanBytes, "setup plan")
	if err != nil {
		return SetupPlan{}, err
	}
	value = normalizeSetupPlan(value)
	if err := ValidateSetupPlan(value); err != nil {
		return SetupPlan{}, err
	}
	return value, nil
}

func FreezeSetupPlan(value SetupPlan) (CanonicalArtifact, error) {
	value = normalizeSetupPlan(value)
	if err := ValidateSetupPlan(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func DecodePlanAuthorizationReference(reader io.Reader) (PlanAuthorizationReference, error) {
	value, err := decodeStrictBoundedJSON[PlanAuthorizationReference](reader, MaxPlanAuthorizationBytes, "plan authorization")
	if err != nil {
		return PlanAuthorizationReference{}, err
	}
	value.AuthorizedAt = value.AuthorizedAt.UTC()
	if err := ValidatePlanAuthorizationReference(value); err != nil {
		return PlanAuthorizationReference{}, err
	}
	return value, nil
}

func FreezePlanAuthorizationReference(value PlanAuthorizationReference) (CanonicalArtifact, error) {
	value.AuthorizedAt = value.AuthorizedAt.UTC()
	if err := ValidatePlanAuthorizationReference(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func DecodeSetupReceipt(reader io.Reader) (SetupReceipt, error) {
	value, err := decodeStrictBoundedJSON[SetupReceipt](reader, MaxSetupReceiptBytes, "setup receipt")
	if err != nil {
		return SetupReceipt{}, err
	}
	value = normalizeSetupReceipt(value)
	if err := ValidateSetupReceipt(value); err != nil {
		return SetupReceipt{}, err
	}
	return value, nil
}

func FreezeSetupReceipt(value SetupReceipt) (CanonicalArtifact, error) {
	value = normalizeSetupReceipt(value)
	if err := ValidateSetupReceipt(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func ValidateSetupProfile(value SetupProfile) error {
	v := &validator{}
	validateSetupHeader(v, "setup_profile", value.Schema, SetupProfileSchemaV1, value.ProjectID)
	validateSetupString(v, "setup_profile.bundle_invalid", "compatible_goalrail_bundle", value.CompatibleGoalrailBundle, true)
	validatePlanningAdapter(v, "planning", value.Planning)
	if len(value.ScaffoldAdapters) == 0 {
		v.add("setup_profile.scaffolds.required", "scaffold_adapters", "at least one supported scaffold adapter is required")
	}
	validateAdapterPins(v, "scaffold_adapters", value.ScaffoldAdapters)
	if value.SharedAdmissionAdapter != nil {
		validateAdapterPin(v, "shared_admission_adapter", *value.SharedAdmissionAdapter)
	}
	validateNoSecretPayload(v, value)
	return v.result()
}

func ValidateSetupPlan(value SetupPlan) error {
	v := &validator{}
	validateSetupHeader(v, "setup_plan", value.Schema, SetupPlanSchemaV1, value.ProjectID)
	validateDigestField(v, "setup_plan.declaration_digest_invalid", "declaration_digest", value.DeclarationDigest)
	validateDigestField(v, "setup_plan.profile_digest_invalid", "setup_profile_digest", value.SetupProfileDigest)
	validateSetupString(v, "setup_plan.platform_invalid", "platform", value.Platform, true)
	switch value.State {
	case SetupPlanComplete:
		if len(value.IncompleteReasonIDs) != 0 {
			v.add("setup_plan.complete_has_reasons", "incomplete_reason_ids", "complete plan cannot carry incomplete reasons")
		}
	case SetupPlanIncomplete:
		if len(value.IncompleteReasonIDs) == 0 {
			v.add("setup_plan.incomplete_reason_required", "incomplete_reason_ids", "incomplete plan requires a stable reason")
		}
	default:
		v.add("setup_plan.state_invalid", "state", "setup plan state must be complete or incomplete")
	}
	validateCanonicalStringSet(v, "incomplete_reason_ids", value.IncompleteReasonIDs, MaxSetupItems)
	validateSetupComponents(v, value.Components, value.State == SetupPlanComplete)
	validateSetupMutations(v, value.Mutations)
	validateSetupPrerequisites(v, value.Prerequisites)
	validateSetupTrustSteps(v, value.TrustSteps)
	validateSetupNetworkAccess(v, value.NetworkAccess)
	validateSetupRollback(v, value.Rollback)
	validateSetupVerifications(v, value.Verification, value.State == SetupPlanComplete)
	if value.ProjectCodeWrites != 0 {
		v.add("setup_plan.project_code_writes_forbidden", "project_code_writes", "setup plan must state zero project code writes")
	}
	validateNoSecretPayload(v, value)
	return v.result()
}

func ValidatePlanAuthorizationReference(value PlanAuthorizationReference) error {
	v := &validator{}
	validateSetupHeader(v, "plan_authorization", value.Schema, PlanAuthorizationSchemaV1, value.ProjectID)
	validateDigestField(v, "plan_authorization.digest_invalid", "plan_digest", value.PlanDigest)
	validateEvidenceReferenceField(v, "plan_authorization.decision_ref_invalid", "decision_ref", value.DecisionRef, true)
	validateEvidenceReferenceField(v, "plan_authorization.actor_ref_invalid", "actor_ref", value.ActorRef, true)
	if value.AuthorizedAt.IsZero() {
		v.add("plan_authorization.time_required", "authorized_at", "authorization time is required")
	}
	validateNoSecretPayload(v, value)
	return v.result()
}

func ValidateSetupReceipt(value SetupReceipt) error {
	v := &validator{}
	validateSetupHeader(v, "setup_receipt", value.Schema, SetupReceiptSchemaV1, value.ProjectID)
	validateDigestField(v, "setup_receipt.plan_digest_invalid", "plan_digest", value.PlanDigest)
	switch value.Status {
	case SetupReceiptSuccess, SetupReceiptFailure, SetupReceiptRefused, SetupReceiptNoWrite:
	default:
		v.add("setup_receipt.status_invalid", "status", "unsupported setup receipt status")
	}
	if value.Authorization == nil {
		if value.Status != SetupReceiptRefused && value.Status != SetupReceiptNoWrite {
			v.add("setup_receipt.authorization_required", "authorization", "mutating setup attempt requires exact-plan authorization")
		}
	} else {
		v.absorb(ValidatePlanAuthorizationReference(*value.Authorization))
		if value.Authorization.PlanDigest != value.PlanDigest || value.Authorization.ProjectID != value.ProjectID {
			v.add("setup_receipt.authorization_mismatch", "authorization", "authorization must bind this project and plan digest")
		}
	}
	validateSetupComponentResults(v, value.Components)
	validateSetupMutationResults(v, value.Mutations)
	validateEvidenceReferenceField(v, "setup_receipt.rollback_ref_invalid", "rollback_ref", value.RollbackRef, value.Status == SetupReceiptFailure)
	validateEvidenceReferenceField(v, "setup_receipt.diagnosis_ref_invalid", "diagnosis_ref", value.DiagnosisRef, value.Status == SetupReceiptSuccess)
	validateEvidenceReferenceSet(v, "blocker_refs", value.BlockerRefs, MaxSetupItems, false)
	validateEvidenceReferenceField(v, "setup_receipt.continuation_ref_invalid", "continuation_ref", value.ContinuationRef, true)
	validateNoSecretPayload(v, value)
	return v.result()
}

func normalizeSetupProfile(value SetupProfile) SetupProfile {
	value.ScaffoldAdapters = append([]SetupAdapterPin(nil), value.ScaffoldAdapters...)
	sort.Slice(value.ScaffoldAdapters, func(first, second int) bool {
		return value.ScaffoldAdapters[first].ID < value.ScaffoldAdapters[second].ID
	})
	if value.SharedAdmissionAdapter != nil {
		copy := *value.SharedAdmissionAdapter
		value.SharedAdmissionAdapter = &copy
	}
	return value
}

func normalizeSetupPlan(value SetupPlan) SetupPlan {
	value.IncompleteReasonIDs = normalizeStringSet(value.IncompleteReasonIDs)
	value.Components = append([]SetupComponent(nil), value.Components...)
	sort.Slice(value.Components, func(first, second int) bool { return value.Components[first].ID < value.Components[second].ID })
	value.Mutations = append([]SetupMutation(nil), value.Mutations...)
	sort.Slice(value.Mutations, func(first, second int) bool { return value.Mutations[first].ID < value.Mutations[second].ID })
	value.Prerequisites = append([]SetupPrerequisite(nil), value.Prerequisites...)
	sort.Slice(value.Prerequisites, func(first, second int) bool { return value.Prerequisites[first].ID < value.Prerequisites[second].ID })
	value.TrustSteps = append([]SetupTrustStep(nil), value.TrustSteps...)
	sort.Slice(value.TrustSteps, func(first, second int) bool { return value.TrustSteps[first].ID < value.TrustSteps[second].ID })
	value.NetworkAccess = append([]SetupNetworkAccess(nil), value.NetworkAccess...)
	sort.Slice(value.NetworkAccess, func(first, second int) bool { return value.NetworkAccess[first].ID < value.NetworkAccess[second].ID })
	value.Rollback = append([]SetupRollbackAction(nil), value.Rollback...)
	sort.Slice(value.Rollback, func(first, second int) bool {
		return value.Rollback[first].MutationID < value.Rollback[second].MutationID
	})
	value.Verification = append([]SetupVerification(nil), value.Verification...)
	for index := range value.Verification {
		value.Verification[index].Argv = append([]string(nil), value.Verification[index].Argv...)
	}
	sort.Slice(value.Verification, func(first, second int) bool { return value.Verification[first].ID < value.Verification[second].ID })
	return value
}

func normalizeSetupReceipt(value SetupReceipt) SetupReceipt {
	if value.Authorization != nil {
		copy := *value.Authorization
		copy.AuthorizedAt = copy.AuthorizedAt.UTC()
		value.Authorization = &copy
	}
	value.Components = append([]SetupComponentResult(nil), value.Components...)
	sort.Slice(value.Components, func(first, second int) bool { return value.Components[first].ID < value.Components[second].ID })
	value.Mutations = append([]SetupMutationResult(nil), value.Mutations...)
	sort.Slice(value.Mutations, func(first, second int) bool { return value.Mutations[first].ID < value.Mutations[second].ID })
	value.BlockerRefs = normalizeStringSet(value.BlockerRefs)
	return value
}

func validateSetupHeader(v *validator, prefix, schema, expected string, projectID ProjectID) {
	if schema != expected {
		v.add(prefix+".schema.invalid", "schema", "unsupported schema")
	}
	if !IsCanonicalID(string(projectID)) || !strings.HasPrefix(string(projectID), "prj_") {
		v.add(prefix+".project_id.invalid", "project_id", "project ID must be canonical")
	}
}

func validatePlanningAdapter(v *validator, path string, value SetupPlanningAdapter) {
	validateAdapterPin(v, path+".adapter", value.Adapter)
	validateSetupString(v, "setup_profile.runtime_invalid", path+".runtime", value.Runtime, true)
	validateSetupString(v, "setup_profile.runtime_version_invalid", path+".runtime_version", value.RuntimeVersion, true)
	validateSetupString(v, "setup_profile.compiler_invalid", path+".compiler", value.Compiler, true)
	validateSetupString(v, "setup_profile.compiler_version_invalid", path+".compiler_version", value.CompilerVersion, true)
}

func validateAdapterPins(v *validator, path string, values []SetupAdapterPin) {
	if len(values) > MaxSetupItems {
		v.add("setup_profile.adapters.too_many", path, "adapter count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		validateAdapterPin(v, itemPath, value)
		if _, exists := ids[value.ID]; exists {
			v.add("setup_profile.adapter.id_duplicate", itemPath+".id", "adapter IDs must be unique")
		}
		ids[value.ID] = struct{}{}
	}
}

func validateAdapterPin(v *validator, path string, value SetupAdapterPin) {
	if !IsCanonicalID(value.ID) {
		v.add("setup_profile.adapter.id_invalid", path+".id", "adapter ID must be canonical")
	}
	validateSetupString(v, "setup_profile.adapter.version_invalid", path+".version", value.Version, true)
	validateEvidenceReferenceField(v, "setup_profile.adapter.source_invalid", path+".source_ref", value.SourceRef, true)
	validateDigestField(v, "setup_profile.adapter.integrity_invalid", path+".integrity", value.Integrity)
}

func validateSetupComponents(v *validator, values []SetupComponent, required bool) {
	if required && len(values) == 0 {
		v.add("setup_plan.components.required", "components", "setup plan requires at least one component")
	}
	if len(values) > MaxSetupItems {
		v.add("setup_plan.components.too_many", "components", "component count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("components[%d]", index)
		validateUniqueComponentID(v, ids, "setup_plan.component", path, value.ID)
		validateSetupString(v, "setup_plan.component.version_invalid", path+".version", value.Version, true)
		validateEvidenceReferenceField(v, "setup_plan.component.source_invalid", path+".source_ref", value.SourceRef, true)
		validateDigestField(v, "setup_plan.component.integrity_invalid", path+".integrity", value.Integrity)
		if value.SizeBytes == 0 {
			v.add("setup_plan.component.size_invalid", path+".size_bytes", "component size must be positive")
		}
		validateSetupPath(v, path+".destination", value.Scope, value.Destination)
		validateEvidenceReferenceField(v, "setup_plan.component.license_invalid", path+".license_ref", value.LicenseRef, true)
		validateEvidenceReferenceField(v, "setup_plan.component.provenance_invalid", path+".provenance_ref", value.ProvenanceRef, true)
	}
}

func validateSetupMutations(v *validator, values []SetupMutation) {
	if len(values) > MaxSetupItems {
		v.add("setup_plan.mutations.too_many", "mutations", "mutation count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("mutations[%d]", index)
		validateUniqueSetupID(v, ids, "setup_plan.mutation", path, value.ID)
		switch value.Kind {
		case SetupMutationInstallFile, SetupMutationEditConfig, SetupMutationInstallHook, SetupMutationInstallAdapter, SetupMutationSelectExecutable:
		default:
			v.add("setup_plan.mutation.kind_invalid", path+".kind", "unsupported setup mutation kind")
		}
		validateSetupPath(v, path+".path", value.Scope, value.Path)
		validateOptionalDigestField(v, "setup_plan.mutation.before_digest_invalid", path+".expected_before_digest", value.ExpectedBeforeDigest)
		validateDigestField(v, "setup_plan.mutation.desired_digest_invalid", path+".desired_digest", value.DesiredDigest)
	}
}

func validateSetupPrerequisites(v *validator, values []SetupPrerequisite) {
	if len(values) > MaxSetupItems {
		v.add("setup_plan.prerequisites.too_many", "prerequisites", "prerequisite count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("prerequisites[%d]", index)
		validateUniqueSetupID(v, ids, "setup_plan.prerequisite", path, value.ID)
		switch value.Kind {
		case SetupPrerequisiteRuntime, SetupPrerequisitePlatform, SetupPrerequisiteFilesystem, SetupPrerequisiteTrust:
		default:
			v.add("setup_plan.prerequisite.kind_invalid", path+".kind", "unsupported prerequisite kind")
		}
		validateSetupString(v, "setup_plan.prerequisite.version_invalid", path+".version_constraint", value.VersionConstraint, true)
		validateEvidenceReferenceField(v, "setup_plan.prerequisite.evidence_invalid", path+".evidence_ref", value.EvidenceRef, true)
	}
}

func validateSetupTrustSteps(v *validator, values []SetupTrustStep) {
	if len(values) > MaxSetupItems {
		v.add("setup_plan.trust_steps.too_many", "trust_steps", "trust-step count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("trust_steps[%d]", index)
		validateUniqueSetupID(v, ids, "setup_plan.trust_step", path, value.ID)
		if !IsCanonicalID(value.AdapterID) {
			v.add("setup_plan.trust_step.adapter_invalid", path+".adapter_id", "adapter ID must be canonical")
		}
		validateEvidenceReferenceField(v, "setup_plan.trust_step.action_invalid", path+".action_ref", value.ActionRef, true)
		if !value.Interactive {
			v.add("setup_plan.trust_step.interactive_required", path+".interactive", "provider trust step must remain interactive")
		}
	}
}

func validateSetupNetworkAccess(v *validator, values []SetupNetworkAccess) {
	if len(values) > MaxSetupItems {
		v.add("setup_plan.network.too_many", "network_access", "network access count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("network_access[%d]", index)
		validateUniqueSetupID(v, ids, "setup_plan.network", path, value.ID)
		if value.Method != "GET" && value.Method != "HEAD" {
			v.add("setup_plan.network.method_invalid", path+".method", "planning supports only GET or HEAD reads")
		}
		parsed, err := url.Parse(value.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || len(value.URL) > MaxSetupStringBytes {
			v.add("setup_plan.network.url_invalid", path+".url", "network URL must be a bounded credential-free HTTPS URL")
		}
		if !IsCanonicalID(value.PurposeID) {
			v.add("setup_plan.network.purpose_invalid", path+".purpose_id", "network purpose must be a canonical identifier")
		}
	}
}

func validateSetupRollback(v *validator, values []SetupRollbackAction) {
	if len(values) > MaxSetupItems {
		v.add("setup_plan.rollback.too_many", "rollback", "rollback action count exceeds the v1 bound")
	}
	for index, value := range values {
		path := fmt.Sprintf("rollback[%d]", index)
		if !IsCanonicalID(value.MutationID) {
			v.add("setup_plan.rollback.mutation_invalid", path+".mutation_id", "rollback mutation ID must be canonical")
		}
		if !IsCanonicalID(value.Action) {
			v.add("setup_plan.rollback.action_invalid", path+".action", "rollback action must be a canonical identifier")
		}
		validateSetupString(v, "setup_plan.rollback.target_invalid", path+".target", value.Target, true)
		validateOptionalDigestField(v, "setup_plan.rollback.digest_invalid", path+".prior_state_digest", value.PriorStateDigest)
	}
}

func validateSetupVerifications(v *validator, values []SetupVerification, required bool) {
	if required && len(values) == 0 {
		v.add("setup_plan.verification.required", "verification", "setup plan requires at least one verification command")
	}
	if len(values) > MaxSetupItems {
		v.add("setup_plan.verification.too_many", "verification", "verification count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("verification[%d]", index)
		validateUniqueSetupID(v, ids, "setup_plan.verification", path, value.ID)
		if len(value.Argv) == 0 || len(value.Argv) > MaxSetupArguments {
			v.add("setup_plan.verification.argv_invalid", path+".argv", "verification argv must be present and bounded")
		}
		for argumentIndex, argument := range value.Argv {
			validateSetupString(v, "setup_plan.verification.argument_invalid", fmt.Sprintf("%s.argv[%d]", path, argumentIndex), argument, true)
		}
		if isShellCommand(value.Argv) {
			v.add("setup_plan.verification.shell_forbidden", path+".argv", "verification must use direct argv, not a shell command")
		}
	}
}

func validateSetupComponentResults(v *validator, values []SetupComponentResult) {
	if len(values) > MaxSetupItems {
		v.add("setup_receipt.components.too_many", "components", "component result count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("components[%d]", index)
		validateUniqueComponentID(v, ids, "setup_receipt.component", path, value.ID)
		validateSetupString(v, "setup_receipt.component.version_invalid", path+".version", value.Version, true)
		validateDigestField(v, "setup_receipt.component.integrity_invalid", path+".integrity", value.Integrity)
		validateSetupActionStatus(v, path+".status", value.Status)
	}
}

func validateSetupMutationResults(v *validator, values []SetupMutationResult) {
	if len(values) > MaxSetupItems {
		v.add("setup_receipt.mutations.too_many", "mutations", "mutation result count exceeds the v1 bound")
	}
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("mutations[%d]", index)
		validateUniqueSetupID(v, ids, "setup_receipt.mutation", path, value.ID)
		validateSetupActionStatus(v, path+".status", value.Status)
		validateOptionalDigestField(v, "setup_receipt.mutation.before_digest_invalid", path+".before_digest", value.BeforeDigest)
		validateOptionalDigestField(v, "setup_receipt.mutation.after_digest_invalid", path+".after_digest", value.AfterDigest)
	}
}

func validateSetupActionStatus(v *validator, path string, status SetupActionStatus) {
	switch status {
	case SetupActionApplied, SetupActionSkipped, SetupActionFailed, SetupActionRolledBack:
	default:
		v.add("setup_receipt.action_status_invalid", path, "unsupported setup action status")
	}
}

// validateUniqueComponentID applies the component rule rather than the
// canonical one. Only a component may carry an ecosystem's own identity; every
// other identifier in a plan is minted by Goalrail and stays canonical.
func validateUniqueComponentID(v *validator, seen map[string]struct{}, prefix, path, id string) {
	if !IsComponentID(id) {
		v.add(prefix+".id_invalid", path+".id", "component ID must be canonical or a bounded namespaced package identity")
	} else if _, exists := seen[id]; exists {
		v.add(prefix+".id_duplicate", path+".id", "IDs must be unique")
	}
	seen[id] = struct{}{}
}

func validateUniqueSetupID(v *validator, seen map[string]struct{}, prefix, path, id string) {
	if !IsCanonicalID(id) {
		v.add(prefix+".id_invalid", path+".id", "ID must be canonical")
	} else if _, exists := seen[id]; exists {
		v.add(prefix+".id_duplicate", path+".id", "IDs must be unique")
	}
	seen[id] = struct{}{}
}

func validateSetupPath(v *validator, path string, scope SetupScope, value string) {
	if len(value) == 0 || len(value) > MaxSetupStringBytes || strings.ContainsRune(value, '\x00') {
		v.add("setup.path.invalid", path, "setup path must be present and bounded")
		return
	}
	switch scope {
	case SetupScopeRepository:
		if err := validateRepositoryRelativePath(value); err != nil {
			v.add("setup.path.repository_invalid", path, err.Error())
		} else if value == "." || value != normalizeRelativePath(value) {
			v.add("setup.path.repository_noncanonical", path, "repository setup path must be normalized")
		}
	case SetupScopeUserLocal:
		if !filepath.IsAbs(value) || filepath.Clean(value) == string(filepath.Separator) || filepath.Clean(value) != value {
			v.add("setup.path.user_local_invalid", path, "user-local setup path must be resolved, absolute, and non-root")
		}
	default:
		v.add("setup.scope.invalid", path, "setup scope must be repository or user_local")
	}
}

func validateSetupString(v *validator, code, path, value string, required bool) {
	if required && strings.TrimSpace(value) == "" {
		v.add(code, path, "value is required")
		return
	}
	if len(value) > MaxSetupStringBytes || strings.ContainsRune(value, '\x00') || hasSecretShapedContent(value) {
		v.add(code, path, "value must be bounded and non-secret")
	}
}

func validateDigestField(v *validator, code, path string, digest SHA256Digest) {
	if !IsSHA256Digest(digest) {
		v.add(code, path, "digest must be a complete lowercase SHA-256 reference")
	}
}

func validateOptionalDigestField(v *validator, code, path string, digest *SHA256Digest) {
	if digest != nil {
		validateDigestField(v, code, path, *digest)
	}
}

func validateEvidenceReferenceField(v *validator, code, path, value string, required bool) {
	if value == "" && !required {
		return
	}
	if !IsEvidenceReference(value) {
		v.add(code, path, "value must be a bounded provider-neutral evidence reference")
	}
}

func validateNoSecretPayload(v *validator, value any) {
	visitStrings(reflect.ValueOf(value), func(text string) {
		if hasSecretShapedContent(text) {
			v.add("payload.secret_forbidden", "", "secret-shaped content is forbidden")
		}
	})
}

func visitStrings(value reflect.Value, visit func(string)) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		visitStrings(value.Elem(), visit)
		return
	}
	switch value.Kind() {
	case reflect.String:
		visit(value.String())
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(time.Time{}) {
			return
		}
		for index := 0; index < value.NumField(); index++ {
			visitStrings(value.Field(index), visit)
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			visitStrings(value.Index(index), visit)
		}
	}
}
