package admission

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

const VerifierVersion = "goalrail-admission-v1"

const (
	ExitAllowed = 0
	ExitDenied  = 3
)

// ChangedPath is frozen Git object data. It contains no checkout-local path,
// wall-clock value, provider response, or mutable ref.
type ChangedPath struct {
	Path         string            `json:"path"`
	PreviousPath string            `json:"previous_path,omitempty"`
	Kind         domain.ChangeKind `json:"kind"`
	OldMode      string            `json:"old_mode"`
	NewMode      string            `json:"new_mode"`
	OldObject    string            `json:"old_object"`
	NewObject    string            `json:"new_object"`
	Commits      []string          `json:"commits"`
}

// FrozenRange is the complete deterministic input collected from explicit Git
// commits. Base governance is authoritative; head governance is validated but
// never used to weaken evaluation of its own range.
type FrozenRange struct {
	BaseRevision          string
	HeadRevision          string
	BaseDeclaration       *domain.ProjectDeclaration
	BaseDeclarationDigest domain.SHA256Digest
	BasePolicy            *domain.ProjectPolicy
	BasePolicyDigest      domain.SHA256Digest
	BaseGovernanceInvalid bool
	HeadDeclaration       *domain.ProjectDeclaration
	HeadDeclarationDigest domain.SHA256Digest
	HeadPolicy            *domain.ProjectPolicy
	HeadPolicyDigest      domain.SHA256Digest
	HeadGovernanceInvalid bool
	Changes               []ChangedPath
	Commits               []string
	// CommitParents carries each frozen commit's parents. Order alone cannot
	// answer ancestry: reverse topological order puts an ancestor earlier, but
	// two commits on parallel branches also have an order and no ancestry
	// relation, so a claim anchored on a side branch would pass a position
	// comparison. Reachability needs the edges.
	CommitParents map[string][]string
	Graph         WorkUnitGraph
	// Projections are bounded semantic parses of the exact committed artifacts
	// whose state affects admission. They are produced by the collector at the
	// evaluated revision and never serialized with the range.
	Projections []SemanticProjection `json:"-"`
}

type Input struct {
	Range  FrozenRange
	Packet domain.AdmissionPacket
	// Candidates are same-invocation authenticated provider observations. A
	// packet read from disk always arrives with none, which is what makes a
	// serialized authority claim non-authoritative.
	Candidates []ProviderCandidate `json:"-"`
}

type PathDecision struct {
	Path                  string
	Kind                  domain.ChangeKind
	Classification        domain.MaterialityClassification
	RuleIDs               []domain.PolicyRuleID
	RequiredEvidenceKinds []string
}

type PolicyEvaluation struct {
	Decisions             []PathDecision
	MaterialPaths         []string
	EvidencePaths         []string
	RequiredEvidenceKinds []string
	ConflictRefs          []string
}

type exceptionEnvelope struct {
	Schema       string                `json:"schema"`
	AuthorityID  string                `json:"authority_id"`
	Class        domain.ExceptionClass `json:"class"`
	ActorRef     string                `json:"actor_ref"`
	PathPrefixes []string              `json:"path_prefixes"`
	EffectScopes []string              `json:"effect_scopes"`
	IssuedAt     time.Time             `json:"issued_at"`
	ExpiresAt    time.Time             `json:"expires_at"`
	// The two fields a restoration claim adds: the artifact whose requirement
	// is claimed to be restored, and that artifact's exact digest, so the claim
	// is a reference rather than a name. There is deliberately no anchor field.
	// When the claim was made is read from where its own artifact entered the
	// history, because any field the claim carries about itself is written by
	// the actor making it — the same reason `IssuedAt` is not read here.
	RequirementRef    string              `json:"requirement_ref"`
	RequirementDigest domain.SHA256Digest `json:"requirement_digest"`
}

// Verify evaluates only its explicit input. It does not read environment,
// filesystem, network, credentials, clocks, refs, or provider state.
func Verify(input Input) (domain.AdmissionResult, error) {
	packet := input.Packet
	if err := domain.ValidateAdmissionPacket(packet); err != nil {
		return domain.AdmissionResult{}, fmt.Errorf("validate admission packet: %w", err)
	}
	result := baseResult(packet)
	if packet.BaseRevision != input.Range.BaseRevision || packet.HeadRevision != input.Range.HeadRevision {
		return deny(result, domain.AdmissionInvalid, domain.ReasonRangeMismatch, "packet:range"), nil
	}

	declaration, declarationDigest, policy, policyDigest, bootstrap, authorityReason := resolveAuthority(input.Range)
	if authorityReason != "" {
		return deny(result, domain.AdmissionInvalid, authorityReason, "repo:root/.goalrail/project.json"), nil
	}
	result.ProjectID = declaration.ProjectID
	result.PolicyDigest = policyDigest
	if packet.ProjectID != declaration.ProjectID || packet.DeclarationDigest != declarationDigest || packet.PolicyDigest != policyDigest {
		return deny(result, domain.AdmissionInvalid, domain.ReasonDeclarationInvalid, "packet:authority"), nil
	}

	policyEvaluation := EvaluatePolicy(*policy, input.Range.Changes)
	result.MaterialPaths = policyEvaluation.MaterialPaths
	if len(policyEvaluation.ConflictRefs) > 0 {
		result.Classification = domain.AdmissionAmbiguous
		result.Outcome = domain.AdmissionDeny
		result.Reasons = []domain.AdmissionReason{{Code: domain.ReasonPolicyConflict, EvidenceRefs: policyEvaluation.ConflictRefs}}
		result.ConflictRefs = policyEvaluation.ConflictRefs
		return result, nil
	}

	if reason, refs := verifyGraphBinding(input.Range.Graph, packet, input.Range.Commits, input.Range.Changes, policyEvaluation.MaterialPaths, declarationDigest, policyDigest); reason != "" {
		classification := domain.AdmissionInvalid
		if reason == domain.ReasonLineageConflict {
			classification = domain.AdmissionAmbiguous
		}
		return denyMany(result, classification, reason, refs), nil
	}
	if reason, refs := verifyPacketEvidence(input.Range.Graph, packet); reason != "" {
		classification := domain.AdmissionInvalid
		if reason == domain.ReasonMaterialPathUnbound || reason == domain.ReasonTrustedTimeMissing {
			classification = domain.AdmissionMissing
		} else if reason == domain.ReasonLineageConflict {
			classification = domain.AdmissionAmbiguous
		}
		return denyMany(result, classification, reason, refs), nil
	}

	missingKinds := missingEvidenceKinds(policyEvaluation.RequiredEvidenceKinds, input.Range.Graph, packet)
	if len(missingKinds) > 0 {
		refs := make([]string, 0, len(missingKinds))
		for _, kind := range missingKinds {
			refs = append(refs, "evidence-kind:"+kind)
		}
		return denyMany(result, domain.AdmissionMissing, domain.ReasonGeneratedEvidenceMissing, refs), nil
	}

	// Semantic projections and provider candidates are composed last, with the
	// owner boundary last inside that step. No later evidence repairs an
	// earlier structural, governance, or graph-binding failure.
	view, reason, refs := buildEffectiveView(input, *policy)
	if reason != "" {
		classification := domain.AdmissionInvalid
		if reason == domain.ReasonTrustedTimeMissing {
			classification = domain.AdmissionMissing
		} else if reason == domain.ReasonLineageConflict {
			classification = domain.AdmissionAmbiguous
		}
		return denyMany(result, classification, reason, refs), nil
	}

	evaluatedAt := input.Range.Graph.Unit.CreatedAt.UTC()
	if packet.EvaluationTime != nil {
		evaluatedAt = packet.EvaluationTime.UTC()
	}
	completeness, err := EvaluateCompleteness(input.Range.Graph, evaluatedAt, view)
	if err != nil {
		return deny(result, domain.AdmissionInvalid, domain.ReasonPacketInvalid, "lineage:exception"), nil
	}
	if len(completeness.ExpiredExceptions) > 0 {
		refs := digestsToRefs(completeness.ExpiredExceptions)
		return denyMany(result, domain.AdmissionInvalid, domain.ReasonExceptionExpired, refs), nil
	}
	if len(completeness.Ambiguous) > 0 {
		refs := graphConflictRefs(input.Range.Graph)
		return denyMany(result, domain.AdmissionAmbiguous, domain.ReasonLineageConflict, refs), nil
	}

	exceptionClass, exceptionAllowed, exceptionObserved, restorationDenial := authorizedException(input.Range, *policy, result.MaterialPaths, evaluatedAt)
	if exceptionAllowed {
		if !view.Satisfied(domain.LineageOwnerDecision) {
			return deny(result, domain.AdmissionMissing, domain.ReasonOwnerDecisionMissing, "lineage:owner_decision"), nil
		}
		if !exceptionMayBypass(completeness.Missing) {
			reason, refs := missingLineageReason(completeness.Missing)
			return denyMany(result, domain.AdmissionMissing, reason, refs), nil
		}
		result.Classification = admissionClass(exceptionClass)
		result.Outcome = domain.AdmissionAllow
		result.Reasons = []domain.AdmissionReason{{Code: domain.ReasonExceptionApplied, EvidenceRefs: []string{"lineage:exception"}}}
		return result, nil
	}
	if restorationDenial != "" {
		return deny(result, domain.AdmissionInvalid, restorationDenial, "lineage:exception"), nil
	}
	if exceptionObserved {
		return deny(result, domain.AdmissionInvalid, domain.ReasonExceptionScopeMismatch, "lineage:exception"), nil
	}

	admissionMissing := withoutClosure(completeness.Missing)
	if len(admissionMissing) > 0 || len(completeness.Unavailable) > 0 {
		missing := append([]domain.LineageRelation(nil), admissionMissing...)
		missing = append(missing, completeness.Unavailable...)
		reason, refs := missingLineageReason(uniqueRelations(missing))
		return denyMany(result, domain.AdmissionMissing, reason, refs), nil
	}
	if completeness.Lifecycle != domain.WorkUnitAdmissionReady && completeness.Lifecycle != domain.WorkUnitClosed {
		return deny(result, domain.AdmissionMissing, domain.ReasonMaterialPathUnbound, "lineage:lifecycle"), nil
	}
	if bootstrap {
		result.Classification = domain.AdmissionBootstrap
		result.Outcome = domain.AdmissionAllow
		result.Reasons = []domain.AdmissionReason{{Code: domain.ReasonBootstrapRange, EvidenceRefs: []string{"git:head/" + packet.HeadRevision}}}
		return result, nil
	}
	result.Classification = domain.AdmissionValid
	result.Outcome = domain.AdmissionAllow
	return result, nil
}

// CanonicalResult is the single output/exit contract used by local and shared
// entrypoints. The trailing newline is part of the public CLI representation.
func CanonicalResult(result domain.AdmissionResult) ([]byte, int, error) {
	artifact, err := domain.FreezeAdmissionResult(result)
	if err != nil {
		return nil, 0, err
	}
	encoded := append(artifact.CanonicalJSON(), '\n')
	if result.Outcome == domain.AdmissionAllow {
		return encoded, ExitAllowed, nil
	}
	return encoded, ExitDenied, nil
}

func EvaluatePolicy(policy domain.ProjectPolicy, changes []ChangedPath) PolicyEvaluation {
	result := PolicyEvaluation{}
	required := make(map[string]struct{})
	material := make(map[string]struct{})
	evidence := make(map[string]struct{})
	conflicts := make(map[string]struct{})
	for _, change := range changes {
		paths := []string{change.Path}
		if change.PreviousPath != "" && change.PreviousPath != change.Path {
			paths = append(paths, change.PreviousPath)
		}
		for _, path := range paths {
			decision := evaluatePath(policy, path, change.Kind)
			result.Decisions = append(result.Decisions, decision)
			if len(decision.RuleIDs) > 1 {
				for _, id := range decision.RuleIDs {
					conflicts["policy-rule:"+string(id)] = struct{}{}
				}
				continue
			}
			switch decision.Classification {
			case domain.MaterialityEvidence:
				evidence[path] = struct{}{}
			case domain.MaterialityGenerated:
				for _, kind := range decision.RequiredEvidenceKinds {
					required[kind] = struct{}{}
				}
			case domain.MaterialityNonMaterial:
			default:
				material[path] = struct{}{}
				for _, kind := range decision.RequiredEvidenceKinds {
					required[kind] = struct{}{}
				}
			}
		}
	}
	result.MaterialPaths = sortedKeys(material)
	result.EvidencePaths = sortedKeys(evidence)
	result.RequiredEvidenceKinds = sortedKeys(required)
	result.ConflictRefs = sortedKeys(conflicts)
	sort.Slice(result.Decisions, func(i, j int) bool {
		if result.Decisions[i].Path != result.Decisions[j].Path {
			return result.Decisions[i].Path < result.Decisions[j].Path
		}
		return result.Decisions[i].Kind < result.Decisions[j].Kind
	})
	return result
}

func evaluatePath(policy domain.ProjectPolicy, path string, kind domain.ChangeKind) PathDecision {
	decision := PathDecision{Path: path, Kind: kind, Classification: domain.MaterialityMaterial}
	priority := -1
	for _, rule := range policy.Rules {
		if !ruleMatches(rule, path, kind) {
			continue
		}
		if rule.Priority > priority {
			priority = rule.Priority
			decision.Classification = rule.Classification
			decision.RequiredEvidenceKinds = append([]string(nil), rule.RequiredEvidenceKinds...)
			decision.RuleIDs = []domain.PolicyRuleID{rule.ID}
		} else if rule.Priority == priority {
			decision.RuleIDs = append(decision.RuleIDs, rule.ID)
		}
	}
	sort.Slice(decision.RuleIDs, func(i, j int) bool { return decision.RuleIDs[i] < decision.RuleIDs[j] })
	sort.Strings(decision.RequiredEvidenceKinds)
	return decision
}

func ruleMatches(rule domain.PolicyPathRule, path string, kind domain.ChangeKind) bool {
	kindMatch := false
	for _, allowed := range rule.ChangeKinds {
		if allowed == kind {
			kindMatch = true
			break
		}
	}
	if !kindMatch {
		return false
	}
	switch rule.Matcher.Kind {
	case domain.PathMatcherExact:
		return path == rule.Matcher.Path
	case domain.PathMatcherPrefix:
		if rule.Matcher.Path == "." {
			return true
		}
		return path == rule.Matcher.Path || strings.HasPrefix(path, strings.TrimSuffix(rule.Matcher.Path, "/")+"/")
	default:
		return false
	}
}

func resolveAuthority(frozen FrozenRange) (*domain.ProjectDeclaration, domain.SHA256Digest, *domain.ProjectPolicy, domain.SHA256Digest, bool, domain.AdmissionReasonCode) {
	if frozen.HeadGovernanceInvalid || frozen.HeadDeclaration == nil || frozen.HeadPolicy == nil || !validGovernance(*frozen.HeadDeclaration, frozen.HeadDeclarationDigest, *frozen.HeadPolicy, frozen.HeadPolicyDigest) {
		return nil, "", nil, "", false, domain.ReasonDeclarationInvalid
	}
	if frozen.BaseDeclaration == nil {
		if frozen.BaseGovernanceInvalid {
			return nil, "", nil, "", false, domain.ReasonDeclarationInvalid
		}
		// Only an explicit no-base-declaration range may let complete head
		// governance establish the project identity.
		return frozen.HeadDeclaration, frozen.HeadDeclarationDigest, frozen.HeadPolicy, frozen.HeadPolicyDigest, true, ""
	}
	if frozen.BaseGovernanceInvalid || frozen.BasePolicy == nil || !validGovernance(*frozen.BaseDeclaration, frozen.BaseDeclarationDigest, *frozen.BasePolicy, frozen.BasePolicyDigest) {
		return nil, "", nil, "", false, domain.ReasonDeclarationInvalid
	}
	// A head commit cannot mint a new project identity for its own range.
	if frozen.HeadDeclaration.ProjectID != frozen.BaseDeclaration.ProjectID {
		return nil, "", nil, "", false, domain.ReasonDeclarationInvalid
	}
	return frozen.BaseDeclaration, frozen.BaseDeclarationDigest, frozen.BasePolicy, frozen.BasePolicyDigest, false, ""
}

func validGovernance(declaration domain.ProjectDeclaration, declarationDigest domain.SHA256Digest, policy domain.ProjectPolicy, policyDigest domain.SHA256Digest) bool {
	frozenDeclaration, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil || frozenDeclaration.Digest() != declarationDigest {
		return false
	}
	frozenPolicy, err := domain.FreezeProjectPolicy(policy)
	return err == nil && frozenPolicy.Digest() == policyDigest && declaration.Policy.Digest == policyDigest && policy.ProjectID == declaration.ProjectID
}

func verifyGraphBinding(graph WorkUnitGraph, packet domain.AdmissionPacket, commits []string, changes []ChangedPath, materialPaths []string, declarationDigest, policyDigest domain.SHA256Digest) (domain.AdmissionReasonCode, []string) {
	unitArtifact, err := domain.FreezeWorkUnit(graph.Unit)
	if err != nil || unitArtifact.Digest() != packet.WorkUnitRef.Digest || packet.WorkUnitRef.Identity != "work-unit:"+string(graph.Unit.ID) {
		return domain.ReasonChangeMismatch, []string{"packet:work-unit"}
	}
	if graph.Unit.ProjectID != packet.ProjectID || graph.Unit.DeclarationDigest != declarationDigest || graph.Unit.PolicyDigest != policyDigest {
		return domain.ReasonDeclarationInvalid, []string{"lineage:work-unit"}
	}
	if !hasRequiredSpine(graph.Unit.RequiredRelations) {
		return domain.ReasonPacketInvalid, []string{"lineage:required-relations"}
	}
	if graph.Unit.IntentRef.ArtifactKind != "intent" {
		return domain.ReasonIntentUnconfirmed, []string{"lineage:confirmed_intent"}
	}
	if graph.Unit.ChangeRef.ArtifactKind != "change" {
		return domain.ReasonChangeMismatch, []string{"lineage:change"}
	}
	if !relationTargets(graph, domain.LineageConfirmedIntent, graph.Unit.IntentRef) || !relationTargets(graph, domain.LineageChange, graph.Unit.ChangeRef) {
		return domain.ReasonLineageConflict, []string{string(graph.Unit.IntentRef.Digest), string(graph.Unit.ChangeRef.Digest)}
	}
	for _, event := range graph.Events {
		if !containsEvidenceReference(event.Sources, packet.WorkUnitRef) {
			return domain.ReasonPacketInvalid, []string{"lineage:source-binding"}
		}
	}
	if ok, refs := graphCommitSetMatches(graph, commits, changes, materialPaths); !ok {
		return domain.ReasonChangeMismatch, refs
	}
	if len(graph.Conflicts) > 0 {
		return domain.ReasonLineageConflict, graphConflictRefs(graph)
	}
	return "", nil
}

func hasRequiredSpine(requirements []domain.LineageRequirement) bool {
	want := RequiredSpine()
	if len(requirements) != len(want) {
		return false
	}
	actual := make(map[domain.LineageRelation]domain.RelationCardinality, len(requirements))
	for _, requirement := range requirements {
		actual[requirement.Relation] = requirement.Cardinality
	}
	for _, requirement := range want {
		if actual[requirement.Relation] != requirement.Cardinality {
			return false
		}
	}
	return true
}

func verifyPacketEvidence(graph WorkUnitGraph, packet domain.AdmissionPacket) (domain.AdmissionReasonCode, []string) {
	if len(packet.Provenance) > 0 && !hasBoundedEvaluationTime(packet) {
		return domain.ReasonTrustedTimeMissing, []string{"packet:evaluation-time"}
	}
	if hasRelation(graph, domain.LineageException) && !hasTrustedEvaluationTime(packet) {
		return domain.ReasonTrustedTimeMissing, []string{"packet:evaluation-time"}
	}
	packetRefs := make(map[string]domain.ContentAddressedEvidenceReference, len(packet.Evidence))
	for _, reference := range packet.Evidence {
		if previous, exists := packetRefs[reference.Identity]; exists && previous != reference {
			return domain.ReasonLineageConflict, []string{string(previous.Digest), string(reference.Digest)}
		}
		packetRefs[reference.Identity] = reference
	}
	provenance := make(map[string]domain.AdmissionProviderProvenance)
	for _, item := range packet.Provenance {
		key := item.AdapterID + "\x00" + string(item.EvidenceDigest)
		provenance[key] = item
	}
	for _, event := range graph.Events {
		for _, reference := range event.Targets {
			if trustedWithoutPacket(reference, packet) {
				continue
			}
			claimed, ok := packetRefs[reference.Identity]
			if !ok {
				return domain.ReasonMaterialPathUnbound, []string{reference.Identity}
			}
			if claimed != reference {
				return domain.ReasonPacketInvalid, []string{reference.Identity}
			}
			item, ok := provenance[reference.AdapterID+"\x00"+string(reference.Digest)]
			if !ok || !item.Authenticated || item.ObservedAt.After(packet.EvaluationTime.UTC()) {
				return domain.ReasonProvenanceUntrusted, []string{reference.Identity}
			}
		}
	}
	return "", nil
}

func hasTrustedEvaluationTime(packet domain.AdmissionPacket) bool {
	if !hasBoundedEvaluationTime(packet) {
		return false
	}
	for _, item := range packet.Provenance {
		if item.Authenticated && item.ProviderRef == packet.TimeAuthorityRef && !item.ObservedAt.After(packet.EvaluationTime.UTC()) {
			return true
		}
	}
	return false
}

func hasBoundedEvaluationTime(packet domain.AdmissionPacket) bool {
	return packet.EvaluationTime != nil &&
		!packet.EvaluationTime.IsZero() &&
		domain.IsEvidenceReference(packet.TimeAuthorityRef)
}

func trustedWithoutPacket(reference domain.ContentAddressedEvidenceReference, packet domain.AdmissionPacket) bool {
	if strings.HasPrefix(reference.SourceRef, "repo:root/") || reference.ArtifactKind == "commit" {
		return true
	}
	if strings.HasPrefix(reference.SourceRef, "git:") {
		return (reference.ArtifactKind == "project_declaration" && reference.Digest == packet.DeclarationDigest) ||
			(reference.ArtifactKind == "project_policy" && reference.Digest == packet.PolicyDigest)
	}
	return false
}

func missingEvidenceKinds(required []string, graph WorkUnitGraph, packet domain.AdmissionPacket) []string {
	present := make(map[string]struct{})
	for _, reference := range packet.Evidence {
		present[reference.ArtifactKind] = struct{}{}
	}
	for _, event := range graph.Events {
		for _, reference := range event.Targets {
			present[reference.ArtifactKind] = struct{}{}
		}
	}
	var missing []string
	for _, kind := range required {
		if _, ok := present[kind]; !ok {
			missing = append(missing, kind)
		}
	}
	return missing
}

func authorizedException(frozen FrozenRange, policy domain.ProjectPolicy, materialPaths []string, evaluatedAt time.Time) (domain.ExceptionClass, bool, bool, domain.AdmissionReasonCode) {
	graph := frozen.Graph
	observed := false
	var restorationDenial domain.AdmissionReasonCode
	for _, event := range graph.Events {
		if event.Relation != domain.LineageException {
			continue
		}
		for _, target := range event.Targets {
			raw := graph.Replicas[target.Digest]
			if len(raw) == 0 {
				continue
			}
			var envelope exceptionEnvelope
			if json.Unmarshal(raw, &envelope) != nil || envelope.Schema != LineageExceptionSchemaV1 || envelope.ExpiresAt.IsZero() || !evaluatedAt.Before(envelope.ExpiresAt.UTC()) {
				continue
			}
			observed = true
			for _, authority := range policy.ExceptionAuthorities {
				if authority.ID != envelope.AuthorityID || authority.Class != envelope.Class || !contains(authority.ActorRefs, envelope.ActorRef) || !contains(authority.ActorRefs, event.ActorRef) {
					continue
				}
				if envelope.IssuedAt.IsZero() || envelope.ExpiresAt.Before(envelope.IssuedAt) || envelope.ExpiresAt.Sub(envelope.IssuedAt) > time.Duration(authority.MaxDurationSeconds)*time.Second {
					continue
				}
				if !subset(envelope.EffectScopes, authority.EffectScopes) || !pathsCovered(materialPaths, authority.PathPrefixes) || !pathsCovered(materialPaths, envelope.PathPrefixes) {
					continue
				}
				if envelope.Class == domain.ExceptionRestoration {
					// A restoration claim that matched its authority is still
					// refused on its own two conditions, and refused by name:
					// a claim made too late and a claim bound to the wrong
					// artifact are different defects with different remedies,
					// and falling through to a generic scope mismatch would
					// tell the author neither.
					if failure := restorationFailure(frozen, policy, event, envelope, materialPaths); failure != "" {
						if restorationDenial == "" {
							restorationDenial = failure
						}
						continue
					}
				}
				return envelope.Class, true, true, ""
			}
		}
	}
	return "", false, observed, restorationDenial
}

func exceptionMayBypass(missing []domain.LineageRelation) bool {
	for _, relation := range missing {
		switch relation {
		case domain.LineageRunSession, domain.LineagePullRequest, domain.LineageReviewIndex,
			domain.LineageCheckSet, domain.LineageTerminalReceipt, domain.LineageClosure:
		default:
			return false
		}
	}
	return true
}

func admissionClass(class domain.ExceptionClass) domain.AdmissionClassification {
	switch class {
	case domain.ExceptionExempted:
		return domain.AdmissionExempted
	case domain.ExceptionBreakGlass:
		return domain.AdmissionBreakGlass
	case domain.ExceptionRestoration:
		// Visibly non-VALID, like every other exception: the direct route is an
		// exception with a recorded claim, not a second ordinary lane.
		return domain.AdmissionExempted
	default:
		return domain.AdmissionBootstrap
	}
}

func baseResult(packet domain.AdmissionPacket) domain.AdmissionResult {
	return domain.AdmissionResult{
		Schema: domain.AdmissionResultSchemaV1, ProjectID: packet.ProjectID, PolicyDigest: packet.PolicyDigest,
		BaseRevision: packet.BaseRevision, HeadRevision: packet.HeadRevision,
		MaterialPaths: []string{}, WorkUnitRef: packet.WorkUnitRef,
		Classification: domain.AdmissionInvalid, Outcome: domain.AdmissionDeny,
		Reasons: []domain.AdmissionReason{}, MissingRefs: []string{}, ConflictRefs: []string{},
		VerifierVersion: VerifierVersion,
	}
}

func deny(result domain.AdmissionResult, classification domain.AdmissionClassification, reason domain.AdmissionReasonCode, reference string) domain.AdmissionResult {
	return denyMany(result, classification, reason, []string{reference})
}

func denyMany(result domain.AdmissionResult, classification domain.AdmissionClassification, reason domain.AdmissionReasonCode, references []string) domain.AdmissionResult {
	refs := uniqueStrings(references)
	sort.Strings(refs)
	result.Classification = classification
	result.Outcome = domain.AdmissionDeny
	result.Reasons = []domain.AdmissionReason{{Code: reason, EvidenceRefs: refs}}
	if classification == domain.AdmissionMissing {
		result.MissingRefs = refs
	}
	if classification == domain.AdmissionAmbiguous {
		result.ConflictRefs = refs
	}
	return result
}

func uniqueStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	return result
}

func relationTargets(graph WorkUnitGraph, relation domain.LineageRelation, reference domain.ContentAddressedEvidenceReference) bool {
	for _, event := range graph.Events {
		if event.Relation != relation {
			continue
		}
		for _, target := range event.Targets {
			if target == reference {
				return true
			}
		}
	}
	return false
}

func containsEvidenceReference(values []domain.ContentAddressedEvidenceReference, want domain.ContentAddressedEvidenceReference) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func graphCommitSetMatches(graph WorkUnitGraph, revisions []string, changes []ChangedPath, materialPaths []string) (bool, []string) {
	allowed := make(map[string]struct{}, len(revisions))
	for _, revision := range revisions {
		allowed[revision] = struct{}{}
	}
	seen := false
	linked := make(map[string]struct{})
	var invalid []string
	for _, event := range graph.Events {
		if event.Relation != domain.LineageCommit {
			continue
		}
		for _, target := range event.Targets {
			if target.ArtifactKind != "commit" {
				invalid = append(invalid, target.Identity)
				continue
			}
			revision := strings.TrimPrefix(target.Identity, "git-commit:")
			if revision == target.Identity {
				revision = strings.TrimPrefix(target.SourceRef, "git:commit/")
			}
			if revision == target.SourceRef {
				revision = strings.TrimPrefix(target.SourceRef, "git:")
			}
			if _, ok := allowed[revision]; !ok {
				invalid = append(invalid, "git:commit/"+revision)
				continue
			}
			seen = true
			linked[revision] = struct{}{}
		}
	}
	material := make(map[string]struct{}, len(materialPaths))
	for _, path := range materialPaths {
		material[path] = struct{}{}
	}
	for _, change := range changes {
		_, pathMaterial := material[change.Path]
		_, previousMaterial := material[change.PreviousPath]
		if !pathMaterial && !previousMaterial {
			continue
		}
		for _, revision := range change.Commits {
			if _, ok := linked[revision]; !ok {
				invalid = append(invalid, "git:commit/"+revision)
			}
		}
	}
	if !seen && len(invalid) == 0 {
		invalid = append(invalid, "lineage:commit")
	}
	sort.Strings(invalid)
	return seen && len(invalid) == 0, invalid
}

func hasRelation(graph WorkUnitGraph, relation domain.LineageRelation) bool {
	for _, event := range graph.Events {
		if event.Relation == relation && len(event.Targets) > 0 {
			return true
		}
	}
	return false
}

func missingLineageReason(relations []domain.LineageRelation) (domain.AdmissionReasonCode, []string) {
	sort.Slice(relations, func(i, j int) bool { return relations[i] < relations[j] })
	reason := domain.ReasonMaterialPathUnbound
	for _, preferred := range []domain.LineageRelation{
		domain.LineageProjectPolicy, domain.LineageConfirmedIntent, domain.LineageChange,
		domain.LineageWorkSpec, domain.LineageRunSession, domain.LineageCommit,
		domain.LineagePullRequest, domain.LineageReviewIndex, domain.LineageCheckSet,
		domain.LineageTerminalReceipt, domain.LineageOwnerDecision,
	} {
		if !containsRelation(relations, preferred) {
			continue
		}
		switch preferred {
		case domain.LineageConfirmedIntent:
			reason = domain.ReasonIntentUnconfirmed
		case domain.LineageChange, domain.LineageCommit:
			reason = domain.ReasonChangeMismatch
		case domain.LineageWorkSpec:
			reason = domain.ReasonWorkSpecMissing
		case domain.LineageRunSession:
			reason = domain.ReasonRunSessionMissing
		case domain.LineagePullRequest:
			reason = domain.ReasonPullRequestMissing
		case domain.LineageReviewIndex:
			reason = domain.ReasonReviewMissing
		case domain.LineageCheckSet:
			reason = domain.ReasonCheckMissing
		case domain.LineageTerminalReceipt:
			reason = domain.ReasonReceiptMissing
		case domain.LineageOwnerDecision:
			reason = domain.ReasonOwnerDecisionMissing
		case domain.LineageProjectPolicy:
			reason = domain.ReasonDeclarationInvalid
		}
		break
	}
	refs := make([]string, len(relations))
	for index, relation := range relations {
		refs[index] = "lineage:" + string(relation)
	}
	return reason, refs
}

func withoutClosure(values []domain.LineageRelation) []domain.LineageRelation {
	result := make([]domain.LineageRelation, 0, len(values))
	for _, value := range values {
		if value != domain.LineageClosure {
			result = append(result, value)
		}
	}
	return result
}

func containsRelation(values []domain.LineageRelation, want domain.LineageRelation) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func graphConflictRefs(graph WorkUnitGraph) []string {
	var refs []string
	for _, conflict := range graph.Conflicts {
		refs = append(refs, digestsToRefs(conflict.Digests)...)
	}
	sort.Strings(refs)
	return refs
}

func digestsToRefs(digests []domain.SHA256Digest) []string {
	refs := make([]string, len(digests))
	for index, digest := range digests {
		refs[index] = string(digest)
	}
	return refs
}

func uniqueRelations(values []domain.LineageRelation) []domain.LineageRelation {
	set := make(map[domain.LineageRelation]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	result := make([]domain.LineageRelation, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	return result
}

// restorationFailure reports why a restoration claim cannot stand, or the empty
// code when it stands. It decides three questions and deliberately not a
// fourth: whether the claim binds an artifact the lineage already carries,
// whether the work it excuses amends a normative artifact in its own scope, and
// whether the claim was recorded before that work. Whether the diff restores
// the bound requirement or moves its boundary is semantic, is reserved to
// review, and an empty return here must not be read as having settled it.
func restorationFailure(frozen FrozenRange, policy domain.ProjectPolicy, event domain.LineageEvent, envelope exceptionEnvelope, materialPaths []string) domain.AdmissionReasonCode {
	// The bound artifact must be one the work unit's own lineage records, with
	// its reference and digest recorded together. Checking that a replica
	// merely exists under the claimed digest proved nothing: any unrelated
	// retained artifact satisfied it while the reference went uncompared.
	if !boundToRecordedArtifact(frozen.Graph, envelope) {
		return domain.ReasonRestorationUnbound
	}
	// A policy that authorizes restoration without declaring which paths are
	// normative cannot answer the amendment question, and a question that
	// cannot be answered is refused rather than passed. Treating an empty
	// declaration as "nothing is normative" turned the prohibition off exactly
	// where the policy had said least.
	if len(policy.NormativePathPrefixes) == 0 {
		return domain.ReasonRestorationUnbound
	}
	// A claim cannot stand where the work amends a normative artifact inside
	// its own scope: there is no unchanged prior requirement left to restore.
	// The scope is any normative path the policy declares, not only the one the
	// claim binds — a claim that restores requirement A while amending
	// requirement B beside it is the case this exists to refuse, and comparing
	// against the bound path alone let it through.
	if amendsNormativePath(frozen, policy, envelope.PathPrefixes) {
		return domain.ReasonRestorationUnbound
	}
	touches := materialTouches(frozen, materialPaths, envelope.PathPrefixes)
	if len(touches) == 0 {
		// Nothing material in scope, so there is no work for the claim to
		// precede. Scope coverage is decided by the caller.
		return ""
	}
	// The anchor is derived from where the claim actually entered the history,
	// never from what the claim says about itself. An `anchor_commit` field
	// would be written by the same actor making the claim, which is precisely
	// the self-report the claim exists to replace: the evidence is read at
	// head, so a claim added afterwards could simply name an earlier commit.
	anchor, inRange := claimAnchor(frozen, event)
	if !inRange {
		// The claim's artifact is not among the changed paths, so it was
		// already committed before this range began and precedes every commit
		// in it. That is the strongest form of the claim, not a missing one.
		return ""
	}
	// Every minimal touch, not the earliest one in the list. Reverse
	// topological order is a partial order: with two parallel branches touching
	// in-scope paths, an anchor ancestral to one of them is not thereby
	// ancestral to the other.
	for _, touch := range touches {
		if anchor == touch || !ancestorWithinRange(frozen, anchor, touch) {
			return domain.ReasonRestorationNotAnchored
		}
	}
	return ""
}

// boundToRecordedArtifact reports whether the claim names a reference and
// digest that the work unit's lineage records together on a target that states
// requirements.
//
// The relation is checked, not merely the pairing: every target in the graph is
// recorded with a reference and a digest, so accepting any of them let a claim
// bind the work unit itself, a commit, or a receipt and call that the
// requirement it restores. Only confirmed intent states a requirement here.
func boundToRecordedArtifact(graph WorkUnitGraph, envelope exceptionEnvelope) bool {
	if envelope.RequirementDigest == "" || envelope.RequirementRef == "" {
		return false
	}
	for _, event := range graph.Events {
		if event.Relation != domain.LineageConfirmedIntent {
			continue
		}
		for _, target := range event.Targets {
			if target.SourceRef == envelope.RequirementRef && target.Digest == envelope.RequirementDigest {
				return true
			}
		}
	}
	return false
}

// amendsNormativePath reports whether any changed path inside the claimed scope
// is one the policy declares normative. The policy owns which paths those are,
// as it owns every other path question; this adds no second classifier.
func amendsNormativePath(frozen FrozenRange, policy domain.ProjectPolicy, prefixes []string) bool {
	for _, change := range frozen.Changes {
		for _, path := range []string{change.Path, change.PreviousPath} {
			if path == "" || !pathsCovered([]string{path}, prefixes) {
				continue
			}
			if pathsCovered([]string{path}, policy.NormativePathPrefixes) {
				return true
			}
		}
	}
	return false
}

// claimAnchor reports the earliest frozen commit that touched the claim's own
// artifact, and whether the artifact was touched inside the range at all.
func claimAnchor(frozen FrozenRange, event domain.LineageEvent) (string, bool) {
	paths := make(map[string]struct{}, len(event.Targets))
	for _, target := range event.Targets {
		if path := repositoryPath(target.SourceRef); path != "" {
			paths[path] = struct{}{}
		}
	}
	if len(paths) == 0 {
		// The claim's artifact is not repository-addressable, so nothing
		// establishes when it entered the history. Treating that as "before the
		// work" would accept exactly the unprovable claim.
		return "", true
	}
	touching := commitsTouching(frozen, paths)
	for _, revision := range frozen.Commits {
		if _, ok := touching[revision]; ok {
			return revision, true
		}
	}
	return "", false
}

// materialTouches returns every frozen commit touching a material path inside
// the claimed scope, in frozen order.
func materialTouches(frozen FrozenRange, materialPaths, prefixes []string) []string {
	inScope := make(map[string]struct{}, len(materialPaths))
	for _, path := range materialPaths {
		if pathsCovered([]string{path}, prefixes) {
			inScope[path] = struct{}{}
		}
	}
	if len(inScope) == 0 {
		return nil
	}
	touching := commitsTouching(frozen, inScope)
	ordered := make([]string, 0, len(touching))
	for _, revision := range frozen.Commits {
		if _, ok := touching[revision]; ok {
			ordered = append(ordered, revision)
		}
	}
	return ordered
}

func commitsTouching(frozen FrozenRange, paths map[string]struct{}) map[string]struct{} {
	touching := make(map[string]struct{}, len(frozen.Changes))
	for _, change := range frozen.Changes {
		_, current := paths[change.Path]
		_, previous := paths[change.PreviousPath]
		if !current && !previous {
			continue
		}
		for _, revision := range change.Commits {
			touching[revision] = struct{}{}
		}
	}
	return touching
}

// repositoryPath extracts the repository-relative path from a reference, or
// empty when the reference does not address one.
func repositoryPath(reference string) string {
	path := strings.TrimPrefix(reference, "repo:root/")
	if path == reference {
		return ""
	}
	return path
}

// ancestorWithinRange reports whether anchor is reachable from descendant by
// walking parents inside the frozen range. A walk that leaves the range ends
// there: the verifier was never given those commits, so it cannot establish
// ancestry for them, and an unprovable claim is not accepted on trust.
func ancestorWithinRange(frozen FrozenRange, anchor, descendant string) bool {
	known := make(map[string]struct{}, len(frozen.Commits))
	for _, revision := range frozen.Commits {
		known[revision] = struct{}{}
	}
	if _, ok := known[anchor]; !ok {
		return false
	}
	seen := map[string]struct{}{descendant: {}}
	queue := []string{descendant}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, parent := range frozen.CommitParents[current] {
			if parent == anchor {
				return true
			}
			if _, ok := known[parent]; !ok {
				continue
			}
			if _, visited := seen[parent]; visited {
				continue
			}
			seen[parent] = struct{}{}
			queue = append(queue, parent)
		}
	}
	return false
}

func pathsCovered(paths, prefixes []string) bool {
	for _, path := range paths {
		covered := false
		for _, prefix := range prefixes {
			if path == prefix || strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/") {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func subset(values, allowed []string) bool {
	for _, value := range values {
		if !contains(allowed, value) {
			return false
		}
	}
	return true
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sortedKeys[T ~string](values map[T]struct{}) []T {
	result := make([]T, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
