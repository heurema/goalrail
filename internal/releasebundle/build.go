package releasebundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

const releaseVersionSymbol = "github.com/heurema/goalrail/internal/harness.releaseVersion"

type BuildOptions struct {
	RepoRoot       string
	DistDir        string
	SourceCacheDir string
	ReleaseVersion string
	FetchSource    SourceFetcher

	readGoVersion func(string) (string, error)
}

type releaseInputs struct {
	sourceLock     SourceLock
	packageJSONRaw []byte
	closure        compilerClosure
	licenseRaw     []byte
}

type compilerPackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Private      bool              `json:"private"`
	Dependencies map[string]string `json:"dependencies"`
}

type LockSummary struct {
	Schema             string     `json:"schema"`
	Runtime            string     `json:"runtime"`
	Compiler           string     `json:"compiler"`
	PackageCount       int        `json:"package_count"`
	InstallScriptCount int        `json:"install_script_count"`
	InstallPolicy      string     `json:"install_policy"`
	CompilerLockDigest string     `json:"compiler_lock_digest"`
	SupportedPlatforms []Platform `json:"supported_platforms"`
}

func CheckSourceLock(repoRoot string) (LockSummary, error) {
	resolvedRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return LockSummary{}, fmt.Errorf("resolve repository root: %w", err)
	}
	inputs, err := loadReleaseInputs(resolvedRepo)
	if err != nil {
		return LockSummary{}, err
	}
	platforms := make([]Platform, 0, len(inputs.sourceLock.Platforms))
	installScripts := 0
	for _, source := range inputs.sourceLock.Platforms {
		platforms = append(platforms, source.Platform())
	}
	for _, packageEntry := range inputs.closure.Packages {
		if packageEntry.HasInstallScript {
			installScripts++
		}
	}
	sortPlatforms(platforms)
	return LockSummary{
		Schema:             SourceLockSchemaV1,
		Runtime:            inputs.sourceLock.Runtime.ID + "@" + inputs.sourceLock.Runtime.Version,
		Compiler:           inputs.sourceLock.Compiler.ID + "@" + inputs.sourceLock.Compiler.Version,
		PackageCount:       len(inputs.closure.Packages),
		InstallScriptCount: installScripts,
		InstallPolicy:      "never-run-package-scripts",
		CompilerLockDigest: inputs.closure.Digest,
		SupportedPlatforms: platforms,
	}, nil
}

func Build(ctx context.Context, options BuildOptions) (ReleaseMetadata, error) {
	paths, err := validateBuildPaths(options.RepoRoot, options.DistDir, options.SourceCacheDir)
	if err != nil {
		return ReleaseMetadata{}, err
	}
	if !validVersion(options.ReleaseVersion) || !strings.HasPrefix(options.ReleaseVersion, "v") {
		return ReleaseMetadata{}, fmt.Errorf("release version %q is not a bounded v-prefixed version", options.ReleaseVersion)
	}
	inputs, err := loadReleaseInputs(paths.repoRoot)
	if err != nil {
		return ReleaseMetadata{}, err
	}
	if err := os.MkdirAll(paths.distDir, 0o755); err != nil {
		return ReleaseMetadata{}, fmt.Errorf("create distribution directory: %w", err)
	}
	if err := os.MkdirAll(paths.cacheDir, 0o755); err != nil {
		return ReleaseMetadata{}, fmt.Errorf("create source cache: %w", err)
	}
	fetch := options.FetchSource
	if fetch == nil {
		return ReleaseMetadata{}, fmt.Errorf("release build requires an explicit bounded source fetcher")
	}
	versionReader := options.readGoVersion
	if versionReader == nil {
		versionReader = readGoBinaryVersion
	}

	temporaryRoot, err := os.MkdirTemp("", "goalrail-release-build-*")
	if err != nil {
		return ReleaseMetadata{}, fmt.Errorf("create release build directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	compilerRoot := filepath.Join(temporaryRoot, "compiler")
	compilerFiles, err := compilerBasePayload(compilerRoot, inputs)
	if err != nil {
		return ReleaseMetadata{}, err
	}
	extractedCompiler, err := cacheCompilerSources(ctx, paths.cacheDir, compilerRoot, inputs.closure, fetch)
	if err != nil {
		return ReleaseMetadata{}, err
	}
	compilerFiles = append(compilerFiles, extractedCompiler...)
	sortPayloadFiles(compilerFiles)
	if err := verifyCompilerEntrypoint(compilerFiles, inputs.sourceLock.Compiler, inputs.closure); err != nil {
		return ReleaseMetadata{}, err
	}

	platformMetadata := make([]ReleasePlatformMetadata, 0, len(inputs.sourceLock.Platforms))
	for _, platformSource := range inputs.sourceLock.Platforms {
		platform := platformSource.Platform()
		platformRoot := filepath.Join(temporaryRoot, platform.Key())
		if err := os.MkdirAll(platformRoot, 0o755); err != nil {
			return ReleaseMetadata{}, fmt.Errorf("create platform staging directory: %w", err)
		}

		binaryPath := filepath.Join(paths.distDir, platform.Key(), "gr")
		binaryVersion, err := versionReader(binaryPath)
		if err != nil {
			return ReleaseMetadata{}, fmt.Errorf("read %s Goalrail binary identity: %w", platform.Key(), err)
		}
		if binaryVersion != options.ReleaseVersion {
			return ReleaseMetadata{}, fmt.Errorf("%s Goalrail binary version = %q, want %q", platform.Key(), binaryVersion, options.ReleaseVersion)
		}
		goalrailPayload, err := payloadFromExisting(binaryPath, "bin/gr", "goalrail", 0o755)
		if err != nil {
			return ReleaseMetadata{}, fmt.Errorf("stage %s Goalrail binary: %w", platform.Key(), err)
		}
		licensePayload, err := payloadFromBytes(platformRoot, "licenses/goalrail-LICENSE", "goalrail", 0o644, inputs.licenseRaw)
		if err != nil {
			return ReleaseMetadata{}, fmt.Errorf("stage Goalrail license: %w", err)
		}
		runtimeFiles, err := cacheRuntimeSource(ctx, paths.cacheDir, platformRoot, platformSource, inputs.sourceLock.Runtime, fetch)
		if err != nil {
			return ReleaseMetadata{}, err
		}

		payload := make([]payloadFile, 0, len(compilerFiles)+len(runtimeFiles)+2)
		payload = append(payload, compilerFiles...)
		payload = append(payload, goalrailPayload, licensePayload)
		payload = append(payload, runtimeFiles...)
		sortPayloadFiles(payload)
		if err := validateUniquePayload(payload); err != nil {
			return ReleaseMetadata{}, err
		}

		manifest := buildManifest(options.ReleaseVersion, platformSource, inputs, payload)
		if err := validateSetupManifest(manifest, inputs); err != nil {
			return ReleaseMetadata{}, fmt.Errorf("validate generated %s manifest: %w", platform.Key(), err)
		}
		manifestRaw, err := canonicalJSON(manifest)
		if err != nil {
			return ReleaseMetadata{}, err
		}
		manifestName := setupManifestName(options.ReleaseVersion, platform)
		manifestPath := filepath.Join(paths.distDir, manifestName)
		if err := writeNewArtifact(manifestPath, manifestRaw, 0o644); err != nil {
			return ReleaseMetadata{}, err
		}
		archiveName := setupArchiveName(options.ReleaseVersion, platform)
		archivePath := filepath.Join(paths.distDir, archiveName)
		if err := writeSetupArchive(archivePath, payload, manifestRaw); err != nil {
			return ReleaseMetadata{}, fmt.Errorf("write %s: %w", archiveName, err)
		}

		minimalName := minimalArchiveName(options.ReleaseVersion, platform)
		minimalArtifact, err := inspectPublishedArtifact(filepath.Join(paths.distDir, minimalName), minimalName)
		if err != nil {
			return ReleaseMetadata{}, fmt.Errorf("inspect minimal archive for %s: %w", platform.Key(), err)
		}
		setupArtifact, err := inspectPublishedArtifact(archivePath, archiveName)
		if err != nil {
			return ReleaseMetadata{}, err
		}
		manifestArtifact, err := inspectPublishedArtifact(manifestPath, manifestName)
		if err != nil {
			return ReleaseMetadata{}, err
		}
		platformMetadata = append(platformMetadata, ReleasePlatformMetadata{
			Platform: platform,
			GoalrailBinary: BinaryIdentity{
				ComponentID: "goalrail", Path: "bin/gr", Kind: "go-buildinfo", Version: options.ReleaseVersion,
				SHA256: goalrailPayload.record.SHA256, SourceIntegrity: goalrailPayload.record.SHA256,
			},
			MinimalArchive: minimalArtifact,
			SetupArchive:   setupArtifact,
			SetupManifest:  manifestArtifact,
		})
	}
	sort.Slice(platformMetadata, func(i, j int) bool { return platformMetadata[i].Platform.Key() < platformMetadata[j].Platform.Key() })

	metadata := ReleaseMetadata{
		Schema:           ReleaseMetadataSchema,
		ReleaseVersion:   options.ReleaseVersion,
		ChecksumArtifact: ChecksumsName,
		Compatibility: ReleaseCompatibility{
			GovernanceContract:    "goalrail-governance-v1",
			SetupProfileSchema:    "goalrail.setup-profile/v1",
			SetupManifestSchema:   SetupManifestSchemaV1,
			RuntimeID:             inputs.sourceLock.Runtime.ID,
			RuntimeVersion:        inputs.sourceLock.Runtime.Version,
			CompilerID:            inputs.sourceLock.Compiler.ID,
			CompilerVersion:       inputs.sourceLock.Compiler.Version,
			CompilerInstallPolicy: "never-run-package-scripts",
		},
		SupportedPlatforms: platformMetadata,
	}
	metadataRaw, err := canonicalJSON(metadata)
	if err != nil {
		return ReleaseMetadata{}, err
	}
	if err := writeNewArtifact(filepath.Join(paths.distDir, CurrentMetadataName), metadataRaw, 0o644); err != nil {
		return ReleaseMetadata{}, err
	}
	if err := writeChecksums(paths.distDir, metadata); err != nil {
		return ReleaseMetadata{}, err
	}
	return metadata, nil
}

type resolvedBuildPaths struct {
	repoRoot string
	distDir  string
	cacheDir string
}

func validateBuildPaths(repoRoot, distDir, cacheDir string) (resolvedBuildPaths, error) {
	resolvedRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return resolvedBuildPaths{}, fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedDist, err := filepath.Abs(distDir)
	if err != nil {
		return resolvedBuildPaths{}, fmt.Errorf("resolve distribution directory: %w", err)
	}
	resolvedCache, err := filepath.Abs(cacheDir)
	if err != nil {
		return resolvedBuildPaths{}, fmt.Errorf("resolve source cache: %w", err)
	}
	for label, candidate := range map[string]string{"distribution directory": resolvedDist, "source cache": resolvedCache} {
		if candidate == resolvedRepo || strings.HasPrefix(candidate, resolvedRepo+string(filepath.Separator)) {
			return resolvedBuildPaths{}, fmt.Errorf("%s must stay outside the repository checkout", label)
		}
	}
	return resolvedBuildPaths{repoRoot: resolvedRepo, distDir: resolvedDist, cacheDir: resolvedCache}, nil
}

func loadReleaseInputs(repoRoot string) (releaseInputs, error) {
	sourceRaw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(SourceLockPath)))
	if err != nil {
		return releaseInputs{}, fmt.Errorf("read setup source lock: %w", err)
	}
	source, err := decodeStrictJSON[SourceLock](strings.NewReader(string(sourceRaw)), "setup source lock")
	if err != nil {
		return releaseInputs{}, err
	}
	if err := validateSourceLock(source); err != nil {
		return releaseInputs{}, err
	}
	packageRaw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(CompilerPackagePath)))
	if err != nil {
		return releaseInputs{}, fmt.Errorf("read compiler package root: %w", err)
	}
	var packageRoot compilerPackageJSON
	if err := json.Unmarshal(packageRaw, &packageRoot); err != nil {
		return releaseInputs{}, fmt.Errorf("decode compiler package root: %w", err)
	}
	if !packageRoot.Private || len(packageRoot.Dependencies) != 1 || packageRoot.Dependencies[source.Compiler.ID] != source.Compiler.Version {
		return releaseInputs{}, fmt.Errorf("compiler package root must be private and pin only %s@%s", source.Compiler.ID, source.Compiler.Version)
	}
	lockRaw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(source.Compiler.LockPath)))
	if err != nil {
		return releaseInputs{}, fmt.Errorf("read compiler lock: %w", err)
	}
	closure, err := loadCompilerClosure(lockRaw, source.Compiler)
	if err != nil {
		return releaseInputs{}, err
	}
	if packageRoot.Name != closure.RootName || packageRoot.Version != closure.RootVersion {
		return releaseInputs{}, fmt.Errorf("compiler package root and lock root identities disagree")
	}
	licenseRaw, err := os.ReadFile(filepath.Join(repoRoot, "LICENSE"))
	if err != nil {
		return releaseInputs{}, fmt.Errorf("read Goalrail license: %w", err)
	}
	return releaseInputs{sourceLock: source, packageJSONRaw: packageRaw, closure: closure, licenseRaw: licenseRaw}, nil
}

func compilerBasePayload(root string, inputs releaseInputs) ([]payloadFile, error) {
	packagePayload, err := payloadFromBytes(root, "compiler/package.json", "compiler-lock", 0o644, inputs.packageJSONRaw)
	if err != nil {
		return nil, fmt.Errorf("stage compiler package root: %w", err)
	}
	lockPayload, err := payloadFromBytes(root, "compiler/package-lock.json", "compiler-lock", 0o644, inputs.closure.Raw)
	if err != nil {
		return nil, fmt.Errorf("stage compiler lock: %w", err)
	}
	return []payloadFile{lockPayload, packagePayload}, nil
}

func payloadFromExisting(sourcePath, bundlePath, componentID string, mode int64) (payloadFile, error) {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return payloadFile{}, err
	}
	if !info.Mode().IsRegular() {
		return payloadFile{}, fmt.Errorf("source is not a regular file")
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return payloadFile{}, err
	}
	defer file.Close()
	hash := newSHA256Writer()
	written, err := io.Copy(hash, file)
	if err != nil {
		return payloadFile{}, err
	}
	if written != info.Size() {
		return payloadFile{}, fmt.Errorf("read %d bytes, want %d", written, info.Size())
	}
	return payloadFile{
		record: ManifestFile{
			Path: bundlePath, ComponentID: componentID, SizeBytes: info.Size(),
			SHA256: hash.Digest(), Mode: fmt.Sprintf("%04o", mode),
		},
		sourcePath: sourcePath,
	}, nil
}

type sha256Writer struct {
	hash hash.Hash
}

func newSHA256Writer() *sha256Writer {
	return &sha256Writer{hash: sha256.New()}
}

func (writer *sha256Writer) Write(raw []byte) (int, error) {
	return writer.hash.Write(raw)
}

func (writer *sha256Writer) Digest() string {
	return "sha256:" + hex.EncodeToString(writer.hash.Sum(nil))
}

func buildManifest(releaseVersion string, platformSource PlatformSource, inputs releaseInputs, payload []payloadFile) SetupBundleManifest {
	byPath := make(map[string]ManifestFile, len(payload))
	files := make([]ManifestFile, 0, len(payload))
	for _, entry := range payload {
		byPath[entry.record.Path] = entry.record
		files = append(files, entry.record)
	}
	components := buildManifestComponents(releaseVersion, platformSource, inputs, byPath["bin/gr"].SHA256)
	rootPackage := inputs.closure.byPath["node_modules/"+inputs.sourceLock.Compiler.ID]
	binaries := []BinaryIdentity{
		{
			ComponentID: "goalrail", Path: "bin/gr", Kind: "go-buildinfo", Version: releaseVersion,
			SHA256: byPath["bin/gr"].SHA256, SourceIntegrity: byPath["bin/gr"].SHA256,
		},
		{
			ComponentID: "node", Path: "runtime/node/bin/node", Kind: "node-distribution", Version: inputs.sourceLock.Runtime.Version,
			SHA256: byPath["runtime/node/bin/node"].SHA256, SourceIntegrity: "sha256:" + platformSource.RuntimeSHA256,
		},
		{
			ComponentID: npmComponentID(rootPackage.Path), Path: "compiler/" + inputs.sourceLock.Compiler.Entrypoint,
			Kind: "npm-bin", Version: rootPackage.Version,
			SHA256: byPath["compiler/"+inputs.sourceLock.Compiler.Entrypoint].SHA256, SourceIntegrity: rootPackage.Integrity,
		},
	}
	sort.Slice(binaries, func(i, j int) bool { return binaries[i].Path < binaries[j].Path })
	return SetupBundleManifest{
		Schema:                SetupManifestSchemaV1,
		ReleaseVersion:        releaseVersion,
		Platform:              platformSource.Platform(),
		ArchiveName:           setupArchiveName(releaseVersion, platformSource.Platform()),
		ManifestPath:          BundleManifestPath,
		CompilerLockDigest:    inputs.closure.Digest,
		CompilerInstallPolicy: "never-run-package-scripts",
		Components:            components,
		BinaryIdentities:      binaries,
		Files:                 files,
	}
}

func buildManifestComponents(releaseVersion string, platformSource PlatformSource, inputs releaseInputs, goalrailDigest string) []ManifestComponent {
	components := []ManifestComponent{
		{
			ID: "goalrail", Name: "gr", Kind: "native-binary", Version: releaseVersion,
			Integrity: goalrailDigest, LicenseRef: "bundle:licenses/goalrail-LICENSE",
			ProvenanceRef: "release:" + releaseVersion, Dependencies: []ManifestDependency{},
		},
		{
			ID: "node", Name: inputs.sourceLock.Runtime.ID, Kind: "private-runtime", Version: inputs.sourceLock.Runtime.Version,
			Integrity: "sha256:" + platformSource.RuntimeSHA256, LicenseRef: "bundle:licenses/node-LICENSE",
			ProvenanceRef: platformSource.RuntimeURL + "#sha256=" + platformSource.RuntimeSHA256,
			Dependencies:  []ManifestDependency{},
		},
		{
			ID: "compiler-lock", Name: "goalrail-private-planning-runtime", Kind: "dependency-lock", Version: inputs.sourceLock.Compiler.Version,
			Integrity: inputs.closure.Digest, LicenseRef: "bundle:compiler/package-lock.json",
			ProvenanceRef: "repository:" + CompilerLockPath + "#" + inputs.closure.Digest,
			Dependencies: []ManifestDependency{{
				Name: inputs.sourceLock.Compiler.ID, Requested: inputs.sourceLock.Compiler.Version,
				ComponentID: npmComponentID("node_modules/" + inputs.sourceLock.Compiler.ID),
			}},
		},
	}
	for _, packageEntry := range inputs.closure.Packages {
		components = append(components, ManifestComponent{
			ID: npmComponentID(packageEntry.Path), Name: packageEntry.Name, Kind: "npm-package", Version: packageEntry.Version,
			Integrity: packageEntry.Integrity, LicenseRef: "spdx:" + packageEntry.License,
			ProvenanceRef:    packageEntry.Resolved + "#" + packageEntry.Integrity,
			HasInstallScript: packageEntry.HasInstallScript,
			Dependencies:     append([]ManifestDependency(nil), packageEntry.Dependencies...),
		})
	}
	sort.Slice(components, func(i, j int) bool { return components[i].ID < components[j].ID })
	return components
}

func validateUniquePayload(payload []payloadFile) error {
	seen := make(map[string]struct{}, len(payload))
	for _, entry := range payload {
		if _, duplicate := seen[entry.record.Path]; duplicate {
			return fmt.Errorf("setup payload repeats %s", entry.record.Path)
		}
		seen[entry.record.Path] = struct{}{}
	}
	return nil
}

func verifyCompilerEntrypoint(files []payloadFile, compiler CompilerSource, closure compilerClosure) error {
	entrypoint := "compiler/" + compiler.Entrypoint
	packageJSON := "compiler/node_modules/" + compiler.ID + "/package.json"
	var entrypointFound, packageFound bool
	for _, file := range files {
		switch file.record.Path {
		case entrypoint:
			entrypointFound = true
		case packageJSON:
			packageFound = true
		}
	}
	if !entrypointFound || !packageFound {
		return fmt.Errorf("compiler source does not contain its pinned package.json and entrypoint")
	}
	root := closure.byPath["node_modules/"+compiler.ID]
	if root.Version != compiler.Version {
		return fmt.Errorf("compiler entrypoint package version mismatch")
	}
	return nil
}

func writeSetupArchive(target string, payload []payloadFile, manifestRaw []byte) (err error) {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	gzipWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range payload {
		mode, err := strconv.ParseInt(entry.record.Mode, 8, 64)
		if err != nil {
			return err
		}
		if err := writeTarFile(tarWriter, entry.record.Path, mode, entry.record.SizeBytes, entry.sourcePath, nil); err != nil {
			return err
		}
	}
	if err := writeTarFile(tarWriter, BundleManifestPath, 0o644, int64(len(manifestRaw)), "", manifestRaw); err != nil {
		return err
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	return nil
}

func writeTarFile(writer *tar.Writer, name string, mode, size int64, sourcePath string, raw []byte) error {
	header := &tar.Header{
		Name: name, Mode: mode, Size: size, Typeflag: tar.TypeReg,
		ModTime: time.Unix(0, 0).UTC(), AccessTime: time.Time{}, ChangeTime: time.Time{},
		Uid: 0, Gid: 0, Uname: "", Gname: "", Format: tar.FormatPAX,
	}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	if raw != nil {
		_, err := writer.Write(raw)
		return err
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(writer, file)
	if err != nil {
		return err
	}
	if written != size {
		return fmt.Errorf("archive source %s changed: read %d bytes, want %d", sourcePath, written, size)
	}
	return nil
}

func writeNewArtifact(target string, raw []byte, mode os.FileMode) error {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create release artifact %s: %w", filepath.Base(target), err)
	}
	written, writeErr := file.Write(raw)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write release artifact %s: %w", filepath.Base(target), writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close release artifact %s: %w", filepath.Base(target), closeErr)
	}
	if written != len(raw) {
		return fmt.Errorf("write release artifact %s: wrote %d bytes, want %d", filepath.Base(target), written, len(raw))
	}
	return nil
}

func inspectPublishedArtifact(filePath, name string) (PublishedArtifact, error) {
	payload, err := payloadFromExisting(filePath, name, "release-asset", 0o644)
	if err != nil {
		return PublishedArtifact{}, err
	}
	return PublishedArtifact{Name: name, SizeBytes: payload.record.SizeBytes, SHA256: payload.record.SHA256}, nil
}

func writeChecksums(distDir string, metadata ReleaseMetadata) error {
	artifacts := []PublishedArtifact{{Name: CurrentMetadataName}}
	for _, platform := range metadata.SupportedPlatforms {
		artifacts = append(artifacts, platform.MinimalArchive, platform.SetupArchive, platform.SetupManifest)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	var builder strings.Builder
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if _, duplicate := seen[artifact.Name]; duplicate {
			return fmt.Errorf("checksum input repeats %s", artifact.Name)
		}
		seen[artifact.Name] = struct{}{}
		inspected, err := inspectPublishedArtifact(filepath.Join(distDir, artifact.Name), artifact.Name)
		if err != nil {
			return err
		}
		builder.WriteString(strings.TrimPrefix(inspected.SHA256, "sha256:"))
		builder.WriteString("  ")
		builder.WriteString(inspected.Name)
		builder.WriteByte('\n')
	}
	return writeNewArtifact(filepath.Join(distDir, ChecksumsName), []byte(builder.String()), 0o644)
}

func readGoBinaryVersion(filePath string) (string, error) {
	info, err := buildinfo.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	stamped, err := linkerReleaseVersion(info.Settings)
	if err != nil {
		return "", err
	}
	if stamped != "" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" && info.Main.Version != stamped {
			return "", fmt.Errorf("linker release %q disagrees with module version %q", stamped, info.Main.Version)
		}
		return stamped, nil
	}
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "", fmt.Errorf("binary has neither a verified release linker stamp nor a module release version")
	}
	return info.Main.Version, nil
}

func ReadGoBinaryVersion(filePath string) (string, error) {
	return readGoBinaryVersion(filePath)
}

func linkerReleaseVersion(settings []debug.BuildSetting) (string, error) {
	var stamped string
	for _, setting := range settings {
		if setting.Key != "-ldflags" {
			continue
		}
		fields := strings.Fields(setting.Value)
		for index := 0; index < len(fields); index++ {
			var assignment string
			switch {
			case fields[index] == "-X" && index+1 < len(fields):
				index++
				assignment = fields[index]
			case strings.HasPrefix(fields[index], "-X="):
				assignment = strings.TrimPrefix(fields[index], "-X=")
			default:
				continue
			}
			name, value, ok := strings.Cut(assignment, "=")
			if !ok || name != releaseVersionSymbol {
				continue
			}
			if stamped != "" && stamped != value {
				return "", fmt.Errorf("binary carries conflicting release linker stamps")
			}
			stamped = value
		}
	}
	if stamped != "" && (!validVersion(stamped) || !strings.HasPrefix(stamped, "v")) {
		return "", fmt.Errorf("binary carries invalid release linker stamp %q", stamped)
	}
	return stamped, nil
}

func minimalArchiveName(version string, platform Platform) string {
	return "gr_" + version + "_" + platform.Key() + ".tar.gz"
}

func setupArchiveName(version string, platform Platform) string {
	return "goalrail-setup_" + version + "_" + platform.Key() + ".tar.gz"
}

func setupManifestName(version string, platform Platform) string {
	return "goalrail-setup_" + version + "_" + platform.Key() + ".manifest.json"
}
