// Package admissionlocal renders replaceable local feedback adapters. The
// adapters delegate to Goalrail commands and deliberately own no policy or
// admission decision logic.
package admissionlocal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	WorkUnitTrailer      = "Goalrail-Work-Unit"
	PrekAdapterID        = "prek"
	PrekAdapterVersion   = "1"
	PrekAdapterSourceRef = "builtin:local-feedback-prek-v1"
	maxCommitMessage     = 128 << 10
)

var ErrTrailerMissing = errors.New("Goalrail work-unit trailer is missing")

// ValidateCommitMessage returns the trailer's work-unit ID. It is an early
// index only: callers must still run the canonical lineage verifier.
func ValidateCommitMessage(reader io.Reader) (domain.WorkUnitID, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxCommitMessage+1))
	if err != nil {
		return "", fmt.Errorf("read commit message: %w", err)
	}
	if len(raw) > maxCommitMessage {
		return "", fmt.Errorf("commit message exceeds %d bytes", maxCommitMessage)
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return "", errors.New("commit message contains NUL")
	}

	lines := splitLines(string(raw))
	start := trailerBlockStart(lines)
	var found domain.WorkUnitID
	for _, line := range lines[start:] {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != WorkUnitTrailer {
			continue
		}
		candidate := domain.WorkUnitID(strings.TrimSpace(value))
		if !domain.IsCanonicalID(string(candidate)) || !strings.HasPrefix(string(candidate), "wu_") {
			return "", errors.New("Goalrail work-unit trailer is not a canonical work-unit ID")
		}
		if found != "" {
			return "", errors.New("Goalrail work-unit trailer is duplicated")
		}
		found = candidate
	}
	if found == "" {
		return "", ErrTrailerMissing
	}
	return found, nil
}

func splitLines(message string) []string {
	scanner := bufio.NewScanner(strings.NewReader(strings.ReplaceAll(message, "\r\n", "\n")))
	scanner.Buffer(make([]byte, 1024), maxCommitMessage)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trailerBlockStart(lines []string) int {
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) == "" {
			return index + 1
		}
	}
	return 0
}

type Shims struct {
	Doctor    []byte
	Verify    []byte
	CommitMsg []byte
}

// RenderShims returns POSIX shims that only exec diagnosis, the public
// verifier, or the trailer validator. The executable is fixed at setup-plan
// time; shell metacharacters are refused rather than escaped heuristically.
func RenderShims(executable string) (Shims, error) {
	if !filepath.IsAbs(executable) || filepath.Clean(executable) != executable || !safeShellWord(executable) {
		return Shims{}, errors.New("local shims require a normalized absolute executable without shell metacharacters")
	}
	return Shims{
		Doctor:    []byte("#!/bin/sh\nset -eu\nexec " + executable + " doctor --repo \"${1:-.}\" --json\n"),
		Verify:    []byte("#!/bin/sh\nset -eu\nexec " + executable + " verify-lineage \"$@\"\n"),
		CommitMsg: []byte("#!/bin/sh\nset -eu\nexec " + executable + " commit-msg --file \"$1\"\n"),
	}, nil
}

func safeShellWord(value string) bool {
	for _, character := range value {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/._-", character) {
			return false
		}
	}
	return value != ""
}

// PrekTemplateDigest identifies the adapter template, not a machine-specific
// executable path rendered into it.
func PrekTemplateDigest() domain.SHA256Digest {
	return domain.DigestCanonicalJSON([]byte(prekTemplate))
}

// RenderPrekAdapter emits nothing unless the committed setup profile contains
// the exact current prek adapter pin. Goalrail does not install or execute prek.
func RenderPrekAdapter(profile domain.SetupProfile, executable string) ([]byte, bool, error) {
	var selected *domain.SetupAdapterPin
	for index := range profile.ScaffoldAdapters {
		if profile.ScaffoldAdapters[index].ID == PrekAdapterID {
			if selected != nil {
				return nil, false, errors.New("setup profile selects prek more than once")
			}
			selected = &profile.ScaffoldAdapters[index]
		}
	}
	if selected == nil {
		return nil, false, nil
	}
	if selected.Version != PrekAdapterVersion || selected.SourceRef != PrekAdapterSourceRef || selected.Integrity != PrekTemplateDigest() {
		return nil, false, errors.New("setup profile does not pin the current Goalrail prek adapter")
	}
	shims, err := RenderShims(executable)
	if err != nil {
		return nil, false, err
	}
	_ = shims
	rendered := strings.ReplaceAll(prekTemplate, "{{GR}}", executable)
	return []byte(rendered), true, nil
}

const prekTemplate = `repos:
  - repo: local
    hooks:
      - id: goalrail-diagnosis
        name: Goalrail diagnosis (advisory)
        entry: {{GR}} doctor --repo . --json
        language: system
        pass_filenames: false
        always_run: true
`
