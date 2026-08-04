package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

func setRemoteDefault(t *testing.T, root, remote, branch, commit string) {
	t.Helper()
	if _, err := git(root, "update-ref", "refs/remotes/"+remote+"/"+branch, commit); err != nil {
		t.Fatal(err)
	}
	if _, err := git(root, "symbolic-ref", "refs/remotes/"+remote+"/HEAD", "refs/remotes/"+remote+"/"+branch); err != nil {
		t.Fatal(err)
	}
}

func TestResolveBasePrefersTheRemoteDefaultOverItsLocalShadow(t *testing.T) {
	root := repository(t)
	remoteCommit, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	setRemoteDefault(t, root, "origin", "main", remoteCommit)

	write(t, root, "local-only.txt", "local\n")
	commit(t, root, "move local main")
	localCommit, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	if localCommit == remoteCommit {
		t.Fatal("the fixture did not create a divergent local shadow")
	}

	resolved, err := ResolveBase(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Ref != "origin/main" || resolved.Commit != remoteCommit {
		t.Fatalf("resolved %+v, want origin/main at %s", resolved, remoteCommit)
	}
}

func TestResolveBaseMakesAnExplicitRefAuthoritative(t *testing.T) {
	root := repository(t)
	remoteCommit, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	setRemoteDefault(t, root, "origin", "main", remoteCommit)
	write(t, root, "local-only.txt", "local\n")
	commit(t, root, "move local main")
	localCommit, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveBase(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Ref != "main" || resolved.Commit != localCommit {
		t.Fatalf("resolved %+v, want explicit main at %s", resolved, localCommit)
	}
}

func TestResolveBaseRefusesAbsentAmbiguousAndInvalidDefaults(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		_, err := ResolveBase(repository(t), "")
		if err == nil || !strings.Contains(err.Error(), "--base") {
			t.Fatalf("absent metadata produced %v", err)
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
		root := repository(t)
		commit, err := Resolve(root, "main")
		if err != nil {
			t.Fatal(err)
		}
		setRemoteDefault(t, root, "origin", "main", commit)
		setRemoteDefault(t, root, "upstream", "main", commit)
		_, err = ResolveBase(root, "")
		if err == nil || !strings.Contains(err.Error(), "found 2") || !strings.Contains(err.Error(), "--base") {
			t.Fatalf("ambiguous metadata produced %v", err)
		}
	})

	t.Run("invalid explicit", func(t *testing.T) {
		_, err := ResolveBase(repository(t), "does-not-exist")
		if err == nil || !strings.Contains(err.Error(), "explicit --base") {
			t.Fatalf("invalid explicit base produced %v", err)
		}
	})
}

func TestResolveBaseStaysOfflineWithAnUnreachableRemote(t *testing.T) {
	root := repository(t)
	commit, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	setRemoteDefault(t, root, "origin", "main", commit)
	if _, err := git(root, "remote", "add", "origin", "ssh://unreachable.invalid/repository"); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(t.TempDir(), "network-attempted")
	ssh := filepath.Join(t.TempDir(), "ssh")
	if err := os.WriteFile(ssh, []byte("#!/bin/sh\ntouch \""+marker+"\"\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", ssh)

	resolved, err := ResolveBase(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Ref != "origin/main" || resolved.Commit != commit {
		t.Fatalf("resolved %+v", resolved)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("base discovery attempted network access: %v", err)
	}
}

func TestInvalidBaseHasNoMaterializationGateOrReviewerSideEffects(t *testing.T) {
	for _, test := range []struct {
		name      string
		explicit  string
		ambiguous bool
	}{
		{name: "invalid explicit", explicit: "does-not-exist"},
		{name: "ambiguous omitted", ambiguous: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := branchWithWork(t)
			if test.ambiguous {
				base, err := Resolve(root, "main")
				if err != nil {
					t.Fatal(err)
				}
				setRemoteDefault(t, root, "origin", "main", base)
				setRemoteDefault(t, root, "upstream", "main", base)
			}

			gateMarker := filepath.Join(t.TempDir(), "gate-ran")
			reviewerMarker := filepath.Join(t.TempDir(), "reviewer-ran")
			t.Setenv("REVIEWER_MARKER", reviewerMarker)
			stubReviewer(t, "codex", `touch "$REVIEWER_MARKER"; cat >/dev/null; echo report`)

			_, err := Run(context.Background(), Input{
				RepositoryRoot: root,
				StateRoot:      t.TempDir(),
				BaseRef:        test.explicit,
				Selection: Selection{
					Reviewer: ambient.ScaffoldCodex,
					Mode:     "cross",
					Reason:   "test",
				},
				Gate: "touch " + gateMarker,
			})
			if err == nil || !strings.Contains(err.Error(), "--base") {
				t.Fatalf("invalid base produced %v", err)
			}
			for _, path := range []string{
				filepath.Join(root, InstructionsPath),
				gateMarker,
				reviewerMarker,
			} {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("invalid input created %s: %v", path, statErr)
				}
			}
		})
	}
}
