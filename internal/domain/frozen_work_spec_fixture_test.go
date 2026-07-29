package domain

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// frozenFixtureDigest pins the identity of a checked-in goalrail.work-spec/v0
// document. It exists to catch a silent fork of the canonical schema: adding,
// renaming, reordering, or re-typing a WorkSpec field would move this digest and
// invalidate every frozen WorkSpec that already exists.
//
// If this test fails, the canonical schema changed. That is a versioning
// decision, not a fixture to update.
const frozenFixtureDigest = "sha256:f328866a631fe0e31c0da68b1ba0956d170cedb121b303114dc701bfbd858b63"

func TestFrozenWorkSpecV0DigestIsUnchanged(t *testing.T) {
	raw, err := os.ReadFile("testdata/frozen-work-spec-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := DecodeWorkSpec(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := FreezeWorkSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if string(frozen.Digest()) != frozenFixtureDigest {
		t.Fatalf(
			"frozen WorkSpec digest = %s, want %s: goalrail.work-spec/v0 changed",
			frozen.Digest(),
			frozenFixtureDigest,
		)
	}
}

func TestWorkSpecRejectsAnEscalationPathField(t *testing.T) {
	// The escalation channel is a reserved constant path precisely so that the
	// canonical WorkSpec needs no new field. DecodeWorkSpec rejecting one is the
	// property that keeps the digest above stable.
	raw, err := os.ReadFile("testdata/frozen-work-spec-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	withField := strings.Replace(
		string(raw),
		`"paths": ["inside"],`,
		`"paths": ["inside"],
  "escalation_path": ".goalrail/blocked.md",`,
		1,
	)
	if withField == string(raw) {
		t.Fatal("fixture shape changed; the unknown-field probe patched nothing")
	}
	if _, err := DecodeWorkSpec(strings.NewReader(withField)); err == nil {
		t.Fatal("an unknown WorkSpec field was accepted, forking goalrail.work-spec/v0")
	}
}
