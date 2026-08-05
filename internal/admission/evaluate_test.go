package admission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/admissionlocal"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/lineage"
)

func TestPreparedSharedVerdictDoesNotDependOnLocalHookPresence(t *testing.T) {
	input := validInput(t)
	input.Range.Graph.Events = withoutBypassRelations(input.Range.Graph.Events)
	if _, err := admissionlocal.ValidateCommitMessage(strings.NewReader(
		"feat: indexed only\n\nGoalrail-Work-Unit: wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
	)); err != nil {
		t.Fatalf("valid early work-unit index: %v", err)
	}
	want, wantCode := runSerializedEntrypoint(t, input)

	directory := t.TempDir()
	hook := filepath.Join(directory, "pre-push")
	shims, err := admissionlocal.RenderShims("/opt/goalrail/bin/gr")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name    string
		prepare func(*testing.T)
	}{
		{name: "absent", prepare: func(*testing.T) {}},
		{name: "deleted", prepare: func(t *testing.T) {
			if err := os.WriteFile(hook, shims.Verify, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(hook); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "no-verify", prepare: func(t *testing.T) {
			// The hook exists, but the shared entrypoint is invoked directly,
			// which is the observable state after Git bypasses it.
			if err := os.WriteFile(hook, shims.Verify, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			_ = os.Remove(hook)
			fixture.prepare(t)
			got, gotCode := runSerializedEntrypoint(t, input)
			if string(got) != string(want) || gotCode != wantCode {
				t.Fatalf("local hook state changed shared verdict\nbefore: %s (%d)\nafter: %s (%d)", want, wantCode, got, gotCode)
			}
		})
	}
	got, _ := runSerializedEntrypoint(t, input)
	var result domain.AdmissionResult
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionMissing || result.Outcome != domain.AdmissionDeny {
		t.Fatalf("bypassed hook result = %+v", result)
	}
}

const (
	testBase = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testHead = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestPolicyMatcherUsesPathKindPriorityAndDefaultMaterial(t *testing.T) {
	policy := testPolicy(t)
	policy.Rules = append(policy.Rules,
		domain.PolicyPathRule{
			ID: "exact-readme", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherExact, Path: "README.md"},
			ChangeKinds: []domain.ChangeKind{domain.ChangeModify}, Priority: 300,
			Classification: domain.MaterialityNonMaterial,
		},
		domain.PolicyPathRule{
			ID: "generated", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: "gen"},
			ChangeKinds: []domain.ChangeKind{domain.ChangeAdd}, Priority: 250,
			Classification: domain.MaterialityGenerated, RequiredEvidenceKinds: []string{"generator_receipt"},
		},
	)
	evaluation := EvaluatePolicy(policy, []ChangedPath{
		{Path: "README.md", Kind: domain.ChangeModify},
		{Path: "README.md", Kind: domain.ChangeDelete},
		{Path: "gen/client.go", Kind: domain.ChangeAdd},
		{Path: ".goalrail/evidence/item.json", Kind: domain.ChangeAdd},
		{Path: "misc.txt", Kind: domain.ChangeModify},
	})
	if !reflect.DeepEqual(evaluation.MaterialPaths, []string{"README.md", "misc.txt"}) {
		t.Fatalf("material paths = %#v", evaluation.MaterialPaths)
	}
	if !reflect.DeepEqual(evaluation.RequiredEvidenceKinds, []string{"generator_receipt"}) {
		t.Fatalf("required evidence = %#v", evaluation.RequiredEvidenceKinds)
	}
	if !reflect.DeepEqual(evaluation.EvidencePaths, []string{".goalrail/evidence/item.json"}) {
		t.Fatalf("evidence paths = %#v", evaluation.EvidencePaths)
	}
	if missing := missingEvidenceKinds(evaluation.RequiredEvidenceKinds, WorkUnitGraph{}, domain.AdmissionPacket{}); !reflect.DeepEqual(missing, []string{"generator_receipt"}) {
		t.Fatalf("missing generated evidence = %#v", missing)
	}
	packet := domain.AdmissionPacket{Evidence: []domain.ContentAddressedEvidenceReference{testReference("generator_receipt", "generator-receipt:fixture", "repo:root/gen/receipt.json", "goalrail")}}
	if missing := missingEvidenceKinds(evaluation.RequiredEvidenceKinds, WorkUnitGraph{}, packet); len(missing) != 0 {
		t.Fatalf("present generated evidence remained missing: %#v", missing)
	}
}

func TestPolicyMatcherReportsOnlyOverlappingEqualPriorityAsAmbiguous(t *testing.T) {
	policy := testPolicy(t)
	policy.Rules = []domain.PolicyPathRule{
		{
			ID: "first", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: "internal"},
			ChangeKinds: []domain.ChangeKind{domain.ChangeModify}, Priority: 100,
			Classification: domain.MaterialityMaterial,
		},
		{
			ID: "second", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherExact, Path: "internal/app.go"},
			ChangeKinds: []domain.ChangeKind{domain.ChangeModify}, Priority: 100,
			Classification: domain.MaterialityNonMaterial,
		},
	}
	if _, err := domain.FreezeProjectPolicy(policy); err != nil {
		t.Fatalf("shared priority is valid before matching: %v", err)
	}
	evaluation := EvaluatePolicy(policy, []ChangedPath{{Path: "internal/app.go", Kind: domain.ChangeModify}})
	want := []string{"policy-rule:first", "policy-rule:second"}
	if !reflect.DeepEqual(evaluation.ConflictRefs, want) {
		t.Fatalf("conflict refs = %#v", evaluation.ConflictRefs)
	}
}

func TestAdmissionGoldenCorpusAndEntrypointsAreByteIdentical(t *testing.T) {
	type goldenCase struct {
		name           string
		build          func(*testing.T) Input
		classification domain.AdmissionClassification
		outcome        domain.AdmissionOutcome
		reason         domain.AdmissionReasonCode
	}
	cases := []goldenCase{
		{name: "valid", build: validInput, classification: domain.AdmissionValid, outcome: domain.AdmissionAllow},
		{name: "missing", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.Graph.Events = withoutRelation(input.Range.Graph.Events, domain.LineageTerminalReceipt)
			return input
		}, classification: domain.AdmissionMissing, outcome: domain.AdmissionDeny, reason: domain.ReasonReceiptMissing},
		{name: "ambiguous", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.Graph.Conflicts = []RelationConflict{{
				Relation: domain.LineageWorkSpec,
				Digests:  []domain.SHA256Digest{testDigest("conflict-a"), testDigest("conflict-b")},
			}}
			return input
		}, classification: domain.AdmissionAmbiguous, outcome: domain.AdmissionDeny, reason: domain.ReasonLineageConflict},
		{name: "malformed", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.Graph.Unit.RequiredRelations = input.Range.Graph.Unit.RequiredRelations[:len(input.Range.Graph.Unit.RequiredRelations)-1]
			refreezeWorkUnit(t, &input)
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonPacketInvalid},
		{name: "invalid-head-policy", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.HeadGovernanceInvalid = true
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonDeclarationInvalid},
		{name: "out-of-scope", build: func(t *testing.T) Input {
			input := validInput(t)
			for index := range input.Range.Graph.Events {
				if input.Range.Graph.Events[index].Relation == domain.LineageCommit {
					input.Range.Graph.Events[index].Targets[0].Identity = "git-commit:" + strings.Repeat("c", 40)
					input.Range.Graph.Events[index].Targets[0].SourceRef = "git:" + strings.Repeat("c", 40)
					refreezeEvent(t, &input.Range.Graph.Events[index])
				}
			}
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonChangeMismatch},
		{name: "unlinked-material-commit", build: func(t *testing.T) Input {
			input := validInput(t)
			materialCommit := strings.Repeat("c", 40)
			input.Range.Commits = append(input.Range.Commits, materialCommit)
			input.Range.Changes[0].Commits = []string{materialCommit}
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonChangeMismatch},
		{name: "self-weakened", build: func(t *testing.T) Input {
			input := validInput(t)
			headPolicy := *input.Range.HeadPolicy
			headPolicy.Rules = append([]domain.PolicyPathRule(nil), headPolicy.Rules...)
			headPolicy.Rules[0].Classification = domain.MaterialityNonMaterial
			policyArtifact, err := domain.FreezeProjectPolicy(headPolicy)
			if err != nil {
				t.Fatal(err)
			}
			headDeclaration := *input.Range.HeadDeclaration
			headDeclaration.Policy.Digest = policyArtifact.Digest()
			declarationArtifact, err := domain.FreezeProjectDeclaration(headDeclaration)
			if err != nil {
				t.Fatal(err)
			}
			input.Range.HeadPolicy = &headPolicy
			input.Range.HeadPolicyDigest = policyArtifact.Digest()
			input.Range.HeadDeclaration = &headDeclaration
			input.Range.HeadDeclarationDigest = declarationArtifact.Digest()
			return input
		}, classification: domain.AdmissionValid, outcome: domain.AdmissionAllow},
		{name: "expired-exception", build: func(t *testing.T) Input {
			input := validInput(t)
			addException(t, &input, input.Packet.EvaluationTime.Add(-2*time.Hour), input.Packet.EvaluationTime.Add(-time.Hour))
			input.Range.Graph.Events = withoutBypassRelations(input.Range.Graph.Events)
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonExceptionExpired},
		{name: "permitted-exception", build: func(t *testing.T) Input {
			input := validInput(t)
			addException(t, &input, input.Packet.EvaluationTime.Add(-time.Minute), input.Packet.EvaluationTime.Add(30*time.Minute))
			input.Range.Graph.Events = withoutBypassRelations(input.Range.Graph.Events)
			return input
		}, classification: domain.AdmissionBreakGlass, outcome: domain.AdmissionAllow, reason: domain.ReasonExceptionApplied},
		{name: "out-of-scope-exception", build: func(t *testing.T) Input {
			input := validInput(t)
			addException(t, &input, input.Packet.EvaluationTime.Add(-time.Minute), input.Packet.EvaluationTime.Add(30*time.Minute))
			for eventIndex := range input.Range.Graph.Events {
				if input.Range.Graph.Events[eventIndex].Relation != domain.LineageException {
					continue
				}
				target := input.Range.Graph.Events[eventIndex].Targets[0]
				var envelope exceptionEnvelope
				if err := json.Unmarshal(input.Range.Graph.Replicas[target.Digest], &envelope); err != nil {
					t.Fatal(err)
				}
				envelope.PathPrefixes = []string{"docs"}
				raw, err := json.Marshal(envelope)
				if err != nil {
					t.Fatal(err)
				}
				delete(input.Range.Graph.Replicas, target.Digest)
				target.Digest = domain.DigestCanonicalJSON(raw)
				target.SourceRef = "repo:root/.goalrail/evidence/sha256/" + strings.TrimPrefix(string(target.Digest), "sha256:")
				input.Range.Graph.Events[eventIndex].Targets[0] = target
				refreezeEvent(t, &input.Range.Graph.Events[eventIndex])
				input.Range.Graph.Replicas[target.Digest] = raw
			}
			return input
		}, classification: domain.AdmissionInvalid, outcome: domain.AdmissionDeny, reason: domain.ReasonExceptionScopeMismatch},
		{name: "migration", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.BaseDeclaration = nil
			input.Range.BasePolicy = nil
			input.Range.BaseDeclarationDigest = ""
			input.Range.BasePolicyDigest = ""
			return input
		}, classification: domain.AdmissionBootstrap, outcome: domain.AdmissionAllow, reason: domain.ReasonBootstrapRange},
		{name: "locally-bypassed", build: func(t *testing.T) Input {
			input := validInput(t)
			input.Range.Graph.Events = withoutBypassRelations(input.Range.Graph.Events)
			return input
		}, classification: domain.AdmissionMissing, outcome: domain.AdmissionDeny, reason: domain.ReasonRunSessionMissing},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := test.build(t)
			localBytes, localCode := runPureEntrypoint(t, input)
			sharedBytes, sharedCode := runSerializedEntrypoint(t, input)
			if string(localBytes) != string(sharedBytes) || localCode != sharedCode {
				t.Fatalf("entrypoints differ\nlocal: %s (%d)\nshared: %s (%d)", localBytes, localCode, sharedBytes, sharedCode)
			}
			var result domain.AdmissionResult
			if err := json.Unmarshal(localBytes, &result); err != nil {
				t.Fatal(err)
			}
			if result.Classification != test.classification || result.Outcome != test.outcome {
				t.Fatalf("result = %s/%s, want %s/%s: %s", result.Classification, result.Outcome, test.classification, test.outcome, localBytes)
			}
			if test.reason != "" && (len(result.Reasons) == 0 || result.Reasons[0].Code != test.reason) {
				t.Fatalf("reason = %#v, want %s", result.Reasons, test.reason)
			}
			if test.name == "self-weakened" && !reflect.DeepEqual(result.MaterialPaths, []string{"internal/app.go"}) {
				t.Fatalf("head policy weakened base evaluation: %#v", result.MaterialPaths)
			}
		})
	}
}

func TestAdmissionCoreIgnoresEnvironmentOrderClockAndCheckout(t *testing.T) {
	input := validInput(t)
	first, firstCode := runPureEntrypoint(t, input)
	t.Setenv("GOALRAIL_HIDDEN_OVERRIDE", "allow")
	t.Setenv("TZ", "Pacific/Kiritimati")
	sort.Slice(input.Range.Graph.Events, func(i, j int) bool {
		return input.Range.Graph.Events[i].SemanticDigest > input.Range.Graph.Events[j].SemanticDigest
	})
	input.Range.Graph.Replicas = map[domain.SHA256Digest][]byte{}
	second, secondCode := runPureEntrypoint(t, input)
	if string(first) != string(second) || firstCode != secondCode {
		t.Fatalf("non-semantic ambient state changed admission\nfirst: %s\nsecond: %s", first, second)
	}
}

func TestExternalEvidenceRequiresExactAuthenticatedProvenance(t *testing.T) {
	input := validInput(t)
	var external domain.ContentAddressedEvidenceReference
	for index := range input.Range.Graph.Events {
		if input.Range.Graph.Events[index].Relation != domain.LineageReviewIndex {
			continue
		}
		external = input.Range.Graph.Events[index].Targets[0]
		external.SourceRef = "github:review-index/42"
		external.AdapterID = "github"
		input.Range.Graph.Events[index].Targets[0] = external
		refreezeEvent(t, &input.Range.Graph.Events[index])
	}
	substituted := external
	substituted.Digest = testDigest("substituted-review")
	input.Packet.Evidence = []domain.ContentAddressedEvidenceReference{substituted}
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonPacketInvalid {
		t.Fatalf("substituted result = %+v", result)
	}
	input.Packet.Evidence = []domain.ContentAddressedEvidenceReference{external}
	result, err = Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonProvenanceUntrusted {
		t.Fatalf("untrusted result = %+v", result)
	}
	input.Packet.Provenance = []domain.AdmissionProviderProvenance{{
		AdapterID: "github", ProviderRef: "github:pull/42", EvidenceDigest: external.Digest,
		ObservedAt: input.Packet.EvaluationTime.Add(-time.Minute), Authenticated: true,
	}}
	result, err = Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionValid {
		t.Fatalf("authenticated result = %+v", result)
	}
}

func TestTimeBoundExceptionRequiresAuthenticatedExplicitTime(t *testing.T) {
	input := validInput(t)
	addException(t, &input, input.Packet.EvaluationTime.Add(-time.Minute), input.Packet.EvaluationTime.Add(30*time.Minute))
	input.Packet.Provenance = nil
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionMissing || result.Reasons[0].Code != domain.ReasonTrustedTimeMissing {
		t.Fatalf("untrusted exception time result = %+v", result)
	}
}

func TestExceptionClassesRemainVisible(t *testing.T) {
	want := map[domain.ExceptionClass]domain.AdmissionClassification{
		domain.ExceptionExempted:   domain.AdmissionExempted,
		domain.ExceptionBreakGlass: domain.AdmissionBreakGlass,
		domain.ExceptionBootstrap:  domain.AdmissionBootstrap,
	}
	for class, classification := range want {
		if got := admissionClass(class); got != classification {
			t.Fatalf("exception class %s = %s, want %s", class, got, classification)
		}
	}
}

func TestExceptionMayBypassAdmissionReadyClosure(t *testing.T) {
	if !exceptionMayBypass([]domain.LineageRelation{domain.LineageClosure}) {
		t.Fatal("admission-ready exception must not require a post-integration closure")
	}
	if exceptionMayBypass([]domain.LineageRelation{domain.LineageConfirmedIntent}) {
		t.Fatal("exception must not bypass confirmed intent")
	}
}

func validInput(t *testing.T) Input {
	t.Helper()
	policy := testPolicy(t)
	policyArtifact, err := domain.FreezeProjectPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	declaration := domain.ProjectDeclaration{
		Schema: domain.ProjectSchemaV1, ProjectID: policy.ProjectID, ContractVersion: domain.GovernanceContractV1,
		Policy:       domain.CommittedArtifactReference{Schema: domain.PolicySchemaV1, Path: domain.DefaultProjectPolicyPath, Digest: policyArtifact.Digest()},
		Bootstrap:    domain.CommittedArtifactReference{Schema: domain.BootstrapSchemaV1, Path: domain.DefaultProjectBootstrapPath, Digest: testDigest("bootstrap")},
		SetupProfile: domain.CommittedArtifactReference{Schema: domain.SetupProfileSchemaV1, Path: domain.DefaultProjectSetupProfilePath, Digest: testDigest("setup")},
	}
	declarationArtifact, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	intentRef := testReference("intent", "intent:goalrail-v1", "repo:root/openspec/changes/change/intent.md", "goalrail")
	changeRef := testReference("change", "change:goalrail-v1", "repo:root/openspec/changes/change", "goalrail")
	unit := domain.WorkUnit{
		Schema: domain.WorkUnitSchemaV1, ID: "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ProjectID: policy.ProjectID,
		DeclarationDigest: declarationArtifact.Digest(), PolicyDigest: policyArtifact.Digest(),
		IntentRef: intentRef, ChangeRef: changeRef,
		CreatedAt: time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC), Lifecycle: domain.WorkUnitOpen,
		RequiredRelations: lineage.DefaultRequirements(),
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		t.Fatal(err)
	}
	unitRef := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "work_unit", Identity: "work-unit:" + string(unit.ID), Version: "1", Digest: unitArtifact.Digest(),
		SourceRef: "repo:root/.goalrail/work-units/" + string(unit.ID) + "/unit.json", AdapterID: "goalrail",
	}
	evaluationTime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	events := testEvents(t, unit, unitRef, declaration, declarationArtifact.Digest(), policyArtifact.Digest(), testHead)
	packet := domain.AdmissionPacket{
		Schema: domain.AdmissionPacketSchemaV1, ProjectID: policy.ProjectID,
		DeclarationDigest: declarationArtifact.Digest(), PolicyDigest: policyArtifact.Digest(),
		BaseRevision: testBase, HeadRevision: testHead, WorkUnitRef: unitRef,
		EvaluationTime: &evaluationTime, TimeAuthorityRef: "provider:trusted-evaluation-time",
		Evidence: []domain.ContentAddressedEvidenceReference{}, Provenance: []domain.AdmissionProviderProvenance{},
	}
	return Input{
		Range: FrozenRange{
			BaseRevision: testBase, HeadRevision: testHead,
			BaseDeclaration: &declaration, BaseDeclarationDigest: declarationArtifact.Digest(),
			BasePolicy: &policy, BasePolicyDigest: policyArtifact.Digest(),
			HeadDeclaration: &declaration, HeadDeclarationDigest: declarationArtifact.Digest(),
			HeadPolicy: &policy, HeadPolicyDigest: policyArtifact.Digest(),
			Changes: []ChangedPath{{Path: "internal/app.go", Kind: domain.ChangeModify}},
			Commits: []string{testHead},
			Graph:   WorkUnitGraph{Unit: unit, Events: events, Replicas: map[domain.SHA256Digest][]byte{}},
		},
		Packet: packet,
	}
}

func testPolicy(t *testing.T) domain.ProjectPolicy {
	t.Helper()
	return domain.ProjectPolicy{
		Schema: domain.PolicySchemaV1, ProjectID: "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Version: 1,
		Rules: []domain.PolicyPathRule{
			{
				ID: "material-source", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: "internal"},
				ChangeKinds: []domain.ChangeKind{domain.ChangeAdd, domain.ChangeModify, domain.ChangeDelete, domain.ChangeRename, domain.ChangeMode, domain.ChangeSubmodule},
				Priority:    200, Classification: domain.MaterialityMaterial,
			},
			{
				ID: "goalrail-evidence", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: ".goalrail"},
				ChangeKinds: []domain.ChangeKind{domain.ChangeAdd, domain.ChangeModify, domain.ChangeDelete, domain.ChangeRename, domain.ChangeMode, domain.ChangeSubmodule},
				Priority:    100, Classification: domain.MaterialityEvidence,
			},
		},
		ExceptionAuthorities: []domain.PolicyExceptionAuthority{{
			ID: "owner-break-glass", Class: domain.ExceptionBreakGlass,
			ActorRefs: []string{"user:owner"}, PathPrefixes: []string{"internal"}, EffectScopes: []string{"material_change"},
			MaxDurationSeconds: 3600, OwnerDecisionRequired: true,
		}},
		OwnerDecision: domain.PolicyOwnerDecision{
			Required: true, AuthorityRefs: []string{"user:owner"},
			Outcomes: []domain.OwnerDecisionOutcome{domain.OwnerDecisionAllow, domain.OwnerDecisionReject},
		},
	}
}

func testEvents(t *testing.T, unit domain.WorkUnit, unitRef domain.ContentAddressedEvidenceReference, declaration domain.ProjectDeclaration, declarationDigest, policyDigest domain.SHA256Digest, commit string) []domain.LineageEvent {
	t.Helper()
	var events []domain.LineageEvent
	for _, requirement := range lineage.DefaultRequirements() {
		if requirement.Relation == domain.LineageClosure {
			continue
		}
		var targets []domain.ContentAddressedEvidenceReference
		switch requirement.Relation {
		case domain.LineageProjectPolicy:
			targets = []domain.ContentAddressedEvidenceReference{
				{ArtifactKind: "project_declaration", Identity: "project:" + string(declaration.ProjectID), Version: "1", Digest: declarationDigest, SourceRef: "repo:root/.goalrail/project.json", AdapterID: "goalrail"},
				{ArtifactKind: "project_policy", Identity: "policy:" + string(declaration.ProjectID), Version: "1", Digest: policyDigest, SourceRef: "repo:root/.goalrail/policy.json", AdapterID: "goalrail"},
			}
		case domain.LineageConfirmedIntent:
			targets = []domain.ContentAddressedEvidenceReference{unit.IntentRef}
		case domain.LineageChange:
			targets = []domain.ContentAddressedEvidenceReference{unit.ChangeRef}
		case domain.LineageCommit:
			targets = []domain.ContentAddressedEvidenceReference{{
				ArtifactKind: "commit", Identity: "git-commit:" + commit, Version: "1", Digest: testDigest(commit), SourceRef: "git:" + commit, AdapterID: "git",
			}}
		default:
			kind := relationArtifactKind(requirement.Relation)
			targets = []domain.ContentAddressedEvidenceReference{testReference(kind, kind+":fixture", "repo:root/.goalrail/index/"+kind+".json", "goalrail")}
		}
		event := domain.LineageEvent{
			Schema: domain.LineageEventSchemaV1, WorkUnitID: unit.ID, Relation: requirement.Relation, Cardinality: requirement.Cardinality,
			Sources: []domain.ContentAddressedEvidenceReference{unitRef}, Targets: targets,
			ActorRef: "user:owner", AdapterID: "goalrail", ObservedAt: time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC),
		}
		refreezeEvent(t, &event)
		events = append(events, event)
	}
	return events
}

func relationArtifactKind(relation domain.LineageRelation) string {
	switch relation {
	case domain.LineageWorkSpec:
		return "work_spec"
	case domain.LineageRunSession:
		return "run_session"
	case domain.LineagePullRequest:
		return "pull_request"
	case domain.LineageReviewIndex:
		return "review_index"
	case domain.LineageCheckSet:
		return "check_set"
	case domain.LineageTerminalReceipt:
		return "terminal_receipt"
	case domain.LineageOwnerDecision:
		return "owner_decision"
	default:
		return string(relation)
	}
}

func addException(t *testing.T, input *Input, issuedAt, expiresAt time.Time) {
	t.Helper()
	envelope := exceptionEnvelope{
		Schema: LineageExceptionSchemaV1, AuthorityID: "owner-break-glass", Class: domain.ExceptionBreakGlass,
		ActorRef: "user:owner", PathPrefixes: []string{"internal"}, EffectScopes: []string{"material_change"},
		IssuedAt: issuedAt, ExpiresAt: expiresAt,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	digest := domain.DigestCanonicalJSON(raw)
	target := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "exception", Identity: "exception:owner-break-glass", Version: "1", Digest: digest,
		SourceRef: "repo:root/.goalrail/evidence/sha256/" + strings.TrimPrefix(string(digest), "sha256:"), AdapterID: "goalrail",
	}
	event := domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: input.Range.Graph.Unit.ID,
		Relation: domain.LineageException, Cardinality: domain.RelationSingle,
		Sources: []domain.ContentAddressedEvidenceReference{input.Packet.WorkUnitRef}, Targets: []domain.ContentAddressedEvidenceReference{target},
		ActorRef: "user:owner", AdapterID: "goalrail", ObservedAt: issuedAt,
	}
	refreezeEvent(t, &event)
	input.Range.Graph.Events = append(input.Range.Graph.Events, event)
	input.Range.Graph.Replicas[digest] = raw
	input.Packet.Provenance = append(input.Packet.Provenance, domain.AdmissionProviderProvenance{
		AdapterID: "goalrail", ProviderRef: input.Packet.TimeAuthorityRef,
		EvidenceDigest: testDigest("trusted-time"), ObservedAt: input.Packet.EvaluationTime.UTC(), Authenticated: true,
	})
}

func withoutBypassRelations(events []domain.LineageEvent) []domain.LineageEvent {
	result := append([]domain.LineageEvent(nil), events...)
	for _, relation := range []domain.LineageRelation{
		domain.LineageRunSession, domain.LineagePullRequest, domain.LineageReviewIndex,
		domain.LineageCheckSet, domain.LineageTerminalReceipt,
	} {
		result = withoutRelation(result, relation)
	}
	return result
}

func withoutRelation(events []domain.LineageEvent, relation domain.LineageRelation) []domain.LineageEvent {
	result := make([]domain.LineageEvent, 0, len(events))
	for _, event := range events {
		if event.Relation != relation {
			result = append(result, event)
		}
	}
	return result
}

func refreezeEvent(t *testing.T, event *domain.LineageEvent) {
	t.Helper()
	event.SemanticDigest = ""
	artifact, err := domain.FreezeLineageEvent(*event)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := domain.DecodeLineageEvent(strings.NewReader(string(artifact.CanonicalJSON())))
	if err != nil {
		t.Fatal(err)
	}
	*event = decoded
}

func refreezeWorkUnit(t *testing.T, input *Input) {
	t.Helper()
	artifact, err := domain.FreezeWorkUnit(input.Range.Graph.Unit)
	if err != nil {
		t.Fatal(err)
	}
	input.Packet.WorkUnitRef.Digest = artifact.Digest()
}

func runPureEntrypoint(t *testing.T, input Input) ([]byte, int) {
	t.Helper()
	result, err := Verify(input)
	if err != nil {
		t.Fatal(err)
	}
	canonical, code, err := CanonicalResult(result)
	if err != nil {
		t.Fatal(err)
	}
	return canonical, code
}

func runSerializedEntrypoint(t *testing.T, input Input) ([]byte, int) {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var transported Input
	if err := json.Unmarshal(raw, &transported); err != nil {
		t.Fatal(err)
	}
	transported.Range.Graph.Replicas = make(map[domain.SHA256Digest][]byte, len(input.Range.Graph.Replicas))
	for digest, replica := range input.Range.Graph.Replicas {
		transported.Range.Graph.Replicas[digest] = append([]byte(nil), replica...)
	}
	return runPureEntrypoint(t, transported)
}

func testReference(kind, identity, source, adapter string) domain.ContentAddressedEvidenceReference {
	return domain.ContentAddressedEvidenceReference{
		ArtifactKind: kind, Identity: identity, Version: "1", Digest: testDigest(identity), SourceRef: source, AdapterID: adapter,
	}
}

func testDigest(value string) domain.SHA256Digest {
	return domain.DigestCanonicalJSON([]byte(value))
}
