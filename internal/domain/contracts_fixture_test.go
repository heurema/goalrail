package domain

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type canonicalContractFixture struct {
	name     string
	artifact CanonicalArtifact
	refreeze func([]byte) (CanonicalArtifact, error)
}

func TestCanonicalContractFixturesMatchCheckedInGoldens(t *testing.T) {
	fixtures := canonicalContractFixtures(t)
	entries, err := os.ReadDir(filepath.Join("testdata", "contracts-v1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(fixtures) {
		t.Fatalf("golden fixture count changed: got %d, want %d", len(entries), len(fixtures))
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			golden, err := os.ReadFile(filepath.Join("testdata", "contracts-v1", fixture.name))
			if err != nil {
				t.Fatal(err)
			}
			golden = bytes.TrimSuffix(golden, []byte("\n"))
			if !bytes.Equal(golden, fixture.artifact.CanonicalJSON()) {
				t.Fatalf("canonical JSON differs from %s", fixture.name)
			}
			if fixture.artifact.Digest() != DigestCanonicalJSON(golden) {
				t.Fatalf("digest does not identify %s", fixture.name)
			}
		})
	}
}

func TestCanonicalContractGoldensRoundTripByteIdentically(t *testing.T) {
	for _, fixture := range canonicalContractFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			roundTripped, err := fixture.refreeze(fixture.artifact.CanonicalJSON())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(roundTripped.CanonicalJSON(), fixture.artifact.CanonicalJSON()) ||
				roundTripped.Digest() != fixture.artifact.Digest() {
				t.Fatal("canonical contract did not round-trip byte-identically")
			}
		})
	}
}

func TestEquivalentSemanticContractInputsAreByteIdentical(t *testing.T) {
	assertEquivalent := func(name string, left, right CanonicalArtifact) {
		t.Helper()
		if !bytes.Equal(left.CanonicalJSON(), right.CanonicalJSON()) || left.Digest() != right.Digest() {
			t.Fatalf("%s equivalent inputs produced different canonical artifacts", name)
		}
	}

	project := validProjectDeclaration()
	assertEquivalent("project", mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectDeclaration(project) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectDeclaration(project) }))

	policyLeft, policyRight := validProjectPolicy(), validProjectPolicy()
	policyRight.Rules[0], policyRight.Rules[1] = policyRight.Rules[1], policyRight.Rules[0]
	assertEquivalent("policy", mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectPolicy(policyLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectPolicy(policyRight) }))

	profileLeft, profileRight := validSetupProfile(), validSetupProfile()
	profileRight.ScaffoldAdapters[0], profileRight.ScaffoldAdapters[1] = profileRight.ScaffoldAdapters[1], profileRight.ScaffoldAdapters[0]
	assertEquivalent("setup profile", mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupProfile(profileLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupProfile(profileRight) }))

	planLeft, planRight := validSetupPlan(), validSetupPlan()
	planRight.Components[0], planRight.Components[1] = planRight.Components[1], planRight.Components[0]
	planRight.Mutations[0], planRight.Mutations[1] = planRight.Mutations[1], planRight.Mutations[0]
	assertEquivalent("setup plan", mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupPlan(planLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupPlan(planRight) }))

	authorizationLeft, authorizationRight := validPlanAuthorization(), validPlanAuthorization()
	authorizationRight.AuthorizedAt = authorizationLeft.AuthorizedAt.In(time.FixedZone("equivalent", 3*60*60))
	assertEquivalent("plan authorization", mustContract(t, func() (CanonicalArtifact, error) { return FreezePlanAuthorizationReference(authorizationLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezePlanAuthorizationReference(authorizationRight) }))

	receipt := validSetupReceipt()
	assertEquivalent("setup receipt", mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupReceipt(receipt) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupReceipt(receipt) }))

	workUnitLeft, workUnitRight := validWorkUnit(), validWorkUnit()
	workUnitRight.RequiredRelations[0], workUnitRight.RequiredRelations[1] = workUnitRight.RequiredRelations[1], workUnitRight.RequiredRelations[0]
	assertEquivalent("work unit", mustContract(t, func() (CanonicalArtifact, error) { return FreezeWorkUnit(workUnitLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeWorkUnit(workUnitRight) }))

	eventLeft, eventRight := validLineageEvent(), validLineageEvent()
	eventRight.Sources[0], eventRight.Sources[1] = eventRight.Sources[1], eventRight.Sources[0]
	eventRight.Targets[0], eventRight.Targets[1] = eventRight.Targets[1], eventRight.Targets[0]
	assertEquivalent("lineage event", mustContract(t, func() (CanonicalArtifact, error) { return FreezeLineageEvent(eventLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeLineageEvent(eventRight) }))

	packetLeft, packetRight := validAdmissionPacket(), validAdmissionPacket()
	packetRight.Evidence[0], packetRight.Evidence[1] = packetRight.Evidence[1], packetRight.Evidence[0]
	packetRight.Provenance[0], packetRight.Provenance[1] = packetRight.Provenance[1], packetRight.Provenance[0]
	assertEquivalent("admission packet", mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionPacket(packetLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionPacket(packetRight) }))

	resultLeft, resultRight := validAdmissionResult(), validAdmissionResult()
	resultRight.MaterialPaths[0], resultRight.MaterialPaths[1] = resultRight.MaterialPaths[1], resultRight.MaterialPaths[0]
	assertEquivalent("admission result", mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionResult(resultLeft) }), mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionResult(resultRight) }))
}

func canonicalContractFixtures(t *testing.T) []canonicalContractFixture {
	t.Helper()
	return []canonicalContractFixture{
		{name: "project.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectDeclaration(validProjectDeclaration()) }), refreeze: refreezeProject},
		{name: "policy.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeProjectPolicy(validProjectPolicy()) }), refreeze: refreezePolicy},
		{name: "setup-profile.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupProfile(validSetupProfile()) }), refreeze: refreezeSetupProfile},
		{name: "setup-plan.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupPlan(validSetupPlan()) }), refreeze: refreezeSetupPlan},
		{name: "plan-authorization.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezePlanAuthorizationReference(validPlanAuthorization()) }), refreeze: refreezePlanAuthorization},
		{name: "setup-receipt.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeSetupReceipt(validSetupReceipt()) }), refreeze: refreezeSetupReceipt},
		{name: "work-unit.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeWorkUnit(validWorkUnit()) }), refreeze: refreezeWorkUnit},
		{name: "lineage-event.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeLineageEvent(validLineageEvent()) }), refreeze: refreezeLineageEvent},
		{name: "admission-packet.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionPacket(validAdmissionPacket()) }), refreeze: refreezeAdmissionPacket},
		{name: "admission-result.json", artifact: mustContract(t, func() (CanonicalArtifact, error) { return FreezeAdmissionResult(validAdmissionResult()) }), refreeze: refreezeAdmissionResult},
	}
}

func mustContract(t *testing.T, freeze func() (CanonicalArtifact, error)) CanonicalArtifact {
	t.Helper()
	artifact, err := freeze()
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func refreezeProject(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeProjectDeclaration(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeProjectDeclaration(value)
}

func refreezePolicy(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeProjectPolicy(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeProjectPolicy(value)
}

func refreezeSetupProfile(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeSetupProfile(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeSetupProfile(value)
}

func refreezeSetupPlan(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeSetupPlan(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeSetupPlan(value)
}

func refreezePlanAuthorization(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodePlanAuthorizationReference(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezePlanAuthorizationReference(value)
}

func refreezeSetupReceipt(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeSetupReceipt(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeSetupReceipt(value)
}

func refreezeWorkUnit(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeWorkUnit(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeWorkUnit(value)
}

func refreezeLineageEvent(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeLineageEvent(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeLineageEvent(value)
}

func refreezeAdmissionPacket(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeAdmissionPacket(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeAdmissionPacket(value)
}

func refreezeAdmissionResult(raw []byte) (CanonicalArtifact, error) {
	value, err := DecodeAdmissionResult(bytes.NewReader(raw))
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return FreezeAdmissionResult(value)
}
