package localrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultMaxObservedPaths  = 256
	DefaultMaxGitStatusBytes = 1 << 20
	defaultMaxGitOutputBytes = 64 << 10
	defaultMaxGitErrorBytes  = 16 << 10
)

var ErrWorktreeObservationTooLarge = errors.New("worktree observation exceeds configured bound")

type GitObserver struct {
	MaxPaths       int
	MaxStatusBytes int
}

func (observer GitObserver) ResolveRepository(
	ctx context.Context,
	requestedRoot string,
	baseRevision string,
) (string, string, error) {
	root, err := filepath.Abs(requestedRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository root symlinks: %w", err)
	}
	root = filepath.Clean(root)
	top, err := observer.git(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	top, err = filepath.EvalSymlinks(strings.TrimSpace(top))
	if err != nil {
		return "", "", fmt.Errorf("resolve Git top level: %w", err)
	}
	top = filepath.Clean(top)
	if top != root {
		return "", "", fmt.Errorf("declared repository root %q is not Git top level %q", root, top)
	}

	resolvedRevision, err := observer.git(ctx, root, "rev-parse", "--verify", baseRevision+"^{commit}")
	if err != nil {
		return "", "", fmt.Errorf("resolve pinned revision: %w", err)
	}
	resolvedRevision = strings.ToLower(strings.TrimSpace(resolvedRevision))
	head, err := observer.git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	if strings.ToLower(strings.TrimSpace(head)) != resolvedRevision {
		return "", "", fmt.Errorf("repository HEAD does not match pinned base revision")
	}
	return root, resolvedRevision, nil
}

func (observer GitObserver) Observe(ctx context.Context, root string) (WorktreeObservation, error) {
	head, err := observer.git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return WorktreeObservation{}, err
	}
	raw, err := observer.gitStatusBytes(
		ctx,
		root,
		"-c",
		"status.renames=false",
		"status",
		"--porcelain=v1",
		"-z",
		"--untracked-files=all",
	)
	if err != nil {
		return WorktreeObservation{}, err
	}
	ignoredRaw, err := observer.gitStatusBytes(
		ctx,
		root,
		"ls-files",
		"-z",
		"--others",
		"--ignored",
		"--exclude-standard",
	)
	if err != nil {
		return WorktreeObservation{}, err
	}
	entries, err := observer.parseEntries(root, raw, ignoredRaw)
	if err != nil {
		return WorktreeObservation{}, err
	}
	observation := WorktreeObservation{
		Head:    strings.ToLower(strings.TrimSpace(head)),
		Entries: entries,
	}
	digest, err := observationDigest(observation)
	if err != nil {
		return WorktreeObservation{}, err
	}
	observation.Digest = digest
	return observation, nil
}

func CompareWorktrees(
	baseline WorktreeObservation,
	terminal WorktreeObservation,
	scope []string,
) WorktreeDelta {
	before := make(map[string]WorktreeEntry, len(baseline.Entries))
	after := make(map[string]WorktreeEntry, len(terminal.Entries))
	paths := make(map[string]struct{}, len(baseline.Entries)+len(terminal.Entries))
	for _, entry := range baseline.Entries {
		before[entry.Path] = entry
		paths[entry.Path] = struct{}{}
	}
	for _, entry := range terminal.Entries {
		after[entry.Path] = entry
		paths[entry.Path] = struct{}{}
	}

	changed := make([]string, 0)
	violations := make([]string, 0)
	for path := range paths {
		if before[path] == after[path] {
			continue
		}
		changed = append(changed, path)
		if !pathInScope(path, scope) {
			violations = append(violations, path)
		}
	}
	sort.Strings(changed)
	sort.Strings(violations)
	return WorktreeDelta{
		BaselineDigest:  baseline.Digest,
		TerminalDigest:  terminal.Digest,
		HeadChanged:     baseline.Head != terminal.Head,
		ChangedPaths:    changed,
		ScopeViolations: violations,
	}
}

func (observer GitObserver) parseEntries(
	root string,
	statusRaw []byte,
	ignoredRaw []byte,
) ([]WorktreeEntry, error) {
	maximum := observer.MaxPaths
	if maximum <= 0 {
		maximum = DefaultMaxObservedPaths
	}
	entries := make([]WorktreeEntry, 0, maximum)
	seen := make(map[string]struct{}, maximum)
	appendEntry := func(status, path string) error {
		path = filepath.ToSlash(path)
		if path == "" || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return fmt.Errorf("Git reported an invalid repository path")
		}
		if _, exists := seen[path]; exists {
			return nil
		}
		if len(entries) >= maximum {
			return fmt.Errorf(
				"%w: dirty path count exceeds %d",
				ErrWorktreeObservationTooLarge,
				maximum,
			)
		}
		seen[path] = struct{}{}
		entry := WorktreeEntry{
			Path:   path,
			Status: status,
		}
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(fullPath)
		if err == nil {
			entry.Mode = uint32(info.Mode())
			entry.Digest, err = hashPath(fullPath, info)
			if err != nil {
				return fmt.Errorf("hash worktree path %q: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect worktree path %q: %w", path, err)
		}
		entries = append(entries, entry)
		return nil
	}

	for _, record := range bytes.Split(statusRaw, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("invalid Git status record")
		}
		if err := appendEntry(string(record[:2]), string(record[3:])); err != nil {
			return nil, err
		}
	}
	for _, record := range bytes.Split(ignoredRaw, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if err := appendEntry("!!", string(record)); err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Path < entries[right].Path
	})
	return entries, nil
}

func (observer GitObserver) git(ctx context.Context, root string, arguments ...string) (string, error) {
	output, err := observer.gitBytes(ctx, root, arguments...)
	return string(output), err
}

func (observer GitObserver) gitBytes(
	ctx context.Context,
	root string,
	arguments ...string,
) ([]byte, error) {
	return observer.runGitBytes(ctx, root, defaultMaxGitOutputBytes, arguments...)
}

func (observer GitObserver) gitStatusBytes(
	ctx context.Context,
	root string,
	arguments ...string,
) ([]byte, error) {
	maximum := observer.MaxStatusBytes
	if maximum <= 0 {
		maximum = DefaultMaxGitStatusBytes
	}
	return observer.runGitBytes(ctx, root, maximum, arguments...)
}

func (GitObserver) runGitBytes(
	ctx context.Context,
	root string,
	maximum int,
	arguments ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, arguments...)...)
	stdout := boundedBuffer{maximum: maximum}
	stderr := boundedBuffer{maximum: defaultMaxGitErrorBytes}
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	if stdout.exceeded {
		return nil, fmt.Errorf(
			"%w: Git output exceeds %d bytes",
			ErrWorktreeObservationTooLarge,
			maximum,
		)
	}
	if stderr.exceeded {
		return nil, fmt.Errorf("read Git state: error output exceeds %d bytes", defaultMaxGitErrorBytes)
	}
	if runErr != nil {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = runErr.Error()
		}
		return nil, fmt.Errorf("read Git state: %s", reason)
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	maximum  int
	exceeded bool
}

func (buffer *boundedBuffer) Write(chunk []byte) (int, error) {
	received := len(chunk)
	remaining := buffer.maximum - buffer.buffer.Len()
	if remaining > 0 {
		if remaining < len(chunk) {
			_, _ = buffer.buffer.Write(chunk[:remaining])
		} else {
			_, _ = buffer.buffer.Write(chunk)
		}
	}
	if len(chunk) > remaining {
		buffer.exceeded = true
	}
	return received, nil
}

func (buffer *boundedBuffer) Bytes() []byte {
	return buffer.buffer.Bytes()
}

func (buffer *boundedBuffer) String() string {
	return buffer.buffer.String()
}

func hashPath(path string, info os.FileInfo) (string, error) {
	hasher := sha256.New()
	switch {
	case info.Mode().IsRegular():
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(hasher, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(hasher, target); err != nil {
			return "", err
		}
	default:
		if _, err := io.WriteString(hasher, info.Mode().String()); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func observationDigest(observation WorktreeObservation) (string, error) {
	observation.Digest = ""
	encoded, err := json.Marshal(observation)
	if err != nil {
		return "", fmt.Errorf("encode worktree observation: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func pathInScope(path string, scope []string) bool {
	path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	for _, candidate := range scope {
		candidate = filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate)))
		if candidate == "." || path == candidate || strings.HasPrefix(path, candidate+"/") {
			return true
		}
	}
	return false
}
