package openspec

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestIntentResolverVerifiesConfirmedIdentityAndDigest(t *testing.T) {
	root := t.TempDir()
	raw := []byte(minimalIntent("confirmed", "None.", true))
	path := filepath.Join(root, "intent.md")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	reference := domain.WorkSpecIntentReference{
		ID:          "INTENT-TEST",
		Version:     1,
		ArtifactRef: "intent.md",
		Digest:      "sha256:" + hex.EncodeToString(sum[:]),
	}
	if err := (IntentResolver{}).Verify(root, reference); err != nil {
		t.Fatal(err)
	}

	reference.Digest = "sha256:" + strings.Repeat("0", 64)
	if err := (IntentResolver{}).Verify(root, reference); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestIntentResolverRejectsCandidateAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	raw := []byte(minimalIntent("candidate", "None.", false))
	if err := os.WriteFile(filepath.Join(root, "intent.md"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	reference := domain.WorkSpecIntentReference{
		ID:          "INTENT-TEST",
		Version:     1,
		ArtifactRef: "intent.md",
		Digest:      "sha256:" + hex.EncodeToString(sum[:]),
	}
	if err := (IntentResolver{}).Verify(root, reference); err == nil || !strings.Contains(err.Error(), "not confirmed") {
		t.Fatalf("expected candidate rejection, got %v", err)
	}

	outside := t.TempDir()
	raw = []byte(minimalIntent("confirmed", "None.", true))
	outsidePath := filepath.Join(outside, "intent.md")
	if err := os.WriteFile(outsidePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(root, "escaped.md")); err != nil {
		t.Fatal(err)
	}
	sum = sha256.Sum256(raw)
	reference.ArtifactRef = "escaped.md"
	reference.Digest = "sha256:" + hex.EncodeToString(sum[:])
	if err := (IntentResolver{}).Verify(root, reference); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
}
