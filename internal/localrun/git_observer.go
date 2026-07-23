package localrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultMaxObservedPaths = 256

type GitObserver struct {
	MaxPaths int
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
	raw, err := observer.gitBytes(
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
	entries, err := observer.parseEntries(root, raw)
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
		ChangedPaths:    changed,
		ScopeViolations: violations,
	}
}

func (observer GitObserver) parseEntries(root string, raw []byte) ([]WorktreeEntry, error) {
	maximum := observer.MaxPaths
	if maximum <= 0 {
		maximum = DefaultMaxObservedPaths
	}
	records := bytes.Split(raw, []byte{0})
	entries := make([]WorktreeEntry, 0, len(records))
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("invalid Git status record")
		}
		if len(entries) >= maximum {
			return nil, fmt.Errorf("dirty path count exceeds %d", maximum)
		}
		path := filepath.ToSlash(string(record[3:]))
		if path == "" || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, fmt.Errorf("Git reported an invalid repository path")
		}
		entry := WorktreeEntry{
			Path:   path,
			Status: string(record[:2]),
		}
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(fullPath)
		if err == nil {
			entry.Mode = uint32(info.Mode())
			entry.Digest, err = hashPath(fullPath, info)
			if err != nil {
				return nil, fmt.Errorf("hash worktree path %q: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect worktree path %q: %w", path, err)
		}
		entries = append(entries, entry)
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

func (GitObserver) gitBytes(ctx context.Context, root string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, arguments...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = err.Error()
		}
		return nil, fmt.Errorf("read Git state: %s", reason)
	}
	return stdout.Bytes(), nil
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
