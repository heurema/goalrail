package doctor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

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
	defer installed.close()
	observation.Runtime = observeComponent(installed, reason, "runtime", profile.Planning.Runtime, profile.Planning.RuntimeVersion)
	observation.Compiler = observeComponent(installed, reason, "compiler", profile.Planning.Compiler, profile.Planning.CompilerVersion)
	return observation
}

// installedBundle is the record authorized setup left behind: where it
// installed, and what it installed there.
//
// The location is held open as a confined root rather than as a string. Opening
// with O_NOFOLLOW refuses a substituted symbolic link only at the final pathname
// component, so a parent directory replaced by a link to an external tree would
// be followed before any per-file protection applied, and a component could be
// verified against bytes outside the bundle entirely. Every file this reads is
// therefore opened through the root, which cannot traverse out of it.
type installedBundle struct {
	root     *os.Root
	path     string
	manifest releasebundle.SetupBundleManifest
}

func (bundle *installedBundle) close() {
	if bundle != nil && bundle.root != nil {
		_ = bundle.root.Close()
	}
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
	relative := strings.Join([]string{".local", "share", "goalrail", "bundles", version, platform.Key()}, "/")
	path := filepath.Join(home, filepath.FromSlash(relative))
	// Descending from the home directory rather than opening the bundle path
	// outright is what confines the ancestors. Opening the bundle path directly
	// resolves it the ordinary way first, so a bundle directory — or any
	// directory above it — replaced by a link to an external tree would become
	// the confined root itself, and every later read would be confined to
	// somewhere else entirely.
	homeRoot, err := os.OpenRoot(home)
	if err != nil {
		return nil, "no installed Goalrail bundle corresponds to this binary; run the exact Goalrail setup plan"
	}
	defer homeRoot.Close()
	root, err := homeRoot.OpenRoot(relative)
	if err != nil {
		return nil, "no installed Goalrail bundle corresponds to this binary; run the exact Goalrail setup plan"
	}
	raw, err := readInRoot(root, releasebundle.BundleManifestPath, "installed bundle manifest", maxInstalledManifest)
	if err != nil {
		root.Close()
		return nil, "the installed bundle manifest could not be read; run the exact Goalrail setup plan"
	}
	// The bundle contracts own what a manifest must be, and that decoder checks
	// far more than this reader could restate: canonical encoding, safe relative
	// paths, digest shapes, sorted and unique records, and every binary identity
	// binding an exact file. Accepting a manifest on a schema check alone would
	// let an edited one name a path that leaves the bundle and still be trusted
	// to say which bytes are authentic.
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(raw))
	if err != nil {
		root.Close()
		return nil, "the installed bundle manifest is not a valid bundle manifest"
	}
	switch {
	case manifest.ReleaseVersion != version:
		root.Close()
		return nil, "the installed bundle manifest names a different release than this binary"
	case manifest.Platform != platform:
		root.Close()
		return nil, "the installed bundle manifest names a different platform than this binary"
	}
	return &installedBundle{root: root, path: path, manifest: manifest}, ""
}

// readInRoot reads a bundle file through the confined root, applying the same
// descriptor discipline the rest of the codebase uses for resolved artifacts.
func readInRoot(root *os.Root, name, label string, limit int) ([]byte, error) {
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, _, err := boundedio.ReadOpenFile(file, label, limit)
	return raw, err
}

// observeComponent reports one declared planning component by matching it
// against the installed bundle's own records.
//
// The declared value is matched against component identity and is never joined
// onto a path: the path comes from the manifest's binary identity for that
// component, which the bundle contract requires to bind an exact file.
//
// The component's own version and its entrypoint's version must agree. The
// manifest decoder binds an identity to a file but does not require the two
// version fields to say the same thing, so an edited manifest could state a
// component at one version and its entrypoint at the version a profile
// demands, and a reader that preferred either one would report ready on a
// description that contradicts itself. Disagreement is the manifest being
// invalid, not the component being absent or stale, so it is reported as such.
func observeComponent(installed *installedBundle, reason, kind, declared, required string) ComponentReadiness {
	component := ComponentReadiness{Kind: kind, ID: declared, RequiredVersion: required}
	if strings.TrimSpace(declared) == "" {
		component.State = ComponentNotRequired
		return component
	}
	if installed == nil {
		component.State = ComponentMissing
		component.Detail = reason
		return component
	}
	named, reason := componentFor(installed.manifest, declared)
	if reason != "" {
		component.State = ComponentMissing
		component.Detail = reason
		return component
	}
	identity, found := binaryIdentityFor(installed.manifest, named.ID)
	if !found {
		component.State = ComponentMissing
		component.Detail = "the installed bundle records no entrypoint for this " + kind + " component"
		return component
	}
	if named.Version != identity.Version {
		component.State = ComponentInvalid
		component.Detail = "the installed bundle manifest gives this component and its entrypoint different versions"
		return component
	}
	component.ObservedVersion = named.Version
	// The manifest's own spelling is kept, prefix included: this field sits
	// beside other digest fields in one report, and a bare sum would fail the
	// domain's digest contract while looking like every other one.
	component.RequiredIntegrity = domain.SHA256Digest(identity.SHA256)
	record, found := fileRecordFor(installed.manifest, identity.Path)
	if !found {
		component.State = ComponentInvalid
		component.Detail = "the installed bundle manifest records no file for this component entrypoint"
		return component
	}
	return verifyInstalledFile(component, installed, record, required)
}

// verifyInstalledFile settles a component's state from the bytes on disk.
//
// The version comparison comes first because an installation of the wrong
// version is incompatible whether or not its bytes are intact, and reporting a
// digest mismatch there would name the wrong problem.
func verifyInstalledFile(component ComponentReadiness, installed *installedBundle, record releasebundle.ManifestFile, required string) ComponentReadiness {
	component.Path = filepath.Join(installed.path, filepath.FromSlash(record.Path))
	if component.ObservedVersion != strings.TrimPrefix(required, "v") &&
		component.ObservedVersion != required {
		component.State = ComponentIncompatible
		component.Detail = "the installed component's version differs from the exact declared version"
		return component
	}
	file, err := installed.root.OpenFile(record.Path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		component.State = ComponentMissing
		component.Detail = "the file the installed manifest records for this component could not be read inside the bundle"
		return component
	}
	defer file.Close()
	// The manifest records a mode as well as a digest, and only one of them
	// says whether the file can still be run. A runtime whose bytes are intact
	// but whose executable bit was cleared is not ready by any reading a user
	// would recognise, and nothing here executes it to find out.
	info, err := file.Stat()
	if err != nil {
		component.State = ComponentMissing
		component.Detail = "the file the installed manifest records for this component could not be read"
		return component
	}
	if observedMode := fmt.Sprintf("%04o", info.Mode().Perm()); observedMode != record.Mode {
		component.State = ComponentUnverifiedIntegrity
		component.Detail = "the installed component's mode is " + observedMode + ", not the " + record.Mode + " its manifest records"
		return component
	}
	observed, _, err := boundedio.DigestOpenFile(file, "installed component", maxInstalledComponentFile)
	if err != nil {
		component.State = ComponentMissing
		component.Detail = "the file the installed manifest records for this component could not be read"
		return component
	}
	if observed != normalizeDigest(record.SHA256) {
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

// componentFor resolves a declared component against the manifest. The empty
// reason means it was found.
//
// An exact identifier wins outright, and a name is consulted only when no
// identifier matches. The bundle contract requires component identifiers to be
// unique and sorted but says nothing about names, so an accepted manifest may
// carry two components sharing one name — and a scan that took the first match
// in either field would resolve the declared runtime to whichever component
// sorted earlier, then report the entrypoint of something else. An independent
// review found exactly that after this lookup was shared between the runtime
// and the compiler; the runtime had previously matched an identifier alone.
//
// A name that matches more than one component is ambiguous rather than
// resolvable, and saying so is the only honest answer available.
func componentFor(manifest releasebundle.SetupBundleManifest, declared string) (releasebundle.ManifestComponent, string) {
	for _, component := range manifest.Components {
		if component.ID == declared {
			return component, ""
		}
	}
	var selected releasebundle.ManifestComponent
	matches := 0
	for _, component := range manifest.Components {
		if component.Name == declared {
			selected, matches = component, matches+1
		}
	}
	switch matches {
	case 1:
		return selected, ""
	case 0:
		return releasebundle.ManifestComponent{}, "the installed bundle carries no component with this declared identity"
	default:
		return releasebundle.ManifestComponent{}, "the installed bundle carries more than one component under this declared name"
	}
}

// fileRecordFor returns the manifest's record for one installed path. The bundle
// contract requires every binary identity to bind an exact file, so the absence
// of one is the manifest contradicting itself rather than an ordinary miss.
func fileRecordFor(manifest releasebundle.SetupBundleManifest, path string) (releasebundle.ManifestFile, bool) {
	for _, file := range manifest.Files {
		if file.Path == path {
			return file, true
		}
	}
	return releasebundle.ManifestFile{}, false
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
