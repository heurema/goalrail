package admission

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

// The three commits of the recorded failure. `#85` put the implementation
// first, the snapshot second and its confirmation third; after that backfill a
// verifier that reads only relations sees a confirmed snapshot bound to the
// work unit and no violation. What distinguishes the honest sequence from the
// dishonest one is which commit the claim sits in, so the fixture is a history
// rather than a set of timestamps.
const (
	commitBeforeWork = "cccccccccccccccccccccccccccccccccccccccc"
	commitOfWork     = "dddddddddddddddddddddddddddddddddddddddd"
	commitAfterWork  = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

func TestARestorationClaimMustPrecedeTheWorkItExcuses(t *testing.T) {
	for _, test := range []struct {
		name            string
		anchor          string
		bindRequirement bool
		wantOutcome     domain.AdmissionOutcome
		wantReason      domain.AdmissionReasonCode
		wantClass       domain.AdmissionClassification
	}{
		{
			// The `#85` shape: the claim arrives after the code it excuses.
			name: "claimed after the work", anchor: commitAfterWork, bindRequirement: true,
			wantOutcome: domain.AdmissionDeny, wantReason: domain.ReasonRestorationNotAnchored,
			wantClass: domain.AdmissionInvalid,
		},
		{
			// The claim in the same commit as the work is not before it. An
			// author who decided while implementing has not met the condition.
			name: "claimed in the same commit", anchor: commitOfWork, bindRequirement: true,
			wantOutcome: domain.AdmissionDeny, wantReason: domain.ReasonRestorationNotAnchored,
			wantClass: domain.AdmissionInvalid,
		},
		{
			name: "claimed before the work", anchor: commitBeforeWork, bindRequirement: true,
			wantOutcome: domain.AdmissionAllow, wantReason: domain.ReasonExceptionApplied,
			// Visibly not VALID: the direct route stays an exception with a
			// recorded claim rather than a second ordinary lane.
			wantClass: domain.AdmissionExempted,
		},
		{
			// Naming a requirement in prose is what the claim replaces. Without
			// a digest that resolves to retained evidence there is nothing to
			// bind, and the failure is reported separately because its remedy
			// differs: this one can be corrected, a late claim cannot.
			name: "names a requirement it cannot bind", anchor: commitBeforeWork, bindRequirement: false,
			wantOutcome: domain.AdmissionDeny, wantReason: domain.ReasonRestorationUnbound,
			wantClass: domain.AdmissionInvalid,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := restorationInput(t, test.anchor, test.bindRequirement)
			result, err := Verify(input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != test.wantOutcome {
				t.Fatalf("outcome = %s, want %s (reasons %v)", result.Outcome, test.wantOutcome, result.Reasons)
			}
			if result.Classification != test.wantClass {
				t.Fatalf("classification = %s, want %s", result.Classification, test.wantClass)
			}
			if len(result.Reasons) == 0 || result.Reasons[0].Code != test.wantReason {
				t.Fatalf("reason = %v, want %s", result.Reasons, test.wantReason)
			}
		})
	}
}

// The issue's own diagnostic, as a check: a change that needs a governing
// contract to move in the same act is not a defect fix. If the artifact the
// claim binds is amended in the same range, there is no unchanged prior
// requirement to restore.
func TestARestorationCannotAmendWhatItBinds(t *testing.T) {
	input := restorationInput(t, commitBeforeWork, true)
	input.Range.Changes = append(input.Range.Changes, ChangedPath{
		Path: "openspec/specs/work-unit-lineage/spec.md", Kind: domain.ChangeModify,
		Commits: []string{commitOfWork},
	})
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationUnbound {
		t.Fatalf("a claim amended the requirement it binds and stood: %s %v", result.Outcome, result.Reasons)
	}
}

// The two failures must stay distinguishable. A caller told only "the exception
// did not apply" cannot tell an author whose claim is repairable from one whose
// claim can never be.
func TestTheTwoRestorationFailuresAreDistinctReasons(t *testing.T) {
	late, err := Verify(restorationInput(t, commitAfterWork, true))
	if err != nil {
		t.Fatal(err)
	}
	unbound, err := Verify(restorationInput(t, commitBeforeWork, false))
	if err != nil {
		t.Fatal(err)
	}
	if late.Reasons[0].Code == unbound.Reasons[0].Code {
		t.Fatalf("both failures report %s", late.Reasons[0].Code)
	}
}

// A claim anchored outside the frozen range cannot be shown to precede
// anything: the verifier was never given those commits. Accepting it would mean
// trusting an assertion, which is the thing the claim exists to replace.
func TestAnAnchorOutsideTheFrozenRangeIsNotProof(t *testing.T) {
	input := restorationInput(t, testBase, true)
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationNotAnchored {
		t.Fatalf("an unprovable anchor was accepted: %s %v", result.Outcome, result.Reasons)
	}
}

// The ordering condition belongs to the restoration class alone. A break-glass
// action is invoked during an emergency, so requiring it to precede its own
// work would describe something that cannot happen.
func TestOrderingDoesNotBindBreakGlass(t *testing.T) {
	// The same late anchor, under the class whose timing is inherent to what it
	// is. The break-glass authority already exists in the fixture policy.
	input := claimInput(t, domain.ExceptionBreakGlass, "owner-break-glass", commitAfterWork, true)
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionAllow || result.Classification != domain.AdmissionBreakGlass {
		t.Fatalf("break-glass inherited the ordering condition: %s %s %v", result.Outcome, result.Classification, result.Reasons)
	}
}

// restorationInput builds the three-commit history with a restoration claim
// anchored where the caller asks.
func restorationInput(t *testing.T, anchor string, bindRequirement bool) Input {
	t.Helper()
	return claimInput(t, domain.ExceptionRestoration, "owner-restoration", anchor, bindRequirement)
}

func claimInput(t *testing.T, class domain.ExceptionClass, authorityID, anchor string, bindRequirement bool) Input {
	t.Helper()
	policy := testPolicy(t)
	policy.ExceptionAuthorities = append(policy.ExceptionAuthorities, domain.PolicyExceptionAuthority{
		ID: "owner-restoration", Class: domain.ExceptionRestoration,
		ActorRefs: []string{"user:owner"},
		// Wide enough to cover the specification the claim binds, so the
		// amend-what-you-bind case is decided by its own condition rather than
		// by falling outside the claim's scope.
		PathPrefixes: []string{"internal", "openspec"}, EffectScopes: []string{"material_change"},
		MaxDurationSeconds: 3600, OwnerDecisionRequired: true,
	})
	input := validInputWithPolicyAndCommit(t, policy, commitOfWork)

	input.Range.Commits = []string{commitBeforeWork, commitOfWork, commitAfterWork}
	input.Range.CommitParents = map[string][]string{
		commitBeforeWork: {testBase},
		commitOfWork:     {commitBeforeWork},
		commitAfterWork:  {commitOfWork},
	}
	// The material path is touched by exactly one commit, which is what the
	// claim must precede.
	input.Range.Changes = []ChangedPath{{
		Path: "internal/app.go", Kind: domain.ChangeModify, Commits: []string{commitOfWork},
	}}

	// The artifact the claim binds: a retained requirement, present under
	// exactly the digest the claim names.
	requirement := []byte(`{"requirement":"exceptions are first-class lineage evidence"}`)
	requirementDigest := domain.DigestCanonicalJSON(requirement)
	if bindRequirement {
		input.Range.Graph.Replicas[requirementDigest] = requirement
	} else {
		requirementDigest = testDigest("requirement-that-was-never-retained")
	}

	// One hour end to end, which is the authority's maximum duration.
	issuedAt := input.Packet.EvaluationTime.Add(-30 * time.Minute)
	addClaim(t, &input, class, authorityID, restorationClaim{
		anchor: anchor, requirementDigest: requirementDigest,
		issuedAt: issuedAt, expiresAt: input.Packet.EvaluationTime.Add(30 * time.Minute),
	})
	return input
}

type restorationClaim struct {
	anchor            string
	requirementDigest domain.SHA256Digest
	issuedAt          time.Time
	expiresAt         time.Time
}

func addClaim(t *testing.T, input *Input, class domain.ExceptionClass, authorityID string, claim restorationClaim) {
	t.Helper()
	envelope := exceptionEnvelope{
		Schema: LineageExceptionSchemaV1, AuthorityID: authorityID, Class: class,
		ActorRef: "user:owner", PathPrefixes: []string{"internal", "openspec"}, EffectScopes: []string{"material_change"},
		IssuedAt: claim.issuedAt, ExpiresAt: claim.expiresAt,
		RequirementRef:    "repo:root/openspec/specs/work-unit-lineage/spec.md",
		RequirementDigest: claim.requirementDigest,
		AnchorCommit:      claim.anchor,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	digest := domain.DigestCanonicalJSON(raw)
	target := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "exception", Identity: "exception:" + authorityID, Version: "1", Digest: digest,
		SourceRef: "repo:root/.goalrail/evidence/sha256/" + strings.TrimPrefix(string(digest), "sha256:"), AdapterID: "goalrail",
	}
	event := domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: input.Range.Graph.Unit.ID,
		Relation: domain.LineageException, Cardinality: domain.RelationSingle,
		Sources: []domain.ContentAddressedEvidenceReference{input.Packet.WorkUnitRef}, Targets: []domain.ContentAddressedEvidenceReference{target},
		ActorRef: "user:owner", AdapterID: "goalrail", ObservedAt: claim.issuedAt,
	}
	refreezeEvent(t, &event)
	input.Range.Graph.Events = append(input.Range.Graph.Events, event)
	input.Range.Graph.Replicas[digest] = raw
	input.Packet.Provenance = append(input.Packet.Provenance, domain.AdmissionProviderProvenance{
		AdapterID: "goalrail", ProviderRef: input.Packet.TimeAuthorityRef,
		EvidenceDigest: testDigest("trusted-time"), ObservedAt: input.Packet.EvaluationTime.UTC(), Authenticated: true,
	})
	rebuildProjections(input)
}
