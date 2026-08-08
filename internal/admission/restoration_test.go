package admission

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

// The commits of the recorded failure. `#85` put the implementation first, the
// snapshot second and its confirmation third; after that backfill a verifier
// that reads only relations sees a confirmed snapshot bound to the work unit
// and no violation. What separates the honest sequence from the dishonest one
// is where the claim itself entered the history, so the fixture is a history
// rather than a set of fields.
const (
	commitBeforeWork = "cccccccccccccccccccccccccccccccccccccccc"
	commitOfWork     = "dddddddddddddddddddddddddddddddddddddddd"
	commitAfterWork  = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	// A second branch touching an in-scope material path, so a merge range can
	// be built where one branch is not an ancestor of the other.
	commitParallelWork = "ffffffffffffffffffffffffffffffffffffffff"
	commitMerge        = "1111111111111111111111111111111111111111"

	// The artifact a claim binds must be one the lineage already records with
	// its reference and digest together. The confirmed Intent Snapshot is
	// exactly that, and is the artifact a restoration claim names in practice.
	requirementPath = "openspec/changes/change/intent.md"
	claimPath       = ".goalrail/evidence/restoration-claim.json"
)

func TestARestorationClaimMustPrecedeTheWorkItExcuses(t *testing.T) {
	for _, test := range []struct {
		name        string
		claimCommit string
		wantOutcome domain.AdmissionOutcome
		wantReason  domain.AdmissionReasonCode
		wantClass   domain.AdmissionClassification
	}{
		{
			// The `#85` shape: the claim's own artifact enters the history
			// after the code it excuses.
			name: "claim committed after the work", claimCommit: commitAfterWork,
			wantOutcome: domain.AdmissionDeny, wantReason: domain.ReasonRestorationNotAnchored,
			wantClass: domain.AdmissionInvalid,
		},
		{
			// The claim in the same commit as the work is not before it. An
			// author who decided while implementing has not met the condition.
			name: "claim committed with the work", claimCommit: commitOfWork,
			wantOutcome: domain.AdmissionDeny, wantReason: domain.ReasonRestorationNotAnchored,
			wantClass: domain.AdmissionInvalid,
		},
		{
			name: "claim committed before the work", claimCommit: commitBeforeWork,
			wantOutcome: domain.AdmissionAllow, wantReason: domain.ReasonExceptionApplied,
			// Visibly not VALID: the direct route stays an exception with a
			// recorded claim rather than a second ordinary lane.
			wantClass: domain.AdmissionExempted,
		},
		{
			// The claim's artifact is not touched inside the range at all, so
			// it was already committed before the range began. That precedes
			// every commit in it — the strongest form of the claim.
			name: "claim already committed before the range", claimCommit: "",
			wantOutcome: domain.AdmissionAllow, wantReason: domain.ReasonExceptionApplied,
			wantClass: domain.AdmissionExempted,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Verify(restorationInput(t, claimFixture{claimCommit: test.claimCommit}))
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

// The order must not be readable from anything the claim says about itself.
//
// The first version of this check read an `anchor_commit` field out of the
// claim's envelope. The evidence is collected at head, so a claim written after
// the fact could name an earlier commit and pass — the gate was bypassable by
// typing a string. The field is gone; this test fails to compile if it returns.
func TestTheClaimCannotDeclareItsOwnOrder(t *testing.T) {
	raw, err := json.Marshal(exceptionEnvelope{
		Schema: LineageExceptionSchemaV1, Class: domain.ExceptionRestoration,
		RequirementRef: "repo:root/" + requirementPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, selfReported := range []string{"anchor_commit", "issued_at\":\"1970"} {
		if strings.Contains(string(raw), selfReported) {
			t.Fatalf("the envelope carries %q, which the actor making the claim writes", selfReported)
		}
	}

	// And the behaviour, not only the shape: the claim's own `issued_at`
	// precedes the work in every fixture here, and a claim whose artifact
	// entered the history late is still refused.
	result, err := Verify(restorationInput(t, claimFixture{claimCommit: commitAfterWork}))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationNotAnchored {
		t.Fatalf("a claim backdated in its own fields was accepted: %s %v", result.Outcome, result.Reasons)
	}
}

// Binding must compare the named artifact, not merely find something under the
// claimed digest. A claim pointing at any unrelated retained artifact passed
// the first version of this check because the reference went uncompared.
func TestBindingComparesTheNamedArtifact(t *testing.T) {
	for _, test := range []struct {
		name    string
		fixture claimFixture
	}{
		{"digest of an unrelated artifact", claimFixture{claimCommit: commitBeforeWork, bindUnrelatedDigest: true}},
		{"reference naming nothing recorded", claimFixture{claimCommit: commitBeforeWork, bindUnrecordedRef: true}},
		{"no binding at all", claimFixture{claimCommit: commitBeforeWork, bindNothing: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Verify(restorationInput(t, test.fixture))
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationUnbound {
				t.Fatalf("an unbound claim stood: %s %v", result.Outcome, result.Reasons)
			}
		})
	}
}

// Reverse topological order is a partial order. With two branches touching
// in-scope material paths, an anchor ancestral to one branch is not thereby
// ancestral to the other, and checking only the first touch in the list
// accepted a claim that preceded neither.
func TestEveryMaterialTouchIsCheckedNotTheFirst(t *testing.T) {
	input := restorationInput(t, claimFixture{claimCommit: commitBeforeWork, parallelBranch: true})
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationNotAnchored {
		t.Fatalf("a claim preceding one branch but not the other stood: %s %v", result.Outcome, result.Reasons)
	}
}

// The issue's own diagnostic, as a check: a change that needs a governing
// contract to move in the same act is not a defect fix. It must hold for any
// requirement inside the claim's scope, not only the one the claim binds —
// otherwise restoring requirement A while amending requirement B beside it
// passes.
func TestARestorationCannotAmendANormativePathInItsScope(t *testing.T) {
	for _, amended := range []string{
		// The artifact the claim binds, and a different one beside it. The
		// second is the case a bound-path comparison let through.
		requirementPath,
		"openspec/specs/lineage-admission/spec.md",
	} {
		t.Run(amended, func(t *testing.T) {
			input := restorationInput(t, claimFixture{claimCommit: commitBeforeWork, amend: amended})
			result, err := Verify(input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != domain.AdmissionDeny || result.Reasons[0].Code != domain.ReasonRestorationUnbound {
				t.Fatalf("a claim amended %s and stood: %s %v", amended, result.Outcome, result.Reasons)
			}
		})
	}
}

// The two failures must stay distinguishable. A caller told only "the exception
// did not apply" cannot tell an author whose claim is repairable from one whose
// claim can never be.
func TestTheTwoRestorationFailuresAreDistinctReasons(t *testing.T) {
	late, err := Verify(restorationInput(t, claimFixture{claimCommit: commitAfterWork}))
	if err != nil {
		t.Fatal(err)
	}
	unbound, err := Verify(restorationInput(t, claimFixture{claimCommit: commitBeforeWork, bindNothing: true}))
	if err != nil {
		t.Fatal(err)
	}
	if late.Reasons[0].Code == unbound.Reasons[0].Code {
		t.Fatalf("both failures report %s", late.Reasons[0].Code)
	}
}

// The ordering condition belongs to the restoration class alone. A break-glass
// action is invoked during an emergency, so requiring it to precede its own
// work would describe something that cannot happen.
func TestOrderingDoesNotBindBreakGlass(t *testing.T) {
	input := claimInput(t, domain.ExceptionBreakGlass, "owner-break-glass",
		claimFixture{claimCommit: commitAfterWork, bindNothing: true})
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != domain.AdmissionAllow || result.Classification != domain.AdmissionBreakGlass {
		t.Fatalf("break-glass inherited the restoration conditions: %s %s %v", result.Outcome, result.Classification, result.Reasons)
	}
}

// bindCommit records a second material commit in the lineage, so a merge range
// is refused on the ordering question rather than on an unbound commit.
func bindCommit(t *testing.T, input *Input, commit string) {
	t.Helper()
	for index := range input.Range.Graph.Events {
		event := input.Range.Graph.Events[index]
		if event.Relation != domain.LineageCommit {
			continue
		}
		event.Targets = append(append([]domain.ContentAddressedEvidenceReference(nil), event.Targets...),
			domain.ContentAddressedEvidenceReference{
				ArtifactKind: "commit", Identity: "git-commit:" + commit, Version: "1",
				Digest: testDigest(commit), SourceRef: "git:" + commit, AdapterID: "git",
			})
		refreezeEvent(t, &event)
		input.Range.Graph.Events[index] = event
		return
	}
	t.Fatal("no commit relation to bind")
}

// recordedTarget finds the target the lineage already carries for a reference,
// failing loudly rather than inventing one: a fixture that fabricates the
// artifact it binds would not exercise the binding at all.
func recordedTarget(t *testing.T, graph WorkUnitGraph, reference string) domain.ContentAddressedEvidenceReference {
	t.Helper()
	for _, event := range graph.Events {
		for _, target := range event.Targets {
			if target.SourceRef == reference {
				return target
			}
		}
	}
	t.Fatalf("no lineage target references %s", reference)
	return domain.ContentAddressedEvidenceReference{}
}

type claimFixture struct {
	// claimCommit is where the claim's own artifact enters the history. Empty
	// means it was committed before the range began.
	claimCommit         string
	bindUnrelatedDigest bool
	bindUnrecordedRef   bool
	bindNothing         bool
	amend               string
	parallelBranch      bool
}

func restorationInput(t *testing.T, fixture claimFixture) Input {
	t.Helper()
	return claimInput(t, domain.ExceptionRestoration, "owner-restoration", fixture)
}

func claimInput(t *testing.T, class domain.ExceptionClass, authorityID string, fixture claimFixture) Input {
	t.Helper()
	policy := testPolicy(t)
	policy.NormativePathPrefixes = []string{"openspec"}
	policy.ExceptionAuthorities = append(policy.ExceptionAuthorities, domain.PolicyExceptionAuthority{
		ID: "owner-restoration", Class: domain.ExceptionRestoration,
		ActorRefs: []string{"user:owner"},
		// Wide enough to cover the specification the claim binds and the
		// evidence the claim itself is, so those cases are decided by their own
		// conditions rather than by falling outside the claim's scope.
		PathPrefixes: []string{".goalrail", "internal", "openspec"}, EffectScopes: []string{"material_change"},
		MaxDurationSeconds: 3600, OwnerDecisionRequired: true,
	})
	input := validInputWithPolicyAndCommit(t, policy, commitOfWork)

	input.Range.Commits = []string{commitBeforeWork, commitOfWork, commitAfterWork}
	input.Range.CommitParents = map[string][]string{
		commitBeforeWork: {testBase},
		commitOfWork:     {commitBeforeWork},
		commitAfterWork:  {commitOfWork},
	}
	input.Range.Changes = []ChangedPath{{
		Path: "internal/app.go", Kind: domain.ChangeModify, Commits: []string{commitOfWork},
	}}
	if fixture.parallelBranch {
		// Two branches off the same parent, merged. Neither is an ancestor of
		// the other, and both touch in-scope material paths.
		input.Range.Commits = []string{commitBeforeWork, commitOfWork, commitParallelWork, commitMerge}
		input.Range.CommitParents = map[string][]string{
			commitBeforeWork:   {testBase},
			commitOfWork:       {commitBeforeWork},
			commitParallelWork: {testBase},
			commitMerge:        {commitOfWork, commitParallelWork},
		}
		input.Range.Changes = append(input.Range.Changes, ChangedPath{
			Path: "internal/other.go", Kind: domain.ChangeModify, Commits: []string{commitParallelWork},
		})
		bindCommit(t, &input, commitParallelWork)
	}
	if fixture.amend != "" {
		input.Range.Changes = append(input.Range.Changes, ChangedPath{
			Path: fixture.amend, Kind: domain.ChangeModify, Commits: []string{commitOfWork},
		})
	}
	if fixture.claimCommit != "" {
		input.Range.Changes = append(input.Range.Changes, ChangedPath{
			Path: claimPath, Kind: domain.ChangeAdd, Commits: []string{fixture.claimCommit},
		})
	}

	// One hour end to end, which is the authority's maximum duration, and
	// issued before the work in every case: the ordering answer must not come
	// from this field.
	issuedAt := input.Packet.EvaluationTime.Add(-30 * time.Minute)
	addClaim(t, &input, class, authorityID, fixture, issuedAt)
	return input
}

func addClaim(t *testing.T, input *Input, class domain.ExceptionClass, authorityID string, fixture claimFixture, issuedAt time.Time) {
	t.Helper()

	// The artifact the claim binds is already recorded by the work unit's
	// lineage, with its reference and digest on one target. That pairing is
	// what binding compares.
	requirementTarget := recordedTarget(t, input.Range.Graph, "repo:root/"+requirementPath)

	envelope := exceptionEnvelope{
		Schema: LineageExceptionSchemaV1, AuthorityID: authorityID, Class: class,
		ActorRef: "user:owner", PathPrefixes: []string{".goalrail", "internal", "openspec"},
		EffectScopes: []string{"material_change"},
		IssuedAt:     issuedAt, ExpiresAt: input.Packet.EvaluationTime.Add(30 * time.Minute),
		RequirementRef: requirementTarget.SourceRef, RequirementDigest: requirementTarget.Digest,
	}
	switch {
	case fixture.bindNothing:
		envelope.RequirementRef, envelope.RequirementDigest = "", ""
	case fixture.bindUnrelatedDigest:
		// A digest that resolves to a retained artifact, but not to the one the
		// reference names.
		envelope.RequirementDigest = input.Packet.WorkUnitRef.Digest
	case fixture.bindUnrecordedRef:
		envelope.RequirementRef = "repo:root/openspec/changes/never-recorded/intent.md"
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	digest := domain.DigestCanonicalJSON(raw)
	target := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "exception", Identity: "exception:" + authorityID, Version: "1", Digest: digest,
		SourceRef: "repo:root/" + claimPath, AdapterID: "goalrail",
	}
	event := domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: input.Range.Graph.Unit.ID,
		Relation: domain.LineageException, Cardinality: domain.RelationSingle,
		Sources:  []domain.ContentAddressedEvidenceReference{input.Packet.WorkUnitRef},
		Targets:  []domain.ContentAddressedEvidenceReference{target},
		ActorRef: "user:owner", AdapterID: "goalrail", ObservedAt: issuedAt,
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
