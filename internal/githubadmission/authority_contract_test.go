package githubadmission

import (
	"bytes"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	projectstate "github.com/heurema/goalrail/internal/project"
)

// The authority a provider reports and the authority a policy permits have to
// be the same vocabulary, and nothing but a test can hold them together.
//
// They were not. This adapter reports a provider permission, while the canon
// default policy permitted only a role, so the exact membership check in the
// verifier left the owner decision unsatisfied and shared admission denied
// every approved review. Nothing failed: the end-to-end fixture supplied the
// role directly and never crossed the seam.
func TestTheDefaultPolicyPermitsEveryAuthorityThisAdapterReports(t *testing.T) {
	files, err := projectstate.RenderProjectCanon("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	var policyRaw []byte
	for _, file := range files {
		if strings.HasSuffix(file.Path, domain.DefaultProjectPolicyPath) || file.Path == domain.DefaultProjectPolicyPath {
			policyRaw = file.Content
		}
	}
	if len(policyRaw) == 0 {
		t.Fatalf("the rendered canon carries no project policy: %d files", len(files))
	}
	policy, err := domain.DecodeProjectPolicy(bytes.NewReader(policyRaw))
	if err != nil {
		t.Fatal(err)
	}

	for _, permission := range authorizingPermissions() {
		reported := "github-permission:" + permission
		permitted := false
		for _, candidate := range policy.OwnerDecision.AuthorityRefs {
			if candidate == reported {
				permitted = true
				break
			}
		}
		if !permitted {
			t.Fatalf("this adapter reports %q, which the default policy does not permit: %#v",
				reported, policy.OwnerDecision.AuthorityRefs)
		}
	}
}
