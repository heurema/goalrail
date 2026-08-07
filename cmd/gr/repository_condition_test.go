package main

import (
	"bytes"
	"errors"
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
	// Both shapes of failure Git can hand back after actually running. The
	// first version of this survey used only an absent repository, and a
	// repository Git ran against and refused — the disputed-ownership case a
	// shared or container-mounted checkout produces — still relayed `fatal:`,
	// its exit status, and the `git config` line it suggests pasting.
	for _, condition := range []struct {
		name  string
		repo  func(*testing.T) string
		named []string
	}{
		{
			name:  "absent repository",
			repo:  func(t *testing.T) string { return t.TempDir() },
			named: []string{"not inside a Git repository", "unmanaged"},
		},
		{
			name:  "repository Git refuses",
			repo:  refusedRepository,
			named: []string{"Git refused to resolve"},
		},
	} {
		t.Run(condition.name, func(t *testing.T) {
			repository := condition.repo(t)
			// The seam is command and subcommand: `setup plan` resolves a
			// repository as surely as `init` does, and a survey that stopped at
			// the top level passed while it still relayed.
			for _, command := range [][]string{
				{"init", "--repo", repository},
				{"migrate", "--repo", repository},
				{"update", "--repo", repository},
				{"doctor", "--repo", repository},
				{"setup", "plan", "--repo", repository},
			} {
				t.Run(strings.Join(command[:len(command)-2], " "), func(t *testing.T) {
					var stdout, stderr bytes.Buffer
					err := run(t.Context(), command, strings.NewReader(""), &stdout, &stderr, productionService)

					said := stdout.String() + stderr.String()
					if err != nil {
						said += err.Error()
					}
					for _, foreign := range []string{"fatal:", "exit status", "git config"} {
						if strings.Contains(said, foreign) {
							t.Fatalf("%v relays what another program said: %q", command, said)
						}
					}
					if !containsAny(said, condition.named) {
						t.Fatalf("%v did not name the condition: %q", command, said)
					}
				})
			}
		})
	}
}

// refusedRepository is a repository Git runs against and declines to resolve.
// Disputed ownership is the refusal users actually meet, but it needs a second
// account to stage; an unreadable format version reaches the same branch — Git
// ran, and said no for a reason that is not the path being outside a
// repository — without a privileged test.
func refusedRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "core.repositoryformatversion", "99"},
	} {
		command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	return repository
}

func containsAny(said string, phrases []string) bool {
	for _, phrase := range phrases {
		if strings.Contains(said, phrase) {
			return true
		}
	}
	return false
}

// Naming the condition must not become a way of losing a failure. A discovery
// that could not run at all is a different answer from a path that is simply not
// a repository, and a caller reads the difference.
func TestABrokenDiscoveryIsNotReportedAsAnAbsentRepository(t *testing.T) {
	outside := t.TempDir()
	// The absent-repository answer is taken first, while Git is still available:
	// with it gone, every run produces the other outcome and the comparison
	// would compare a thing with itself.
	absent := absentRepositoryError(t, outside)

	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"init", "--repo", outside}, strings.NewReader(""), &stdout, &stderr, productionService)
	if err == nil {
		t.Fatal("initialization succeeded with no Git available")
	}
	if strings.Contains(err.Error(), "not inside a Git repository") {
		t.Fatalf("a discovery that could not run was reported as an absent repository: %v", err)
	}

	// Wording alone is half the requirement. A caller reading only the status
	// must be able to tell the two apart, and the first version of this change
	// left both at the same code while its test checked only the sentence.
	if exitStatusOf(err) == exitStatusOf(absent) {
		t.Fatalf("a broken discovery and an absent repository share exit status %d", exitStatusOf(err))
	}
}

func absentRepositoryError(t *testing.T, outside string) error {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"init", "--repo", outside}, strings.NewReader(""), &stdout, &stderr, productionService)
	if err == nil {
		t.Fatal("initialization succeeded outside a repository")
	}
	return err
}

func exitStatusOf(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
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
