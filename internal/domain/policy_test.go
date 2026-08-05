package domain

import (
	"bytes"
	"testing"
)

func TestProjectPolicyCanonicalDigestIgnoresSemanticSetOrder(t *testing.T) {
	left := validProjectPolicy()
	right := validProjectPolicy()
	right.Rules[0], right.Rules[1] = right.Rules[1], right.Rules[0]
	right.Rules[0].ChangeKinds[0], right.Rules[0].ChangeKinds[1] = right.Rules[0].ChangeKinds[1], right.Rules[0].ChangeKinds[0]
	right.ExceptionAuthorities[0].ActorRefs[0], right.ExceptionAuthorities[0].ActorRefs[1] = right.ExceptionAuthorities[0].ActorRefs[1], right.ExceptionAuthorities[0].ActorRefs[0]
	right.OwnerDecision.Outcomes[0], right.OwnerDecision.Outcomes[1] = right.OwnerDecision.Outcomes[1], right.OwnerDecision.Outcomes[0]

	leftFrozen, err := FreezeProjectPolicy(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeProjectPolicy(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) || leftFrozen.Digest() != rightFrozen.Digest() {
		t.Fatal("equivalent policies must have byte-identical canonical form")
	}
}

func TestProjectPolicyAllowsSharedPriorityAndRejectsUnknownMatcher(t *testing.T) {
	policy := validProjectPolicy()
	policy.Rules[1].Priority = policy.Rules[0].Priority
	if _, err := FreezeProjectPolicy(policy); err != nil {
		t.Fatalf("shared priority must be resolved only when rules overlap a concrete change: %v", err)
	}
	policy.Rules[0].Matcher.Kind = "glob"
	if _, err := FreezeProjectPolicy(policy); err == nil {
		t.Fatal("unknown matcher was accepted")
	}
}

func validProjectPolicy() ProjectPolicy {
	return ProjectPolicy{
		Schema:    PolicySchemaV1,
		ProjectID: "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Version:   1,
		Rules: []PolicyPathRule{
			{
				ID:                    "go-source",
				Matcher:               PolicyPathMatcher{Kind: PathMatcherPrefix, Path: "internal"},
				ChangeKinds:           []ChangeKind{ChangeModify, ChangeAdd},
				Priority:              200,
				Classification:        MaterialityMaterial,
				RequiredEvidenceKinds: []string{"terminal_receipt", "work_spec"},
			},
			{
				ID:                    "generated-docs",
				Matcher:               PolicyPathMatcher{Kind: PathMatcherPrefix, Path: "docs/generated"},
				ChangeKinds:           []ChangeKind{ChangeDelete, ChangeModify},
				Priority:              100,
				Classification:        MaterialityGenerated,
				RequiredEvidenceKinds: []string{"generator_receipt"},
			},
		},
		ExceptionAuthorities: []PolicyExceptionAuthority{
			{
				ID:                    "owner-break-glass",
				Class:                 ExceptionBreakGlass,
				ActorRefs:             []string{"user:owner-b", "user:owner-a"},
				PathPrefixes:          []string{"internal", "cmd"},
				EffectScopes:          []string{"integration", "material_change"},
				MaxDurationSeconds:    3600,
				OwnerDecisionRequired: true,
			},
		},
		OwnerDecision: PolicyOwnerDecision{
			Required:      true,
			AuthorityRefs: []string{"user:owner-a"},
			Outcomes:      []OwnerDecisionOutcome{OwnerDecisionReject, OwnerDecisionAllow},
		},
	}
}
