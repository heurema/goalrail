package lineage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	projectstate "github.com/heurema/goalrail/internal/project"
)

func TestBeginCreatesCanonicalAnchorAndInitialRelationsIdempotently(t *testing.T) {
	repository := managedRepositoryFixture(t)
	_, thisFile, _, _ := runtime.Caller(0)
	sourceChange := filepath.Clean(filepath.Join(
		filepath.Dir(thisFile),
		"..", "..", "openspec", "changes", "archive", "2026-08-05-project-lineage-admission-v0",
	))
	targetChange := filepath.Join(repository, "openspec", "changes", "project-lineage-admission-v0")
	copyTree(t, sourceChange, targetChange)

	fixedTime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	options := BeginOptions{
		Repository: repository,
		ChangeID:   "project-lineage-admission-v0",
		ActorRef:   "user:test-owner",
		Now:        func() time.Time { return fixedTime },
		NewWorkUnitID: func() (domain.WorkUnitID, error) {
			return "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
		},
	}
	receipt, err := Begin(context.Background(), options)
	if err != nil {
		var chain []string
		for current := err; current != nil; current = errors.Unwrap(current) {
			chain = append(chain, current.Error())
		}
		t.Fatal(strings.Join(chain, "\ncaused by: "))
	}
	if !receipt.Created || len(receipt.EventRefs) != 3 {
		t.Fatalf("begin receipt = %+v", receipt)
	}
	rawAnchor, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(receipt.AnchorRef)))
	if err != nil {
		t.Fatal(err)
	}
	unit, err := domain.DecodeWorkUnit(bytes.NewReader(rawAnchor))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rawAnchor, artifact.CanonicalJSON()) || artifact.Digest() != receipt.AnchorDigest {
		t.Fatal("stored work-unit anchor is not the reported canonical artifact")
	}
	for _, eventRef := range receipt.EventRefs {
		raw, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(eventRef)))
		if err != nil {
			t.Fatal(err)
		}
		event, err := domain.DecodeLineageEvent(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		frozen, err := domain.FreezeLineageEvent(event)
		if err != nil || !bytes.Equal(raw, frozen.CanonicalJSON()) {
			t.Fatalf("event %s is not canonical: %v", eventRef, err)
		}
	}

	repeated, err := Begin(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Created || repeated.AnchorDigest != receipt.AnchorDigest {
		t.Fatalf("idempotent begin = %+v", repeated)
	}

	unit.CreatedAt = fixedTime.Add(time.Minute)
	store, err := NewStore(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin(unit, nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("incompatible anchor error = %v, want ErrConflict", err)
	}
}

func TestDigestChangeSnapshotIsOrderIndependentAndRejectsSymlinks(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "b.md"), []byte("b\n"))
	writeTestFile(t, filepath.Join(directory, "nested", "a.md"), []byte("a\n"))
	first, err := DigestChangeSnapshot(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(directory, "b.md"), time.Now(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	second, err := DigestChangeSnapshot(directory)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("change snapshot digest depends on filesystem metadata or traversal order")
	}
	if err := os.Symlink(filepath.Join(directory, "b.md"), filepath.Join(directory, "link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := DigestChangeSnapshot(directory); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func managedRepositoryFixture(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	command := exec.Command("git", "init", "--quiet", repository)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	files, err := projectstate.RenderProjectCanon("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		writeTestFile(t, filepath.Join(repository, filepath.FromSlash(file.Path)), file.Content)
	}
	return repository
}

func copyTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
