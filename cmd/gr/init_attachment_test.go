package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// A health report names initialization as the remedy for a disconnected
// repository-scope scaffold, so initialization has to perform it. It did not:
// nothing wrote that registration, and a user following correct advice observed
// no change and had no remaining move.
func TestInitializationRegistersAnExplicitlySelectedRepositoryScaffold(t *testing.T) {
	repository, home, executable := attachmentFixture(t)

	writes := registerRepositoryScaffolds(repository, string(ambient.ScaffoldClaudeCode), home, executable)

	if len(writes) != 1 || writes[0].Action != "registered" {
		t.Fatalf("initialization wrote no registration: %#v", writes)
	}
	if _, err := os.Stat(filepath.Join(repository, ambient.RepositorySettingsPath)); err != nil {
		t.Fatalf("the registration file is absent: %v", err)
	}
	if writes[0].Trust == "" || len(writes[0].Events) == 0 {
		t.Fatalf("the write was not disclosed with its events and trust state: %#v", writes[0])
	}
}

// The registration must not be something a commit could hand to a teammate:
// trust is a standing consent to run a command in one's own sessions, and one
// person cannot give it for everyone.
func TestTheRegistrationIsMadeUnshareableByThisCloneAlone(t *testing.T) {
	repository, home, executable := attachmentFixture(t)

	writes := registerRepositoryScaffolds(repository, string(ambient.ScaffoldClaudeCode), home, executable)

	if len(writes) != 1 || len(writes[0].IgnoreEntries) == 0 {
		t.Fatalf("no ignore entry was recorded: %#v", writes)
	}
	if ignored, err := ambient.IgnoreState(repository, ambient.RepositorySettingsPath); err != nil || !ignored {
		t.Fatalf("the registration path is not ignored: ignored=%t err=%v", ignored, err)
	}
	// The rule lives in this clone's own exclude file, which no commit carries,
	// rather than in a shared ignore file the repository would ship.
	if _, err := os.Stat(filepath.Join(repository, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("a shared ignore file was created or changed: %v", err)
	}
	exclude, err := os.ReadFile(filepath.Join(repository, ".git", "info", "exclude"))
	if err != nil || !strings.Contains(string(exclude), ambient.RepositorySettingsPath) {
		t.Fatalf("the clone's own rule does not carry the path: %v", err)
	}
}

// A machine with no supported scaffold has its configuration left alone rather
// than guessed at.
func TestInitializationWritesNothingForAScaffoldThatIsNotPresent(t *testing.T) {
	repository, home, executable := attachmentFixture(t)

	if writes := registerRepositoryScaffolds(repository, "", home, executable); len(writes) != 0 {
		t.Fatalf("an absent scaffold was configured anyway: %#v", writes)
	}
	if _, err := os.Stat(filepath.Join(repository, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("scaffold configuration was created: %v", err)
	}
}

// A scaffold this machine actually carries is registered without being named.
func TestInitializationRegistersADetectedScaffold(t *testing.T) {
	repository, home, executable := attachmentFixture(t)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	writes := registerRepositoryScaffolds(repository, "", home, executable)

	if len(writes) != 1 || writes[0].Action != "registered" {
		t.Fatalf("a detected scaffold was not registered: %#v", writes)
	}
}

// Repeating initialization leaves a correct registration exactly as it is.
func TestRepeatingInitializationLeavesTheRegistrationAlone(t *testing.T) {
	repository, home, executable := attachmentFixture(t)
	registerRepositoryScaffolds(repository, string(ambient.ScaffoldClaudeCode), home, executable)
	before, err := os.ReadFile(filepath.Join(repository, ambient.RepositorySettingsPath))
	if err != nil {
		t.Fatal(err)
	}

	writes := registerRepositoryScaffolds(repository, string(ambient.ScaffoldClaudeCode), home, executable)

	if len(writes) != 1 || writes[0].Action != "unchanged" {
		t.Fatalf("a correct registration was rewritten: %#v", writes)
	}
	after, err := os.ReadFile(filepath.Join(repository, ambient.RepositorySettingsPath))
	if err != nil || string(before) != string(after) {
		t.Fatalf("the registration is not byte-identical after a second run: %v", err)
	}
}

func attachmentFixture(t *testing.T) (repository, home, executable string) {
	t.Helper()
	home = t.TempDir()
	repository = filepath.Join(t.TempDir(), "repository")
	if err := os.MkdirAll(repository, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{{"init", "-q"}, {"config", "core.excludesFile", os.DevNull}} {
		command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	executable = filepath.Join(t.TempDir(), "gr")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repository, home, executable
}
