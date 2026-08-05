package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	projectdoctor "github.com/heurema/goalrail/internal/doctor"
	projectstate "github.com/heurema/goalrail/internal/project"
	"github.com/heurema/goalrail/internal/review"
)

type forbiddenTerminal struct{ reads int }

func (terminal *forbiddenTerminal) Read([]byte) (int, error) {
	terminal.reads++
	return 0, errors.New("terminal input is forbidden")
}

func runThroughDispatcher(t *testing.T, terminal io.Reader, arguments ...string) ([]byte, []byte, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(
		context.Background(),
		arguments,
		terminal,
		&stdout,
		&stderr,
		productionService,
	)
	return stdout.Bytes(), stderr.Bytes(), err
}

func publicReviewRepository(t *testing.T) (root, base string) {
	t.Helper()
	root = scratchRepository(t)
	writeFile(t, filepath.Join(root, "README.md"), "base\n")
	gitCommand(t, root, "add", "README.md")
	gitCommand(t, root, "commit", "-m", "base")
	gitCommand(t, root, "branch", "-M", "main")
	base = strings.TrimSpace(gitOutput(t, root, "rev-parse", "HEAD"))
	gitCommand(t, root, "update-ref", "refs/remotes/origin/main", base)
	gitCommand(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	gitCommand(t, root, "switch", "-c", "work")
	writeFile(t, filepath.Join(root, "feature.txt"), "first\n")
	gitCommand(t, root, "add", "feature.txt")
	gitCommand(t, root, "commit", "-m", "feature")
	return root, base
}

func installClaudeReviewStub(t *testing.T, script string) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "claude")
	writeFile(t, path, "#!/bin/sh\n"+script)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func doctorExitCode(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return -1
}

func doctorThroughDispatcher(t *testing.T, terminal io.Reader, root, stateRoot string) (projectdoctor.Diagnosis, int) {
	t.Helper()
	stdout, _, err := runThroughDispatcher(t, terminal,
		"doctor", "--repo", root, "--state-dir", stateRoot,
		"--scaffold", "claude-code", "--json",
	)
	var diagnosis projectdoctor.Diagnosis
	if decodeErr := json.Unmarshal(stdout, &diagnosis); decodeErr != nil {
		t.Fatalf("doctor did not return JSON: %v\n%s", decodeErr, stdout)
	}
	return diagnosis, doctorExitCode(err)
}

func TestPublicReviewFlowNeedsNoTerminalAndKeepsDoctorAdvisory(t *testing.T) {
	root, _ := publicReviewRepository(t)
	home, stateRoot := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_THREAD_ID", "desktop-thread")
	t.Setenv("CODEX_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")
	providerLog := filepath.Join(t.TempDir(), "provider.log")
	t.Setenv("PROVIDER_LOG", providerLog)
	installClaudeReviewStub(t, `printf 'called\n' >>"$PROVIDER_LOG"; cat >/dev/null; printf 'no material findings\n'`)
	terminal := &forbiddenTerminal{}

	initialized, _, err := runThroughDispatcher(t, terminal,
		"init", "--repo", root, "--scaffold", "claude-code",
	)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	var initResult projectstate.InitializeReport
	if err := json.Unmarshal(initialized, &initResult); err != nil {
		t.Fatalf("init did not return JSON: %v\n%s", err, initialized)
	}

	firstOutput, _, err := runThroughDispatcher(t, terminal,
		"review", "--repo", root, "--state-dir", stateRoot, "--deadline", "30s",
	)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	var first reviewReport
	if err := json.Unmarshal(firstOutput, &first); err != nil {
		t.Fatalf("review did not return JSON: %v\n%s", err, firstOutput)
	}
	if first.Author != string(ambient.ScaffoldCodex) ||
		first.Reviewer != string(ambient.ScaffoldClaudeCode) ||
		first.Mode != "cross" || first.BaseRef != "origin/main" {
		t.Fatalf("desktop detection or omitted-base routing was wrong: %+v", first)
	}

	current, currentExit := doctorThroughDispatcher(t, terminal, root, stateRoot)
	if current.Review.State != string(review.StateCurrent) {
		t.Fatalf("review did not become current: %+v", current.Review)
	}

	writeFile(t, filepath.Join(root, "after-review.txt"), "later\n")
	gitCommand(t, root, "add", "after-review.txt")
	gitCommand(t, root, "commit", "-m", "advance reviewed branch")
	stale, staleExit := doctorThroughDispatcher(t, terminal, root, stateRoot)
	if stale.Review.State != string(review.StateStale) {
		t.Fatalf("new commit did not make the review stale: %+v", stale.Review)
	}

	secondOutput, _, err := runThroughDispatcher(t, terminal,
		"review", "--repo", root, "--state-dir", stateRoot, "--deadline", "30s",
	)
	if err != nil {
		t.Fatalf("re-review: %v", err)
	}
	var second reviewReport
	if err := json.Unmarshal(secondOutput, &second); err != nil {
		t.Fatalf("re-review did not return JSON: %v\n%s", err, secondOutput)
	}
	recurrent, recurrentExit := doctorThroughDispatcher(t, terminal, root, stateRoot)
	if recurrent.Review.State != string(review.StateCurrent) {
		t.Fatalf("re-review did not become current: %+v", recurrent.Review)
	}

	if err := os.WriteFile(second.Receipt, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	unreadable, unreadableExit := doctorThroughDispatcher(t, terminal, root, stateRoot)
	if unreadable.Review.State != "unreadable" {
		t.Fatalf("corrupt receipt was not reported unreadable: %+v", unreadable.Review)
	}

	for name, snapshot := range map[string]struct {
		working bool
		exit    int
	}{
		"stale":      {stale.Working, staleExit},
		"recurrent":  {recurrent.Working, recurrentExit},
		"unreadable": {unreadable.Working, unreadableExit},
	} {
		if snapshot.working != current.Working || snapshot.exit != currentExit {
			t.Fatalf("%s review state changed doctor verdict/exit: current=(%v,%d) got=(%v,%d)",
				name, current.Working, currentExit, snapshot.working, snapshot.exit)
		}
	}
	if terminal.reads != 0 {
		t.Fatalf("public flow tried to read the terminal %d time(s)", terminal.reads)
	}
	providerCalls, err := os.ReadFile(providerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(providerCalls), "called\n") != 2 {
		t.Fatalf("provider calls = %q, want exactly two reviews", providerCalls)
	}
}

func TestPublicReviewRejectsMalformedAndAmbiguousBaseBeforeSideEffects(t *testing.T) {
	for _, test := range []struct {
		name      string
		malformed bool
	}{
		{name: "malformed explicit", malformed: true},
		{name: "ambiguous omitted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, base := publicReviewRepository(t)
			if !test.malformed {
				gitCommand(t, root, "update-ref", "refs/remotes/upstream/main", base)
				gitCommand(t, root, "symbolic-ref", "refs/remotes/upstream/HEAD", "refs/remotes/upstream/main")
			}
			t.Setenv("CODEX_THREAD_ID", "desktop-thread")
			t.Setenv("CODEX_SESSION_ID", "")
			t.Setenv("CLAUDECODE", "")
			providerMarker, gateMarker := filepath.Join(t.TempDir(), "provider"), filepath.Join(t.TempDir(), "gate")
			t.Setenv("PROVIDER_MARKER", providerMarker)
			t.Setenv("GATE_MARKER", gateMarker)
			installClaudeReviewStub(t, `touch "$PROVIDER_MARKER"; cat >/dev/null; echo report`)
			terminal := &forbiddenTerminal{}
			arguments := []string{
				"review", "--repo", root, "--state-dir", t.TempDir(),
				"--gate", `touch "$GATE_MARKER"`,
			}
			if test.malformed {
				arguments = append(arguments, "--base=")
			}
			_, _, err := runThroughDispatcher(t, terminal, arguments...)
			if err == nil || !strings.Contains(err.Error(), "--base") {
				t.Fatalf("input error = %v", err)
			}
			if terminal.reads != 0 {
				t.Fatalf("input error tried to prompt %d time(s)", terminal.reads)
			}
			for _, path := range []string{
				providerMarker,
				gateMarker,
				filepath.Join(root, review.InstructionsPath),
			} {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("invalid input created %s: %v", path, statErr)
				}
			}
		})
	}
}

func TestExplicitAuthorOverrideWinsAndInvalidValuesFail(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "codex-thread")
	t.Setenv("CLAUDECODE", "claude-session")
	for _, test := range []struct {
		value string
		want  ambient.Scaffold
	}{
		{value: "codex", want: ambient.ScaffoldCodex},
		{value: "claude-code", want: ambient.ScaffoldClaudeCode},
	} {
		got, err := resolveAuthor(test.value)
		if err != nil || got != test.want {
			t.Fatalf("override %q = %q, %v; want %q", test.value, got, err, test.want)
		}
	}
	if _, err := resolveAuthor("clade-code"); err == nil {
		t.Fatal("invalid explicit author was accepted")
	}
}
