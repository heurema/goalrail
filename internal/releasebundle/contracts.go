package releasebundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	SourceLockSchemaV1    = "goalrail.setup-source-lock/v1"
	SetupManifestSchemaV1 = "goalrail.setup-bundle-manifest/v1"
	ReleaseMetadataSchema = "goalrail.release-metadata/v1"

	SourceLockPath      = "release/setup/source-lock.json"
	CompilerPackagePath = "release/setup/compiler/package.json"
	CompilerLockPath    = "release/setup/compiler/package-lock.json"
	CurrentMetadataName = "current-release.json"
	ChecksumsName       = "checksums.txt"
	BundleManifestPath  = "manifest.json"

	maxContractBytes  = 8 << 20
	maxReferenceBytes = 1024
)

var (
	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern = regexp.MustCompile(`^v?[0-9][0-9A-Za-z.+-]{0,63}$`)

	currentPlatforms = []Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}
)

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

func (platform Platform) Key() string {
	return platform.OS + "_" + platform.Arch
}

type SourceLock struct {
	Schema    string           `json:"schema"`
	Runtime   RuntimeSource    `json:"runtime"`
	Compiler  CompilerSource   `json:"compiler"`
	Platforms []PlatformSource `json:"platforms"`
}

type RuntimeSource struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	LicenseRef    string `json:"license_ref"`
	ProvenanceRef string `json:"provenance_ref"`
}

type CompilerSource struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	LockPath      string `json:"lock_path"`
	Entrypoint    string `json:"entrypoint"`
	LicenseRef    string `json:"license_ref"`
	ProvenanceRef string `json:"provenance_ref"`
}

type PlatformSource struct {
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	RuntimeArchive string `json:"runtime_archive"`
	RuntimeURL     string `json:"runtime_url"`
	RuntimeSHA256  string `json:"runtime_sha256"`
}

func (source PlatformSource) Platform() Platform {
	return Platform{OS: source.OS, Arch: source.Arch}
}

type SetupBundleManifest struct {
	Schema                string              `json:"schema"`
	ReleaseVersion        string              `json:"release_version"`
	Platform              Platform            `json:"platform"`
	ArchiveName           string              `json:"archive_name"`
	ManifestPath          string              `json:"manifest_path"`
	CompilerLockDigest    string              `json:"compiler_lock_digest"`
	CompilerInstallPolicy string              `json:"compiler_install_policy"`
	Components            []ManifestComponent `json:"components"`
	BinaryIdentities      []BinaryIdentity    `json:"binary_identities"`
	Files                 []ManifestFile      `json:"files"`
}

type ManifestComponent struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Kind             string               `json:"kind"`
	Version          string               `json:"version"`
	Integrity        string               `json:"integrity"`
	LicenseRef       string               `json:"license_ref"`
	ProvenanceRef    string               `json:"provenance_ref"`
	HasInstallScript bool                 `json:"has_install_script"`
	Dependencies     []ManifestDependency `json:"dependencies"`
}

type ManifestDependency struct {
	Name        string `json:"name"`
	Requested   string `json:"requested"`
	ComponentID string `json:"component_id"`
}

type BinaryIdentity struct {
	ComponentID     string `json:"component_id"`
	Path            string `json:"path"`
	Kind            string `json:"kind"`
	Version         string `json:"version"`
	SHA256          string `json:"sha256"`
	SourceIntegrity string `json:"source_integrity"`
}

type ManifestFile struct {
	Path        string `json:"path"`
	ComponentID string `json:"component_id"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	Mode        string `json:"mode"`
}

type ReleaseMetadata struct {
	Schema             string                    `json:"schema"`
	ReleaseVersion     string                    `json:"release_version"`
	ChecksumArtifact   string                    `json:"checksum_artifact"`
	Compatibility      ReleaseCompatibility      `json:"compatibility"`
	SupportedPlatforms []ReleasePlatformMetadata `json:"supported_platforms"`
}

type ReleaseCompatibility struct {
	GovernanceContract    string `json:"governance_contract"`
	SetupProfileSchema    string `json:"setup_profile_schema"`
	SetupManifestSchema   string `json:"setup_manifest_schema"`
	RuntimeID             string `json:"runtime_id"`
	RuntimeVersion        string `json:"runtime_version"`
	CompilerID            string `json:"compiler_id"`
	CompilerVersion       string `json:"compiler_version"`
	CompilerInstallPolicy string `json:"compiler_install_policy"`
}

type ReleasePlatformMetadata struct {
	Platform       Platform          `json:"platform"`
	GoalrailBinary BinaryIdentity    `json:"goalrail_binary"`
	MinimalArchive PublishedArtifact `json:"minimal_archive"`
	SetupArchive   PublishedArtifact `json:"setup_archive"`
	SetupManifest  PublishedArtifact `json:"setup_manifest"`
}

type PublishedArtifact struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// DecodeReleaseMetadata accepts only the canonical bounded contract published
// as current-release.json. Source-lock equality remains a release-side check;
// this decoder is the read-only consumer boundary used before installation.
func DecodeReleaseMetadata(reader io.Reader) (ReleaseMetadata, error) {
	metadata, err := decodeCanonicalContract[ReleaseMetadata](reader, "current release metadata")
	if err != nil {
		return ReleaseMetadata{}, err
	}
	if err := validateReleaseMetadataShape(metadata); err != nil {
		return ReleaseMetadata{}, err
	}
	return metadata, nil
}

// DecodeSetupBundleManifest accepts only a canonical bounded manifest whose
// internal component, file, and binary references agree structurally.
func DecodeSetupBundleManifest(reader io.Reader) (SetupBundleManifest, error) {
	manifest, err := decodeCanonicalContract[SetupBundleManifest](reader, "setup bundle manifest")
	if err != nil {
		return SetupBundleManifest{}, err
	}
	if err := validateSetupManifestShape(manifest); err != nil {
		return SetupBundleManifest{}, err
	}
	return manifest, nil
}

type npmLock struct {
	Name            string                `json:"name"`
	Version         string                `json:"version"`
	LockfileVersion int                   `json:"lockfileVersion"`
	Packages        map[string]npmPackage `json:"packages"`
}

type npmPackage struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Resolved             string            `json:"resolved"`
	Integrity            string            `json:"integrity"`
	License              string            `json:"license"`
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	PeerDependenciesMeta map[string]struct {
		Optional bool `json:"optional"`
	} `json:"peerDependenciesMeta"`
	OS               []string `json:"os"`
	CPU              []string `json:"cpu"`
	Dev              bool     `json:"dev"`
	DevOptional      bool     `json:"devOptional"`
	Optional         bool     `json:"optional"`
	HasInstallScript bool     `json:"hasInstallScript"`
	Link             bool     `json:"link"`
}

type compilerClosure struct {
	Raw         []byte
	Digest      string
	RootName    string
	RootVersion string
	Packages    []lockedPackage
	byPath      map[string]lockedPackage
}

type lockedPackage struct {
	Path             string
	Name             string
	Version          string
	Resolved         string
	Integrity        string
	License          string
	Dependencies     []ManifestDependency
	HasInstallScript bool
}

func decodeStrictJSON[T any](reader io.Reader, label string) (T, error) {
	var zero T
	raw, err := io.ReadAll(io.LimitReader(reader, maxContractBytes+1))
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", label, err)
	}
	if len(raw) > maxContractBytes {
		return zero, fmt.Errorf("read %s: exceeds %d bytes", label, maxContractBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s: %w", label, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected trailing JSON value")
		}
		return zero, fmt.Errorf("decode %s: %w", label, err)
	}
	return value, nil
}

func decodeCanonicalContract[T any](reader io.Reader, label string) (T, error) {
	var zero T
	raw, err := io.ReadAll(io.LimitReader(reader, maxContractBytes+1))
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", label, err)
	}
	if len(raw) > maxContractBytes {
		return zero, fmt.Errorf("read %s: exceeds %d bytes", label, maxContractBytes)
	}
	value, err := decodeStrictJSON[T](bytes.NewReader(raw), label)
	if err != nil {
		return zero, err
	}
	canonical, err := canonicalJSON(value)
	if err != nil {
		return zero, err
	}
	if !bytes.Equal(raw, canonical) {
		return zero, fmt.Errorf("%s is not canonical JSON", label)
	}
	return value, nil
}

func canonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return append(raw, '\n'), nil
}

func validateSourceLock(value SourceLock) error {
	if value.Schema != SourceLockSchemaV1 {
		return fmt.Errorf("source lock schema = %q, want %q", value.Schema, SourceLockSchemaV1)
	}
	if value.Runtime.ID != "node" || !validVersion(value.Runtime.Version) {
		return fmt.Errorf("runtime identity is not the pinned node version")
	}
	if !validReference(value.Runtime.LicenseRef) || !validReference(value.Runtime.ProvenanceRef) {
		return fmt.Errorf("runtime license or provenance reference is invalid")
	}
	if value.Compiler.ID != "@fission-ai/openspec" || !validVersion(value.Compiler.Version) {
		return fmt.Errorf("compiler identity is not the pinned OpenSpec package")
	}
	if value.Compiler.LockPath != CompilerLockPath {
		return fmt.Errorf("compiler lock path = %q, want %q", value.Compiler.LockPath, CompilerLockPath)
	}
	if !safeArchivePath(value.Compiler.Entrypoint) || !strings.HasPrefix(value.Compiler.Entrypoint, "node_modules/") {
		return fmt.Errorf("compiler entrypoint %q is unsafe", value.Compiler.Entrypoint)
	}
	if !validReference(value.Compiler.LicenseRef) || !validReference(value.Compiler.ProvenanceRef) {
		return fmt.Errorf("compiler license or provenance reference is invalid")
	}

	platforms := make([]Platform, 0, len(value.Platforms))
	seen := make(map[string]struct{}, len(value.Platforms))
	for _, source := range value.Platforms {
		platform := source.Platform()
		if platform.OS == "" || platform.Arch == "" {
			return fmt.Errorf("source lock has an empty platform")
		}
		if _, duplicate := seen[platform.Key()]; duplicate {
			return fmt.Errorf("source lock repeats platform %s", platform.Key())
		}
		seen[platform.Key()] = struct{}{}
		platforms = append(platforms, platform)
		if path.Base(source.RuntimeArchive) != source.RuntimeArchive || source.RuntimeArchive == "." {
			return fmt.Errorf("runtime archive %q is not a filename", source.RuntimeArchive)
		}
		parsed, err := url.Parse(source.RuntimeURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "nodejs.org" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("runtime URL %q is not a bounded nodejs.org HTTPS source", source.RuntimeURL)
		}
		if path.Base(parsed.Path) != source.RuntimeArchive {
			return fmt.Errorf("runtime URL and archive disagree for %s", platform.Key())
		}
		if !sha256Pattern.MatchString(source.RuntimeSHA256) {
			return fmt.Errorf("runtime SHA-256 is invalid for %s", platform.Key())
		}
	}
	sortPlatforms(platforms)
	if !equalPlatforms(platforms, currentPlatforms) {
		return fmt.Errorf("source lock platforms %v do not equal current release platforms %v", platforms, currentPlatforms)
	}
	return nil
}

func loadCompilerClosure(raw []byte, compiler CompilerSource) (compilerClosure, error) {
	if len(raw) == 0 || len(raw) > maxContractBytes {
		return compilerClosure{}, fmt.Errorf("compiler lock size is invalid")
	}
	var lock npmLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return compilerClosure{}, fmt.Errorf("decode compiler lock: %w", err)
	}
	if lock.LockfileVersion != 3 {
		return compilerClosure{}, fmt.Errorf("compiler lockfileVersion = %d, want 3", lock.LockfileVersion)
	}
	root, ok := lock.Packages[""]
	if !ok {
		return compilerClosure{}, fmt.Errorf("compiler lock has no root package")
	}
	if lock.Name == "" || lock.Version == "" || root.Name != lock.Name || root.Version != lock.Version {
		return compilerClosure{}, fmt.Errorf("compiler lock root identity is incomplete or inconsistent")
	}
	if len(root.Dependencies) != 1 || root.Dependencies[compiler.ID] != compiler.Version {
		return compilerClosure{}, fmt.Errorf("compiler lock root must pin only %s@%s", compiler.ID, compiler.Version)
	}
	if len(lock.Packages) < 2 || len(lock.Packages) > 1024 {
		return compilerClosure{}, fmt.Errorf("compiler lock package count is outside bounds")
	}

	byPath := make(map[string]lockedPackage, len(lock.Packages)-1)
	paths := make([]string, 0, len(lock.Packages)-1)
	for packagePath, value := range lock.Packages {
		if packagePath == "" {
			continue
		}
		if !safeLockPackagePath(packagePath) {
			return compilerClosure{}, fmt.Errorf("compiler package path %q is unsafe", packagePath)
		}
		if value.Version == "" || len(value.Version) > 128 {
			return compilerClosure{}, fmt.Errorf("compiler package %s has no bounded exact version", packagePath)
		}
		if value.Dev || value.DevOptional || value.Optional || value.Link || len(value.OS) != 0 || len(value.CPU) != 0 || len(value.OptionalDependencies) != 0 {
			return compilerClosure{}, fmt.Errorf("compiler package %s is dev, optional, linked, or platform-dependent", packagePath)
		}
		for peer := range value.PeerDependencies {
			if !value.PeerDependenciesMeta[peer].Optional {
				return compilerClosure{}, fmt.Errorf("compiler package %s has unpinned required peer dependency %s", packagePath, peer)
			}
		}
		if err := validateNPMSource(value.Resolved, value.Integrity); err != nil {
			return compilerClosure{}, fmt.Errorf("compiler package %s: %w", packagePath, err)
		}
		if value.License == "" || len(value.License) > 128 {
			return compilerClosure{}, fmt.Errorf("compiler package %s has no bounded license identity", packagePath)
		}
		name, err := packageNameFromLockPath(packagePath)
		if err != nil {
			return compilerClosure{}, err
		}
		if value.Name != "" && value.Name != name {
			return compilerClosure{}, fmt.Errorf("compiler package path %s names %q", packagePath, value.Name)
		}
		byPath[packagePath] = lockedPackage{
			Path: packagePath, Name: name, Version: value.Version, Resolved: value.Resolved,
			Integrity: value.Integrity, License: value.License, HasInstallScript: value.HasInstallScript,
		}
		paths = append(paths, packagePath)
	}

	for packagePath, value := range lock.Packages {
		if packagePath == "" {
			continue
		}
		current := byPath[packagePath]
		names := make([]string, 0, len(value.Dependencies))
		for name := range value.Dependencies {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			target, ok := resolveLockedDependency(packagePath, name, byPath)
			if !ok {
				return compilerClosure{}, fmt.Errorf("compiler package %s dependency %s is not in the exact lock tree", packagePath, name)
			}
			current.Dependencies = append(current.Dependencies, ManifestDependency{
				Name: name, Requested: value.Dependencies[name], ComponentID: npmComponentID(target.Path),
			})
		}
		byPath[packagePath] = current
	}

	sort.Strings(paths)
	packages := make([]lockedPackage, 0, len(paths))
	for _, packagePath := range paths {
		packages = append(packages, byPath[packagePath])
	}
	rootPath := "node_modules/" + compiler.ID
	rootPackage, ok := byPath[rootPath]
	if !ok || rootPackage.Version != compiler.Version {
		return compilerClosure{}, fmt.Errorf("compiler lock does not contain exact root %s@%s", compiler.ID, compiler.Version)
	}
	if !strings.Contains(compiler.ProvenanceRef, rootPackage.Integrity) {
		return compilerClosure{}, fmt.Errorf("compiler provenance does not bind the root package integrity")
	}
	return compilerClosure{
		Raw: append([]byte(nil), raw...), Digest: digestBytes(raw),
		RootName: lock.Name, RootVersion: lock.Version, Packages: packages, byPath: byPath,
	}, nil
}

func resolveLockedDependency(packagePath, dependency string, packages map[string]lockedPackage) (lockedPackage, bool) {
	current := packagePath
	for {
		candidate := current + "/node_modules/" + dependency
		if target, ok := packages[candidate]; ok {
			return target, true
		}
		index := strings.LastIndex(current, "/node_modules/")
		if index < 0 {
			break
		}
		current = current[:index]
	}
	target, ok := packages["node_modules/"+dependency]
	return target, ok
}

func packageNameFromLockPath(packagePath string) (string, error) {
	index := strings.LastIndex(packagePath, "node_modules/")
	if index < 0 {
		return "", fmt.Errorf("compiler package path %q has no node_modules segment", packagePath)
	}
	remainder := packagePath[index+len("node_modules/"):]
	parts := strings.Split(remainder, "/")
	if len(parts) == 1 && parts[0] != "" && !strings.HasPrefix(parts[0], "@") {
		return parts[0], nil
	}
	if len(parts) == 2 && strings.HasPrefix(parts[0], "@") && parts[1] != "" {
		return parts[0] + "/" + parts[1], nil
	}
	return "", fmt.Errorf("compiler package path %q has an invalid package name", packagePath)
}

func safeLockPackagePath(value string) bool {
	if !strings.HasPrefix(value, "node_modules/") || !safeArchivePath(value) {
		return false
	}
	segments := strings.Split(value, "/")
	for index, segment := range segments {
		if segment == "node_modules" {
			if index+1 >= len(segments) || segments[index+1] == "node_modules" || segments[index+1] == "" {
				return false
			}
		}
	}
	return true
}

func validateNPMSource(resolved, integrity string) error {
	parsed, err := url.Parse(resolved)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "registry.npmjs.org" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("resolved source %q is not bounded registry.npmjs.org HTTPS", resolved)
	}
	algorithm, encoded, ok := strings.Cut(integrity, "-")
	if !ok || algorithm != "sha512" {
		return fmt.Errorf("integrity %q is not an exact SHA-512 SRI", integrity)
	}
	digest, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(digest) != sha512Size {
		return fmt.Errorf("integrity %q is not a complete SHA-512 SRI", integrity)
	}
	return nil
}

const sha512Size = 64

func validVersion(value string) bool {
	return versionPattern.MatchString(value)
}

func validReference(value string) bool {
	return value != "" && len(value) <= maxReferenceBytes && !strings.ContainsAny(value, "\r\n\x00")
}

func safeArchivePath(value string) bool {
	return value != "" && value == path.Clean(value) && value != "." && !strings.HasPrefix(value, "/") && !strings.Contains(value, "\\") && !strings.HasPrefix(value, "../") && value != ".."
}

func sortPlatforms(platforms []Platform) {
	sort.Slice(platforms, func(i, j int) bool {
		if platforms[i].OS == platforms[j].OS {
			return platforms[i].Arch < platforms[j].Arch
		}
		return platforms[i].OS < platforms[j].OS
	})
}

func equalPlatforms(first, second []Platform) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func npmComponentID(packagePath string) string {
	return "npm:" + packagePath
}
