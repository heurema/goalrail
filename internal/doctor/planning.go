package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/releasebundle"
)

type ComponentState string

const (
	ComponentReady               ComponentState = "ready"
	ComponentMissing             ComponentState = "missing"
	ComponentIncompatible        ComponentState = "incompatible"
	ComponentUnverifiedIntegrity ComponentState = "unverified_integrity"
	ComponentInvalid             ComponentState = "invalid"
	ComponentNotRequired         ComponentState = "not_required"
)

// maxInstalledManifest bounds the installed bundle manifest. It describes a
// bundle rather than containing one, so a file beyond this size is not a
// manifest this reader should try to understand.
const maxInstalledManifest = 4 << 20

// maxInstalledComponentFile bounds one inspected component file. The private
// runtime is the largest thing the bundle carries and is well inside this.
const maxInstalledComponentFile = int64(512 << 20)

type ComponentReadiness struct {
	Kind              string              `json:"kind"`
	ID                string              `json:"id"`
	RequiredVersion   string              `json:"required_version,omitempty"`
	RequiredIntegrity domain.SHA256Digest `json:"required_integrity,omitempty"`
	ObservedVersion   string              `json:"observed_version,omitempty"`
	Path              string              `json:"path,omitempty"`
	State             ComponentState      `json:"state"`
	Detail            string              `json:"detail,omitempty"`
}

type PlanningObservation struct {
	Bundle   ComponentReadiness `json:"bundle"`
	Runtime  ComponentReadiness `json:"runtime"`
	Compiler ComponentReadiness `json:"compiler"`
}

type PlanningObserver interface {
	ObservePlanning(context.Context, domain.SetupProfile) PlanningObservation
}

type PlanningObserverFunc func(context.Context, domain.SetupProfile) PlanningObservation

func (function PlanningObserverFunc) ObservePlanning(ctx context.Context, profile domain.SetupProfile) PlanningObservation {
	return function(ctx, profile)
}

// defaultPlanningObserver answers whether the declared planning toolchain is
// available and compatible by inspecting the bundle authorized setup installed.
//
// It runs nothing. A promoted requirement forbids initialization, update, and
// diagnosis from executing the runtime, a package runner, or the stock planning
// CLI, and asking a program for its version is executing it. Reading the
// installed manifest answers availability, version, and integrity at once, and
// answers them about the installation the declared profile pins rather than
// about whatever the executable lookup path happens to reach first.
//
// It also takes no path from repository content. The profile states which
// components are required and at which versions; the manifest states where they
// live. A profile value is therefore matched against component identity and
// never resolved, so a repository that writes a path where a program name
// belongs selects nothing at all.
type defaultPlanningObserver struct {
	// home is where authorized setup installs. An empty value resolves no
	// bundle rather than falling back to another root.
	home string

	// version is this binary's own identity, which names the bundle it was
	// installed from. It is a field rather than a direct read of the package
	// variable for the reason the version resolver already documents: a test
	// binary reports the development marker in every state, so no test could
	// otherwise exercise the released shapes this observer is written for.
	version string
}

func (observer defaultPlanningObserver) ObservePlanning(_ context.Context, profile domain.SetupProfile) PlanningObservation {
	observation := PlanningObservation{
		Bundle: ComponentReadiness{
			Kind: "goalrail_bundle", ID: "goalrail", RequiredVersion: profile.CompatibleGoalrailBundle,
			ObservedVersion: observer.version,
		},
	}
	if compatibleGoalrailBundle(observer.version, profile.CompatibleGoalrailBundle) {
		observation.Bundle.State = ComponentReady
	} else if strings.TrimSpace(observer.version) == "" || observer.version == "unknown" || observer.version == "(devel)" {
		observation.Bundle.State = ComponentUnverifiedIntegrity
		observation.Bundle.Detail = "this development binary has no release identity compatible with the declared setup bundle"
	} else {
		observation.Bundle.State = ComponentIncompatible
		observation.Bundle.Detail = "the running Goalrail release is outside the declared compatible bundle range"
	}

	installed, reason := loadInstalledBundle(observer.home, observer.version)
	observation.Runtime = observeRuntime(installed, reason, profile)
	observation.Compiler = observeCompiler(installed, reason, profile)
	return observation
}

// installedBundle is the record authorized setup left behind: where it
// installed, and what it installed there.
type installedBundle struct {
	root     string
	manifest releasebundle.SetupBundleManifest
}

// loadInstalledBundle resolves and reads the bundle this binary was installed
// from. The empty result carries the reason, which becomes the component detail
// so a report never says only that something is missing.
//
// The location is derived rather than recorded because a release refuses to
// publish an artifact whose stamped version is not its tag, so a binary that
// came from a bundle reports exactly that bundle's release version. That leaves
// no need for a pointer file, and the alternatives are closed: the stable
// executable is a byte copy that carries no link back, and enumerating sibling
// bundle directories would have to guess which one installed this process.
func loadInstalledBundle(home, version string) (*installedBundle, string) {
	if strings.TrimSpace(home) == "" {
		return nil, "no home directory was resolved, so no installed Goalrail bundle could be located"
	}
	if !harness.IsReleaseVersion(version) {
		return nil, fmt.Sprintf("this build reports %q, which is not a release version, so it corresponds to no installed Goalrail bundle", version)
	}
	platform := releasebundle.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	root := filepath.Join(home, ".local", "share", "goalrail", "bundles", version, platform.Key())
	raw, err := boundedio.ReadRegularFile(
		filepath.Join(root, releasebundle.BundleManifestPath), "installed bundle manifest", maxInstalledManifest)
	if err != nil {
		return nil, "the installed bundle manifest could not be read; run the exact Goalrail setup plan"
	}
	var manifest releasebundle.SetupBundleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, "the installed bundle manifest is not valid JSON"
	}
	switch {
	case manifest.Schema != releasebundle.SetupManifestSchemaV1:
		return nil, "the installed bundle manifest carries an unsupported schema"
	case manifest.ReleaseVersion != version:
		return nil, "the installed bundle manifest names a different release than this binary"
	case manifest.Platform != platform:
		return nil, "the installed bundle manifest names a different platform than this binary"
	}
	return &installedBundle{root: root, manifest: manifest}, ""
}

// observeRuntime reports the declared runtime by matching it against the
// installed bundle's component identity, then verifying the exact file that
// component's binary identity names.
func observeRuntime(installed *installedBundle, reason string, profile domain.SetupProfile) ComponentReadiness {
	component := ComponentReadiness{
		Kind: "runtime", ID: profile.Planning.Runtime, RequiredVersion: profile.Planning.RuntimeVersion,
	}
	if strings.TrimSpace(profile.Planning.Runtime) == "" {
		component.State = ComponentNotRequired
		return component
	}
	if installed == nil {
		component.State = ComponentMissing
		component.Detail = reason
		return component
	}
	identity, found := binaryIdentityFor(installed.manifest, profile.Planning.Runtime)
	if !found {
		component.State = ComponentMissing
		component.Detail = "the installed bundle carries no runtime component with this declared identity"
		return component
	}
	component.ObservedVersion = identity.Version
	component.RequiredIntegrity = domain.SHA256Digest(normalizeDigest(identity.SHA256))
	return verifyInstalledFile(component, installed.root, identity.Path, identity.SHA256, profile.Planning.RuntimeVersion)
}

// observeCompiler reports the declared compiler the same way. Its component is
// named rather than binary-identified, because the planning compiler is a
// package rather than an executable, so the verified file is the one the
// manifest records for it.
func observeCompiler(installed *installedBundle, reason string, profile domain.SetupProfile) ComponentReadiness {
	component := ComponentReadiness{
		Kind: "compiler", ID: profile.Planning.Compiler, RequiredVersion: profile.Planning.CompilerVersion,
	}
	if strings.TrimSpace(profile.Planning.Compiler) == "" {
		component.State = ComponentNotRequired
		return component
	}
	if installed == nil {
		component.State = ComponentMissing
		component.Detail = reason
		return component
	}
	named, found := componentNamed(installed.manifest, profile.Planning.Compiler)
	if !found {
		component.State = ComponentMissing
		component.Detail = "the installed bundle carries no compiler component with this declared identity"
		return component
	}
	component.ObservedVersion = named.Version
	file, found := entrypointFor(installed.manifest, named.ID)
	if !found {
		component.State = ComponentMissing
		component.Detail = "the installed bundle records no file for this compiler component"
		return component
	}
	component.RequiredIntegrity = domain.SHA256Digest(normalizeDigest(file.SHA256))
	return verifyInstalledFile(component, installed.root, file.Path, file.SHA256, profile.Planning.CompilerVersion)
}

// verifyInstalledFile settles a component's state from the bytes on disk.
//
// The version comparison comes first because an installation of the wrong
// version is incompatible whether or not its bytes are intact, and reporting a
// digest mismatch there would name the wrong problem.
func verifyInstalledFile(component ComponentReadiness, root, relative, expected, required string) ComponentReadiness {
	component.Path = filepath.Join(root, filepath.FromSlash(relative))
	if component.ObservedVersion != strings.TrimPrefix(required, "v") &&
		component.ObservedVersion != required {
		component.State = ComponentIncompatible
		component.Detail = "the installed component's version differs from the exact declared version"
		return component
	}
	observed, _, err := boundedio.DigestRegularFile(component.Path, "installed component", maxInstalledComponentFile)
	if err != nil {
		component.State = ComponentMissing
		component.Detail = "the file the installed manifest records for this component could not be read"
		return component
	}
	if observed != normalizeDigest(expected) {
		component.State = ComponentUnverifiedIntegrity
		component.Detail = "the installed component's bytes do not match the digest its manifest records"
		return component
	}
	component.State = ComponentReady
	return component
}

// binaryIdentityFor finds the manifest's binary identity for a declared
// component. The declared value is matched against component identity and is
// never joined onto a path: the path comes from the manifest.
func binaryIdentityFor(manifest releasebundle.SetupBundleManifest, declared string) (releasebundle.BinaryIdentity, bool) {
	for _, identity := range manifest.BinaryIdentities {
		if identity.ComponentID == declared {
			return identity, true
		}
	}
	return releasebundle.BinaryIdentity{}, false
}

func componentNamed(manifest releasebundle.SetupBundleManifest, declared string) (releasebundle.ManifestComponent, bool) {
	for _, component := range manifest.Components {
		if component.Name == declared || component.ID == declared {
			return component, true
		}
	}
	return releasebundle.ManifestComponent{}, false
}

// entrypointFor selects one file to verify for a component that has no binary
// identity. The bundle's whole tree was verified against this manifest when it
// was installed and that walk is not repeated here; this states which single
// record the diagnosis stands behind.
func entrypointFor(manifest releasebundle.SetupBundleManifest, componentID string) (releasebundle.ManifestFile, bool) {
	var selected releasebundle.ManifestFile
	found := false
	for _, file := range manifest.Files {
		if file.ComponentID != componentID {
			continue
		}
		if !found || file.Path < selected.Path {
			selected, found = file, true
		}
	}
	return selected, found
}

// normalizeDigest accepts both spellings the bundle contracts use — a bare
// lowercase hex sum and one prefixed with its algorithm — so a comparison never
// fails on notation.
func normalizeDigest(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
}

func compatibleGoalrailBundle(version, constraint string) bool {
	if constraint != ">=0.2.0 <0.3.0" {
		return false
	}
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	return majorErr == nil && minorErr == nil && major == 0 && minor == 2
}
