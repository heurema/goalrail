// Package project discovers committed Goalrail project identity without using
// checkout-local markers or caller-selected alternate Git repositories.
package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrNotRepository = errors.New("not a Git repository")
	ErrNoWorktree    = errors.New("Git repository has no worktree")
)

const maxGitDiscoveryOutput = 16 << 10

// ResolveWorktreeRoot returns the physical root of the worktree containing
// start. Git repository-selection environment variables are ignored so the
// answer always describes start rather than a caller-selected alternate repo.
func ResolveWorktreeRoot(ctx context.Context, start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		return "", fmt.Errorf("resolve worktree root: empty start path")
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve worktree start: %w", err)
	}
	physicalStart, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve worktree start symlinks: %w", err)
	}
	info, err := os.Stat(physicalStart)
	if err != nil {
		return "", fmt.Errorf("stat worktree start: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("resolve worktree root: start path is not a directory")
	}

	inside, err := gitOutput(ctx, physicalStart, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		// Only Git saying so means "not a repository". Every other way this can
		// fail — Git absent, a cancelled context, a refused directory, disputed
		// ownership — is the discovery breaking, and a caller that reads the
		// distinction would otherwise be told a working machine has no
		// repository here.
		var failure *gitCommandError
		if errors.As(err, &failure) && failure.ran && mentionsNoRepository(failure.message) {
			return "", fmt.Errorf("%w: %s", ErrNotRepository, failure.message)
		}
		return "", fmt.Errorf("resolve Git worktree root: %w", err)
	}
	if strings.TrimSpace(inside) != "true" {
		return "", ErrNoWorktree
	}

	top, err := gitOutput(ctx, physicalStart, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve Git worktree top: %w", err)
	}
	top = strings.TrimSpace(top)
	if top == "" || strings.ContainsRune(top, '\x00') || strings.Contains(top, "\n") {
		return "", fmt.Errorf("resolve Git worktree top: invalid Git output")
	}
	physicalTop, err := filepath.EvalSymlinks(top)
	if err != nil {
		return "", fmt.Errorf("resolve Git worktree top symlinks: %w", err)
	}
	physicalTop = filepath.Clean(physicalTop)
	if physicalTop == string(filepath.Separator) {
		return "", fmt.Errorf("resolve Git worktree top: filesystem root is not a safe project boundary")
	}
	relative, err := filepath.Rel(physicalTop, physicalStart)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve Git worktree top: Git returned a root outside the requested directory")
	}
	return physicalTop, nil
}

func gitOutput(ctx context.Context, directory string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, arguments...)...)
	command.Env = gitDiscoveryEnvironment()
	stdout := &boundedBuffer{limit: maxGitDiscoveryOutput}
	stderr := &boundedBuffer{limit: maxGitDiscoveryOutput}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if stderr.exceeded {
			message = "Git error output exceeded the discovery bound"
		}
		var exit *exec.ExitError
		return "", &gitCommandError{err: err, message: message, ran: errors.As(err, &exit)}
	}
	if stdout.exceeded {
		return "", fmt.Errorf("Git output exceeded %d bytes", maxGitDiscoveryOutput)
	}
	return stdout.String(), nil
}

// gitCommandError separates a Git command that ran and refused from one that
// never ran. Only the first can carry Git's own verdict about the directory.
type gitCommandError struct {
	err     error
	message string
	ran     bool
}

func (failure *gitCommandError) Error() string {
	if failure.message == "" {
		return failure.err.Error()
	}
	return failure.err.Error() + ": " + failure.message
}

func (failure *gitCommandError) Unwrap() error { return failure.err }

// mentionsNoRepository matches the one verdict this package translates. The
// discovery environment pins the C locale so the phrase is Git's own English
// rather than whatever the machine is configured to print.
func mentionsNoRepository(message string) bool {
	return strings.Contains(strings.ToLower(message), "not a git repository")
}

func gitDiscoveryEnvironment() []string {
	removed := map[string]bool{
		"GIT_DIR":                          true,
		"GIT_WORK_TREE":                    true,
		"GIT_COMMON_DIR":                   true,
		"GIT_INDEX_FILE":                   true,
		"GIT_OBJECT_DIRECTORY":             true,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
		"GIT_CEILING_DIRECTORIES":          true,
		"GIT_DISCOVERY_ACROSS_FILESYSTEM":  true,
		"GIT_NAMESPACE":                    true,
		"GIT_OPTIONAL_LOCKS":               true,
	}
	environment := os.Environ()
	kept := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		injectedConfig := name == "GIT_CONFIG_COUNT" || name == "GIT_CONFIG_PARAMETERS" ||
			strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_")
		if found && (removed[name] || injectedConfig || name == "LC_ALL" || name == "LANG") {
			continue
		}
		kept = append(kept, entry)
	}
	// A pinned locale, so a refusal this package must classify is reported in
	// the words it is classified by.
	kept = append(kept, "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "LANG=C")
	return kept
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *boundedBuffer) Write(payload []byte) (int, error) {
	originalLength := len(payload)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(payload) > remaining {
			payload = payload[:remaining]
		}
		_, _ = buffer.buffer.Write(payload)
	}
	if originalLength > remaining {
		buffer.exceeded = true
	}
	return originalLength, nil
}

func (buffer *boundedBuffer) String() string {
	return buffer.buffer.String()
}
