// Package harness installs and inspects the repository-side harness: the
// OpenSpec overlay carrying the Goalrail schema, its currency against the copy
// this binary was built with, and the report of what an installation changed.
//
// The canonical overlay lives in the binary. A repository holds a materialized
// copy, so every question about that copy is answered by comparing digests
// against the canon rather than by trusting the copy or a version recorded
// beside it. Nothing here executes a package runner, a Node runtime, or the
// stock OpenSpec CLI: the overlay is inert content, and comparing it needs no
// toolchain.
package harness

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// canonFS carries the overlay this binary materializes. It is a committed copy
// of the repository's own `openspec/schemas/goalrail-intent` tree; a test pins
// the two byte-identical, because Go embedding cannot reach outside the package
// directory and a second unpinned source of truth is exactly the drift this
// package exists to detect.
//
//go:embed canon
var canonFS embed.FS

// canonRoot is the embedded directory prefix, stripped from every path the rest
// of the package deals in.
const canonRoot = "canon"

// SchemaName is the workflow schema the overlay installs. It is also the value
// the OpenSpec configuration's managed key must name.
const SchemaName = "goalrail-intent"

// OverlayDirectory is where the schema and its templates live inside a
// repository, relative to the repository root.
const OverlayDirectory = "openspec/schemas/" + SchemaName

// ConfigPath is the OpenSpec configuration this package manages one key of. The
// rest of the file is the repository's own content.
const ConfigPath = "openspec/config.yaml"

// PinnedInvocation is the exact command a harnessed repository is driven by.
//
// It is reported rather than executed. The explicit schema argument is not
// decoration: the pinned version resolves `new change` to the default schema
// even when the configuration names this one, so a repository holding only the
// files still fails on its first change without it.
const PinnedInvocation = "OPENSPEC_TELEMETRY=0 npx --yes " + pinnedPackage

// pinnedPackage is the version of the stock CLI this release was verified against.
// It moves only inside a Goalrail change, and the user is never asked about it.
const pinnedPackage = "@fission-ai/openspec@1.6.0"

// PinnedNewChange is the invocation whose omission the pinned version's defect
// punishes.
//
// It names the stock CLI resolved from the registry, which is what a machine
// carrying no installed bundle would have to reach for. Where authorized setup
// has placed the pinned toolchain, InstalledNewChange names that instead: the
// bundle exists so the compiler is fetched once, pinned by digest and verified
// per file, and an invocation that resolves the package again would discard
// that at the moment it was established.
const PinnedNewChange = PinnedInvocation + " new change <name> --schema " + SchemaName

// NewChangeArguments is the part of the invocation that does not depend on how
// the compiler was obtained.
const NewChangeArguments = " new change <name> --schema " + SchemaName

// InstalledNewChange renders the invocation for a verified installed toolchain:
// the bundle's own runtime executing the bundle's own compiler entrypoint, with
// no name left for a package manager to resolve.
func InstalledNewChange(runtimePath, compilerPath string) string {
	return "OPENSPEC_TELEMETRY=0 " + shellArgument(runtimePath) + " " + shellArgument(compilerPath) + NewChangeArguments
}

// shellArgument quotes a path so the reported command survives a home directory
// carrying a space or a quote. A reported command a reader cannot paste is a
// command they will retype wrongly.
func shellArgument(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\n'\"\\$`&;|<>()*?[]#~=%!{}") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// CanonFile is one file of the overlay, with the digest it must have to be
// current.
type CanonFile struct {
	// Path is relative to the repository root.
	Path string `json:"path"`

	// Digest is the SHA-256 of the canonical content, hex-encoded and prefixed.
	Digest string `json:"digest"`
}

// Canon is one version of the overlay: its files and its own identity.
type Canon struct {
	// ID identifies the version as a whole, derived from the per-file digests so
	// that one string answers "which overlay is this".
	ID string `json:"id"`

	Files []CanonFile `json:"files"`
}

// Digest returns the canonical digest of one path, and whether the canon
// carries that path at all.
func (canon Canon) Digest(relativePath string) (string, bool) {
	for _, file := range canon.Files {
		if file.Path == relativePath {
			return file.Digest, true
		}
	}
	return "", false
}

// Paths returns every path the canon defines, in stable order.
func (canon Canon) Paths() []string {
	paths := make([]string, 0, len(canon.Files))
	for _, file := range canon.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

// Content returns the canonical bytes for one path.
func Content(relativePath string) ([]byte, error) {
	embedded, err := embeddedPath(relativePath)
	if err != nil {
		return nil, err
	}
	return canonFS.ReadFile(embedded)
}

// CurrentCanon is the overlay this binary was built with.
func CurrentCanon() (Canon, error) {
	entries, err := collectCanonPaths()
	if err != nil {
		return Canon{}, err
	}
	files := make([]CanonFile, 0, len(entries))
	for _, relativePath := range entries {
		content, readErr := Content(relativePath)
		if readErr != nil {
			return Canon{}, readErr
		}
		files = append(files, CanonFile{Path: relativePath, Digest: Digest(content)})
	}
	return Canon{ID: canonIdentity(files), Files: files}, nil
}

// Digest is the one hashing rule this package uses, so a digest printed by a
// report and a digest compared by a check can never disagree.
func Digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// canonIdentity derives a canon's own identity from its files. Deriving it
// rather than declaring it means a canon cannot claim an identity its content
// does not support.
func canonIdentity(files []CanonFile) string {
	ordered := make([]CanonFile, len(files))
	copy(ordered, files)
	sort.Slice(ordered, func(first, second int) bool {
		return ordered[first].Path < ordered[second].Path
	})
	var builder strings.Builder
	for _, file := range ordered {
		builder.WriteString(file.Path)
		builder.WriteString(" ")
		builder.WriteString(file.Digest)
		builder.WriteString("\n")
	}
	return Digest([]byte(builder.String()))
}

// collectCanonPaths walks the embedded tree and returns repository-relative
// paths in stable order.
func collectCanonPaths() ([]string, error) {
	var found []string
	err := fs.WalkDir(canonFS, canonRoot, func(walked string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, relErr := repositoryPath(walked)
		if relErr != nil {
			return relErr
		}
		found = append(found, relative)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read embedded canon: %w", err)
	}
	if len(found) == 0 {
		// A build that embedded nothing would otherwise install an empty overlay
		// and report success.
		return nil, errors.New("embedded canon is empty")
	}
	sort.Strings(found)
	return found, nil
}

// repositoryPath converts an embedded path into the repository-relative path the
// file is materialized at.
func repositoryPath(embedded string) (string, error) {
	trimmed := strings.TrimPrefix(embedded, canonRoot+"/")
	if trimmed == embedded {
		return "", fmt.Errorf("embedded path %q is outside the canon", embedded)
	}
	return path.Join(OverlayDirectory, trimmed), nil
}

// embeddedPath is the inverse of repositoryPath.
func embeddedPath(relativePath string) (string, error) {
	trimmed := strings.TrimPrefix(relativePath, OverlayDirectory+"/")
	if trimmed == relativePath {
		return "", fmt.Errorf("path %q is not part of the overlay", relativePath)
	}
	return path.Join(canonRoot, trimmed), nil
}
