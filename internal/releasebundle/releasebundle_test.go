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
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestRepositorySourceLockPinsCompletePlatformIndependentClosure(t *testing.T) {
	summary, err := CheckSourceLock(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Runtime != "node@22.18.0" || summary.Compiler != "@fission-ai/openspec@1.6.0" {
		t.Fatalf("unexpected toolchain summary: %+v", summary)
	}
	if summary.PackageCount != 80 || summary.InstallScriptCount != 1 || summary.InstallPolicy != "never-run-package-scripts" {
		t.Fatalf("dependency closure summary = %+v", summary)
	}
	if !equalPlatforms(summary.SupportedPlatforms, currentPlatforms) {
		t.Fatalf("platforms = %+v, want %+v", summary.SupportedPlatforms, currentPlatforms)
	}
}

func TestCompilerClosureRejectsUnverifiableOrPlatformDependentNodes(t *testing.T) {
	compiler := CompilerSource{
		ID: "@fission-ai/openspec", Version: "1.6.0",
		ProvenanceRef: "npm:@fission-ai/openspec@1.6.0#sha512-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	}
	base := map[string]any{
		"name": "fixture", "version": "1.0.0", "lockfileVersion": 3,
		"packages": map[string]any{
			"": map[string]any{"name": "fixture", "version": "1.0.0", "dependencies": map[string]string{"@fission-ai/openspec": "1.6.0"}},
			"node_modules/@fission-ai/openspec": map[string]any{
				"version": "1.6.0", "resolved": "https://registry.npmjs.org/@fission-ai/openspec/-/openspec-1.6.0.tgz",
				"integrity": "sha512-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==", "license": "MIT",
			},
		},
	}

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing integrity",
			mutate: func(lock map[string]any) {
				packages := lock["packages"].(map[string]any)
				packages["node_modules/@fission-ai/openspec"].(map[string]any)["integrity"] = ""
			},
			want: "integrity",
		},
		{
			name: "platform selector",
			mutate: func(lock map[string]any) {
				packages := lock["packages"].(map[string]any)
				packages["node_modules/@fission-ai/openspec"].(map[string]any)["os"] = []string{"darwin"}
			},
			want: "platform-dependent",
		},
		{
			name: "unresolved dependency",
			mutate: func(lock map[string]any) {
				packages := lock["packages"].(map[string]any)
				packages["node_modules/@fission-ai/openspec"].(map[string]any)["dependencies"] = map[string]string{"missing": "1.0.0"}
			},
			want: "not in the exact lock tree",
		},
		{
			name: "required peer dependency",
			mutate: func(lock map[string]any) {
				packages := lock["packages"].(map[string]any)
				packages["node_modules/@fission-ai/openspec"].(map[string]any)["peerDependencies"] = map[string]string{"unbound-peer": "1.0.0"}
			},
			want: "unpinned required peer dependency",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			var mutated map[string]any
			if err := json.Unmarshal(raw, &mutated); err != nil {
				t.Fatal(err)
			}
			test.mutate(mutated)
			raw, err = json.Marshal(mutated)
			if err != nil {
				t.Fatal(err)
			}
			_, err = loadCompilerClosure(raw, compiler)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestReleaseVersionComesFromTheVerifierBoundLinkerAssignment(t *testing.T) {
	settings := []debug.BuildSetting{{
		Key:   "-ldflags",
		Value: "-s -X " + releaseVersionSymbol + "=v0.2.0",
	}}
	version, err := linkerReleaseVersion(settings)
	if err != nil {
		t.Fatal(err)
	}
	if version != "v0.2.0" {
		t.Fatalf("version = %q, want v0.2.0", version)
	}
	settings[0].Value += " -X " + releaseVersionSymbol + "=v0.2.1"
	if _, err := linkerReleaseVersion(settings); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("conflicting linker stamps error = %v", err)
	}
}

func TestReleaseWorkflowVerifiesSetupAssetsBeforePublication(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(workflow)
	for _, required := range []string{
		"-X github.com/heurema/goalrail/internal/harness.releaseVersion=${GITHUB_REF_NAME}",
		"COPYFILE_DISABLE=1 tar",
		"goalrail-release build",
		"goalrail-release verify",
		"current-release.json",
		"goalrail-setup_*.manifest.json",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("release workflow does not contain %q", required)
		}
	}
	verifyIndex := strings.Index(text, "goalrail-release verify")
	publishIndex := strings.Index(text, "gh release create")
	if verifyIndex < 0 || publishIndex < 0 || verifyIndex >= publishIndex {
		t.Fatalf("release verification must run before publication")
	}
}

func TestBuildAndVerifyReleaseBundleFromExactCachedSources(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	dist := filepath.Join(root, "dist")
	cache := filepath.Join(root, "source-cache")
	for _, directory := range []string{repository, dist, cache} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	license := []byte("fixture license\n")
	writeTestArtifact(t, filepath.Join(repository, "LICENSE"), license, 0o644)

	compilerArchive := makeTarGzip(t, map[string]tarFixture{
		"package/package.json": {
			raw: []byte(`{"name":"@fission-ai/openspec","version":"1.6.0","license":"MIT","scripts":{"postinstall":"node scripts/postinstall.js"},"bin":{"openspec":"./bin/openspec.js"}}`), mode: 0o644,
		},
		"package/bin/openspec.js": {raw: []byte("#!/usr/bin/env node\n"), mode: 0o755},
	})
	compilerSum := sha512.Sum512(compilerArchive)
	compilerIntegrity := "sha512-" + base64.StdEncoding.EncodeToString(compilerSum[:])
	compilerURL := "https://registry.npmjs.org/@fission-ai/openspec/-/openspec-1.6.0.tgz"
	writeTestArtifact(t, filepath.Join(cache, "npm-"+bareSHA256([]byte(compilerURL))+".tgz"), compilerArchive, 0o600)

	packageRoot := []byte("{\n  \"name\": \"goalrail-private-planning-runtime\",\n  \"version\": \"1.0.0\",\n  \"private\": true,\n  \"dependencies\": {\n    \"@fission-ai/openspec\": \"1.6.0\"\n  }\n}\n")
	compilerLock := map[string]any{
		"name": "goalrail-private-planning-runtime", "version": "1.0.0", "lockfileVersion": 3,
		"packages": map[string]any{
			"": map[string]any{"name": "goalrail-private-planning-runtime", "version": "1.0.0", "dependencies": map[string]string{"@fission-ai/openspec": "1.6.0"}},
			"node_modules/@fission-ai/openspec": map[string]any{
				"version": "1.6.0", "resolved": compilerURL, "integrity": compilerIntegrity,
				"license": "MIT", "hasInstallScript": true,
			},
		},
	}
	compilerLockRaw, err := json.MarshalIndent(compilerLock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	compilerLockRaw = append(compilerLockRaw, '\n')
	writeTestArtifact(t, filepath.Join(repository, filepath.FromSlash(CompilerPackagePath)), packageRoot, 0o644)
	writeTestArtifact(t, filepath.Join(repository, filepath.FromSlash(CompilerLockPath)), compilerLockRaw, 0o644)

	runtimeVersion := "1.2.3"
	platformSources := make([]PlatformSource, 0, len(currentPlatforms))
	for _, platform := range currentPlatforms {
		archiveName := fmt.Sprintf("node-v%s-%s-%s.tar.gz", runtimeVersion, platform.OS, platform.Arch)
		prefix := strings.TrimSuffix(archiveName, ".tar.gz")
		runtimeArchive := makeTarGzip(t, map[string]tarFixture{
			prefix + "/bin/node": {raw: []byte("fixture node " + runtimeVersion + "\n"), mode: 0o755},
			prefix + "/LICENSE":  {raw: []byte("node fixture license\n"), mode: 0o644},
		})
		sum := sha256.Sum256(runtimeArchive)
		platformSources = append(platformSources, PlatformSource{
			OS: platform.OS, Arch: platform.Arch, RuntimeArchive: archiveName,
			RuntimeURL:    "https://nodejs.org/dist/v" + runtimeVersion + "/" + archiveName,
			RuntimeSHA256: hex.EncodeToString(sum[:]),
		})
		writeTestArtifact(t, filepath.Join(cache, archiveName), runtimeArchive, 0o600)

		platformDir := filepath.Join(dist, platform.Key())
		if err := os.MkdirAll(platformDir, 0o755); err != nil {
			t.Fatal(err)
		}
		binary := []byte("fixture gr " + platform.Key() + "\n")
		writeTestArtifact(t, filepath.Join(platformDir, "gr"), binary, 0o755)
		minimal := makeTarGzip(t, map[string]tarFixture{
			"gr": {raw: binary, mode: 0o755}, "LICENSE": {raw: license, mode: 0o644},
		})
		writeTestArtifact(t, filepath.Join(dist, minimalArchiveName("v0.2.0", platform)), minimal, 0o644)
	}
	sourceLock := SourceLock{
		Schema: SourceLockSchemaV1,
		Runtime: RuntimeSource{
			ID: "node", Version: runtimeVersion,
			LicenseRef:    "https://github.com/nodejs/node/blob/v1.2.3/LICENSE",
			ProvenanceRef: "https://nodejs.org/dist/v1.2.3/SHASUMS256.txt",
		},
		Compiler: CompilerSource{
			ID: "@fission-ai/openspec", Version: "1.6.0", LockPath: CompilerLockPath,
			Entrypoint:    "node_modules/@fission-ai/openspec/bin/openspec.js",
			LicenseRef:    "npm:@fission-ai/openspec@1.6.0#MIT",
			ProvenanceRef: "npm:@fission-ai/openspec@1.6.0#" + compilerIntegrity,
		},
		Platforms: platformSources,
	}
	sourceLockRaw, err := json.MarshalIndent(sourceLock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	sourceLockRaw = append(sourceLockRaw, '\n')
	writeTestArtifact(t, filepath.Join(repository, filepath.FromSlash(SourceLockPath)), sourceLockRaw, 0o644)

	readVersion := func(string) (string, error) { return "v0.2.0", nil }
	metadata, err := Build(context.Background(), BuildOptions{
		RepoRoot: repository, DistDir: dist, SourceCacheDir: cache, ReleaseVersion: "v0.2.0",
		FetchSource: func(context.Context, string, int64) ([]byte, error) {
			return nil, fmt.Errorf("network fetch was not expected")
		},
		readGoVersion: readVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.SupportedPlatforms) != 4 {
		t.Fatalf("published platforms = %d, want 4", len(metadata.SupportedPlatforms))
	}
	if err := Verify(context.Background(), VerifyOptions{
		RepoRoot: repository, DistDir: dist, SourceCacheDir: cache, ReleaseVersion: "v0.2.0",
		readGoVersion: readVersion,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, CurrentMetadataName)); !os.IsNotExist(err) {
		t.Fatalf("release output appeared inside repository: %v", err)
	}

	firstManifest := filepath.Join(dist, metadata.SupportedPlatforms[0].SetupManifest.Name)
	raw, err := os.ReadFile(firstManifest)
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = ' '
	if err := os.WriteFile(firstManifest, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = Verify(context.Background(), VerifyOptions{
		RepoRoot: repository, DistDir: dist, SourceCacheDir: cache, ReleaseVersion: "v0.2.0",
		readGoVersion: readVersion,
	})
	if err == nil || !strings.Contains(err.Error(), "identity disagrees") {
		t.Fatalf("tampered manifest error = %v", err)
	}
}

type tarFixture struct {
	raw  []byte
	mode int64
}

func makeTarGzip(t *testing.T, files map[string]tarFixture) []byte {
	t.Helper()
	var result bytes.Buffer
	gzipWriter := gzip.NewWriter(&result)
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	tarWriter := tar.NewWriter(gzipWriter)
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	sortStrings(paths)
	for _, filePath := range paths {
		fixture := files[filePath]
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: filePath, Mode: fixture.mode, Size: int64(len(fixture.raw)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(0, 0),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(fixture.raw); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}

func writeTestArtifact(t *testing.T, filePath string, raw []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, raw, mode); err != nil {
		t.Fatal(err)
	}
}

func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		for current := index; current > 0 && values[current] < values[current-1]; current-- {
			values[current], values[current-1] = values[current-1], values[current]
		}
	}
}

var _ io.Writer = (*containsWriter)(nil)
