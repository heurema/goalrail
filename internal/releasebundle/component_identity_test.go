package releasebundle

import (
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

// The producer and the consumer must agree on what a component may be called.
//
// A setup plan bounds a component identity; a release that published one beyond
// that bound would ship a bundle no plan could freeze, which is the failure this
// pair of rules exists to prevent — reintroduced at a different threshold rather
// than removed.
func TestAPublishedComponentIdentityIsOneASetupPlanCanExpress(t *testing.T) {
	long := "npm:node_modules/" + strings.Repeat("a", 200)
	if domain.IsComponentID(long) {
		t.Fatalf("the consumer accepts %d characters, so this case proves nothing", len(long))
	}

	manifest := SetupBundleManifest{
		Schema:                SetupManifestSchemaV1,
		ReleaseVersion:        "v0.2.0",
		Platform:              Platform{OS: "linux", Arch: "arm64"},
		ArchiveName:           setupArchiveName("v0.2.0", Platform{OS: "linux", Arch: "arm64"}),
		ManifestPath:          BundleManifestPath,
		CompilerLockDigest:    "sha256:" + strings.Repeat("a", 64),
		CompilerInstallPolicy: "never-run-package-scripts",
		Components: []ManifestComponent{{
			ID: long, Name: "overlong", Kind: "npm-package", Version: "1.0.0",
			Integrity:  "sha256:" + strings.Repeat("b", 64),
			LicenseRef: "spdx:MIT", ProvenanceRef: "npm:overlong@1.0.0",
		}},
		Files: []ManifestFile{{
			Path: "compiler/x", ComponentID: long, SizeBytes: 1,
			SHA256: "sha256:" + strings.Repeat("c", 64), Mode: "0644",
		}},
	}
	err := validateSetupManifestShape(manifest)
	if err == nil {
		t.Fatal("a manifest naming a component no setup plan can express was accepted")
	}
	if !strings.Contains(err.Error(), "component 0") {
		t.Fatalf("the refusal does not name the component: %v", err)
	}
}
