package releasebundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxNPMArchiveBytes     = 64 << 20
	maxRuntimeArchiveBytes = 128 << 20
)

type SourceFetcher func(context.Context, string, int64) ([]byte, error)

type payloadFile struct {
	record     ManifestFile
	sourcePath string
}

func cacheCompilerSources(ctx context.Context, cacheRoot, extractRoot string, closure compilerClosure, fetch SourceFetcher) ([]payloadFile, error) {
	files := make([]payloadFile, 0, len(closure.Packages)*8)
	seen := make(map[string]struct{})
	for _, packageEntry := range closure.Packages {
		cacheName := "npm-" + bareSHA256([]byte(packageEntry.Resolved)) + ".tgz"
		archivePath, err := ensureCachedSource(ctx, cacheRoot, cacheName, packageEntry.Resolved, packageEntry.Integrity, maxNPMArchiveBytes, fetch)
		if err != nil {
			return nil, fmt.Errorf("cache %s@%s: %w", packageEntry.Name, packageEntry.Version, err)
		}
		packageFiles, err := extractNPMArchive(archivePath, extractRoot, packageEntry, seen)
		if err != nil {
			return nil, err
		}
		files = append(files, packageFiles...)
	}
	sortPayloadFiles(files)
	return files, nil
}

func cacheRuntimeSource(ctx context.Context, cacheRoot, extractRoot string, source PlatformSource, runtime RuntimeSource, fetch SourceFetcher) ([]payloadFile, error) {
	archivePath, err := ensureCachedSource(ctx, cacheRoot, source.RuntimeArchive, source.RuntimeURL, "sha256:"+source.RuntimeSHA256, maxRuntimeArchiveBytes, fetch)
	if err != nil {
		return nil, fmt.Errorf("cache runtime for %s: %w", source.Platform().Key(), err)
	}
	return extractRuntimeArchive(archivePath, extractRoot, source, runtime)
}

func ensureCachedSource(ctx context.Context, cacheRoot, name, sourceURL, integrity string, maxBytes int64, fetch SourceFetcher) (string, error) {
	if path.Base(name) != name || name == "." {
		return "", fmt.Errorf("cache name %q is unsafe", name)
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", fmt.Errorf("create source cache: %w", err)
	}
	target := filepath.Join(cacheRoot, name)
	if info, err := os.Lstat(target); err == nil {
		if !info.Mode().IsRegular() || info.Size() > maxBytes {
			return "", fmt.Errorf("cached source %s is not a bounded regular file", name)
		}
		raw, err := os.ReadFile(target)
		if err != nil {
			return "", fmt.Errorf("read cached source %s: %w", name, err)
		}
		if err := verifySourceIntegrity(raw, integrity); err != nil {
			return "", fmt.Errorf("cached source %s: %w", name, err)
		}
		return target, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect cached source %s: %w", name, err)
	}
	if fetch == nil {
		return "", fmt.Errorf("cached source %s is absent and network fetching is disabled", name)
	}
	raw, err := fetch(ctx, sourceURL, maxBytes)
	if err != nil {
		return "", err
	}
	if err := verifySourceIntegrity(raw, integrity); err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(cacheRoot, ".source-*")
	if err != nil {
		return "", fmt.Errorf("create temporary cached source: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return "", fmt.Errorf("protect temporary cached source: %w", err)
	}
	if _, err := temporary.Write(raw); err != nil {
		return "", fmt.Errorf("write temporary cached source: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("sync temporary cached source: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary cached source: %w", err)
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", fmt.Errorf("place cached source %s: %w", name, err)
	}
	committed = true
	return target, nil
}

func verifySourceIntegrity(raw []byte, integrity string) error {
	algorithm, encoded, ok := strings.Cut(integrity, ":")
	if ok && algorithm == "sha256" {
		if !sha256Pattern.MatchString(encoded) {
			return fmt.Errorf("invalid SHA-256 integrity")
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != encoded {
			return fmt.Errorf("SHA-256 integrity mismatch")
		}
		return nil
	}
	algorithm, encoded, ok = strings.Cut(integrity, "-")
	if !ok || algorithm != "sha512" {
		return fmt.Errorf("unsupported source integrity %q", integrity)
	}
	want, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(want) != sha512Size {
		return fmt.Errorf("invalid SHA-512 integrity")
	}
	sum := sha512.Sum512(raw)
	if !bytes.Equal(sum[:], want) {
		return fmt.Errorf("SHA-512 integrity mismatch")
	}
	return nil
}

func extractNPMArchive(archivePath, extractRoot string, packageEntry lockedPackage, seen map[string]struct{}) ([]payloadFile, error) {
	archive, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open compiler source %s: %w", packageEntry.Name, err)
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, fmt.Errorf("open compiler source gzip %s: %w", packageEntry.Name, err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	prefix := "package/"
	componentID := npmComponentID(packageEntry.Path)
	var files []payloadFile
	foundPackageJSON := false
	var packageJSONPath string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read compiler source %s: %w", packageEntry.Name, err)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("compiler source %s contains non-regular entry %q", packageEntry.Name, header.Name)
		}
		if !strings.HasPrefix(header.Name, prefix) {
			return nil, fmt.Errorf("compiler source %s entry %q has no package root", packageEntry.Name, header.Name)
		}
		relative := strings.TrimPrefix(header.Name, prefix)
		if !safeArchivePath(relative) {
			return nil, fmt.Errorf("compiler source %s entry %q is unsafe", packageEntry.Name, header.Name)
		}
		bundlePath := path.Join("compiler", packageEntry.Path, relative)
		if _, duplicate := seen[bundlePath]; duplicate {
			return nil, fmt.Errorf("compiler source repeats bundle path %s", bundlePath)
		}
		seen[bundlePath] = struct{}{}
		mode := normalizedMode(header.Mode)
		payload, err := materializePayload(extractRoot, bundlePath, componentID, mode, io.LimitReader(tarReader, header.Size), header.Size)
		if err != nil {
			return nil, fmt.Errorf("extract compiler source %s: %w", packageEntry.Name, err)
		}
		files = append(files, payload)
		if relative == "package.json" {
			foundPackageJSON = true
			packageJSONPath = payload.sourcePath
		}
	}
	if !foundPackageJSON {
		return nil, fmt.Errorf("compiler source %s has no package.json", packageEntry.Name)
	}
	if err := verifyLockedPackageJSON(packageJSONPath, packageEntry); err != nil {
		return nil, err
	}
	return files, nil
}

func verifyLockedPackageJSON(filePath string, packageEntry lockedPackage) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read compiler source package.json for %s: %w", packageEntry.Name, err)
	}
	var value struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		License      string            `json:"license"`
		Dependencies map[string]string `json:"dependencies"`
		Scripts      map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("decode compiler source package.json for %s: %w", packageEntry.Name, err)
	}
	wantDependencies := make(map[string]string, len(packageEntry.Dependencies))
	for _, dependency := range packageEntry.Dependencies {
		wantDependencies[dependency.Name] = dependency.Requested
	}
	hasInstallScript := value.Scripts["preinstall"] != "" || value.Scripts["install"] != "" || value.Scripts["postinstall"] != ""
	if value.Name != packageEntry.Name || value.Version != packageEntry.Version || value.License != packageEntry.License || !equalStringMaps(value.Dependencies, wantDependencies) || hasInstallScript != packageEntry.HasInstallScript {
		return fmt.Errorf("compiler source package.json for %s does not equal its lock identity", packageEntry.Name)
	}
	return nil
}

func equalStringMaps(first, second map[string]string) bool {
	if len(first) != len(second) {
		return false
	}
	for key, value := range first {
		if second[key] != value {
			return false
		}
	}
	return true
}

func extractRuntimeArchive(archivePath, extractRoot string, source PlatformSource, runtime RuntimeSource) ([]payloadFile, error) {
	archive, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open runtime source: %w", err)
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, fmt.Errorf("open runtime source gzip: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	prefix := strings.TrimSuffix(source.RuntimeArchive, ".tar.gz") + "/"
	required := map[string]struct {
		bundlePath string
		mode       int64
	}{
		"bin/node": {bundlePath: "runtime/node/bin/node", mode: 0o755},
		"LICENSE":  {bundlePath: "licenses/node-LICENSE", mode: 0o644},
	}
	found := make(map[string]bool, len(required))
	files := make([]payloadFile, 0, len(required))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read runtime source: %w", err)
		}
		if !strings.HasPrefix(header.Name, prefix) {
			continue
		}
		relative := strings.TrimPrefix(header.Name, prefix)
		target, wanted := required[relative]
		if !wanted {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("runtime source entry %s is not regular", relative)
		}
		if found[relative] {
			return nil, fmt.Errorf("runtime source repeats %s", relative)
		}
		payload, err := materializePayload(extractRoot, target.bundlePath, runtime.ID, target.mode, io.LimitReader(tarReader, header.Size), header.Size)
		if err != nil {
			return nil, fmt.Errorf("extract runtime source %s: %w", relative, err)
		}
		files = append(files, payload)
		found[relative] = true
	}
	for relative := range required {
		if !found[relative] {
			return nil, fmt.Errorf("runtime source has no %s", relative)
		}
	}
	sortPayloadFiles(files)
	return files, nil
}

func materializePayload(root, bundlePath, componentID string, mode int64, reader io.Reader, size int64) (payloadFile, error) {
	if !safeArchivePath(bundlePath) || size < 0 {
		return payloadFile{}, fmt.Errorf("invalid payload path or size")
	}
	target := filepath.Join(root, filepath.FromSlash(bundlePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return payloadFile{}, err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(mode))
	if err != nil {
		return payloadFile{}, err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), reader)
	closeErr := file.Close()
	if copyErr != nil {
		return payloadFile{}, copyErr
	}
	if closeErr != nil {
		return payloadFile{}, closeErr
	}
	if written != size {
		return payloadFile{}, fmt.Errorf("wrote %d bytes, want %d", written, size)
	}
	return payloadFile{
		record: ManifestFile{
			Path: bundlePath, ComponentID: componentID, SizeBytes: size,
			SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)), Mode: fmt.Sprintf("%04o", mode),
		},
		sourcePath: target,
	}, nil
}

func payloadFromBytes(root, bundlePath, componentID string, mode int64, raw []byte) (payloadFile, error) {
	return materializePayload(root, bundlePath, componentID, mode, bytes.NewReader(raw), int64(len(raw)))
}

func normalizedMode(mode int64) int64 {
	if mode&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func sortPayloadFiles(files []payloadFile) {
	sort.Slice(files, func(i, j int) bool { return files[i].record.Path < files[j].record.Path })
}

func bareSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
