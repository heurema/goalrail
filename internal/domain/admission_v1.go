package domain

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type (
	AdmissionClassification string
	AdmissionOutcome        string
	AdmissionReasonCode     string
)

const (
	AdmissionPacketSchemaV1 = "goalrail.admission-packet/v1"
	AdmissionResultSchemaV1 = "goalrail.admission-result/v1"
	MaxAdmissionPacketBytes = 256 << 10
	MaxAdmissionResultBytes = 256 << 10
	MaxAdmissionEvidence    = 512
	MaxAdmissionPaths       = 4096
	MaxAdmissionReasons     = 256

	AdmissionValid      AdmissionClassification = "VALID"
	AdmissionMissing    AdmissionClassification = "MISSING"
	AdmissionAmbiguous  AdmissionClassification = "AMBIGUOUS"
	AdmissionInvalid    AdmissionClassification = "INVALID"
	AdmissionExempted   AdmissionClassification = "EXEMPTED"
	AdmissionBreakGlass AdmissionClassification = "BREAK_GLASS"
	AdmissionBootstrap  AdmissionClassification = "BOOTSTRAP"

	AdmissionAllow AdmissionOutcome = "allow"
	AdmissionDeny  AdmissionOutcome = "deny"

	ReasonDeclarationInvalid       AdmissionReasonCode = "DECLARATION_INVALID"
	ReasonPolicyConflict           AdmissionReasonCode = "POLICY_CONFLICT"
	ReasonMaterialPathUnbound      AdmissionReasonCode = "MATERIAL_PATH_UNBOUND"
	ReasonIntentUnconfirmed        AdmissionReasonCode = "INTENT_UNCONFIRMED"
	ReasonChangeMismatch           AdmissionReasonCode = "CHANGE_MISMATCH"
	ReasonWorkSpecMissing          AdmissionReasonCode = "WORK_SPEC_MISSING"
	ReasonRunSessionMissing        AdmissionReasonCode = "RUN_SESSION_MISSING"
	ReasonPullRequestMissing       AdmissionReasonCode = "PULL_REQUEST_MISSING"
	ReasonReviewMissing            AdmissionReasonCode = "REVIEW_MISSING"
	ReasonCheckMissing             AdmissionReasonCode = "CHECK_MISSING"
	ReasonReceiptMissing           AdmissionReasonCode = "RECEIPT_MISSING"
	ReasonOwnerDecisionMissing     AdmissionReasonCode = "OWNER_DECISION_MISSING"
	ReasonLineageConflict          AdmissionReasonCode = "LINEAGE_CONFLICT"
	ReasonExceptionExpired         AdmissionReasonCode = "EXCEPTION_EXPIRED"
	ReasonExceptionScopeMismatch   AdmissionReasonCode = "EXCEPTION_SCOPE_MISMATCH"
	ReasonActivationUnverified     AdmissionReasonCode = "ACTIVATION_UNVERIFIED"
	ReasonBootstrapRange           AdmissionReasonCode = "BOOTSTRAP_RANGE"
	ReasonExceptionApplied         AdmissionReasonCode = "EXCEPTION_APPLIED"
	ReasonTrustedTimeMissing       AdmissionReasonCode = "TRUSTED_TIME_MISSING"
	ReasonPacketInvalid            AdmissionReasonCode = "PACKET_INVALID"
	ReasonProvenanceUntrusted      AdmissionReasonCode = "PROVENANCE_UNTRUSTED"
	ReasonRangeMismatch            AdmissionReasonCode = "RANGE_MISMATCH"
	ReasonGeneratedEvidenceMissing AdmissionReasonCode = "GENERATED_EVIDENCE_MISSING"
)

var admissionReasonRegistry = []AdmissionReasonCode{
	ReasonActivationUnverified,
	ReasonBootstrapRange,
	ReasonChangeMismatch,
	ReasonCheckMissing,
	ReasonDeclarationInvalid,
	ReasonExceptionApplied,
	ReasonExceptionExpired,
	ReasonExceptionScopeMismatch,
	ReasonGeneratedEvidenceMissing,
	ReasonIntentUnconfirmed,
	ReasonLineageConflict,
	ReasonMaterialPathUnbound,
	ReasonOwnerDecisionMissing,
	ReasonPacketInvalid,
	ReasonPolicyConflict,
	ReasonProvenanceUntrusted,
	ReasonPullRequestMissing,
	ReasonRangeMismatch,
	ReasonReceiptMissing,
	ReasonReviewMissing,
	ReasonRunSessionMissing,
	ReasonTrustedTimeMissing,
	ReasonWorkSpecMissing,
}

type AdmissionProviderProvenance struct {
	AdapterID      string       `json:"adapter_id"`
	ProviderRef    string       `json:"provider_ref"`
	EvidenceDigest SHA256Digest `json:"evidence_digest"`
	ObservedAt     time.Time    `json:"observed_at"`
	Authenticated  bool         `json:"authenticated"`
}

type AdmissionPacket struct {
	Schema            string                              `json:"schema"`
	ProjectID         ProjectID                           `json:"project_id"`
	DeclarationDigest SHA256Digest                        `json:"declaration_digest"`
	PolicyDigest      SHA256Digest                        `json:"policy_digest"`
	BaseRevision      string                              `json:"base_revision"`
	HeadRevision      string                              `json:"head_revision"`
	WorkUnitRef       ContentAddressedEvidenceReference   `json:"work_unit_ref"`
	EvaluationTime    *time.Time                          `json:"evaluation_time"`
	TimeAuthorityRef  string                              `json:"time_authority_ref"`
	Evidence          []ContentAddressedEvidenceReference `json:"evidence"`
	Provenance        []AdmissionProviderProvenance       `json:"provenance"`
}

type AdmissionReason struct {
	Code         AdmissionReasonCode `json:"code"`
	EvidenceRefs []string            `json:"evidence_refs"`
}

type AdmissionResult struct {
	Schema          string                            `json:"schema"`
	ProjectID       ProjectID                         `json:"project_id"`
	PolicyDigest    SHA256Digest                      `json:"policy_digest"`
	BaseRevision    string                            `json:"base_revision"`
	HeadRevision    string                            `json:"head_revision"`
	MaterialPaths   []string                          `json:"material_paths"`
	WorkUnitRef     ContentAddressedEvidenceReference `json:"work_unit_ref"`
	Classification  AdmissionClassification           `json:"classification"`
	Outcome         AdmissionOutcome                  `json:"outcome"`
	Reasons         []AdmissionReason                 `json:"reasons"`
	MissingRefs     []string                          `json:"missing_refs"`
	ConflictRefs    []string                          `json:"conflict_refs"`
	VerifierVersion string                            `json:"verifier_version"`
}

func KnownAdmissionReasonCodes() []AdmissionReasonCode {
	return append([]AdmissionReasonCode(nil), admissionReasonRegistry...)
}

func IsAdmissionReasonCode(value AdmissionReasonCode) bool {
	index := sort.Search(len(admissionReasonRegistry), func(index int) bool {
		return admissionReasonRegistry[index] >= value
	})
	return index < len(admissionReasonRegistry) && admissionReasonRegistry[index] == value
}

func DecodeAdmissionPacket(reader io.Reader) (AdmissionPacket, error) {
	value, err := decodeStrictBoundedJSON[AdmissionPacket](reader, MaxAdmissionPacketBytes, "admission packet")
	if err != nil {
		return AdmissionPacket{}, err
	}
	value = normalizeAdmissionPacket(value)
	if err := ValidateAdmissionPacket(value); err != nil {
		return AdmissionPacket{}, err
	}
	return value, nil
}

func FreezeAdmissionPacket(value AdmissionPacket) (CanonicalArtifact, error) {
	value = normalizeAdmissionPacket(value)
	if err := ValidateAdmissionPacket(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func DecodeAdmissionResult(reader io.Reader) (AdmissionResult, error) {
	value, err := decodeStrictBoundedJSON[AdmissionResult](reader, MaxAdmissionResultBytes, "admission result")
	if err != nil {
		return AdmissionResult{}, err
	}
	value = normalizeAdmissionResult(value)
	if err := ValidateAdmissionResult(value); err != nil {
		return AdmissionResult{}, err
	}
	return value, nil
}

func FreezeAdmissionResult(value AdmissionResult) (CanonicalArtifact, error) {
	value = normalizeAdmissionResult(value)
	if err := ValidateAdmissionResult(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func ValidateAdmissionPacket(value AdmissionPacket) error {
	v := &validator{}
	if value.Schema != AdmissionPacketSchemaV1 {
		v.add("admission_packet.schema.invalid", "schema", "unsupported admission-packet schema")
	}
	validateAdmissionProjectID(v, "admission_packet", value.ProjectID)
	validateDigestField(v, "admission_packet.declaration_digest_invalid", "declaration_digest", value.DeclarationDigest)
	validateDigestField(v, "admission_packet.policy_digest_invalid", "policy_digest", value.PolicyDigest)
	validateAdmissionRange(v, value.BaseRevision, value.HeadRevision)
	validateContentAddressedReference(v, "work_unit_ref", value.WorkUnitRef)
	if value.WorkUnitRef.ArtifactKind != "work_unit" {
		v.add("admission_packet.work_unit_kind_invalid", "work_unit_ref.artifact_kind", "work-unit reference must identify a work_unit artifact")
	}
	if value.EvaluationTime == nil {
		if value.TimeAuthorityRef != "" {
			v.add("admission_packet.time_without_value", "time_authority_ref", "time authority cannot be retained without evaluation time")
		}
	} else {
		if value.EvaluationTime.IsZero() {
			v.add("admission_packet.time_invalid", "evaluation_time", "evaluation time must be non-zero")
		}
		if !IsEvidenceReference(value.TimeAuthorityRef) {
			v.add("admission_packet.time_authority_invalid", "time_authority_ref", "evaluation time requires a bounded trusted authority reference")
		}
	}
	if len(value.Provenance) > 0 && (value.EvaluationTime == nil || value.EvaluationTime.IsZero() || !IsEvidenceReference(value.TimeAuthorityRef)) {
		v.add("admission_packet.provenance_time_required", "evaluation_time", "provider provenance requires a non-zero evaluation time and bounded trusted authority reference")
	}
	validateContentAddressedReferenceSet(v, "evidence", value.Evidence, false)
	if len(value.Evidence) > MaxAdmissionEvidence {
		v.add("admission_packet.evidence.too_many", "evidence", "admission evidence count exceeds the v1 bound")
	}
	validateAdmissionProvenance(v, value.Provenance)
	validateNoSecretPayload(v, value)
	return v.result()
}

func ValidateAdmissionResult(value AdmissionResult) error {
	v := &validator{}
	if value.Schema != AdmissionResultSchemaV1 {
		v.add("admission_result.schema.invalid", "schema", "unsupported admission-result schema")
	}
	validateAdmissionProjectID(v, "admission_result", value.ProjectID)
	validateDigestField(v, "admission_result.policy_digest_invalid", "policy_digest", value.PolicyDigest)
	validateAdmissionRange(v, value.BaseRevision, value.HeadRevision)
	if len(value.MaterialPaths) > MaxAdmissionPaths {
		v.add("admission_result.material_paths.too_many", "material_paths", "material path count exceeds the v1 bound")
	}
	for index, path := range value.MaterialPaths {
		if err := validateRepositoryRelativePath(path); err != nil {
			v.add("admission_result.material_path.invalid", fmt.Sprintf("material_paths[%d]", index), err.Error())
		}
	}
	if duplicateStrings(value.MaterialPaths) {
		v.add("admission_result.material_path.duplicate", "material_paths", "material paths must be unique")
	}
	validateContentAddressedReference(v, "work_unit_ref", value.WorkUnitRef)
	validateAdmissionClassification(v, value.Classification)
	if value.Outcome != AdmissionAllow && value.Outcome != AdmissionDeny {
		v.add("admission_result.outcome.invalid", "outcome", "admission outcome must be allow or deny")
	}
	switch value.Classification {
	case AdmissionValid:
		if value.Outcome != AdmissionAllow {
			v.add("admission_result.valid_must_allow", "outcome", "VALID classification must allow admission")
		}
		if len(value.MissingRefs) != 0 || len(value.ConflictRefs) != 0 {
			v.add("admission_result.valid_evidence_conflict", "classification", "VALID result cannot retain missing or conflicting evidence")
		}
	case AdmissionMissing, AdmissionAmbiguous, AdmissionInvalid:
		if value.Outcome != AdmissionDeny {
			v.add("admission_result.denial_required", "outcome", "MISSING, AMBIGUOUS, and INVALID must deny admission")
		}
	case AdmissionExempted, AdmissionBreakGlass, AdmissionBootstrap:
		if value.Outcome != AdmissionAllow {
			v.add("admission_result.exception_allow_required", "outcome", "authorized exception and bootstrap classifications must allow admission")
		}
	}
	validateAdmissionReasons(v, value.Reasons)
	if value.Classification != AdmissionValid && len(value.Reasons) == 0 {
		v.add("admission_result.reason.required", "reasons", "non-VALID result requires at least one stable reason")
	}
	validateEvidenceReferenceSet(v, "missing_refs", value.MissingRefs, MaxAdmissionEvidence, false)
	validateEvidenceReferenceSet(v, "conflict_refs", value.ConflictRefs, MaxAdmissionEvidence, false)
	if value.Classification == AdmissionMissing && len(value.MissingRefs) == 0 {
		v.add("admission_result.missing_ref.required", "missing_refs", "MISSING result requires a missing evidence reference")
	}
	if value.Classification == AdmissionAmbiguous && len(value.ConflictRefs) < 2 {
		v.add("admission_result.conflict_refs.required", "conflict_refs", "AMBIGUOUS result requires every conflicting claimant")
	}
	validateSetupString(v, "admission_result.verifier_version.invalid", "verifier_version", value.VerifierVersion, true)
	validateNoSecretPayload(v, value)
	return v.result()
}

func normalizeAdmissionPacket(value AdmissionPacket) AdmissionPacket {
	if value.EvaluationTime != nil {
		copy := value.EvaluationTime.UTC()
		value.EvaluationTime = &copy
	}
	value.Evidence = normalizeContentAddressedReferences(value.Evidence)
	value.Provenance = append([]AdmissionProviderProvenance(nil), value.Provenance...)
	for index := range value.Provenance {
		value.Provenance[index].ObservedAt = value.Provenance[index].ObservedAt.UTC()
	}
	sort.Slice(value.Provenance, func(first, second int) bool {
		left, right := value.Provenance[first], value.Provenance[second]
		if left.AdapterID != right.AdapterID {
			return left.AdapterID < right.AdapterID
		}
		if left.ProviderRef != right.ProviderRef {
			return left.ProviderRef < right.ProviderRef
		}
		if left.EvidenceDigest != right.EvidenceDigest {
			return left.EvidenceDigest < right.EvidenceDigest
		}
		if !left.ObservedAt.Equal(right.ObservedAt) {
			return left.ObservedAt.Before(right.ObservedAt)
		}
		return !left.Authenticated && right.Authenticated
	})
	return value
}

func normalizeAdmissionResult(value AdmissionResult) AdmissionResult {
	value.MaterialPaths = append([]string(nil), value.MaterialPaths...)
	for index := range value.MaterialPaths {
		value.MaterialPaths[index] = normalizeRelativePath(value.MaterialPaths[index])
	}
	sort.Strings(value.MaterialPaths)
	value.Reasons = append([]AdmissionReason(nil), value.Reasons...)
	for index := range value.Reasons {
		value.Reasons[index].EvidenceRefs = normalizeStringSet(value.Reasons[index].EvidenceRefs)
	}
	sort.Slice(value.Reasons, func(first, second int) bool {
		return value.Reasons[first].Code < value.Reasons[second].Code
	})
	value.MissingRefs = normalizeStringSet(value.MissingRefs)
	value.ConflictRefs = normalizeStringSet(value.ConflictRefs)
	return value
}

func validateAdmissionProjectID(v *validator, prefix string, value ProjectID) {
	if !IsCanonicalID(string(value)) || !strings.HasPrefix(string(value), "prj_") {
		v.add(prefix+".project_id.invalid", "project_id", "project ID must be canonical")
	}
}

func validateAdmissionRange(v *validator, base, head string) {
	if !gitCommitPattern.MatchString(base) {
		v.add("admission.range.base_invalid", "base_revision", "base revision must be a full lowercase Git commit ID")
	}
	if !gitCommitPattern.MatchString(head) {
		v.add("admission.range.head_invalid", "head_revision", "head revision must be a full lowercase Git commit ID")
	}
	if base != "" && base == head {
		v.add("admission.range.empty", "head_revision", "admission range must contain a distinct head revision")
	}
}

func validateAdmissionProvenance(v *validator, values []AdmissionProviderProvenance) {
	if len(values) > MaxAdmissionEvidence {
		v.add("admission_packet.provenance.too_many", "provenance", "provenance count exceeds the v1 bound")
	}
	for index, value := range values {
		path := fmt.Sprintf("provenance[%d]", index)
		if !IsCanonicalID(value.AdapterID) {
			v.add("admission_packet.provenance.adapter_invalid", path+".adapter_id", "adapter ID must be canonical")
		}
		if !IsEvidenceReference(value.ProviderRef) {
			v.add("admission_packet.provenance.provider_invalid", path+".provider_ref", "provider reference must be bounded and non-secret")
		}
		validateDigestField(v, "admission_packet.provenance.digest_invalid", path+".evidence_digest", value.EvidenceDigest)
		if value.ObservedAt.IsZero() {
			v.add("admission_packet.provenance.time_required", path+".observed_at", "provider observation time is required")
		}
	}
}

func validateAdmissionClassification(v *validator, value AdmissionClassification) {
	switch value {
	case AdmissionValid, AdmissionMissing, AdmissionAmbiguous, AdmissionInvalid,
		AdmissionExempted, AdmissionBreakGlass, AdmissionBootstrap:
	default:
		v.add("admission_result.classification.invalid", "classification", "unsupported admission classification")
	}
}

func validateAdmissionReasons(v *validator, values []AdmissionReason) {
	if len(values) > MaxAdmissionReasons {
		v.add("admission_result.reasons.too_many", "reasons", "reason count exceeds the v1 bound")
	}
	seen := make(map[AdmissionReasonCode]struct{}, len(values))
	for index, value := range values {
		path := fmt.Sprintf("reasons[%d]", index)
		if !IsAdmissionReasonCode(value.Code) {
			v.add("admission_result.reason.unknown", path+".code", "reason code is not in the v1 registry")
		}
		if _, exists := seen[value.Code]; exists {
			v.add("admission_result.reason.duplicate", path+".code", "reason codes must be unique")
		}
		seen[value.Code] = struct{}{}
		validateEvidenceReferenceSet(v, path+".evidence_refs", value.EvidenceRefs, MaxAdmissionEvidence, false)
	}
}
