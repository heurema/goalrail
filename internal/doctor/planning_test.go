package doctor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/releasebundle"
)

// installedVersion is a release tag, because only a released binary corresponds
// to an installed bundle. A test binary reports the development marker, so the
// released shapes can only be exercised by naming one.
const installedVersion = "v0.2.0"

const (
	runtimeEntry  = "runtime/node/bin/node"
	compilerEntry = "node_modules/@fission-ai/openspec/bin/openspec.js"
	compilerID    = "npm:node_modules/@fission-ai/openspec"
)

// A promoted requirement forbids diagnosis from executing the planning runtime,
// a package runner, or the stock planning CLI. Asking a program for its version
// is executing it, so the boundary cannot be checked by inspecting a verdict —
// only by watching whether anything ran. These tests place decoys where the
// components would be resolved and assert that neither is invoked.

func TestPlanningObservationExecutesNothing(t *testing.T) {
	marker, decoys := decoyPath(t, "node", "openspec")
	t.Setenv("PATH", decoys)

	observation := defaultPlanningObserver{home: t.TempDir(), version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if invocations := recorded(t, marker); len(invocations) != 0 {
		t.Fatalf("diagnosis executed %d program(s): %v", len(invocations), invocations)
	}
	if observation.Runtime.State == "" || observation.Compiler.State == "" {
		t.Fatalf("diagnosis reported no runtime or compiler state: %#v", observation)
	}
}

// The declared runtime value is repository content. A checked-out repository
// that writes a path where a program name belongs must select nothing at all,
// rather than be caught by a rule about which characters a name may contain.
func TestDeclaredRuntimePathIsNeverResolvedOrRun(t *testing.T) {
	marker, decoys := decoyPath(t, "node", "openspec")
	t.Setenv("PATH", decoys)
	worktree := t.TempDir()
	planted := filepath.Join(worktree, "tools", "probe.sh")
	writeExecutable(t, planted, marker)

	observation := defaultPlanningObserver{home: t.TempDir(), version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile(planted))

	if invocations := recorded(t, marker); len(invocations) != 0 {
		t.Fatalf("a repository-supplied path was executed: %v", invocations)
	}
	if observation.Runtime.Path == planted {
		t.Fatalf("diagnosis resolved a repository-supplied path: %s", observation.Runtime.Path)
	}
}

// A complete installed bundle still reaches a ready verdict. Removing the
// execution must not make readiness unreachable, or every managed project would
// report setup required forever.
func TestVerifiedBundleReachesReady(t *testing.T) {
	home := installBundle(t, installedVersion, nil)

	observation := defaultPlanningObserver{home: home, version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if observation.Runtime.State != ComponentReady || observation.Compiler.State != ComponentReady {
		t.Fatalf("verified bundle did not reach ready: runtime=%#v compiler=%#v", observation.Runtime, observation.Compiler)
	}
	if observation.Runtime.ObservedVersion != "22.18.0" || observation.Compiler.ObservedVersion != "1.6.0" {
		t.Fatalf("observed identities came from somewhere other than the manifest: %#v %#v",
			observation.Runtime, observation.Compiler)
	}
}

// The bytes decide. An installed component altered after installation is an
// unverified integrity identity, which is one of the states the requirement
// already names, and it is reached without running the file.
func TestAlteredComponentIsUnverifiedIntegrity(t *testing.T) {
	home := installBundle(t, installedVersion, func(root string) {
		writeFile(t, filepath.Join(root, "runtime", "node", "bin", "node"), "tampered")
	})

	observation := defaultPlanningObserver{home: home, version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if observation.Runtime.State != ComponentUnverifiedIntegrity {
		t.Fatalf("altered runtime state = %q, want %q", observation.Runtime.State, ComponentUnverifiedIntegrity)
	}
}

// An installation that never ran authorized setup is reported as not ready,
// whatever unrelated toolchain the machine carries. The decoys satisfy the
// declared version exactly, so a probe that resolved them would have said ready.
func TestUninstalledToolchainIsNotCreditedToThePath(t *testing.T) {
	_, decoys := decoyPath(t, "node", "openspec")
	t.Setenv("PATH", decoys)

	observation := defaultPlanningObserver{home: t.TempDir(), version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if observation.Runtime.State != ComponentMissing || observation.Compiler.State != ComponentMissing {
		t.Fatalf("an unrelated toolchain was credited: runtime=%#v compiler=%#v", observation.Runtime, observation.Compiler)
	}
	if observation.Runtime.Detail == "" {
		t.Fatal("a not-ready component named no reason")
	}
}

// A build that was never published corresponds to no installed bundle, and the
// resolver says so rather than constructing a partial path and reading whatever
// happens to be there.
func TestOnlyAReleaseVersionResolvesABundle(t *testing.T) {
	home := installBundle(t, installedVersion, nil)
	for _, version := range []string{
		"", "unknown", "(devel)", "v0.2.0+dirty", "v0.2.1-0.20260806090733-23e30a5c8f6d", "0.2.0",
	} {
		if installed, reason := loadInstalledBundle(home, version); installed != nil || reason == "" {
			t.Fatalf("version %q resolved a bundle (%v) or gave no reason", version, installed)
		}
	}
	if installed, reason := loadInstalledBundle(home, installedVersion); installed == nil {
		t.Fatalf("the installed release version resolved nothing: %s", reason)
	}
}

func TestAbsentHomeResolvesNoBundle(t *testing.T) {
	if installed, reason := loadInstalledBundle("", installedVersion); installed != nil || reason == "" {
		t.Fatalf("an empty home resolved a bundle (%v) or gave no reason", installed)
	}
}

// Every way the manifest can fail to describe this installation is a stated
// reason, never a partial read.
func TestManifestRefusals(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		mutate  func(root string)
		version string
	}{
		{name: "absent", mutate: func(root string) { removeFile(t, filepath.Join(root, "manifest.json")) }},
		{name: "malformed", mutate: func(root string) { writeFile(t, filepath.Join(root, "manifest.json"), "{") }},
		{name: "unsupported schema", mutate: func(root string) { rewriteManifest(t, root, "schema", "goalrail.setup-bundle-manifest/v99") }},
		{name: "other release", mutate: func(root string) { rewriteManifest(t, root, "release_version", "v0.9.9") }},
		{name: "other platform", mutate: func(root string) { rewriteManifest(t, root, "platform", map[string]string{"os": "plan9", "arch": "sparc"}) }},
		{name: "not a regular file", mutate: func(root string) {
			path := filepath.Join(root, "manifest.json")
			removeFile(t, path)
			if err := os.Symlink(os.DevNull, path); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			home := installBundle(t, installedVersion, testCase.mutate)
			installed, reason := loadInstalledBundle(home, installedVersion)
			if installed != nil {
				t.Fatalf("a %s manifest was accepted", testCase.name)
			}
			if reason == "" {
				t.Fatalf("a %s manifest was refused without a reason", testCase.name)
			}
		})
	}
}

// The prohibition is a property of this package, not of one function. Keeping it
// as an assertion means a returning execution fails a test rather than depending
// on a reviewer noticing it.
func TestDiagnosisPackageCannotExecuteAnything(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(contents), `"os/exec"`) {
			t.Errorf("%s imports os/exec; diagnosis reports the toolchain by inspection and runs nothing", name)
		}
	}
}

func canonProfile(runtimeID string) domain.SetupProfile {
	return domain.SetupProfile{
		Schema:                   domain.SetupProfileSchemaV1,
		CompatibleGoalrailBundle: ">=0.2.0 <0.3.0",
		Planning: domain.SetupPlanningAdapter{
			Adapter:         domain.SetupAdapterPin{ID: "openspec", Version: "1.6.0"},
			Runtime:         runtimeID,
			RuntimeVersion:  "22.18.0",
			Compiler:        "@fission-ai/openspec",
			CompilerVersion: "1.6.0",
		},
	}
}

// decoyPath installs one executable per name that records its own invocation and
// prints the exact declared version, so a probe that runs it would be satisfied
// rather than fail for an unrelated reason. It returns the marker file and the
// PATH value that reaches the decoys first.
func decoyPath(t *testing.T, names ...string) (string, string) {
	t.Helper()
	directory := t.TempDir()
	marker := filepath.Join(t.TempDir(), "invocations")
	for _, name := range names {
		writeExecutable(t, filepath.Join(directory, name), marker)
	}
	return marker, directory + string(os.PathListSeparator) + os.Getenv("PATH")
}

func writeExecutable(t *testing.T, path, marker string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho " + filepath.Base(path) + " \"$@\" >> " + marker + "\n" +
		"echo 22.18.0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// installBundle lays out what authorized setup leaves behind: the two component
// files and a manifest that records their digests. The optional mutation runs
// afterwards, so a case can damage exactly one thing about an otherwise
// complete installation.
func installBundle(t *testing.T, version string, mutate func(root string)) string {
	t.Helper()
	home := t.TempDir()
	platform := releasebundle.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	root := filepath.Join(home, ".local", "share", "goalrail", "bundles", version, platform.Key())

	runtimeBytes, compilerBytes := "installed-node-binary", "installed-openspec-entrypoint"
	writeFile(t, filepath.Join(root, filepath.FromSlash(runtimeEntry)), runtimeBytes)
	writeFile(t, filepath.Join(root, filepath.FromSlash(compilerEntry)), compilerBytes)

	manifest := releasebundle.SetupBundleManifest{
		Schema:         releasebundle.SetupManifestSchemaV1,
		ReleaseVersion: version,
		Platform:       platform,
		ManifestPath:   releasebundle.BundleManifestPath,
		Components: []releasebundle.ManifestComponent{
			{ID: "node", Name: "node", Kind: "private-runtime", Version: "22.18.0"},
			{ID: compilerID, Name: "@fission-ai/openspec", Kind: "npm-package", Version: "1.6.0"},
		},
		BinaryIdentities: []releasebundle.BinaryIdentity{{
			ComponentID: "node", Path: runtimeEntry, Kind: "node",
			Version: "22.18.0", SHA256: digestOf(runtimeBytes),
		}},
		Files: []releasebundle.ManifestFile{
			{Path: runtimeEntry, ComponentID: "node", SizeBytes: int64(len(runtimeBytes)), SHA256: digestOf(runtimeBytes), Mode: "0755"},
			{Path: compilerEntry, ComponentID: compilerID, SizeBytes: int64(len(compilerBytes)), SHA256: digestOf(compilerBytes), Mode: "0644"},
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, releasebundle.BundleManifestPath), string(raw))
	if mutate != nil {
		mutate(root)
	}
	return home
}

// rewriteManifest changes one field of an otherwise complete manifest, so a
// refusal case differs from the accepted one in exactly the thing it names.
func rewriteManifest(t *testing.T, root, field string, value any) {
	t.Helper()
	path := filepath.Join(root, releasebundle.BundleManifestPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document[field] = value
	rewritten, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(rewritten))
}

func digestOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func recorded(t *testing.T, marker string) []string {
	t.Helper()
	contents, err := os.ReadFile(marker)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var invocations []string
	for _, line := range strings.Split(strings.TrimSpace(string(contents)), "\n") {
		if strings.TrimSpace(line) != "" {
			invocations = append(invocations, line)
		}
	}
	return invocations
}
