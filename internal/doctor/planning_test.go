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

// The exact paths and component identifiers the release builder emits, so the
// fixture exercises the contract rather than a convenient shape of its own.
const (
	goalrailEntry = "bin/gr"
	runtimeEntry  = "runtime/node/bin/node"
	compilerEntry = "compiler/node_modules/@fission-ai/openspec/bin/openspec.js"
	compilerOther = "compiler/node_modules/@fission-ai/openspec/LICENSE"
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
		{name: "unsupported schema", mutate: func(root string) {
			mutateManifest(t, root, func(m *releasebundle.SetupBundleManifest) { m.Schema = "goalrail.setup-bundle-manifest/v99" })
		}},
		{name: "other release", mutate: func(root string) {
			mutateManifest(t, root, func(m *releasebundle.SetupBundleManifest) { m.ReleaseVersion = "v0.9.9" })
		}},
		{name: "other platform", mutate: func(root string) {
			mutateManifest(t, root, func(m *releasebundle.SetupBundleManifest) {
				m.Platform = releasebundle.Platform{OS: "plan9", Arch: "sparc"}
			})
		}},
		{name: "not canonical", mutate: func(root string) {
			path := filepath.Join(root, releasebundle.BundleManifestPath)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, "  "+string(raw))
		}},
		{name: "component path escapes the bundle", mutate: func(root string) {
			mutateManifest(t, root, func(m *releasebundle.SetupBundleManifest) {
				m.Files[3].Path = "../../../../tmp/node"
				m.BinaryIdentities[2].Path = "../../../../tmp/node"
			})
		}},
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

// A package owns many files, and only one of them is the entrypoint. Verifying
// whichever file sorted first would leave the entrypoint replaceable while an
// intact licence kept the component green.
func TestReplacedCompilerEntrypointIsCaught(t *testing.T) {
	home := installBundle(t, installedVersion, func(root string) {
		writeFile(t, filepath.Join(root, filepath.FromSlash(compilerEntry)), "replaced-entrypoint")
	})

	observation := defaultPlanningObserver{home: home, version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if observation.Compiler.State == ComponentReady {
		t.Fatalf("a replaced compiler entrypoint stayed ready: %#v", observation.Compiler)
	}
	if !strings.HasSuffix(observation.Compiler.Path, filepath.FromSlash(compilerEntry)) {
		t.Fatalf("the compiler was verified at %q, not at its manifest entrypoint", observation.Compiler.Path)
	}
}

// A manifest is a description, not an authority over where the reader may look.
// One edited to point a component outside the bundle must be refused before any
// path it carries is opened.
func TestManifestCannotPointOutsideTheBundle(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "node")
	writeFile(t, outside, "planted-outside-the-bundle")
	home := installBundle(t, installedVersion, func(root string) {
		mutateManifest(t, root, func(m *releasebundle.SetupBundleManifest) {
			relative, err := filepath.Rel(root, outside)
			if err != nil {
				t.Fatal(err)
			}
			m.Files[3].Path = filepath.ToSlash(relative)
			m.Files[3].SHA256 = digestOf("planted-outside-the-bundle")
			m.BinaryIdentities[2].Path = filepath.ToSlash(relative)
			m.BinaryIdentities[2].SHA256 = digestOf("planted-outside-the-bundle")
		})
	})

	if installed, reason := loadInstalledBundle(home, installedVersion); installed != nil || reason == "" {
		t.Fatalf("a manifest pointing outside the bundle was accepted (%v)", installed)
	}
}

// O_NOFOLLOW refuses a substituted link only at the last pathname component, so
// a parent directory replaced by a link to an external tree would otherwise be
// followed and its bytes verified as if they were the bundle's.
func TestSymlinkedParentDirectoryIsNotFollowed(t *testing.T) {
	elsewhere := t.TempDir()
	writeFile(t, filepath.Join(elsewhere, "bin", "node"), "installed-node-binary")
	home := installBundle(t, installedVersion, func(root string) {
		nodeDirectory := filepath.Join(root, "runtime", "node")
		if err := os.RemoveAll(nodeDirectory); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, nodeDirectory); err != nil {
			t.Fatal(err)
		}
	})

	observation := defaultPlanningObserver{home: home, version: installedVersion}.
		ObservePlanning(context.Background(), canonProfile("node"))

	if observation.Runtime.State == ComponentReady {
		t.Fatal("a component reached through a symlinked parent directory was reported ready")
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

// installBundle lays out what authorized setup leaves behind: the bundle files
// and a manifest that satisfies the bundle contract in full, because the reader
// under test hands the bytes to the contract's own decoder. The optional
// mutation runs afterwards, so a case can damage exactly one thing about an
// otherwise complete installation.
func installBundle(t *testing.T, version string, mutate func(root string)) string {
	t.Helper()
	home := t.TempDir()
	platform := releasebundle.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	root := filepath.Join(home, ".local", "share", "goalrail", "bundles", version, platform.Key())

	contents := map[string]string{
		goalrailEntry: "installed-gr-binary",
		runtimeEntry:  "installed-node-binary",
		// A file of the compiler package that sorts before its entrypoint. A
		// reader that picked a component's file by any rule of its own would
		// verify this one and never notice a replaced entrypoint.
		compilerOther: "installed-openspec-licence",
		compilerEntry: "installed-openspec-entrypoint",
	}
	for relative, body := range contents {
		writeFile(t, filepath.Join(root, filepath.FromSlash(relative)), body)
	}

	file := func(relative, component, mode string) releasebundle.ManifestFile {
		return releasebundle.ManifestFile{
			Path: relative, ComponentID: component,
			SizeBytes: int64(len(contents[relative])), SHA256: digestOf(contents[relative]), Mode: mode,
		}
	}
	component := func(id, name, kind, version string) releasebundle.ManifestComponent {
		return releasebundle.ManifestComponent{
			ID: id, Name: name, Kind: kind, Version: version,
			Integrity: digestOf(id), LicenseRef: "bundle:licenses/" + id, ProvenanceRef: "release:" + id,
			Dependencies: []releasebundle.ManifestDependency{},
		}
	}
	manifest := releasebundle.SetupBundleManifest{
		Schema:                releasebundle.SetupManifestSchemaV1,
		ReleaseVersion:        version,
		Platform:              platform,
		ArchiveName:           "goalrail-setup_" + version + "_" + platform.Key() + ".tar.gz",
		ManifestPath:          releasebundle.BundleManifestPath,
		CompilerLockDigest:    digestOf("compiler-lock"),
		CompilerInstallPolicy: "never-run-package-scripts",
		Components: []releasebundle.ManifestComponent{
			component("compiler-lock", "goalrail-private-planning-runtime", "dependency-lock", "1.6.0"),
			component("goalrail", "gr", "native-binary", version),
			component("node", "node", "private-runtime", "22.18.0"),
			component(compilerID, "@fission-ai/openspec", "npm-package", "1.6.0"),
		},
		BinaryIdentities: []releasebundle.BinaryIdentity{
			{ComponentID: "goalrail", Path: goalrailEntry, Kind: "go-buildinfo", Version: version,
				SHA256: digestOf(contents[goalrailEntry]), SourceIntegrity: digestOf("goalrail-source")},
			{ComponentID: compilerID, Path: compilerEntry, Kind: "npm-bin", Version: "1.6.0",
				SHA256: digestOf(contents[compilerEntry]), SourceIntegrity: digestOf("compiler-source")},
			{ComponentID: "node", Path: runtimeEntry, Kind: "node-distribution", Version: "22.18.0",
				SHA256: digestOf(contents[runtimeEntry]), SourceIntegrity: digestOf("node-source")},
		},
		Files: []releasebundle.ManifestFile{
			file(goalrailEntry, "goalrail", "0755"),
			file(compilerOther, compilerID, "0644"),
			file(compilerEntry, compilerID, "0755"),
			file(runtimeEntry, "node", "0755"),
		},
	}
	writeFile(t, filepath.Join(root, releasebundle.BundleManifestPath), canonicalManifest(t, manifest))
	if mutate != nil {
		mutate(root)
	}
	return home
}

// canonicalManifest renders the manifest the way the bundle contract requires it
// to have been written, because its decoder refuses anything else.
func canonicalManifest(t *testing.T, manifest releasebundle.SetupBundleManifest) string {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw) + "\n"
}

// mutateManifest rewrites one typed field of an otherwise complete manifest and
// renders it canonically again, so a refusal case fails for the reason it names
// rather than for the encoding a hand-edited document would have.
func mutateManifest(t *testing.T, root string, change func(*releasebundle.SetupBundleManifest)) {
	t.Helper()
	path := filepath.Join(root, releasebundle.BundleManifestPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest releasebundle.SetupBundleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	change(&manifest)
	writeFile(t, path, canonicalManifest(t, manifest))
}

func digestOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
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
