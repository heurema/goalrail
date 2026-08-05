package domain

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSetupContractsCanonicalizeSetOrder(t *testing.T) {
	left := validSetupPlan()
	right := validSetupPlan()
	right.Components[0], right.Components[1] = right.Components[1], right.Components[0]
	right.Mutations[0], right.Mutations[1] = right.Mutations[1], right.Mutations[0]

	leftFrozen, err := FreezeSetupPlan(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeSetupPlan(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) {
		t.Fatal("equivalent setup plans must produce byte-identical canonical JSON")
	}
}

func TestIncompleteSetupPlanCanReportUnresolvedEvidenceWithoutInventingComponents(t *testing.T) {
	plan := validSetupPlan()
	plan.State = SetupPlanIncomplete
	plan.IncompleteReasonIDs = []string{"RELEASE_METADATA_UNAVAILABLE"}
	plan.Components = nil
	plan.Mutations = nil
	plan.Rollback = nil
	plan.Verification = nil
	if _, err := FreezeSetupPlan(plan); err != nil {
		t.Fatalf("incomplete plan was rejected: %v", err)
	}

	plan.State = SetupPlanComplete
	plan.IncompleteReasonIDs = nil
	if _, err := FreezeSetupPlan(plan); err == nil {
		t.Fatal("complete plan accepted no components or verification")
	}
}

func TestSetupContractsRejectProhibitedPayloads(t *testing.T) {
	receipt := validSetupReceipt()
	receipt.ContinuationRef = "raw feature prompt copied here"
	if _, err := FreezeSetupReceipt(receipt); err == nil {
		t.Fatal("raw request content was accepted as a continuation reference")
	}

	plan := validSetupPlan()
	plan.Components[0].Destination = "/Users/example/access_token=top-secret-value"
	if _, err := FreezeSetupPlan(plan); err == nil {
		t.Fatal("secret-shaped setup content was accepted")
	}

	raw := strings.Replace(
		string(mustFreezeSetupReceipt(t, validSetupReceipt()).CanonicalJSON()),
		`"continuation_ref":`,
		`"raw_prompt":"do the feature","continuation_ref":`,
		1,
	)
	if _, err := DecodeSetupReceipt(strings.NewReader(raw)); err == nil {
		t.Fatal("unknown raw_prompt field was accepted")
	}
}

func TestSetupReceiptAuthorizationMustBindExactPlan(t *testing.T) {
	receipt := validSetupReceipt()
	receipt.Authorization.PlanDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := FreezeSetupReceipt(receipt); err == nil {
		t.Fatal("receipt accepted authorization for a different plan")
	}
}

func TestPlanAuthorizationCanonicalizesTimeToUTC(t *testing.T) {
	left := validPlanAuthorization()
	right := left
	right.AuthorizedAt = left.AuthorizedAt.In(time.FixedZone("offset", -7*60*60))
	leftFrozen, err := FreezePlanAuthorizationReference(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezePlanAuthorizationReference(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) {
		t.Fatal("equivalent authorization instants must be byte-identical")
	}
}

func validSetupProfile() SetupProfile {
	shared := SetupAdapterPin{
		ID:        "github-actions",
		Version:   "1.0.0",
		SourceRef: "release:github-actions-v1",
		Integrity: digestWith('d'),
	}
	return SetupProfile{
		Schema:                   SetupProfileSchemaV1,
		ProjectID:                "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CompatibleGoalrailBundle: ">=0.2.0 <0.3.0",
		Planning: SetupPlanningAdapter{
			Adapter: SetupAdapterPin{
				ID:        "openspec",
				Version:   "1.6.0",
				SourceRef: "npm:fission-ai/openspec@1.6.0",
				Integrity: digestWith('c'),
			},
			Runtime:         "node",
			RuntimeVersion:  "22.18.0",
			Compiler:        "openspec",
			CompilerVersion: "1.6.0",
		},
		ScaffoldAdapters: []SetupAdapterPin{
			{ID: "codex", Version: "1.0.0", SourceRef: "bundle:codex-v1", Integrity: digestWith('a')},
			{ID: "claude", Version: "1.0.0", SourceRef: "bundle:claude-v1", Integrity: digestWith('b')},
		},
		SharedAdmissionAdapter: &shared,
	}
}

func validSetupPlan() SetupPlan {
	before := digestWith('1')
	return SetupPlan{
		Schema:             SetupPlanSchemaV1,
		ProjectID:          "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DeclarationDigest:  digestWith('2'),
		SetupProfileDigest: digestWith('3'),
		Platform:           "darwin-arm64",
		State:              SetupPlanComplete,
		Components: []SetupComponent{
			{
				ID: "goalrail", Version: "0.2.0", SourceRef: "release:goalrail-v0.2.0-darwin-arm64",
				Integrity: digestWith('4'), SizeBytes: 1024, Destination: "/Users/example/.local/share/goalrail/0.2.0/gr",
				Scope: SetupScopeUserLocal, LicenseRef: "license:goalrail", ProvenanceRef: "release:goalrail-v0.2.0",
			},
			{
				ID: "openspec-runtime", Version: "1.6.0", SourceRef: "release:openspec-runtime-v1.6.0-darwin-arm64",
				Integrity: digestWith('5'), SizeBytes: 2048, Destination: "/Users/example/.local/share/goalrail/runtime/openspec-1.6.0",
				Scope: SetupScopeUserLocal, LicenseRef: "license:openspec", ProvenanceRef: "release:openspec-v1.6.0",
			},
		},
		Mutations: []SetupMutation{
			{ID: "select-gr", Kind: SetupMutationSelectExecutable, Scope: SetupScopeUserLocal, Path: "/Users/example/.local/bin/gr", ExpectedBeforeDigest: &before, DesiredDigest: digestWith('6')},
			{ID: "codex-hook", Kind: SetupMutationInstallHook, Scope: SetupScopeUserLocal, Path: "/Users/example/.codex/config.toml", ExpectedBeforeDigest: &before, DesiredDigest: digestWith('7')},
		},
		Prerequisites: []SetupPrerequisite{
			{ID: "supported-platform", Kind: SetupPrerequisitePlatform, VersionConstraint: "darwin-arm64", Satisfied: true, EvidenceRef: "platform:darwin-arm64"},
		},
		TrustSteps: []SetupTrustStep{
			{ID: "codex-trust", AdapterID: "codex", ActionRef: "instruction:confirm-codex-hook", Interactive: true},
		},
		NetworkAccess: []SetupNetworkAccess{
			{ID: "fetch-bundle", Method: "GET", URL: "https://releases.goalrail.dev/v0.2.0/setup-darwin-arm64.tar.gz", PurposeID: "fetch-bundle"},
		},
		Rollback: []SetupRollbackAction{
			{MutationID: "codex-hook", Action: "restore-bytes", Target: "/Users/example/.codex/config.toml", PriorStateDigest: &before},
			{MutationID: "select-gr", Action: "restore-link", Target: "/Users/example/.local/bin/gr", PriorStateDigest: &before},
		},
		Verification: []SetupVerification{
			{ID: "doctor", Argv: []string{"/Users/example/.local/bin/gr", "doctor", "--json"}},
		},
		ProjectCodeWrites: 0,
	}
}

func validPlanAuthorization() PlanAuthorizationReference {
	return PlanAuthorizationReference{
		Schema:       PlanAuthorizationSchemaV1,
		ProjectID:    "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PlanDigest:   digestWith('8'),
		DecisionRef:  "decision:setup-plan-1",
		ActorRef:     "user:owner-a",
		AuthorizedAt: time.Date(2026, 8, 4, 9, 30, 0, 0, time.UTC),
	}
}

func validSetupReceipt() SetupReceipt {
	authorization := validPlanAuthorization()
	before := digestWith('1')
	after := digestWith('6')
	return SetupReceipt{
		Schema:        SetupReceiptSchemaV1,
		ProjectID:     authorization.ProjectID,
		PlanDigest:    authorization.PlanDigest,
		Authorization: &authorization,
		Components: []SetupComponentResult{
			{ID: "goalrail", Version: "0.2.0", Integrity: digestWith('4'), Status: SetupActionApplied},
		},
		Mutations: []SetupMutationResult{
			{ID: "select-gr", Status: SetupActionApplied, BeforeDigest: &before, AfterDigest: &after},
		},
		RollbackRef:     "receipt:setup-rollback-1",
		DiagnosisRef:    "doctor:local-ready-1",
		Status:          SetupReceiptSuccess,
		ContinuationRef: "task:feature-request-1",
	}
}

func mustFreezeSetupReceipt(t *testing.T, receipt SetupReceipt) CanonicalArtifact {
	t.Helper()
	frozen, err := FreezeSetupReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return frozen
}

func digestWith(character byte) SHA256Digest {
	return SHA256Digest("sha256:" + string(bytes.Repeat([]byte{character}, 64)))
}
