package releasebundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A pin moves as a decision, not as lock-file churn.
//
// Eighty transitive packages change their digest for any reason at all, so a
// review that only sees a diff sees nothing. The recorded closure is what
// somebody last looked at, and disagreeing with it stops the build until
// somebody looks again.
func TestTheRecordedClosureMustMatchTheComputedOne(t *testing.T) {
	repository := lockFixture(t)

	if _, err := CheckSourceLock(repository); err != nil {
		t.Fatalf("the repository's own lock does not match its record: %v", err)
	}

	for _, testCase := range []struct {
		name   string
		change func(*SourceLock)
		want   string
	}{
		{"package count", func(l *SourceLock) { l.Closure.PackageCount++ }, "packages"},
		{"install script count", func(l *SourceLock) { l.Closure.InstallScriptCount++ }, "install script"},
		{"closure digest", func(l *SourceLock) { l.Closure.Digest = "sha256:" + strings.Repeat("a", 64) }, "digest"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rewriteLock(t, repository, testCase.change)
			_, err := CheckSourceLock(repository)
			if err == nil {
				t.Fatalf("a closure disagreeing on the %s was accepted", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("the refusal does not name what disagreed: %v", err)
			}
		})
	}
}

// The adoption dates are disclosure, and the one inconsistency the record can
// catch on its own is a pin claiming to predate its own publication.
func TestAdoptionDatesAreRequiredAndOrdered(t *testing.T) {
	repository := lockFixture(t)

	for _, testCase := range []struct {
		name   string
		change func(*SourceLock)
		want   string
	}{
		{"runtime publication absent", func(l *SourceLock) { l.Runtime.Adoption.PublishedAt = "" }, "publication date"},
		{"compiler adoption absent", func(l *SourceLock) { l.Compiler.Adoption.AdoptedAt = "" }, "pinned it"},
		{"adopted before published", func(l *SourceLock) {
			l.Runtime.Adoption.AdoptedAt = "2020-01-01T00:00:00Z"
		}, "before it was published"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rewriteLock(t, repository, testCase.change)
			if _, err := CheckSourceLock(repository); err == nil {
				t.Fatalf("an unusable adoption record was accepted: %s", testCase.name)
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("the refusal does not name the problem: %v", err)
			}
		})
	}
}

// lockFixture copies this repository's own release inputs, so the corpus is the
// real pinned set rather than a shape invented to pass.
func lockFixture(t *testing.T) string {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	source := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	repository := t.TempDir()
	for _, relative := range []string{
		filepath.Join("release", "setup", "source-lock.json"),
		filepath.Join("release", "setup", "compiler", "package.json"),
		filepath.Join("release", "setup", "compiler", "package-lock.json"),
		"LICENSE",
	} {
		raw, err := os.ReadFile(filepath.Join(source, relative))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repository, relative)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repository, relative), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repository
}

func rewriteLock(t *testing.T, repository string, change func(*SourceLock)) {
	t.Helper()
	path := filepath.Join(repository, "release", "setup", "source-lock.json")
	original := lockFixture(t)
	raw, err := os.ReadFile(filepath.Join(original, "release", "setup", "source-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock SourceLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	change(&lock)
	rewritten, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(rewritten, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The check belongs on the path every release path shares, not in one command.
// Building and verifying load their inputs the same way, so neither can proceed
// against a closure nobody looked at.
func TestBuildAndVerifyRefuseAnUnreviewedClosure(t *testing.T) {
	repository := lockFixture(t)
	rewriteLock(t, repository, func(l *SourceLock) { l.Closure.PackageCount++ })

	if _, err := loadReleaseInputs(repository); err == nil {
		t.Fatal("the shared input path accepted an unreviewed closure")
	} else if !strings.Contains(err.Error(), "packages") {
		t.Fatalf("the refusal does not name what disagreed: %v", err)
	}
}

// The expanded document carries its own schema identifier, because strict
// decoding makes the two shapes mutually unreadable rather than compatible.
func TestTheExpandedSourceLockCarriesItsOwnSchema(t *testing.T) {
	repository := lockFixture(t)
	rewriteLock(t, repository, func(l *SourceLock) { l.Schema = "goalrail.setup-source-lock/v1" })

	if _, err := CheckSourceLock(repository); err == nil {
		t.Fatal("a document claiming the previous schema was accepted")
	} else if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("the refusal does not name the schema: %v", err)
	}
}
