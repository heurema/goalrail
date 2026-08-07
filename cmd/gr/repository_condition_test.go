package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Every command that resolves a repository states the condition itself.
//
// The rule is a property of the command surface, not of the commands that have
// been corrected: three of eight relayed Git's own message and exit status, and
// the one that did not behaved correctly because it had been repaired rather
// than because anything required it. A survey rather than three separate cases,
// so a command added later is covered without being remembered.
func TestNoCommandRelaysAForeignFailure(t *testing.T) {
	outside := t.TempDir()
	for _, command := range [][]string{
		{"init", "--repo", outside},
		{"migrate", "--repo", outside},
		{"update", "--repo", outside},
		{"doctor", "--repo", outside},
	} {
		t.Run(command[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(t.Context(), command, strings.NewReader(""), &stdout, &stderr, productionService)

			said := stdout.String() + stderr.String()
			if err != nil {
				said += err.Error()
			}
			for _, foreign := range []string{"fatal:", "exit status"} {
				if strings.Contains(said, foreign) {
					t.Fatalf("%s relays what another program said: %q", command[0], said)
				}
			}
			if !strings.Contains(said, "not inside a Git repository") && !strings.Contains(said, "unmanaged") {
				t.Fatalf("%s did not name the condition: %q", command[0], said)
			}
		})
	}
}

// Naming the condition must not become a way of losing a failure. A discovery
// that could not run at all is a different answer from a path that is simply not
// a repository, and a caller reads the difference.
func TestABrokenDiscoveryIsNotReportedAsAnAbsentRepository(t *testing.T) {
	outside := t.TempDir()
	t.Setenv("PATH", t.TempDir())

	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"init", "--repo", outside}, strings.NewReader(""), &stdout, &stderr, productionService)
	if err == nil {
		t.Fatal("initialization succeeded with no Git available")
	}
	if strings.Contains(err.Error(), "not inside a Git repository") {
		t.Fatalf("a discovery that could not run was reported as an absent repository: %v", err)
	}
}

// A repository with no work tree keeps its own sentence, which the condition
// above must not swallow.
func TestARepositoryWithNoWorkTreeKeepsItsOwnCondition(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "-C", bare, "init", "--bare", "-q")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, output)
	}

	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"update", "--repo", bare}, strings.NewReader(""), &stdout, &stderr, productionService)
	if err == nil || !strings.Contains(err.Error(), "no work tree") {
		t.Fatalf("a bare repository lost its own condition: %v", err)
	}
}
