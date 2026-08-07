// Package doctor assembles the project-aware, read-only Goalrail diagnosis.
package doctor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/project"
)

const (
	SchemaV2               = "goalrail.doctor/v2"
	requiredAdmissionCheck = "goalrail/lineage"
)

type Category string

const (
	CategoryUnmanaged         Category = "unmanaged"
	CategoryMigrationRequired Category = "migration_required"
	CategoryDeclaredInvalid   Category = "declared_invalid"
	CategoryGovernanceInvalid Category = "governance_invalid"
	CategorySetupRequired     Category = "setup_required"
	CategoryLocalNotReady     Category = "local_not_ready"
	CategoryAdvisory          Category = "locally_ready_advisory"
	CategoryWorking           Category = "fully_enforced"
)

type Reason struct {
	Code       string `json:"code"`
	Layer      string `json:"layer"`
	State      string `json:"state"`
	Path       string `json:"path,omitempty"`
	Detail     string `json:"detail"`
	NextAction string `json:"next_action,omitempty"`
}

type ClaimDiagnosis struct {
	State             project.ClaimState  `json:"state"`
	Reason            project.ClaimReason `json:"reason,omitempty"`
	Detail            string              `json:"detail,omitempty"`
	DeclarationPath   string              `json:"declaration_path"`
	DeclarationDigest domain.SHA256Digest `json:"declaration_digest,omitempty"`
}

type AttachmentDiagnosis struct {
	ambient.AttachmentState
	Scope            ambient.Scope `json:"scope"`
	SupersededScope  string        `json:"superseded_scope,omitempty"`
	SupersededEvents []string      `json:"superseded_events,omitempty"`
}

type PlanningState struct {
	ProfilePath  string                `json:"profile_path"`
	ProfileState project.ArtifactState `json:"profile_state"`
	Bundle       ComponentReadiness    `json:"bundle"`
	Adapter      ComponentReadiness    `json:"adapter"`
	Runtime      ComponentReadiness    `json:"runtime"`
	Compiler     ComponentReadiness    `json:"compiler"`
	Ready        bool                  `json:"ready"`
}

type PreparedAdmissionState struct {
	Configured    bool   `json:"configured"`
	Prepared      bool   `json:"prepared"`
	AdapterID     string `json:"adapter_id,omitempty"`
	Path          string `json:"path,omitempty"`
	RequiredCheck string `json:"required_check,omitempty"`
	Note          string `json:"note"`
}

type Diagnosis struct {
	Schema     string         `json:"schema"`
	Repository string         `json:"repository"`
	Version    string         `json:"version"`
	Category   Category       `json:"category"`
	Claim      ClaimDiagnosis `json:"claim"`

	Managed               bool             `json:"managed"`
	Initialized           bool             `json:"initialized"`
	InitializedDeprecated bool             `json:"initialized_deprecated"`
	InitializedSemantics  string           `json:"initialized_semantics"`
	ProjectID             domain.ProjectID `json:"project_id,omitempty"`
	ContractVersion       string           `json:"contract_version,omitempty"`

	GoverningArtifacts project.GoverningArtifacts `json:"governing_artifacts"`
	ProjectCanon       project.CanonState         `json:"project_canon"`
	Overlay            harness.OverlayState       `json:"overlay"`
	OpenSpecSchema     string                     `json:"openspec_schema,omitempty"`
	OpenSpecPresent    bool                       `json:"openspec_schema_present"`
	Planning           PlanningState              `json:"planning"`
	Attachments        []AttachmentDiagnosis      `json:"attachments"`
	PreparedAdmission  PreparedAdmissionState     `json:"prepared_admission"`
	SharedAdmission    ActivationEvidence         `json:"shared_admission"`

	Observability harness.ObservabilityState `json:"observability"`
	Update        harness.UpdateAvailability `json:"update"`
	Review        harness.ReviewState        `json:"review"`
	Invocation    string                     `json:"invocation"`

	SetupRequired         bool     `json:"setup_required"`
	LocallyReady          bool     `json:"locally_ready"`
	LineageReady          bool     `json:"lineage_ready"`
	SharedAdmissionActive bool     `json:"shared_admission_active"`
	Working               bool     `json:"working"`
	Reasons               []Reason `json:"reasons,omitempty"`
	Problems              []string `json:"problems,omitempty"`
	NextActions           []string `json:"next_actions,omitempty"`
}

type DiagnoseInput struct {
	RepositoryRoot     string
	Home               string
	StateRoot          string
	Scaffolds          []ambient.Scaffold
	LookupEnvironment  func(string) string
	PlanningObserver   PlanningObserver
	ActivationObserver ActivationObserver
	Review             func() harness.ReviewState
	LatestRelease      func(context.Context) (version string, checkedAt time.Time, err error)
}

func Diagnose(ctx context.Context, input DiagnoseInput) (Diagnosis, error) {
	inspection, err := project.Inspect(ctx, input.RepositoryRoot)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis := Diagnosis{
		Schema: SchemaV2, Repository: inspection.WorktreeRoot, Version: harness.Version,
		Claim: ClaimDiagnosis{
			State: inspection.State, Reason: inspection.Reason, Detail: inspection.Detail,
			DeclarationPath: inspection.DeclarationPath, DeclarationDigest: inspection.DeclarationDigest,
		},
		InitializedDeprecated: true,
		InitializedSemantics:  "deprecated for one release: mirrors managed valid-declaration presence; it no longer reads a checkout-local marker",
		Invocation:            harness.PinnedNewChange,
		Update:                harness.InspectUpdateAvailability(nil, harness.Version),
	}

	if inspection.State == project.ClaimUnmanaged {
		legacy, legacyErr := project.DetectLegacyV018(ctx, inspection.WorktreeRoot)
		if legacyErr != nil {
			return Diagnosis{}, legacyErr
		}
		if legacy.Overlay == project.LegacyOverlayComplete && legacy.Marker != project.LegacyMarkerInvalid {
			diagnosis.Category = CategoryMigrationRequired
			addReason(&diagnosis, Reason{
				Code: "GOALRAIL_PROJECT_MIGRATION_REQUIRED", Layer: "claim", State: "migration_required",
				Detail:     "exact Goalrail v0.1.8 project evidence exists without a committed v1 declaration",
				NextAction: "run `gr migrate` in the project worktree, review the new project files, and commit them",
			})
			return diagnosis, nil
		}
		diagnosis.Category = CategoryUnmanaged
		addReason(&diagnosis, Reason{
			Code: "PROJECT_UNMANAGED", Layer: "claim", State: "unmanaged",
			Detail: "the canonical Goalrail project declaration is absent; no managed-project state was inspected",
		})
		return diagnosis, nil
	}
	if inspection.State != project.ClaimManaged {
		diagnosis.Category = CategoryDeclaredInvalid
		addReason(&diagnosis, Reason{
			Code: "PROJECT_DECLARATION_INVALID", Layer: "claim", State: string(inspection.State),
			Path:       domain.ProjectDeclarationPath,
			Detail:     fmt.Sprintf("%s: %s", inspection.Reason, inspection.Detail),
			NextAction: "restore `.goalrail/project.json` from a trusted Git revision or perform an explicit supported migration",
		})
		return diagnosis, nil
	}

	diagnosis.Managed = true
	diagnosis.Initialized = true
	diagnosis.ProjectID = inspection.Declaration.ProjectID
	diagnosis.ContractVersion = inspection.Declaration.ContractVersion

	diagnosis.GoverningArtifacts, err = project.InspectGoverningArtifacts(inspection)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis.ProjectCanon, err = project.InspectCurrentCanon(inspection)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis.Overlay, err = harness.InspectOverlay(inspection.WorktreeRoot)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis.OpenSpecSchema, diagnosis.OpenSpecPresent, err = harness.ConfiguredSchema(inspection.WorktreeRoot)
	if err != nil {
		return Diagnosis{}, err
	}

	diagnoseGoverningArtifacts(&diagnosis)
	diagnoseProjectCanon(&diagnosis)
	diagnoseOverlay(&diagnosis)
	diagnosePlanning(ctx, &diagnosis, input.PlanningObserver, input.Home)
	diagnoseAttachments(&diagnosis, input)
	diagnosePreparedAdmission(ctx, &diagnosis, input.ActivationObserver)

	diagnosis.Observability = harness.InspectObservability(input.StateRoot, input.LookupEnvironment)
	diagnosis.Update = harness.InspectUpdateAvailability(input.LatestRelease, harness.Version)
	if input.Review != nil {
		diagnosis.Review = input.Review()
	}

	diagnosis.LineageReady = diagnosis.GoverningArtifacts.PolicyReady()
	anyAttachment := false
	for _, attachment := range diagnosis.Attachments {
		anyAttachment = anyAttachment || attachment.Working
	}
	schemaReady := diagnosis.OpenSpecPresent && diagnosis.OpenSpecSchema == harness.SchemaName
	diagnosis.LocallyReady = diagnosis.GoverningArtifacts.SetupReady() && diagnosis.ProjectCanon.Current &&
		diagnosis.Overlay.Settled() && schemaReady && diagnosis.Planning.Ready && anyAttachment
	diagnosis.SharedAdmissionActive = diagnosis.SharedAdmission.State == ActivationActive
	diagnosis.Working = diagnosis.Managed && diagnosis.LocallyReady && diagnosis.LineageReady && diagnosis.SharedAdmissionActive

	diagnosis.SetupRequired = !diagnosis.Planning.Ready || !anyAttachment
	if diagnosis.SetupRequired {
		addReasonOnce(&diagnosis, Reason{
			Code: "GOALRAIL_SETUP_REQUIRED", Layer: "local_readiness", State: "setup_required",
			Detail:     "one or more declared local planning or scaffold components are not ready",
			NextAction: "ask the supported agent to present the exact read-only Goalrail setup plan; do not install an ad hoc global dependency",
		})
	}

	switch {
	case !diagnosis.LineageReady:
		diagnosis.Category = CategoryGovernanceInvalid
	case diagnosis.SetupRequired:
		diagnosis.Category = CategorySetupRequired
	case !diagnosis.LocallyReady:
		diagnosis.Category = CategoryLocalNotReady
	case !diagnosis.SharedAdmissionActive:
		diagnosis.Category = CategoryAdvisory
	default:
		diagnosis.Category = CategoryWorking
	}
	return diagnosis, nil
}

func diagnoseGoverningArtifacts(diagnosis *Diagnosis) {
	for _, finding := range []project.ArtifactFinding{
		diagnosis.GoverningArtifacts.Policy,
		diagnosis.GoverningArtifacts.Bootstrap,
		diagnosis.GoverningArtifacts.SetupProfile,
	} {
		if finding.State == project.ArtifactCurrent {
			continue
		}
		code := "GOVERNING_ARTIFACT_INVALID"
		next := "restore the declaration-bound bytes from a trusted Git revision or use the explicit governed amendment flow"
		if finding.Ownership == project.OwnershipCanon {
			code = "PROJECT_CANON_DRIFT"
			next = "run `gr update`; use `--discard-local-edits` only after reviewing the exact canon-owned diff"
		}
		addReason(diagnosis, Reason{
			Code: code, Layer: "governance", State: string(finding.State), Path: finding.Path,
			Detail: nonEmpty(finding.Detail, "declaration-bound artifact is not current"), NextAction: next,
		})
	}
}

func diagnoseProjectCanon(diagnosis *Diagnosis) {
	for _, findings := range [][]project.CanonPathFinding{diagnosis.ProjectCanon.Files, diagnosis.ProjectCanon.ManagedBlocks} {
		for _, finding := range findings {
			if finding.State == project.CanonPathCurrent {
				continue
			}
			next := "run `gr update`"
			if finding.State == project.CanonPathDrift {
				next = "review the drift, then keep it or run `gr update --discard-local-edits` for the exact canon-owned file or block"
			}
			addReason(diagnosis, Reason{
				Code: "PROJECT_CANON_" + strings.ToUpper(string(finding.State)), Layer: "project_canon",
				State: string(finding.State), Path: finding.Path, Detail: finding.Detail, NextAction: next,
			})
		}
	}
}

func diagnoseOverlay(diagnosis *Diagnosis) {
	if !diagnosis.Overlay.Present {
		addReason(diagnosis, Reason{Code: "HARNESS_OVERLAY_MISSING", Layer: "overlay", State: "missing", Detail: "the declared project has no OpenSpec harness overlay", NextAction: "run `gr update`"})
	}
	for _, finding := range diagnosis.Overlay.Drifted() {
		next := "run `gr update`"
		if finding.State == harness.FileEdited {
			next = "review the edit, then keep it or run `gr update --discard-local-edits`"
		}
		addReason(diagnosis, Reason{
			Code: "HARNESS_OVERLAY_" + strings.ToUpper(string(finding.State)), Layer: "overlay",
			State: string(finding.State), Path: finding.Path, Detail: "overlay path does not match the installed canon", NextAction: next,
		})
	}
	if !diagnosis.OpenSpecPresent || diagnosis.OpenSpecSchema != harness.SchemaName {
		addReason(diagnosis, Reason{
			Code: "PLANNING_SCHEMA_NOT_READY", Layer: "planning", State: "incompatible",
			Path: "openspec/config.yaml", Detail: fmt.Sprintf("OpenSpec schema is %q; Goalrail requires %q", diagnosis.OpenSpecSchema, harness.SchemaName),
			NextAction: "run `gr update`; an owner-selected foreign schema still requires explicit resolution",
		})
	}
}

func diagnosePlanning(ctx context.Context, diagnosis *Diagnosis, observer PlanningObserver, home string) {
	profileFinding := diagnosis.GoverningArtifacts.SetupProfile
	diagnosis.Planning.ProfilePath = profileFinding.Path
	diagnosis.Planning.ProfileState = profileFinding.State
	if profileFinding.State != project.ArtifactCurrent {
		diagnosis.Planning.Bundle = invalidComponent("goalrail_bundle", "goalrail", "setup profile is not valid")
		diagnosis.Planning.Adapter = invalidComponent("planning_adapter", "", "setup profile is not valid")
		diagnosis.Planning.Runtime = invalidComponent("runtime", "", "setup profile is not valid")
		diagnosis.Planning.Compiler = invalidComponent("compiler", "", "setup profile is not valid")
		return
	}
	profile := diagnosis.GoverningArtifacts.SetupValue
	if observer == nil {
		observer = defaultPlanningObserver{home: home, version: harness.Version}
	}
	observation := observer.ObservePlanning(ctx, profile)
	observation.Bundle.Kind, observation.Bundle.ID = "goalrail_bundle", "goalrail"
	observation.Bundle.RequiredVersion = profile.CompatibleGoalrailBundle
	observation.Runtime.Kind, observation.Runtime.ID = "runtime", profile.Planning.Runtime
	observation.Runtime.RequiredVersion = profile.Planning.RuntimeVersion
	observation.Compiler.Kind, observation.Compiler.ID = "compiler", profile.Planning.Compiler
	observation.Compiler.RequiredVersion = profile.Planning.CompilerVersion
	for _, component := range []*ComponentReadiness{&observation.Bundle, &observation.Runtime, &observation.Compiler} {
		if component.State == "" {
			component.State = ComponentInvalid
			component.Detail = nonEmpty(component.Detail, "read-only component observer returned no state")
		}
	}
	diagnosis.Planning.Bundle = observation.Bundle
	diagnosis.Planning.Runtime = observation.Runtime
	diagnosis.Planning.Compiler = observation.Compiler
	diagnosis.Planning.Adapter = ComponentReadiness{
		Kind: "planning_adapter", ID: profile.Planning.Adapter.ID,
		RequiredVersion: profile.Planning.Adapter.Version, RequiredIntegrity: profile.Planning.Adapter.Integrity,
		State: ComponentMissing,
	}
	for _, finding := range diagnosis.ProjectCanon.Files {
		if finding.Path != project.PlanningAdapterDescriptorPath {
			continue
		}
		diagnosis.Planning.Adapter.Path = finding.Path
		switch {
		case finding.State == project.CanonPathMissing:
			diagnosis.Planning.Adapter.State = ComponentMissing
		case finding.ObservedDigest != profile.Planning.Adapter.Integrity:
			diagnosis.Planning.Adapter.State = ComponentIncompatible
			diagnosis.Planning.Adapter.Detail = "adapter bytes do not match the exact setup-profile integrity"
		default:
			diagnosis.Planning.Adapter.State = ComponentReady
		}
	}
	diagnosis.Planning.Ready = componentReady(diagnosis.Planning.Bundle) && componentReady(diagnosis.Planning.Adapter) &&
		componentReady(diagnosis.Planning.Runtime) && componentReady(diagnosis.Planning.Compiler)
	if diagnosis.Planning.Ready {
		return
	}
	for _, component := range []ComponentReadiness{
		diagnosis.Planning.Bundle, diagnosis.Planning.Adapter, diagnosis.Planning.Runtime, diagnosis.Planning.Compiler,
	} {
		if componentReady(component) {
			continue
		}
		addReason(diagnosis, Reason{
			Code: "PLANNING_COMPONENT_" + strings.ToUpper(string(component.State)), Layer: "planning",
			State: string(component.State), Path: component.Path,
			Detail:     fmt.Sprintf("%s %s requires %s; observed %s. %s", component.Kind, component.ID, component.RequiredVersion, component.ObservedVersion, component.Detail),
			NextAction: "request the exact Goalrail setup plan for the declared profile",
		})
	}
}

func diagnoseAttachments(diagnosis *Diagnosis, input DiagnoseInput) {
	scaffolds := input.Scaffolds
	if len(scaffolds) == 0 {
		scaffolds = ambient.SupportedScaffolds()
	} else {
		scaffolds = append([]ambient.Scaffold(nil), scaffolds...)
	}
	sort.Slice(scaffolds, func(i, j int) bool { return scaffolds[i] < scaffolds[j] })
	for _, scaffold := range scaffolds {
		scope, err := ambient.ScopeOf(scaffold)
		if err != nil {
			addReason(diagnosis, Reason{Code: "ATTACHMENT_INVALID", Layer: "attachment", State: "invalid", Detail: err.Error()})
			continue
		}
		state, err := ambient.InspectForProject(scaffold, input.Home, diagnosis.Repository, string(diagnosis.ProjectID))
		if err != nil {
			addReason(diagnosis, Reason{Code: "ATTACHMENT_UNREADABLE", Layer: "attachment", State: "invalid", Detail: err.Error()})
			continue
		}
		attachment := AttachmentDiagnosis{AttachmentState: state, Scope: scope}
		if superseded, present := ambient.SupersededTarget(scaffold, input.Home); present {
			if occupied, occupiedErr := ambient.HasAnyRegistration(superseded); occupiedErr == nil && occupied {
				attachment.SupersededScope = superseded.Path
			}
		}
		if target, targetErr := ambient.RegistrationTarget(scaffold, input.Home, diagnosis.Repository); targetErr == nil {
			attachment.SupersededEvents, _ = ambient.SupersededRegistrations(target)
		}
		diagnosis.Attachments = append(diagnosis.Attachments, attachment)
		if !attachment.Working {
			code := "ATTACHMENT_NOT_READY"
			stateName := "disconnected"
			if attachment.Connected {
				stateName = "trust_or_executable_not_ready"
			}
			addReason(diagnosis, Reason{
				Code: code, Layer: "attachment", State: stateName, Path: attachment.ConfigPath,
				Detail:     fmt.Sprintf("%s local attachment is not working; trust=%s", scaffold, attachment.Trust),
				NextAction: nonEmpty(attachment.NextAction, "request the exact Goalrail setup plan for this scaffold"),
			})
		}
	}
	sort.Slice(diagnosis.Attachments, func(i, j int) bool { return diagnosis.Attachments[i].Scaffold < diagnosis.Attachments[j].Scaffold })
}

func diagnosePreparedAdmission(ctx context.Context, diagnosis *Diagnosis, observer ActivationObserver) {
	profile := diagnosis.GoverningArtifacts.SetupValue
	prepared := PreparedAdmissionState{
		RequiredCheck: requiredAdmissionCheck,
		Path:          project.PreparedAdmissionPath,
		Note:          "repository presence is preparation only; it never proves a protected required check",
	}
	if diagnosis.GoverningArtifacts.SetupProfile.State == project.ArtifactCurrent && profile.SharedAdmissionAdapter != nil {
		prepared.Configured = true
		prepared.AdapterID = profile.SharedAdmissionAdapter.ID
	}
	for _, finding := range diagnosis.ProjectCanon.Files {
		if finding.Path == project.PreparedAdmissionPath && finding.State == project.CanonPathCurrent {
			prepared.Prepared = true
		}
	}
	diagnosis.PreparedAdmission = prepared
	if !prepared.Configured {
		diagnosis.SharedAdmission = ActivationEvidence{
			State: ActivationInactive, Freshness: ActivationFreshUnverified,
			RequiredCheck: requiredAdmissionCheck,
			Detail:        "the committed setup profile configures no shared-admission boundary",
		}
		addReason(diagnosis, Reason{
			Code: "SHARED_ADMISSION_INACTIVE", Layer: "shared_admission", State: string(ActivationInactive),
			Detail:     diagnosis.SharedAdmission.Detail,
			NextAction: "create an explicit governed setup-profile amendment before any external activation action",
		})
		return
	}
	request := ActivationRequest{
		ProjectID: diagnosis.ProjectID, AdapterID: prepared.AdapterID,
		RequiredCheck: requiredAdmissionCheck, PreparedPath: prepared.Path,
	}
	diagnosis.SharedAdmission = activationEvidence(ctx, observer, request)
	if diagnosis.SharedAdmission.State != ActivationActive {
		code := "SHARED_ADMISSION_" + strings.ToUpper(string(diagnosis.SharedAdmission.State))
		next := "inspect or authorize read-only observation of the protected target; activation remains external and owner-authorized"
		addReason(diagnosis, Reason{
			Code: code, Layer: "shared_admission", State: string(diagnosis.SharedAdmission.State),
			Detail: nonEmpty(diagnosis.SharedAdmission.Detail, "protected required-check activation is not currently proven"), NextAction: next,
		})
	}
}

func invalidComponent(kind, id, detail string) ComponentReadiness {
	return ComponentReadiness{Kind: kind, ID: id, State: ComponentInvalid, Detail: detail}
}

func componentReady(component ComponentReadiness) bool {
	return component.State == ComponentReady || component.State == ComponentNotRequired
}

func addReason(diagnosis *Diagnosis, reason Reason) {
	diagnosis.Reasons = append(diagnosis.Reasons, reason)
	diagnosis.Problems = append(diagnosis.Problems, reason.Detail)
	diagnosis.NextActions = append(diagnosis.NextActions, reason.NextAction)
}

func addReasonOnce(diagnosis *Diagnosis, reason Reason) {
	for _, existing := range diagnosis.Reasons {
		if existing.Code == reason.Code {
			return
		}
	}
	addReason(diagnosis, reason)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
