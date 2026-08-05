// Package setup inspects a managed checkout and local machine state without
// writing either, then freezes the exact setup work into the canonical domain
// plan consumed by later authorization and apply milestones.
package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/project"
	"github.com/heurema/goalrail/internal/releasebundle"
)

const (
	DefaultReleaseChannel = "https://github.com/heurema/goalrail/releases"
	maxExistingBundleFile = int64(512 << 20)
)

// ReleaseEvidence is the non-executable metadata an outer release-channel
// adapter obtained. Generate never performs network I/O and never accepts an
// archive payload at this planning boundary.
type ReleaseEvidence struct {
	MetadataRaw []byte
	ManifestRaw []byte
}

// PlanOptions makes every source of machine variance explicit for tests while
// preserving useful current-machine defaults for a later CLI adapter.
type PlanOptions struct {
	RepositoryRoot string
	Home           string
	OS             string
	Arch           string
	Scaffold       ambient.Scaffold
	ReleaseChannel string
	Evidence       *ReleaseEvidence
}

// Issue carries human-readable diagnostics outside the canonical plan. Only
// the stable reason ID is admitted to the authorization-bound artifact.
type Issue struct {
	ReasonID string
	Detail   string
}

// Result contains the normalized plan value, its canonical bytes/digest, and
// any noncanonical inspection detail.
type Result struct {
	Plan     domain.SetupPlan
	Artifact domain.CanonicalArtifact
	Issues   []Issue
}

type planner struct {
	home        string
	repository  string
	platform    releasebundle.Platform
	scaffold    ambient.Scaffold
	channel     string
	profile     domain.SetupProfile
	manifest    releasebundle.SetupBundleManifest
	manifestRaw []byte
	entry       releasebundle.ReleasePlatformMetadata
	plan        domain.SetupPlan
	issues      []Issue
}

// Generate performs only bounded reads and canonicalization. Missing release
// evidence and unsupported machine state produce an incomplete, non-executable
// plan rather than an inferred component or mutation.
func Generate(ctx context.Context, options PlanOptions) (Result, error) {
	normalized, err := normalizeOptions(options)
	if err != nil {
		return Result{}, err
	}

	inspection, err := project.Inspect(ctx, normalized.RepositoryRoot)
	if err != nil {
		return Result{}, fmt.Errorf("inspect project declaration: %w", err)
	}
	if inspection.State != project.ClaimManaged {
		return Result{}, fmt.Errorf("setup planning requires a managed project claim; state=%s reason=%s", inspection.State, inspection.Reason)
	}
	artifacts, err := project.InspectGoverningArtifacts(inspection)
	if err != nil {
		return Result{}, fmt.Errorf("inspect governing artifacts: %w", err)
	}
	if artifacts.SetupProfile.State != project.ArtifactCurrent {
		return Result{}, fmt.Errorf("setup profile is not current: state=%s detail=%s", artifacts.SetupProfile.State, artifacts.SetupProfile.Detail)
	}
	profileArtifact, err := domain.FreezeSetupProfile(artifacts.SetupValue)
	if err != nil {
		return Result{}, fmt.Errorf("freeze setup profile: %w", err)
	}

	p := &planner{
		home:       normalized.Home,
		repository: inspection.WorktreeRoot,
		platform:   releasebundle.Platform{OS: normalized.OS, Arch: normalized.Arch},
		scaffold:   normalized.Scaffold,
		channel:    normalized.ReleaseChannel,
		profile:    artifacts.SetupValue,
		plan: domain.SetupPlan{
			Schema:             domain.SetupPlanSchemaV1,
			ProjectID:          inspection.Declaration.ProjectID,
			DeclarationDigest:  inspection.DeclarationDigest,
			SetupProfileDigest: profileArtifact.Digest(),
			Platform:           normalized.OS + "-" + normalized.Arch,
			State:              domain.SetupPlanIncomplete,
			ProjectCodeWrites:  0,
		},
	}
	p.addNetwork("release-metadata", metadataURL(p.channel), "release_metadata")
	p.addPrerequisite("release-metadata", domain.SetupPrerequisitePlatform, "current", false, "release:current-release-metadata")

	if normalized.Evidence == nil || len(normalized.Evidence.MetadataRaw) == 0 {
		p.addIssue("RELEASE_METADATA_UNAVAILABLE", "current release metadata was not provided to the read-only planner")
		return p.finish()
	}
	metadata, err := releasebundle.DecodeReleaseMetadata(bytes.NewReader(normalized.Evidence.MetadataRaw))
	if err != nil {
		p.addIssue("RELEASE_METADATA_INVALID", err.Error())
		return p.finish()
	}
	if !compatibleRelease(p.profile, metadata) {
		p.addIssue("RELEASE_INCOMPATIBLE", "release metadata does not satisfy the committed setup profile")
		return p.finish()
	}
	p.plan.Prerequisites[0].Satisfied = true
	p.plan.Prerequisites[0].VersionConstraint = metadata.ReleaseVersion
	p.plan.Prerequisites[0].EvidenceRef = "release:" + metadata.ReleaseVersion + "/metadata"

	entry, ok := platformEntry(metadata, p.platform)
	if !ok {
		p.addPrerequisite("platform", domain.SetupPrerequisitePlatform, p.platform.Key(), false, "release:"+metadata.ReleaseVersion+"/platforms")
		p.addIssue("PLATFORM_UNSUPPORTED", "release does not publish a setup bundle for "+p.platform.Key())
		return p.finish()
	}
	p.entry = entry
	p.addPrerequisite("platform", domain.SetupPrerequisitePlatform, p.platform.Key(), true, "release:"+metadata.ReleaseVersion+"/platform/"+p.platform.Key())

	immutableBase := strings.TrimSuffix(p.channel, "/") + "/download/" + metadata.ReleaseVersion + "/"
	p.addNetwork("release-checksums", immutableBase+metadata.ChecksumArtifact, "release_checksums")
	p.addNetwork("setup-manifest", immutableBase+entry.SetupManifest.Name, "setup_manifest")
	p.addNetwork("setup-archive", immutableBase+entry.SetupArchive.Name, "setup_archive")
	if normalized.Evidence == nil || len(normalized.Evidence.ManifestRaw) == 0 {
		p.addIssue("SETUP_MANIFEST_UNAVAILABLE", "platform setup manifest was not provided to the read-only planner")
		return p.finish()
	}
	if digestBytes(normalized.Evidence.ManifestRaw) != domain.SHA256Digest(entry.SetupManifest.SHA256) {
		p.addIssue("SETUP_MANIFEST_DIGEST_MISMATCH", "setup manifest bytes do not match release metadata")
		return p.finish()
	}
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(normalized.Evidence.ManifestRaw))
	if err != nil {
		p.addIssue("SETUP_MANIFEST_INVALID", err.Error())
		return p.finish()
	}
	if !manifestMatchesRelease(manifest, metadata, entry) {
		p.addIssue("SETUP_MANIFEST_RELEASE_MISMATCH", "setup manifest identity disagrees with release metadata")
		return p.finish()
	}
	p.manifest = manifest
	p.manifestRaw = append([]byte(nil), normalized.Evidence.ManifestRaw...)

	if _, ok := scaffoldPin(p.profile, p.scaffold); !ok {
		p.addPrerequisite("scaffold-adapter", domain.SetupPrerequisiteTrust, string(p.scaffold), false, "profile:scaffold-adapters")
		p.addIssue("SCAFFOLD_UNSUPPORTED", "committed setup profile does not pin scaffold "+string(p.scaffold))
		return p.finish()
	}
	p.addPrerequisite("runtime", domain.SetupPrerequisiteRuntime, metadata.Compatibility.RuntimeVersion, true, "manifest:"+metadata.ReleaseVersion+"/runtime/"+metadata.Compatibility.RuntimeID)
	p.addPrerequisite("compiler", domain.SetupPrerequisiteRuntime, metadata.Compatibility.CompilerVersion, true, "manifest:"+metadata.ReleaseVersion+"/compiler/openspec")
	p.addPrerequisite("scaffold-adapter", domain.SetupPrerequisiteTrust, string(p.scaffold), true, "profile:scaffold/"+string(p.scaffold))

	versionRoot := filepath.Join(p.home, ".local", "share", "goalrail", "bundles", metadata.ReleaseVersion, p.platform.Key())
	stableExecutable := filepath.Join(p.home, ".local", "bin", "gr")
	components, componentErr := componentsFromManifest(manifest, versionRoot)
	if componentErr != nil {
		p.addIssue("SETUP_MANIFEST_COMPONENT_INVALID", componentErr.Error())
		return p.finish()
	}
	p.plan.Components = components

	filesystemOK := p.inspectBundleDestination(versionRoot, entry.SetupArchive)
	filesystemOK = p.inspectExecutableDestination(stableExecutable, manifest, entry.GoalrailBinary) && filesystemOK
	filesystemOK = p.inspectScaffoldDestination(stableExecutable) && filesystemOK
	p.addPrerequisite("filesystem", domain.SetupPrerequisiteFilesystem, "safe-regular-owned-paths", filesystemOK, "environment:setup-destinations")
	if !filesystemOK {
		return p.finish()
	}

	if ambient.TrustEvidenceOf(p.scaffold) == ambient.TrustGateObserved {
		p.plan.TrustSteps = append(p.plan.TrustSteps, domain.SetupTrustStep{
			ID: "provider-trust", AdapterID: string(p.scaffold),
			ActionRef: "scaffold:" + string(p.scaffold) + "/hooks-trust", Interactive: true,
		})
	}
	p.plan.Verification = []domain.SetupVerification{
		{ID: "doctor", Argv: []string{stableExecutable, "doctor", "--repo", p.repository, "--scaffold", string(p.scaffold), "--json"}},
		{ID: "version", Argv: []string{stableExecutable, "version"}},
	}
	p.plan.State = domain.SetupPlanComplete
	return p.finish()
}

func normalizeOptions(options PlanOptions) (PlanOptions, error) {
	if strings.TrimSpace(options.RepositoryRoot) == "" {
		options.RepositoryRoot = "."
	}
	repository, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return PlanOptions{}, fmt.Errorf("resolve repository root: %w", err)
	}
	options.RepositoryRoot = filepath.Clean(repository)
	if strings.TrimSpace(options.Home) == "" {
		options.Home, err = os.UserHomeDir()
		if err != nil {
			return PlanOptions{}, fmt.Errorf("resolve user home: %w", err)
		}
	}
	home, err := filepath.Abs(options.Home)
	if err != nil {
		return PlanOptions{}, fmt.Errorf("resolve user home: %w", err)
	}
	options.Home = filepath.Clean(home)
	if options.Home == string(filepath.Separator) {
		return PlanOptions{}, errors.New("setup planning refuses the filesystem root as user home")
	}
	if options.OS == "" {
		options.OS = runtime.GOOS
	}
	if options.Arch == "" {
		options.Arch = runtime.GOARCH
	}
	if options.Scaffold == "" {
		options.Scaffold = ambient.ScaffoldCodex
	}
	if options.ReleaseChannel == "" {
		options.ReleaseChannel = DefaultReleaseChannel
	}
	options.ReleaseChannel = strings.TrimSuffix(options.ReleaseChannel, "/")
	parsed, err := url.Parse(options.ReleaseChannel)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return PlanOptions{}, errors.New("release channel must be a credential-free HTTPS URL without query or fragment")
	}
	return options, nil
}

func (p *planner) finish() (Result, error) {
	if len(p.issues) != 0 {
		p.plan.State = domain.SetupPlanIncomplete
		p.plan.Mutations = nil
		p.plan.Rollback = nil
		p.plan.Verification = nil
		p.plan.IncompleteReasonIDs = make([]string, 0, len(p.issues))
		for _, issue := range p.issues {
			p.plan.IncompleteReasonIDs = append(p.plan.IncompleteReasonIDs, issue.ReasonID)
		}
	}
	artifact, err := domain.FreezeSetupPlan(p.plan)
	if err != nil {
		return Result{}, fmt.Errorf("freeze setup plan: %w", err)
	}
	normalized, err := domain.DecodeSetupPlan(bytes.NewReader(artifact.CanonicalJSON()))
	if err != nil {
		return Result{}, fmt.Errorf("re-open canonical setup plan: %w", err)
	}
	return Result{Plan: normalized, Artifact: artifact, Issues: append([]Issue(nil), p.issues...)}, nil
}

func (p *planner) addIssue(reasonID, detail string) {
	for _, issue := range p.issues {
		if issue.ReasonID == reasonID {
			return
		}
	}
	p.issues = append(p.issues, Issue{ReasonID: reasonID, Detail: detail})
}

func (p *planner) addPrerequisite(id string, kind domain.SetupPrerequisiteKind, constraint string, satisfied bool, evidence string) {
	p.plan.Prerequisites = append(p.plan.Prerequisites, domain.SetupPrerequisite{
		ID: id, Kind: kind, VersionConstraint: constraint, Satisfied: satisfied, EvidenceRef: evidence,
	})
}

func (p *planner) addNetwork(id, rawURL, purpose string) {
	p.plan.NetworkAccess = append(p.plan.NetworkAccess, domain.SetupNetworkAccess{ID: id, Method: "GET", URL: rawURL, PurposeID: purpose})
}

func (p *planner) addMutation(id string, kind domain.SetupMutationKind, scope domain.SetupScope, path string, before *domain.SHA256Digest, desired domain.SHA256Digest) {
	p.plan.Mutations = append(p.plan.Mutations, domain.SetupMutation{
		ID: id, Kind: kind, Scope: scope, Path: path, ExpectedBeforeDigest: before, DesiredDigest: desired,
	})
	action := "remove_owned_path"
	if before != nil {
		action = "restore_prior_file"
	}
	p.plan.Rollback = append(p.plan.Rollback, domain.SetupRollbackAction{
		MutationID: id, Action: action, Target: path, PriorStateDigest: before,
	})
}

func (p *planner) inspectBundleDestination(root string, archive releasebundle.PublishedArtifact) bool {
	info, exists, err := inspectSafePath(p.home, root)
	if err != nil {
		p.addIssue("BUNDLE_DESTINATION_UNSAFE", err.Error())
		return false
	}
	if !exists {
		p.addMutation("install-bundle", domain.SetupMutationInstallFile, domain.SetupScopeUserLocal, root, nil, domain.SHA256Digest(archive.SHA256))
		return true
	}
	if !info.IsDir() {
		p.addIssue("BUNDLE_DESTINATION_CONFLICT", root+" exists and is not a directory")
		return false
	}
	if err := verifyInstalledBundle(root, p.manifest, p.manifestRaw); err != nil {
		p.addIssue("BUNDLE_DESTINATION_CONFLICT", err.Error())
		return false
	}
	return true
}

func (p *planner) inspectExecutableDestination(path string, manifest releasebundle.SetupBundleManifest, identity releasebundle.BinaryIdentity) bool {
	info, exists, err := inspectSafePath(p.home, path)
	if err != nil {
		p.addIssue("EXECUTABLE_DESTINATION_UNSAFE", err.Error())
		return false
	}
	desired := domain.SHA256Digest(identity.SHA256)
	if !exists {
		p.addMutation("select-executable", domain.SetupMutationSelectExecutable, domain.SetupScopeUserLocal, path, nil, desired)
		return true
	}
	if !info.Mode().IsRegular() {
		p.addIssue("EXECUTABLE_DESTINATION_CONFLICT", path+" is not a regular file")
		return false
	}
	file, ok := manifestFile(manifest, identity.Path)
	if !ok || info.Size() != file.SizeBytes || info.Size() > maxExistingBundleFile {
		p.addIssue("EXECUTABLE_DESTINATION_CONFLICT", path+" has an unbounded or unexpected size")
		return false
	}
	expectedMode, err := strconv.ParseUint(file.Mode, 8, 32)
	if err != nil {
		p.addIssue("EXECUTABLE_DESTINATION_CONFLICT", path+" has an invalid planned mode")
		return false
	}
	actual, err := digestFile(path)
	if err != nil {
		p.addIssue("EXECUTABLE_DESTINATION_UNREADABLE", err.Error())
		return false
	}
	if actual != desired {
		p.addIssue("EXECUTABLE_DESTINATION_CONFLICT", path+" contains different executable bytes that cannot be recovered")
		return false
	}
	if info.Mode().Perm() == fs.FileMode(expectedMode) {
		return true
	}
	p.addMutation("select-executable", domain.SetupMutationSelectExecutable, domain.SetupScopeUserLocal, path, digestPointer(actual), desired)
	return true
}

func (p *planner) inspectScaffoldDestination(executable string) bool {
	target, err := ambient.RegistrationTarget(p.scaffold, p.home, p.repository)
	if err != nil {
		p.addIssue("SCAFFOLD_TARGET_UNAVAILABLE", err.Error())
		return false
	}
	allowedRoot := p.home
	scope := domain.SetupScopeUserLocal
	planPath := target.Path
	if target.Scope == ambient.ScopeRepository {
		allowedRoot = p.repository
		scope = domain.SetupScopeRepository
		relative, relErr := filepath.Rel(p.repository, target.Path)
		if relErr != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			p.addIssue("SCAFFOLD_TARGET_UNSAFE", "repository scaffold target escapes the worktree")
			return false
		}
		planPath = filepath.ToSlash(relative)
	}
	if _, _, err := inspectSafePath(allowedRoot, target.Path); err != nil {
		p.addIssue("SCAFFOLD_TARGET_UNSAFE", err.Error())
		return false
	}
	connection, err := ambient.PlanRegistration(target, executable)
	if err != nil {
		p.addIssue("SCAFFOLD_CONFIG_INVALID", err.Error())
		return false
	}
	if connection.AlreadyPresent {
		return true
	}
	preview, err := ambient.PreviewConnection(connection)
	if err != nil {
		p.addIssue("SCAFFOLD_CONFIG_INVALID", err.Error())
		return false
	}
	var before *domain.SHA256Digest
	if preview.BeforeExists {
		before = digestPointer(digestBytes(preview.Before))
	}
	p.addMutation("install-hook", domain.SetupMutationInstallHook, scope, planPath, before, digestBytes(preview.After))
	return true
}

func compatibleRelease(profile domain.SetupProfile, metadata releasebundle.ReleaseMetadata) bool {
	c := metadata.Compatibility
	return c.GovernanceContract == "goalrail-governance-v1" &&
		c.SetupProfileSchema == domain.SetupProfileSchemaV1 &&
		c.SetupManifestSchema == releasebundle.SetupManifestSchemaV1 &&
		c.RuntimeID == profile.Planning.Runtime &&
		c.RuntimeVersion == profile.Planning.RuntimeVersion &&
		c.CompilerID == profile.Planning.Compiler &&
		c.CompilerVersion == profile.Planning.CompilerVersion &&
		c.CompilerInstallPolicy == "never-run-package-scripts" &&
		versionSatisfies(metadata.ReleaseVersion, profile.CompatibleGoalrailBundle)
}

func versionSatisfies(version, constraint string) bool {
	current, ok := parseTriplet(strings.TrimPrefix(version, "v"))
	if !ok {
		return false
	}
	fields := strings.Fields(constraint)
	if len(fields) != 2 || !strings.HasPrefix(fields[0], ">=") || !strings.HasPrefix(fields[1], "<") {
		return false
	}
	lower, lowerOK := parseTriplet(strings.TrimPrefix(fields[0], ">="))
	upper, upperOK := parseTriplet(strings.TrimPrefix(fields[1], "<"))
	return lowerOK && upperOK && compareTriplet(current, lower) >= 0 && compareTriplet(current, upper) < 0
}

func parseTriplet(value string) ([3]int, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var parsed [3]int
	for index, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return [3]int{}, false
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return [3]int{}, false
		}
		parsed[index] = number
	}
	return parsed, true
}

func compareTriplet(left, right [3]int) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func platformEntry(metadata releasebundle.ReleaseMetadata, platform releasebundle.Platform) (releasebundle.ReleasePlatformMetadata, bool) {
	for _, entry := range metadata.SupportedPlatforms {
		if entry.Platform == platform {
			return entry, true
		}
	}
	return releasebundle.ReleasePlatformMetadata{}, false
}

func manifestMatchesRelease(manifest releasebundle.SetupBundleManifest, metadata releasebundle.ReleaseMetadata, entry releasebundle.ReleasePlatformMetadata) bool {
	if manifest.ReleaseVersion != metadata.ReleaseVersion || manifest.Platform != entry.Platform || manifest.ArchiveName != entry.SetupArchive.Name || manifest.CompilerInstallPolicy != metadata.Compatibility.CompilerInstallPolicy {
		return false
	}
	goalrail, ok := binaryIdentity(manifest, "bin/gr")
	if !ok || goalrail != entry.GoalrailBinary {
		return false
	}
	runtimeIdentity, runtimeOK := binaryIdentity(manifest, "runtime/node/bin/node")
	if !runtimeOK || runtimeIdentity.ComponentID != metadata.Compatibility.RuntimeID || runtimeIdentity.Version != metadata.Compatibility.RuntimeVersion {
		return false
	}
	for _, component := range manifest.Components {
		if component.Name == metadata.Compatibility.CompilerID && component.Version == metadata.Compatibility.CompilerVersion {
			return true
		}
	}
	return false
}

func scaffoldPin(profile domain.SetupProfile, scaffold ambient.Scaffold) (domain.SetupAdapterPin, bool) {
	for _, pin := range profile.ScaffoldAdapters {
		if pin.ID == string(scaffold) {
			return pin, true
		}
	}
	return domain.SetupAdapterPin{}, false
}

type componentIdentity struct {
	Component releasebundle.ManifestComponent `json:"component"`
	Files     []releasebundle.ManifestFile    `json:"files"`
}

func componentsFromManifest(manifest releasebundle.SetupBundleManifest, destination string) ([]domain.SetupComponent, error) {
	filesByComponent := make(map[string][]releasebundle.ManifestFile, len(manifest.Components))
	for _, file := range manifest.Files {
		filesByComponent[file.ComponentID] = append(filesByComponent[file.ComponentID], file)
	}
	components := make([]domain.SetupComponent, 0, len(manifest.Components))
	for _, component := range manifest.Components {
		files := filesByComponent[component.ID]
		var size uint64
		for _, file := range files {
			if file.SizeBytes < 0 {
				return nil, fmt.Errorf("component %s has a negative file size", component.ID)
			}
			size += uint64(file.SizeBytes)
		}
		if size == 0 {
			return nil, fmt.Errorf("component %s has no non-empty installed files", component.ID)
		}
		raw, err := json.Marshal(componentIdentity{Component: component, Files: files})
		if err != nil {
			return nil, err
		}
		raw = append(raw, '\n')
		license := component.LicenseRef
		if !domain.IsEvidenceReference(license) {
			license = "manifest:" + manifest.ReleaseVersion + "/" + component.ID + "/license"
		}
		provenance := component.ProvenanceRef
		if !domain.IsEvidenceReference(provenance) {
			provenance = "manifest:" + manifest.ReleaseVersion + "/" + component.ID + "/provenance"
		}
		components = append(components, domain.SetupComponent{
			ID: component.ID, Version: component.Version,
			SourceRef: "release:" + manifest.ReleaseVersion + "/" + manifest.Platform.Key() + "/manifest@" + component.ID,
			Integrity: digestBytes(raw), SizeBytes: size, Destination: destination, Scope: domain.SetupScopeUserLocal,
			LicenseRef: license, ProvenanceRef: provenance,
		})
	}
	return components, nil
}

func inspectSafePath(allowedRoot, target string) (fs.FileInfo, bool, error) {
	root := filepath.Clean(allowedRoot)
	path := filepath.Clean(target)
	if !filepath.IsAbs(root) || !filepath.IsAbs(path) || !pathWithin(root, path) || path == root {
		return nil, false, fmt.Errorf("target %q is outside its allowed non-root scope", path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, false, err
	}
	current := root
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, false, fmt.Errorf("inspect allowed root %s: %w", root, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, false, fmt.Errorf("allowed root %s is not a real directory", root)
	}
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) {
			parent := filepath.Dir(current)
			parentInfo, parentErr := os.Lstat(parent)
			if parentErr != nil || !parentInfo.IsDir() || parentInfo.Mode()&0o222 == 0 {
				return nil, false, fmt.Errorf("nearest existing parent %s is not a writable directory", parent)
			}
			return nil, false, nil
		}
		if statErr != nil {
			return nil, false, fmt.Errorf("inspect %s: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, false, fmt.Errorf("target path contains symlink %s", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, false, fmt.Errorf("target ancestor %s is not a directory", current)
		}
		if index == len(parts)-1 {
			return info, true, nil
		}
	}
	return nil, false, errors.New("target path inspection reached no terminal component")
}

func verifyInstalledBundle(root string, manifest releasebundle.SetupBundleManifest, manifestRaw []byte) error {
	expected := make(map[string]releasebundle.ManifestFile, len(manifest.Files)+1)
	expectedDirectories := map[string]struct{}{".": {}}
	for _, file := range manifest.Files {
		expected[file.Path] = file
		for directory := filepath.ToSlash(filepath.Dir(file.Path)); directory != "."; directory = filepath.ToSlash(filepath.Dir(directory)) {
			expectedDirectories[directory] = struct{}{}
		}
	}
	expected[manifest.ManifestPath] = releasebundle.ManifestFile{
		Path: manifest.ManifestPath, SizeBytes: int64(len(manifestRaw)), SHA256: string(digestBytes(manifestRaw)), Mode: "0644",
	}
	seen := make(map[string]struct{}, len(expected))
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("installed bundle contains symlink %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if _, ok := expectedDirectories[relative]; !ok {
				return fmt.Errorf("installed bundle contains unexpected directory %s", relative)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("installed bundle contains non-regular path %s", path)
		}
		want, ok := expected[relative]
		if !ok {
			return fmt.Errorf("installed bundle contains unexpected file %s", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() != want.SizeBytes || info.Size() > maxExistingBundleFile || fmt.Sprintf("%04o", info.Mode().Perm()) != want.Mode {
			return fmt.Errorf("installed bundle file %s size or mode differs", relative)
		}
		digest, err := digestFile(path)
		if err != nil {
			return err
		}
		if digest != domain.SHA256Digest(want.SHA256) {
			return fmt.Errorf("installed bundle file %s digest differs", relative)
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("installed bundle has %d verified files, want %d", len(seen), len(expected))
	}
	return nil
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func manifestFile(manifest releasebundle.SetupBundleManifest, path string) (releasebundle.ManifestFile, bool) {
	for _, file := range manifest.Files {
		if file.Path == path {
			return file, true
		}
	}
	return releasebundle.ManifestFile{}, false
}

func binaryIdentity(manifest releasebundle.SetupBundleManifest, path string) (releasebundle.BinaryIdentity, bool) {
	for _, identity := range manifest.BinaryIdentities {
		if identity.Path == path {
			return identity, true
		}
	}
	return releasebundle.BinaryIdentity{}, false
}

func digestFile(path string) (domain.SHA256Digest, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() || before.Size() > maxExistingBundleFile {
		return "", fmt.Errorf("digest target %s is not a bounded regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return "", err
	}
	afterOpen, err := os.Lstat(path)
	if err != nil || !afterOpen.Mode().IsRegular() || !os.SameFile(before, opened) || !os.SameFile(opened, afterOpen) {
		return "", fmt.Errorf("digest target %s changed before it was read", path)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	afterRead, err := os.Lstat(path)
	if err != nil || !afterRead.Mode().IsRegular() || !os.SameFile(opened, afterRead) {
		return "", fmt.Errorf("digest target %s changed while it was read", path)
	}
	return domain.SHA256Digest("sha256:" + hex.EncodeToString(hash.Sum(nil))), nil
}

func digestBytes(raw []byte) domain.SHA256Digest {
	return domain.DigestCanonicalJSON(raw)
}

func digestPointer(value domain.SHA256Digest) *domain.SHA256Digest {
	copy := value
	return &copy
}

func metadataURL(channel string) string {
	return strings.TrimSuffix(channel, "/") + "/latest/download/" + releasebundle.CurrentMetadataName
}
