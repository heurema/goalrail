package harness

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedCanonMatchesTheRepositoryOverlay pins the two copies together.
//
// Go embedding cannot reach outside the package directory, so the canon is a
// committed copy of `openspec/schemas/goalrail-intent`. A copy nobody compares
// is the drift this package exists to detect, one layer down: the repository's
// overlay stays the source, and editing either without the other fails here
// rather than shipping a binary that installs something else.
func TestEmbeddedCanonMatchesTheRepositoryOverlay(t *testing.T) {
	canon, err := CurrentCanon()
	if err != nil {
		t.Fatalf("read current canon: %v", err)
	}
	repositoryRoot := filepath.Join("..", "..")

	seen := make(map[string]bool, len(canon.Files))
	for _, file := range canon.Files {
		embedded, contentErr := Content(file.Path)
		if contentErr != nil {
			t.Fatalf("read embedded %s: %v", file.Path, contentErr)
		}
		onDisk, readErr := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(file.Path)))
		if readErr != nil {
			t.Fatalf("read repository %s: %v", file.Path, readErr)
		}
		if !bytes.Equal(embedded, onDisk) {
			t.Errorf("embedded canon differs from the repository overlay at %s; "+
				"copy the repository file into internal/harness/canon", file.Path)
		}
		if Digest(onDisk) != file.Digest {
			t.Errorf("digest for %s does not describe the repository file", file.Path)
		}
		seen[file.Path] = true
	}

	// The other direction matters as much: a template added to the repository and
	// not to the canon would be silently absent from every initialized repository.
	overlayRoot := filepath.Join(repositoryRoot, filepath.FromSlash(OverlayDirectory))
	walkErr := filepath.Walk(overlayRoot, func(walked string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(repositoryRoot, walked)
		if relErr != nil {
			return relErr
		}
		slashed := filepath.ToSlash(relative)
		if !seen[slashed] {
			t.Errorf("repository overlay carries %s, which the embedded canon does not", slashed)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk repository overlay: %v", walkErr)
	}
}

func TestCanonIdentityFollowsContent(t *testing.T) {
	canon, err := CurrentCanon()
	if err != nil {
		t.Fatalf("read current canon: %v", err)
	}
	if canon.ID == "" {
		t.Fatal("canon carries no identity")
	}

	// Reordering must not change the identity: it is a property of the content,
	// not of the order a walk happened to produce.
	reversed := make([]CanonFile, 0, len(canon.Files))
	for index := len(canon.Files) - 1; index >= 0; index-- {
		reversed = append(reversed, canon.Files[index])
	}
	if canonIdentity(reversed) != canon.ID {
		t.Error("canon identity depends on file order")
	}

	// Changing one byte of one file must change it.
	altered := make([]CanonFile, len(canon.Files))
	copy(altered, canon.Files)
	altered[0].Digest = Digest([]byte("something else"))
	if canonIdentity(altered) == canon.ID {
		t.Error("canon identity survived a changed file digest")
	}
}

func TestCanonCoversTheSchemaAndEveryTemplate(t *testing.T) {
	canon, err := CurrentCanon()
	if err != nil {
		t.Fatalf("read current canon: %v", err)
	}
	required := []string{
		OverlayDirectory + "/schema.yaml",
		OverlayDirectory + "/templates/context.md",
		OverlayDirectory + "/templates/intent.md",
		OverlayDirectory + "/templates/proposal.md",
		OverlayDirectory + "/templates/spec.md",
		OverlayDirectory + "/templates/design.md",
		OverlayDirectory + "/templates/tasks.md",
	}
	for _, expected := range required {
		if _, present := canon.Digest(expected); !present {
			t.Errorf("canon is missing %s", expected)
		}
	}
	if len(canon.Files) != len(required) {
		t.Errorf("canon carries %d files, expected exactly %d: %v",
			len(canon.Files), len(required), canon.Paths())
	}
}

// TestCanonCarriesNoConfiguration pins the ownership split. The OpenSpec
// configuration holds a project's own description, so comparing it against a
// canon would report every repository's own prose as drift.
func TestCanonCarriesNoConfiguration(t *testing.T) {
	canon, err := CurrentCanon()
	if err != nil {
		t.Fatalf("read current canon: %v", err)
	}
	for _, file := range canon.Files {
		if file.Path == ConfigPath {
			t.Error("canon carries the OpenSpec configuration, which is repository-owned")
		}
	}
}
