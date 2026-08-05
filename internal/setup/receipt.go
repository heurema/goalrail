package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/doctor"
	"github.com/heurema/goalrail/internal/domain"
)

const (
	ReceiptEmissionSchemaV1  = "goalrail.setup-receipt-emission/v1"
	DiagnosisSummarySchemaV1 = "goalrail.setup-diagnosis-summary/v1"
)

var (
	ErrContinuationRequired = errors.New("setup continuation reference is required")
	ErrDiagnosisNotReady    = errors.New("setup diagnosis did not establish local readiness")
	ErrNoWriteReasonInvalid = errors.New("setup no-write reason is invalid")
)

type NoWriteReason string

const (
	NoWriteRefused              NoWriteReason = "refused"
	NoWriteMissingAuthorization NoWriteReason = "missing-authorization"
)

// ReceiptEmission transports one canonical setup receipt together with only
// the machine-local recovery locator and bounded doctor summary needed by the
// caller. Receipt remains the sole canonical setup outcome artifact.
type ReceiptEmission struct {
	Schema        string                 `json:"schema"`
	Receipt       domain.SetupReceipt    `json:"receipt"`
	ReceiptDigest domain.SHA256Digest    `json:"receipt_digest"`
	RecoveryPath  string                 `json:"recovery_path,omitempty"`
	Diagnosis     *SetupDiagnosisSummary `json:"diagnosis,omitempty"`
}

// SetupDiagnosisSummary deliberately excludes repository paths, diagnostic
// prose, environment values, and request content. Shared activation remains a
// separate observed layer and never upgrades local setup authority.
type SetupDiagnosisSummary struct {
	Schema                string   `json:"schema"`
	Category              string   `json:"category"`
	Managed               bool     `json:"managed"`
	LocallyReady          bool     `json:"locally_ready"`
	LineageReady          bool     `json:"lineage_ready"`
	SharedAdmissionState  string   `json:"shared_admission_state"`
	SharedAdmissionActive bool     `json:"shared_admission_active"`
	Working               bool     `json:"working"`
	ReasonRefs            []string `json:"reason_refs"`
}

type NoWriteReceiptOptions struct {
	PlanPath        string
	ContinuationRef string
	Reason          NoWriteReason
}

// ExecuteApply owns the complete authorized attempt: exact-plan verification,
// bounded mutation, diagnosis, and canonical terminal receipt emission.
func ExecuteApply(ctx context.Context, options VerifyPlanOptions, continuationRef string) (ReceiptEmission, error) {
	if !domain.IsEvidenceReference(continuationRef) {
		return ReceiptEmission{}, ErrContinuationRequired
	}

	verified, err := VerifyPlan(ctx, options)
	if err != nil {
		emission, receiptErr := verificationFailureReceipt(options, continuationRef)
		if receiptErr != nil {
			return ReceiptEmission{}, err
		}
		return emission, err
	}
	return ApplyWithReceipt(ctx, verified, continuationRef)
}

// ApplyWithReceipt applies a non-constructible verified plan and emits exactly
// one terminal canonical receipt. Doctor runs only after apply succeeds.
func ApplyWithReceipt(ctx context.Context, verified VerifiedPlan, continuationRef string) (ReceiptEmission, error) {
	if !domain.IsEvidenceReference(continuationRef) {
		return ReceiptEmission{}, ErrContinuationRequired
	}
	plan, authorization, err := receiptAuthority(verified)
	if err != nil {
		return ReceiptEmission{}, err
	}

	result, applyErr := Apply(ctx, verified)
	if applyErr != nil {
		if result.FailureStage == "" {
			switch {
			case errors.Is(applyErr, ErrApplyStateChanged):
				result.FailureStage = "final-plan-verification"
			case errors.Is(applyErr, ErrApplyInvalid):
				result.FailureStage = "apply-contract"
			default:
				result.FailureStage = "apply"
			}
		}
		emission, receiptErr := buildApplyEmission(plan, authorization, result, nil, continuationRef)
		if receiptErr != nil {
			return ReceiptEmission{}, fmt.Errorf("emit failed setup receipt: %w", receiptErr)
		}
		return emission, applyErr
	}

	diagnosis, diagnoseErr := doctor.Diagnose(ctx, doctor.DiagnoseInput{
		RepositoryRoot: verified.options.RepositoryRoot,
		Home:           verified.options.Home,
		Scaffolds:      []ambient.Scaffold{verified.options.Scaffold},
	})
	if diagnoseErr != nil {
		result.FailureStage = "doctor"
		emission, receiptErr := buildApplyEmission(plan, authorization, result, nil, continuationRef)
		if receiptErr != nil {
			return ReceiptEmission{}, fmt.Errorf("emit doctor-failure setup receipt: %w", receiptErr)
		}
		return emission, diagnoseErr
	}
	emission, err := buildApplyEmission(plan, authorization, result, &diagnosis, continuationRef)
	if err != nil {
		return ReceiptEmission{}, err
	}
	if diagnosis.ProjectID != plan.ProjectID || !diagnosis.Managed || !diagnosis.LocallyReady || !diagnosis.LineageReady {
		return emission, ErrDiagnosisNotReady
	}
	return emission, nil
}

// EmitNoWriteReceipt records a refusal or missing exact authorization without
// accepting authorization, running doctor, or mutating repository/machine state.
func EmitNoWriteReceipt(options NoWriteReceiptOptions) (ReceiptEmission, error) {
	if !domain.IsEvidenceReference(options.ContinuationRef) {
		return ReceiptEmission{}, ErrContinuationRequired
	}
	plan, artifact, err := readCanonicalPlan(options.PlanPath)
	if err != nil {
		return ReceiptEmission{}, err
	}
	if _, err := validateCompletePlan(Result{Plan: plan, Artifact: artifact}); err != nil {
		return ReceiptEmission{}, err
	}

	status := domain.SetupReceiptNoWrite
	blocker := "setup-authorization:missing"
	switch options.Reason {
	case NoWriteRefused:
		status = domain.SetupReceiptRefused
		blocker = "setup-authorization:refused"
	case NoWriteMissingAuthorization:
	default:
		return ReceiptEmission{}, ErrNoWriteReasonInvalid
	}
	receipt := domain.SetupReceipt{
		Schema:          domain.SetupReceiptSchemaV1,
		ProjectID:       plan.ProjectID,
		PlanDigest:      artifact.Digest(),
		Components:      componentResults(plan, false, false),
		Mutations:       mutationResults(plan, ApplyResult{}),
		RollbackRef:     "setup-rollback:not-needed",
		DiagnosisRef:    "doctor:not-run-no-write",
		Status:          status,
		BlockerRefs:     []string{blocker},
		ContinuationRef: options.ContinuationRef,
	}
	return freezeEmission(receipt, "", nil)
}

func receiptAuthority(verified VerifiedPlan) (domain.SetupPlan, domain.PlanAuthorizationReference, error) {
	plan, err := domain.DecodeSetupPlan(bytes.NewReader(verified.attempt.PlanArtifact().CanonicalJSON()))
	if err != nil {
		return domain.SetupPlan{}, domain.PlanAuthorizationReference{}, err
	}
	authorization, err := domain.DecodePlanAuthorizationReference(bytes.NewReader(verified.attempt.AuthorizationArtifact().CanonicalJSON()))
	if err != nil {
		return domain.SetupPlan{}, domain.PlanAuthorizationReference{}, err
	}
	return plan, authorization, nil
}

func verificationFailureReceipt(options VerifyPlanOptions, continuationRef string) (ReceiptEmission, error) {
	plan, planArtifact, err := readCanonicalPlan(options.PlanPath)
	if err != nil {
		return ReceiptEmission{}, err
	}
	authorization, _, err := readCanonicalAuthorization(options.AuthorizationPath)
	if err != nil {
		return ReceiptEmission{}, err
	}
	if authorization.ProjectID != plan.ProjectID || authorization.PlanDigest != planArtifact.Digest() {
		return ReceiptEmission{}, ErrAuthorizationStale
	}
	receipt := domain.SetupReceipt{
		Schema:          domain.SetupReceiptSchemaV1,
		ProjectID:       plan.ProjectID,
		PlanDigest:      planArtifact.Digest(),
		Authorization:   &authorization,
		Components:      componentResults(plan, false, false),
		Mutations:       mutationResults(plan, ApplyResult{}),
		RollbackRef:     "setup-rollback:none-required",
		DiagnosisRef:    "doctor:not-run-verification-failed",
		Status:          domain.SetupReceiptFailure,
		BlockerRefs:     []string{"setup-stage:verify-plan"},
		ContinuationRef: continuationRef,
	}
	return freezeEmission(receipt, "", nil)
}

func buildApplyEmission(
	plan domain.SetupPlan,
	authorization domain.PlanAuthorizationReference,
	result ApplyResult,
	diagnosis *doctor.Diagnosis,
	continuationRef string,
) (ReceiptEmission, error) {
	status := domain.SetupReceiptFailure
	diagnosisRef := "doctor:not-run-setup-failed"
	blockers := applyBlockerRefs(result)
	var summary *SetupDiagnosisSummary
	if diagnosis != nil {
		value := summarizeDiagnosis(*diagnosis)
		summary = &value
		diagnosisRef = diagnosisReference(value)
		if diagnosis.ProjectID == plan.ProjectID && diagnosis.Managed && diagnosis.LocallyReady && diagnosis.LineageReady {
			status = domain.SetupReceiptSuccess
			blockers = nil
		} else {
			blockers = append(blockers, value.ReasonRefs...)
		}
	}
	rollbackRef := "setup-rollback:none-required"
	if result.RecoveryDigest != "" {
		rollbackRef = "setup-recovery:sha256:" + string(result.RecoveryDigest)
	}
	receipt := domain.SetupReceipt{
		Schema:          domain.SetupReceiptSchemaV1,
		ProjectID:       plan.ProjectID,
		PlanDigest:      authorization.PlanDigest,
		Authorization:   &authorization,
		Components:      componentResults(plan, mutationApplied(result, "install-bundle"), result.FailedMutationID == "install-bundle"),
		Mutations:       mutationResults(plan, result),
		RollbackRef:     rollbackRef,
		DiagnosisRef:    diagnosisRef,
		Status:          status,
		BlockerRefs:     boundedReferenceSet(blockers),
		ContinuationRef: continuationRef,
	}
	return freezeEmission(receipt, result.RecoveryPath, summary)
}

func componentResults(plan domain.SetupPlan, installed, failed bool) []domain.SetupComponentResult {
	status := domain.SetupActionSkipped
	if failed {
		status = domain.SetupActionFailed
	} else if installed {
		status = domain.SetupActionApplied
	}
	results := make([]domain.SetupComponentResult, 0, len(plan.Components))
	for _, component := range plan.Components {
		results = append(results, domain.SetupComponentResult{
			ID: component.ID, Version: component.Version, Integrity: component.Integrity, Status: status,
		})
	}
	return results
}

func mutationResults(plan domain.SetupPlan, result ApplyResult) []domain.SetupMutationResult {
	results := make([]domain.SetupMutationResult, 0, len(plan.Mutations))
	for _, mutation := range plan.Mutations {
		outcome := domain.SetupMutationResult{ID: mutation.ID, Status: domain.SetupActionSkipped}
		switch {
		case mutationApplied(result, mutation.ID):
			outcome.Status = domain.SetupActionApplied
			outcome.BeforeDigest = cloneDigest(mutation.ExpectedBeforeDigest)
			after := mutation.DesiredDigest
			outcome.AfterDigest = &after
		case result.FailedMutationID == mutation.ID:
			outcome.Status = domain.SetupActionFailed
		}
		results = append(results, outcome)
	}
	return results
}

func mutationApplied(result ApplyResult, id string) bool {
	for _, applied := range result.AppliedMutationIDs {
		if applied == id {
			return true
		}
	}
	return false
}

func cloneDigest(value *domain.SHA256Digest) *domain.SHA256Digest {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func summarizeDiagnosis(value doctor.Diagnosis) SetupDiagnosisSummary {
	reasons := make([]string, 0, len(value.Reasons))
	for _, reason := range value.Reasons {
		if reason.Code != "" {
			reasons = append(reasons, "doctor-reason:"+reason.Code)
		}
	}
	return SetupDiagnosisSummary{
		Schema: DiagnosisSummarySchemaV1, Category: string(value.Category), Managed: value.Managed,
		LocallyReady: value.LocallyReady, LineageReady: value.LineageReady,
		SharedAdmissionState:  string(value.SharedAdmission.State),
		SharedAdmissionActive: value.SharedAdmissionActive, Working: value.Working,
		ReasonRefs: boundedReferenceSet(reasons),
	}
}

func diagnosisReference(value SetupDiagnosisSummary) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "doctor:summary-invalid"
	}
	return "doctor:sha256:" + string(domain.DigestCanonicalJSON(raw))
}

func applyBlockerRefs(result ApplyResult) []string {
	refs := make([]string, 0, 2+len(result.PendingTrustStepIDs))
	if result.FailureStage != "" {
		refs = append(refs, "setup-stage:"+result.FailureStage)
	}
	if result.FailedMutationID != "" {
		refs = append(refs, "setup-mutation:"+result.FailedMutationID)
	}
	for _, id := range result.PendingTrustStepIDs {
		refs = append(refs, "setup-trust-step:"+id)
	}
	return refs
}

func boundedReferenceSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !domain.IsEvidenceReference(value) {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func freezeEmission(receipt domain.SetupReceipt, recoveryPath string, diagnosis *SetupDiagnosisSummary) (ReceiptEmission, error) {
	artifact, err := domain.FreezeSetupReceipt(receipt)
	if err != nil {
		return ReceiptEmission{}, err
	}
	normalized, err := domain.DecodeSetupReceipt(bytes.NewReader(artifact.CanonicalJSON()))
	if err != nil {
		return ReceiptEmission{}, fmt.Errorf("decode canonical setup receipt: %w", err)
	}
	return ReceiptEmission{
		Schema: ReceiptEmissionSchemaV1, Receipt: normalized, ReceiptDigest: artifact.Digest(),
		RecoveryPath: recoveryPath, Diagnosis: diagnosis,
	}, nil
}
