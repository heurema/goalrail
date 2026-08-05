package domain

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type (
	PolicyRuleID              string
	PathMatcherKind           string
	ChangeKind                string
	MaterialityClassification string
	ExceptionClass            string
	OwnerDecisionOutcome      string
)

const (
	PolicySchemaV1   = "goalrail.policy/v1"
	MaxPolicyBytes   = 128 << 10
	MaxPolicyRules   = 512
	MaxPolicyActors  = 64
	MaxPolicyPaths   = 256
	MaxPolicyEffects = 64

	PathMatcherExact  PathMatcherKind = "exact"
	PathMatcherPrefix PathMatcherKind = "prefix"

	ChangeAdd       ChangeKind = "add"
	ChangeModify    ChangeKind = "modify"
	ChangeDelete    ChangeKind = "delete"
	ChangeRename    ChangeKind = "rename"
	ChangeMode      ChangeKind = "mode"
	ChangeSubmodule ChangeKind = "submodule"

	MaterialityMaterial    MaterialityClassification = "material"
	MaterialityEvidence    MaterialityClassification = "evidence"
	MaterialityGenerated   MaterialityClassification = "generated"
	MaterialityNonMaterial MaterialityClassification = "non_material"

	ExceptionExempted   ExceptionClass = "exempted"
	ExceptionBreakGlass ExceptionClass = "break_glass"
	ExceptionBootstrap  ExceptionClass = "bootstrap"

	OwnerDecisionAllow  OwnerDecisionOutcome = "allow"
	OwnerDecisionReject OwnerDecisionOutcome = "reject"
)

type PolicyPathMatcher struct {
	Kind PathMatcherKind `json:"kind"`
	Path string          `json:"path"`
}

type PolicyPathRule struct {
	ID                    PolicyRuleID              `json:"id"`
	Matcher               PolicyPathMatcher         `json:"matcher"`
	ChangeKinds           []ChangeKind              `json:"change_kinds"`
	Priority              int                       `json:"priority"`
	Classification        MaterialityClassification `json:"classification"`
	RequiredEvidenceKinds []string                  `json:"required_evidence_kinds"`
}

type PolicyExceptionAuthority struct {
	ID                    string         `json:"id"`
	Class                 ExceptionClass `json:"class"`
	ActorRefs             []string       `json:"actor_refs"`
	PathPrefixes          []string       `json:"path_prefixes"`
	EffectScopes          []string       `json:"effect_scopes"`
	MaxDurationSeconds    uint64         `json:"max_duration_seconds"`
	OwnerDecisionRequired bool           `json:"owner_decision_required"`
}

type PolicyOwnerDecision struct {
	Required      bool                   `json:"required"`
	AuthorityRefs []string               `json:"authority_refs"`
	Outcomes      []OwnerDecisionOutcome `json:"outcomes"`
}

type ProjectPolicy struct {
	Schema               string                     `json:"schema"`
	ProjectID            ProjectID                  `json:"project_id"`
	Version              uint32                     `json:"version"`
	Rules                []PolicyPathRule           `json:"rules"`
	ExceptionAuthorities []PolicyExceptionAuthority `json:"exception_authorities"`
	OwnerDecision        PolicyOwnerDecision        `json:"owner_decision"`
}

func DecodeProjectPolicy(reader io.Reader) (ProjectPolicy, error) {
	policy, err := decodeStrictBoundedJSON[ProjectPolicy](reader, MaxPolicyBytes, "project policy")
	if err != nil {
		return ProjectPolicy{}, err
	}
	policy = normalizeProjectPolicy(policy)
	if err := ValidateProjectPolicy(policy); err != nil {
		return ProjectPolicy{}, err
	}
	return policy, nil
}

func FreezeProjectPolicy(policy ProjectPolicy) (CanonicalArtifact, error) {
	policy = normalizeProjectPolicy(policy)
	if err := ValidateProjectPolicy(policy); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(policy)
}

func ValidateProjectPolicy(policy ProjectPolicy) error {
	v := &validator{}
	if policy.Schema != PolicySchemaV1 {
		v.add("policy.schema.invalid", "schema", "unsupported project policy schema")
	}
	if !IsCanonicalID(string(policy.ProjectID)) || !strings.HasPrefix(string(policy.ProjectID), "prj_") {
		v.add("policy.project_id.invalid", "project_id", "policy project ID must be canonical")
	}
	if policy.Version != 1 {
		v.add("policy.version.invalid", "version", "policy version must be 1")
	}
	if len(policy.Rules) == 0 {
		v.add("policy.rules.required", "rules", "at least one policy rule is required")
	}
	if len(policy.Rules) > MaxPolicyRules {
		v.add("policy.rules.too_many", "rules", "policy rule count exceeds the v1 bound")
	}
	ruleIDs := make(map[PolicyRuleID]struct{}, len(policy.Rules))
	for index, rule := range policy.Rules {
		path := fmt.Sprintf("rules[%d]", index)
		if !IsCanonicalID(string(rule.ID)) {
			v.add("policy.rule.id_invalid", path+".id", "rule ID must be canonical")
		} else if _, exists := ruleIDs[rule.ID]; exists {
			v.add("policy.rule.id_duplicate", path+".id", "rule ID must be unique")
		} else {
			ruleIDs[rule.ID] = struct{}{}
		}
		if rule.Priority <= 0 {
			v.add("policy.rule.priority_invalid", path+".priority", "priority must be explicitly positive")
		}
		if rule.Matcher.Kind != PathMatcherExact && rule.Matcher.Kind != PathMatcherPrefix {
			v.add("policy.rule.matcher_kind_invalid", path+".matcher.kind", "matcher kind must be exact or prefix")
		}
		if err := validateRepositoryRelativePath(rule.Matcher.Path); err != nil {
			v.add("policy.rule.matcher_path_invalid", path+".matcher.path", err.Error())
		}
		validateChangeKinds(v, path+".change_kinds", rule.ChangeKinds)
		switch rule.Classification {
		case MaterialityMaterial, MaterialityEvidence, MaterialityGenerated, MaterialityNonMaterial:
		default:
			v.add("policy.rule.classification_invalid", path+".classification", "unsupported materiality classification")
		}
		validateCanonicalStringSet(v, path+".required_evidence_kinds", rule.RequiredEvidenceKinds, MaxPolicyEffects)
	}

	if len(policy.ExceptionAuthorities) > MaxPolicyEffects {
		v.add("policy.exceptions.too_many", "exception_authorities", "exception authority count exceeds the v1 bound")
	}
	exceptionIDs := make(map[string]struct{}, len(policy.ExceptionAuthorities))
	for index, authority := range policy.ExceptionAuthorities {
		path := fmt.Sprintf("exception_authorities[%d]", index)
		if !IsCanonicalID(authority.ID) {
			v.add("policy.exception.id_invalid", path+".id", "exception authority ID must be canonical")
		} else if _, exists := exceptionIDs[authority.ID]; exists {
			v.add("policy.exception.id_duplicate", path+".id", "exception authority ID must be unique")
		} else {
			exceptionIDs[authority.ID] = struct{}{}
		}
		switch authority.Class {
		case ExceptionExempted, ExceptionBreakGlass, ExceptionBootstrap:
		default:
			v.add("policy.exception.class_invalid", path+".class", "unsupported exception class")
		}
		validateEvidenceReferenceSet(v, path+".actor_refs", authority.ActorRefs, MaxPolicyActors, true)
		validateRepositoryPathSet(v, path+".path_prefixes", authority.PathPrefixes, MaxPolicyPaths, true)
		validateCanonicalStringSet(v, path+".effect_scopes", authority.EffectScopes, MaxPolicyEffects)
		if !authority.OwnerDecisionRequired {
			v.add("policy.exception.owner_decision_required", path+".owner_decision_required", "v1 exceptions require an owner decision")
		}
	}

	if !policy.OwnerDecision.Required {
		v.add("policy.owner_decision.required", "owner_decision.required", "v1 admission requires an owner decision")
	}
	validateEvidenceReferenceSet(v, "owner_decision.authority_refs", policy.OwnerDecision.AuthorityRefs, MaxPolicyActors, true)
	if len(policy.OwnerDecision.Outcomes) == 0 {
		v.add("policy.owner_decision.outcomes_required", "owner_decision.outcomes", "owner decision outcomes are required")
	}
	for index, outcome := range policy.OwnerDecision.Outcomes {
		if outcome != OwnerDecisionAllow && outcome != OwnerDecisionReject {
			v.add("policy.owner_decision.outcome_invalid", fmt.Sprintf("owner_decision.outcomes[%d]", index), "unsupported owner decision outcome")
		}
	}
	if duplicateOwnerDecisionOutcomes(policy.OwnerDecision.Outcomes) {
		v.add("policy.owner_decision.outcome_duplicate", "owner_decision.outcomes", "owner decision outcomes must be unique")
	}
	return v.result()
}

func normalizeProjectPolicy(policy ProjectPolicy) ProjectPolicy {
	policy.Rules = append([]PolicyPathRule(nil), policy.Rules...)
	for index := range policy.Rules {
		policy.Rules[index].Matcher.Path = normalizeRelativePath(policy.Rules[index].Matcher.Path)
		policy.Rules[index].ChangeKinds = append([]ChangeKind(nil), policy.Rules[index].ChangeKinds...)
		sort.Slice(policy.Rules[index].ChangeKinds, func(first, second int) bool {
			return policy.Rules[index].ChangeKinds[first] < policy.Rules[index].ChangeKinds[second]
		})
		policy.Rules[index].RequiredEvidenceKinds = normalizeStringSet(policy.Rules[index].RequiredEvidenceKinds)
	}
	sort.Slice(policy.Rules, func(first, second int) bool {
		if policy.Rules[first].Priority != policy.Rules[second].Priority {
			return policy.Rules[first].Priority > policy.Rules[second].Priority
		}
		return policy.Rules[first].ID < policy.Rules[second].ID
	})

	policy.ExceptionAuthorities = append([]PolicyExceptionAuthority(nil), policy.ExceptionAuthorities...)
	for index := range policy.ExceptionAuthorities {
		policy.ExceptionAuthorities[index].ActorRefs = normalizeStringSet(policy.ExceptionAuthorities[index].ActorRefs)
		policy.ExceptionAuthorities[index].PathPrefixes = append([]string(nil), policy.ExceptionAuthorities[index].PathPrefixes...)
		for pathIndex := range policy.ExceptionAuthorities[index].PathPrefixes {
			policy.ExceptionAuthorities[index].PathPrefixes[pathIndex] = normalizeRelativePath(policy.ExceptionAuthorities[index].PathPrefixes[pathIndex])
		}
		sort.Strings(policy.ExceptionAuthorities[index].PathPrefixes)
		policy.ExceptionAuthorities[index].EffectScopes = normalizeStringSet(policy.ExceptionAuthorities[index].EffectScopes)
	}
	sort.Slice(policy.ExceptionAuthorities, func(first, second int) bool {
		return policy.ExceptionAuthorities[first].ID < policy.ExceptionAuthorities[second].ID
	})
	policy.OwnerDecision.AuthorityRefs = normalizeStringSet(policy.OwnerDecision.AuthorityRefs)
	policy.OwnerDecision.Outcomes = append([]OwnerDecisionOutcome(nil), policy.OwnerDecision.Outcomes...)
	sort.Slice(policy.OwnerDecision.Outcomes, func(first, second int) bool {
		return policy.OwnerDecision.Outcomes[first] < policy.OwnerDecision.Outcomes[second]
	})
	return policy
}

func validateChangeKinds(v *validator, path string, kinds []ChangeKind) {
	if len(kinds) == 0 {
		v.add("policy.rule.change_kinds_required", path, "at least one change kind is required")
	}
	seen := make(map[ChangeKind]struct{}, len(kinds))
	for index, kind := range kinds {
		switch kind {
		case ChangeAdd, ChangeModify, ChangeDelete, ChangeRename, ChangeMode, ChangeSubmodule:
		default:
			v.add("policy.rule.change_kind_invalid", fmt.Sprintf("%s[%d]", path, index), "unsupported change kind")
		}
		if _, exists := seen[kind]; exists {
			v.add("policy.rule.change_kind_duplicate", path, "change kinds must be unique")
		}
		seen[kind] = struct{}{}
	}
}

func validateCanonicalStringSet(v *validator, path string, values []string, limit int) {
	if len(values) > limit {
		v.add("policy.set.too_many", path, "value count exceeds the v1 bound")
	}
	for index, value := range values {
		if !IsCanonicalID(value) {
			v.add("policy.set.value_invalid", fmt.Sprintf("%s[%d]", path, index), "value must be a canonical identifier")
		}
	}
	if duplicateStrings(values) {
		v.add("policy.set.value_duplicate", path, "values must be unique")
	}
}

func validateEvidenceReferenceSet(v *validator, path string, values []string, limit int, required bool) {
	if required && len(values) == 0 {
		v.add("policy.references.required", path, "at least one evidence reference is required")
	}
	if len(values) > limit {
		v.add("policy.references.too_many", path, "reference count exceeds the v1 bound")
	}
	for index, value := range values {
		if !IsEvidenceReference(value) {
			v.add("policy.reference.invalid", fmt.Sprintf("%s[%d]", path, index), "reference must be bounded, provider-neutral, and non-secret")
		}
	}
	if duplicateStrings(values) {
		v.add("policy.reference.duplicate", path, "references must be unique")
	}
}

func validateRepositoryPathSet(v *validator, path string, values []string, limit int, required bool) {
	if required && len(values) == 0 {
		v.add("policy.paths.required", path, "at least one repository path is required")
	}
	if len(values) > limit {
		v.add("policy.paths.too_many", path, "path count exceeds the v1 bound")
	}
	for index, value := range values {
		if err := validateRepositoryRelativePath(value); err != nil {
			v.add("policy.path.invalid", fmt.Sprintf("%s[%d]", path, index), err.Error())
		}
	}
	if duplicateStrings(values) {
		v.add("policy.path.duplicate", path, "paths must be unique")
	}
}

func duplicateOwnerDecisionOutcomes(values []OwnerDecisionOutcome) bool {
	seen := make(map[OwnerDecisionOutcome]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
