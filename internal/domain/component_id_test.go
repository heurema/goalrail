package domain

import (
	"os"
	"strings"
	"testing"
)

// Every component identity the published release actually carries must validate.
//
// The corpus is the real thing rather than an invented sample: the defect this
// pins was invisible to every synthetic fixture and appeared the first time a
// setup plan met a manifest from an actual release, where 80 of 83 components
// are named by their ecosystem rather than by a canonical ID.
func TestPublishedComponentIdentitiesValidate(t *testing.T) {
	raw, err := os.ReadFile("testdata/published-setup-component-ids.txt")
	if err != nil {
		t.Fatal(err)
	}
	identities := strings.Fields(string(raw))
	if len(identities) < 80 {
		t.Fatalf("the pinned corpus holds %d identities, too few to describe a release", len(identities))
	}
	for _, identity := range identities {
		if !IsComponentID(identity) {
			t.Errorf("a published component identity is rejected: %q", identity)
		}
	}
}

// The component rule is not the canonical rule widened. Loosening that one would
// weaken every project, run, check and lineage identifier in the domain.
func TestComponentIdentityDoesNotWidenTheCanonicalRule(t *testing.T) {
	for _, value := range []string{"npm:node_modules/zod", "npm:node_modules/@fission-ai/openspec"} {
		if IsCanonicalID(value) {
			t.Errorf("%q now passes as a canonical ID", value)
		}
		if !IsComponentID(value) {
			t.Errorf("%q is not accepted as a component identity", value)
		}
	}
}

func TestComponentIdentityRefusals(t *testing.T) {
	for _, testCase := range []struct{ name, value string }{
		{"empty", ""},
		{"climbing out", "npm:node_modules/../../etc/passwd"},
		{"bare traversal", "npm:.."},
		{"current directory segment", "npm:node_modules/./zod"},
		{"empty segment", "npm:node_modules//zod"},
		{"absolute", "npm:/etc/passwd"},
		{"two namespaces", "npm:other:node_modules/zod"},
		{"control character", "npm:node_modules/zo\x00d"},
		{"newline", "npm:node_modules/zod\nnpm:evil"},
		{"space", "npm:node modules/zod"},
		{"upper-case namespace", "NPM:node_modules/zod"},
		{"unbounded identity", "npm:" + strings.Repeat("a", 200)},
		{"unbounded namespace", strings.Repeat("a", 40) + ":node_modules/zod"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if IsComponentID(testCase.value) {
				t.Fatalf("%q was accepted as a component identity", testCase.value)
			}
		})
	}
}
