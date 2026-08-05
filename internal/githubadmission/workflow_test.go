package githubadmission

import (
	"bytes"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestWorkflowIsPinnedMinimalPermissionAndPreparedOnly(t *testing.T) {
	pin := ReleasePin{
		Version: "v0.2.0", ArchiveURL: "https://github.com/heurema/goalrail/releases/download/v0.2.0/goalrail_linux_amd64_v0.2.0.tar.gz",
		ArchiveName: "goalrail_linux_amd64_v0.2.0.tar.gz", SHA256: domain.SHA256Digest("sha256:" + strings.Repeat("a", 64)),
	}
	raw, err := RenderWorkflow(pin)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range [][]byte{
		[]byte("actions/checkout@" + CheckoutActionCommit), []byte("contents: read"), []byte("pull-requests: read"),
		[]byte("checks: read"), []byte("sha256sum --check --strict"), []byte("github-collect"), []byte("verify-lineage"),
		[]byte("pull_request_review:"), []byte("trap 'rm -f"), []byte("GOALRAIL_VERSION: 'v0.2.0'"),
		[]byte(`grep --fixed-strings --quiet "\"version\":\"${GOALRAIL_VERSION}\""`),
	} {
		if !bytes.Contains(raw, required) {
			t.Fatalf("workflow omitted %q\n%s", required, raw)
		}
	}
	for _, forbidden := range [][]byte{
		[]byte("contents: write"), []byte("pull-requests: write"), []byte("checks: write"),
		[]byte("issues: write"), []byte("branches/"), []byte("rulesets"), []byte("gh pr comment"),
	} {
		if bytes.Contains(raw, forbidden) {
			t.Fatalf("workflow contains external mutation surface %q", forbidden)
		}
	}
	if _, err := RenderWorkflow(ReleasePin{Version: pin.Version, ArchiveURL: pin.ArchiveURL, ArchiveName: pin.ArchiveName}); err == nil {
		t.Fatal("workflow accepted an unpinned release")
	}
}
