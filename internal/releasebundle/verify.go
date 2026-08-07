package releasebundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

type VerifyOptions struct {
	RepoRoot       string
	DistDir        string
	SourceCacheDir string
	ReleaseVersion string

	readGoVersion func(string) (string, error)
}

func Verify(ctx context.Context, options VerifyOptions) error {
	paths, err := validateBuildPaths(options.RepoRoot, options.DistDir, options.SourceCacheDir)
	if err != nil {
		return err
	}
	if !validVersion(options.ReleaseVersion) || !strings.HasPrefix(options.ReleaseVersion, "v") {
		return fmt.Errorf("release version %q is not a bounded v-prefixed version", options.ReleaseVersion)
	}
	inputs, err := loadReleaseInputs(paths.repoRoot)
	if err != nil {
		return err
	}
	versionReader := options.readGoVersion
	if versionReader == nil {
		versionReader = readGoBinaryVersion
	}

	metadataRaw, err := readBoundedRegularFile(filepath.Join(paths.distDir, CurrentMetadataName), maxContractBytes)
	if err != nil {
		return fmt.Errorf("read current release metadata: %w", err)
	}
	metadata, err := decodeStrictJSON[ReleaseMetadata](bytes.NewReader(metadataRaw), "current release metadata")
	if err != nil {
		return err
	}
	canonicalMetadata, err := canonicalJSON(metadata)
	if err != nil {
		return err
	}
	if !bytes.Equal(metadataRaw, canonicalMetadata) {
		return fmt.Errorf("current release metadata is not canonical JSON")
	}
	if err := validateReleaseMetadata(metadata, options.ReleaseVersion, inputs); err != nil {
		return err
	}
	checksums, err := readChecksums(filepath.Join(paths.distDir, ChecksumsName))
	if err != nil {
		return err
	}
	expectedAssets := releaseAssets(metadata)
	if err := validateChecksums(paths.distDir, checksums, expectedAssets); err != nil {
		return err
	}

	temporaryRoot, err := os.MkdirTemp("", "goalrail-release-verify-*")
	if err != nil {
		return fmt.Errorf("create release verification directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	compilerRoot := filepath.Join(temporaryRoot, "compiler")
	compilerFiles, err := compilerBasePayload(compilerRoot, inputs)
	if err != nil {
		return err
	}
	extractedCompiler, err := cacheCompilerSources(ctx, paths.cacheDir, compilerRoot, inputs.closure, nil)
	if err != nil {
		return err
	}
	compilerFiles = append(compilerFiles, extractedCompiler...)
	sortPayloadFiles(compilerFiles)
	if err := verifyCompilerEntrypoint(compilerFiles, inputs.sourceLock.Compiler, inputs.closure); err != nil {
		return err
	}

	for _, platformMetadata := range metadata.SupportedPlatforms {
		platformSource, ok := sourceForPlatform(inputs.sourceLock, platformMetadata.Platform)
		if !ok {
			return fmt.Errorf("metadata platform %s has no source lock", platformMetadata.Platform.Key())
		}
		manifestRaw, err := readBoundedRegularFile(filepath.Join(paths.distDir, platformMetadata.SetupManifest.Name), maxContractBytes)
		if err != nil {
			return fmt.Errorf("read %s: %w", platformMetadata.SetupManifest.Name, err)
		}
		manifest, err := decodeStrictJSON[SetupBundleManifest](bytes.NewReader(manifestRaw), platformMetadata.SetupManifest.Name)
		if err != nil {
			return err
		}
		canonicalManifest, err := canonicalJSON(manifest)
		if err != nil {
			return err
		}
		if !bytes.Equal(manifestRaw, canonicalManifest) {
			return fmt.Errorf("%s is not canonical JSON", platformMetadata.SetupManifest.Name)
		}
		if err := validateSetupManifest(manifest, inputs); err != nil {
			return fmt.Errorf("validate %s: %w", platformMetadata.SetupManifest.Name, err)
		}
		if manifest.ReleaseVersion != options.ReleaseVersion || manifest.Platform != platformMetadata.Platform || manifest.ArchiveName != platformMetadata.SetupArchive.Name {
			return fmt.Errorf("manifest, metadata, and release identity disagree for %s", platformMetadata.Platform.Key())
		}

		platformRoot := filepath.Join(temporaryRoot, "runtime-"+platformMetadata.Platform.Key())
		if err := os.MkdirAll(platformRoot, 0o755); err != nil {
			return err
		}
		runtimeFiles, err := cacheRuntimeSource(ctx, paths.cacheDir, platformRoot, platformSource, inputs.sourceLock.Runtime, nil)
		if err != nil {
			return err
		}
		licensePayload, err := payloadFromBytes(platformRoot, "licenses/goalrail-LICENSE", "goalrail", 0o644, inputs.licenseRaw)
		if err != nil {
			return err
		}
		expectedSources := make([]payloadFile, 0, len(compilerFiles)+len(runtimeFiles)+1)
		expectedSources = append(expectedSources, compilerFiles...)
		expectedSources = append(expectedSources, runtimeFiles...)
		expectedSources = append(expectedSources, licensePayload)
		if err := verifyManifestSources(manifest, expectedSources, platformMetadata.GoalrailBinary); err != nil {
			return fmt.Errorf("verify source closure for %s: %w", platformMetadata.Platform.Key(), err)
		}
		if err := verifySetupArchive(
			filepath.Join(paths.distDir, platformMetadata.SetupArchive.Name), manifestRaw, manifest,
			inputs, platformMetadata.GoalrailBinary, versionReader, temporaryRoot,
		); err != nil {
			return fmt.Errorf("verify %s: %w", platformMetadata.SetupArchive.Name, err)
		}
		if err := verifyMinimalArchive(
			filepath.Join(paths.distDir, platformMetadata.MinimalArchive.Name), inputs.licenseRaw,
			platformMetadata.GoalrailBinary, versionReader, temporaryRoot,
		); err != nil {
			return fmt.Errorf("verify %s: %w", platformMetadata.MinimalArchive.Name, err)
		}
	}
	return nil
}

func validateSetupManifest(manifest SetupBundleManifest, inputs releaseInputs) error {
	if err := validateSetupManifestShape(manifest); err != nil {
		return err
	}
	platformSource, ok := sourceForPlatform(inputs.sourceLock, manifest.Platform)
	if !ok {
		return fmt.Errorf("platform %s is not supported", manifest.Platform.Key())
	}
	if manifest.CompilerLockDigest != inputs.closure.Digest || manifest.CompilerInstallPolicy != "never-run-package-scripts" {
		return fmt.Errorf("compiler lock or install policy is inconsistent")
	}
	if len(manifest.Components) != len(inputs.closure.Packages)+3 {
		return fmt.Errorf("component count = %d, want %d", len(manifest.Components), len(inputs.closure.Packages)+3)
	}
	filesByPath := make(map[string]ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		filesByPath[file.Path] = file
	}
	goalrail, ok := binaryAt(manifest, "bin/gr")
	if !ok {
		return fmt.Errorf("manifest has no Goalrail binary identity")
	}
	expectedComponents := buildManifestComponents(manifest.ReleaseVersion, platformSource, inputs, goalrail.SHA256)
	if !reflect.DeepEqual(manifest.Components, expectedComponents) {
		return fmt.Errorf("manifest components do not equal the pinned dependency closure")
	}
	root := inputs.closure.byPath["node_modules/"+inputs.sourceLock.Compiler.ID]
	expectedBinaries := []BinaryIdentity{
		{ComponentID: "goalrail", Path: "bin/gr", Kind: "go-buildinfo", Version: manifest.ReleaseVersion, SHA256: goalrail.SHA256, SourceIntegrity: goalrail.SHA256},
		{ComponentID: npmComponentID(root.Path), Path: "compiler/" + inputs.sourceLock.Compiler.Entrypoint, Kind: "npm-bin", Version: root.Version, SHA256: filesByPath["compiler/"+inputs.sourceLock.Compiler.Entrypoint].SHA256, SourceIntegrity: root.Integrity},
		{ComponentID: "node", Path: "runtime/node/bin/node", Kind: "node-distribution", Version: inputs.sourceLock.Runtime.Version, SHA256: filesByPath["runtime/node/bin/node"].SHA256, SourceIntegrity: "sha256:" + platformSource.RuntimeSHA256},
	}
	sort.Slice(expectedBinaries, func(i, j int) bool { return expectedBinaries[i].Path < expectedBinaries[j].Path })
	if !reflect.DeepEqual(manifest.BinaryIdentities, expectedBinaries) {
		return fmt.Errorf("manifest binary identities do not equal the pinned release identities")
	}
	return nil
}

func validateSetupManifestShape(manifest SetupBundleManifest) error {
	if manifest.Schema != SetupManifestSchemaV1 {
		return fmt.Errorf("schema = %q, want %q", manifest.Schema, SetupManifestSchemaV1)
	}
	if !validVersion(manifest.ReleaseVersion) || !strings.HasPrefix(manifest.ReleaseVersion, "v") {
		return fmt.Errorf("release version is invalid")
	}
	if manifest.Platform.OS == "" || manifest.Platform.Arch == "" || manifest.ArchiveName != setupArchiveName(manifest.ReleaseVersion, manifest.Platform) || manifest.ManifestPath != BundleManifestPath {
		return fmt.Errorf("archive, manifest path, or platform is inconsistent")
	}
	if !strings.HasPrefix(manifest.CompilerLockDigest, "sha256:") || !sha256Pattern.MatchString(strings.TrimPrefix(manifest.CompilerLockDigest, "sha256:")) || manifest.CompilerInstallPolicy != "never-run-package-scripts" {
		return fmt.Errorf("compiler lock or install policy is invalid")
	}
	if len(manifest.Components) == 0 {
		return fmt.Errorf("manifest has no components")
	}
	componentIDs := make(map[string]struct{}, len(manifest.Components))
	for index, component := range manifest.Components {
		if !domain.IsComponentID(component.ID) || component.Name == "" || component.Kind == "" || component.Version == "" || component.Integrity == "" || !validReference(component.LicenseRef) || !validReference(component.ProvenanceRef) {
			return fmt.Errorf("component %d is incomplete or carries an identity a setup plan cannot express", index)
		}
		if index > 0 && manifest.Components[index-1].ID >= component.ID {
			return fmt.Errorf("components are not uniquely sorted")
		}
		componentIDs[component.ID] = struct{}{}
		for dependencyIndex, dependency := range component.Dependencies {
			if dependency.Name == "" || dependency.Requested == "" || dependency.ComponentID == "" {
				return fmt.Errorf("component %s dependency %d is incomplete", component.ID, dependencyIndex)
			}
			if dependencyIndex > 0 && manifest.Components[index].Dependencies[dependencyIndex-1].Name >= dependency.Name {
				return fmt.Errorf("component %s dependencies are not uniquely sorted", component.ID)
			}
		}
	}
	for _, component := range manifest.Components {
		for _, dependency := range component.Dependencies {
			if _, ok := componentIDs[dependency.ComponentID]; !ok {
				return fmt.Errorf("component %s dependency points to missing %s", component.ID, dependency.ComponentID)
			}
		}
	}
	if len(manifest.Files) == 0 {
		return fmt.Errorf("manifest has no files")
	}
	filesByPath := make(map[string]ManifestFile, len(manifest.Files))
	for index, file := range manifest.Files {
		if !safeArchivePath(file.Path) || file.Path == manifest.ManifestPath || file.SizeBytes < 0 || !strings.HasPrefix(file.SHA256, "sha256:") || !sha256Pattern.MatchString(strings.TrimPrefix(file.SHA256, "sha256:")) {
			return fmt.Errorf("manifest file %d is invalid", index)
		}
		if file.Mode != "0644" && file.Mode != "0755" {
			return fmt.Errorf("manifest file %s has unsupported mode %q", file.Path, file.Mode)
		}
		if _, ok := componentIDs[file.ComponentID]; !ok {
			return fmt.Errorf("manifest file %s has unknown component %s", file.Path, file.ComponentID)
		}
		if index > 0 && manifest.Files[index-1].Path >= file.Path {
			return fmt.Errorf("manifest files are not uniquely sorted")
		}
		filesByPath[file.Path] = file
	}
	if len(manifest.BinaryIdentities) != 3 {
		return fmt.Errorf("binary identity count = %d, want 3", len(manifest.BinaryIdentities))
	}
	for index, identity := range manifest.BinaryIdentities {
		file, ok := filesByPath[identity.Path]
		if !ok || file.ComponentID != identity.ComponentID || file.SHA256 != identity.SHA256 || identity.Version == "" || identity.Kind == "" || identity.SourceIntegrity == "" {
			return fmt.Errorf("binary identity %d does not bind an exact manifest file", index)
		}
		if index > 0 && manifest.BinaryIdentities[index-1].Path >= identity.Path {
			return fmt.Errorf("binary identities are not uniquely sorted")
		}
	}
	return nil
}

func validateReleaseMetadata(metadata ReleaseMetadata, releaseVersion string, inputs releaseInputs) error {
	if err := validateReleaseMetadataShape(metadata); err != nil {
		return err
	}
	if metadata.ReleaseVersion != releaseVersion {
		return fmt.Errorf("current release metadata version is inconsistent")
	}
	expectedCompatibility := ReleaseCompatibility{
		GovernanceContract:    "goalrail-governance-v1",
		SetupProfileSchema:    "goalrail.setup-profile/v1",
		SetupManifestSchema:   SetupManifestSchemaV1,
		RuntimeID:             inputs.sourceLock.Runtime.ID,
		RuntimeVersion:        inputs.sourceLock.Runtime.Version,
		CompilerID:            inputs.sourceLock.Compiler.ID,
		CompilerVersion:       inputs.sourceLock.Compiler.Version,
		CompilerInstallPolicy: "never-run-package-scripts",
	}
	if metadata.Compatibility != expectedCompatibility {
		return fmt.Errorf("current release compatibility does not equal source pins")
	}
	platforms := make([]Platform, 0, len(metadata.SupportedPlatforms))
	for _, entry := range metadata.SupportedPlatforms {
		platforms = append(platforms, entry.Platform)
	}
	sortPlatforms(platforms)
	if !equalPlatforms(platforms, currentPlatforms) {
		return fmt.Errorf("release metadata platforms do not equal current supported platforms")
	}
	return nil
}

func validateReleaseMetadataShape(metadata ReleaseMetadata) error {
	if metadata.Schema != ReleaseMetadataSchema || !validVersion(metadata.ReleaseVersion) || !strings.HasPrefix(metadata.ReleaseVersion, "v") || metadata.ChecksumArtifact != ChecksumsName {
		return fmt.Errorf("current release metadata header is inconsistent")
	}
	compatibility := metadata.Compatibility
	if compatibility.GovernanceContract == "" || compatibility.SetupProfileSchema != "goalrail.setup-profile/v1" || compatibility.SetupManifestSchema != SetupManifestSchemaV1 || compatibility.RuntimeID == "" || compatibility.RuntimeVersion == "" || compatibility.CompilerID == "" || compatibility.CompilerVersion == "" || compatibility.CompilerInstallPolicy != "never-run-package-scripts" {
		return fmt.Errorf("current release compatibility is incomplete")
	}
	if len(metadata.SupportedPlatforms) == 0 {
		return fmt.Errorf("current release metadata has no supported platforms")
	}
	for index, entry := range metadata.SupportedPlatforms {
		if entry.Platform.OS == "" || entry.Platform.Arch == "" || index > 0 && metadata.SupportedPlatforms[index-1].Platform.Key() >= entry.Platform.Key() {
			return fmt.Errorf("release platforms are not uniquely sorted")
		}
		if entry.MinimalArchive.Name != minimalArchiveName(metadata.ReleaseVersion, entry.Platform) || entry.SetupArchive.Name != setupArchiveName(metadata.ReleaseVersion, entry.Platform) || entry.SetupManifest.Name != setupManifestName(metadata.ReleaseVersion, entry.Platform) {
			return fmt.Errorf("release artifact names are inconsistent for %s", entry.Platform.Key())
		}
		for _, artifact := range []PublishedArtifact{entry.MinimalArchive, entry.SetupArchive, entry.SetupManifest} {
			if artifact.SizeBytes <= 0 || !strings.HasPrefix(artifact.SHA256, "sha256:") || !sha256Pattern.MatchString(strings.TrimPrefix(artifact.SHA256, "sha256:")) || filepath.Base(artifact.Name) != artifact.Name {
				return fmt.Errorf("release artifact %s has invalid identity", artifact.Name)
			}
		}
		identity := entry.GoalrailBinary
		if identity.ComponentID != "goalrail" || identity.Path != "bin/gr" || identity.Kind != "go-buildinfo" || identity.Version != metadata.ReleaseVersion || identity.SHA256 != identity.SourceIntegrity || !strings.HasPrefix(identity.SHA256, "sha256:") || !sha256Pattern.MatchString(strings.TrimPrefix(identity.SHA256, "sha256:")) {
			return fmt.Errorf("Goalrail binary metadata is invalid for %s", entry.Platform.Key())
		}
	}
	return nil
}

func verifyManifestSources(manifest SetupBundleManifest, expectedSources []payloadFile, goalrail BinaryIdentity) error {
	manifestFiles := make(map[string]ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		manifestFiles[file.Path] = file
	}
	for _, expected := range expectedSources {
		actual, ok := manifestFiles[expected.record.Path]
		if !ok || actual != expected.record {
			return fmt.Errorf("manifest file %s does not equal its integrity-verified source", expected.record.Path)
		}
	}
	goalrailFile, ok := manifestFiles["bin/gr"]
	if !ok || goalrailFile.ComponentID != "goalrail" || goalrailFile.SHA256 != goalrail.SHA256 || goalrailFile.Mode != "0755" || goalrailFile.SizeBytes <= 0 {
		return fmt.Errorf("manifest Goalrail binary does not equal current release metadata")
	}
	if len(manifest.Files) != len(expectedSources)+1 {
		return fmt.Errorf("manifest contains %d files, want exact source closure of %d", len(manifest.Files), len(expectedSources)+1)
	}
	return nil
}

func verifySetupArchive(archivePath string, manifestRaw []byte, manifest SetupBundleManifest, inputs releaseInputs, goalrail BinaryIdentity, versionReader func(string) (string, error), temporaryRoot string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	expected := make(map[string]ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		expected[file.Path] = file
	}
	seen := make(map[string]struct{}, len(expected)+1)
	goalrailTemporary, err := os.CreateTemp(temporaryRoot, "gr-*")
	if err != nil {
		return err
	}
	goalrailPath := goalrailTemporary.Name()
	defer os.Remove(goalrailPath)
	defer goalrailTemporary.Close()
	nodeVersion := &containsWriter{needle: []byte(inputs.sourceLock.Runtime.Version)}
	var compilerPackageRaw []byte
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("archive entry %s is not regular", header.Name)
		}
		if _, duplicate := seen[header.Name]; duplicate {
			return fmt.Errorf("archive repeats %s", header.Name)
		}
		seen[header.Name] = struct{}{}
		if header.Name == BundleManifestPath {
			if header.Size != int64(len(manifestRaw)) {
				return fmt.Errorf("embedded manifest size mismatch")
			}
			raw, err := io.ReadAll(io.LimitReader(tarReader, header.Size))
			if err != nil || !bytes.Equal(raw, manifestRaw) {
				return fmt.Errorf("embedded manifest does not equal published manifest")
			}
			continue
		}
		record, ok := expected[header.Name]
		if !ok {
			return fmt.Errorf("archive contains unmanifested file %s", header.Name)
		}
		mode, err := strconv.ParseInt(record.Mode, 8, 64)
		if err != nil || header.Mode&0o777 != mode || header.Size != record.SizeBytes {
			return fmt.Errorf("archive metadata disagrees for %s", header.Name)
		}
		hash := sha256.New()
		writers := []io.Writer{hash}
		if header.Name == "bin/gr" {
			writers = append(writers, goalrailTemporary)
		}
		if header.Name == "runtime/node/bin/node" {
			writers = append(writers, nodeVersion)
		}
		var compilerBuffer bytes.Buffer
		if header.Name == "compiler/node_modules/"+inputs.sourceLock.Compiler.ID+"/package.json" {
			if header.Size > 1<<20 {
				return fmt.Errorf("compiler package.json exceeds bounds")
			}
			writers = append(writers, &compilerBuffer)
		}
		written, err := io.Copy(io.MultiWriter(writers...), io.LimitReader(tarReader, header.Size))
		if err != nil || written != header.Size {
			return fmt.Errorf("read archive entry %s", header.Name)
		}
		digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
		if digest != record.SHA256 {
			return fmt.Errorf("archive digest disagrees for %s", header.Name)
		}
		if compilerBuffer.Len() != 0 {
			compilerPackageRaw = append([]byte(nil), compilerBuffer.Bytes()...)
		}
	}
	if len(seen) != len(expected)+1 {
		return fmt.Errorf("archive entries are incomplete")
	}
	if _, ok := seen[BundleManifestPath]; !ok {
		return fmt.Errorf("archive has no embedded manifest")
	}
	if err := goalrailTemporary.Close(); err != nil {
		return err
	}
	version, err := versionReader(goalrailPath)
	if err != nil || version != goalrail.Version {
		return fmt.Errorf("embedded Goalrail version = %q, want %q: %v", version, goalrail.Version, err)
	}
	if !nodeVersion.found {
		return fmt.Errorf("runtime binary does not expose pinned version %s", inputs.sourceLock.Runtime.Version)
	}
	if err := verifyCompilerPackageJSON(compilerPackageRaw, inputs.sourceLock.Compiler); err != nil {
		return err
	}
	return nil
}

func verifyMinimalArchive(archivePath string, licenseRaw []byte, goalrail BinaryIdentity, versionReader func(string) (string, error), temporaryRoot string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	seen := make(map[string]struct{}, 2)
	binary, err := os.CreateTemp(temporaryRoot, "minimal-gr-*")
	if err != nil {
		return err
	}
	binaryPath := binary.Name()
	defer os.Remove(binaryPath)
	defer binary.Close()
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("minimal archive entry %s is not regular", header.Name)
		}
		if _, duplicate := seen[header.Name]; duplicate {
			return fmt.Errorf("minimal archive repeats %s", header.Name)
		}
		seen[header.Name] = struct{}{}
		switch header.Name {
		case "gr":
			hash := sha256.New()
			written, err := io.Copy(io.MultiWriter(hash, binary), io.LimitReader(tarReader, header.Size))
			if err != nil || written != header.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != goalrail.SHA256 {
				return fmt.Errorf("minimal archive Goalrail binary identity mismatch")
			}
		case "LICENSE":
			raw, err := io.ReadAll(io.LimitReader(tarReader, header.Size))
			if err != nil || !bytes.Equal(raw, licenseRaw) {
				return fmt.Errorf("minimal archive license mismatch")
			}
		default:
			return fmt.Errorf("minimal archive contains unexpected %s", header.Name)
		}
	}
	if len(seen) != 2 {
		return fmt.Errorf("minimal archive does not contain exactly gr and LICENSE")
	}
	if err := binary.Close(); err != nil {
		return err
	}
	version, err := versionReader(binaryPath)
	if err != nil || version != goalrail.Version {
		return fmt.Errorf("minimal archive Goalrail version = %q, want %q: %v", version, goalrail.Version, err)
	}
	return nil
}

func verifyCompilerPackageJSON(raw []byte, compiler CompilerSource) error {
	if len(raw) == 0 {
		return fmt.Errorf("setup archive has no compiler package.json")
	}
	var value struct {
		Name    string            `json:"name"`
		Version string            `json:"version"`
		Bin     map[string]string `json:"bin"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("decode compiler package.json: %w", err)
	}
	root := "node_modules/" + compiler.ID + "/"
	entrypoint := strings.TrimPrefix(compiler.Entrypoint, root)
	declaredEntrypoint := strings.TrimPrefix(value.Bin["openspec"], "./")
	if value.Name != compiler.ID || value.Version != compiler.Version || declaredEntrypoint != entrypoint || !safeArchivePath(declaredEntrypoint) {
		return fmt.Errorf("compiler package identity or entrypoint disagrees with source lock")
	}
	return nil
}

type containsWriter struct {
	needle []byte
	tail   []byte
	found  bool
}

func (writer *containsWriter) Write(raw []byte) (int, error) {
	if writer.found || len(writer.needle) == 0 {
		writer.found = true
		return len(raw), nil
	}
	combined := make([]byte, 0, len(writer.tail)+len(raw))
	combined = append(combined, writer.tail...)
	combined = append(combined, raw...)
	if bytes.Contains(combined, writer.needle) {
		writer.found = true
	}
	retain := len(writer.needle) - 1
	if retain > len(combined) {
		retain = len(combined)
	}
	writer.tail = append(writer.tail[:0], combined[len(combined)-retain:]...)
	return len(raw), nil
}

func readChecksums(filePath string) (map[string]string, error) {
	raw, err := readBoundedRegularFile(filePath, 64<<10)
	if err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	result := make(map[string]string)
	for index, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		parts := strings.Split(line, "  ")
		if len(parts) != 2 || !sha256Pattern.MatchString(parts[0]) || filepath.Base(parts[1]) != parts[1] || parts[1] == ChecksumsName {
			return nil, fmt.Errorf("checksums line %d is invalid", index+1)
		}
		if _, duplicate := result[parts[1]]; duplicate {
			return nil, fmt.Errorf("checksums repeat %s", parts[1])
		}
		result[parts[1]] = "sha256:" + parts[0]
	}
	return result, nil
}

func releaseAssets(metadata ReleaseMetadata) []PublishedArtifact {
	assets := []PublishedArtifact{{Name: CurrentMetadataName}}
	for _, platform := range metadata.SupportedPlatforms {
		assets = append(assets, platform.MinimalArchive, platform.SetupArchive, platform.SetupManifest)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	return assets
}

func validateChecksums(distDir string, checksums map[string]string, assets []PublishedArtifact) error {
	if len(checksums) != len(assets) {
		return fmt.Errorf("checksums cover %d assets, want %d", len(checksums), len(assets))
	}
	for _, artifact := range assets {
		actual, err := inspectPublishedArtifact(filepath.Join(distDir, artifact.Name), artifact.Name)
		if err != nil {
			return err
		}
		if artifact.Name != CurrentMetadataName && (artifact.SizeBytes != actual.SizeBytes || artifact.SHA256 != actual.SHA256) {
			return fmt.Errorf("metadata identity disagrees for %s", artifact.Name)
		}
		if checksums[artifact.Name] != actual.SHA256 {
			return fmt.Errorf("checksum identity disagrees for %s", artifact.Name)
		}
	}
	return nil
}

func readBoundedRegularFile(filePath string, maxBytes int64) ([]byte, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, fmt.Errorf("file is not a bounded regular file")
	}
	return os.ReadFile(filePath)
}

func binaryAt(manifest SetupBundleManifest, filePath string) (BinaryIdentity, bool) {
	for _, identity := range manifest.BinaryIdentities {
		if identity.Path == filePath {
			return identity, true
		}
	}
	return BinaryIdentity{}, false
}

func sourceForPlatform(source SourceLock, platform Platform) (PlatformSource, bool) {
	for _, candidate := range source.Platforms {
		if candidate.Platform() == platform {
			return candidate, true
		}
	}
	return PlatformSource{}, false
}
