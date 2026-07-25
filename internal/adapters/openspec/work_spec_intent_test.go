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

func TestIntentResolverVerifiesContextBoundConfirmedIntent(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "test-change")
	if err := os.MkdirAll(changeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := []byte(minimalFlowIntent("confirmed", "None.", true))
	if err := os.WriteFile(filepath.Join(changeDir, "intent.md"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(changeDir, "context.md"),
		[]byte(minimalContext("sufficient", "None.")),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	reference := domain.WorkSpecIntentReference{
		ID:          "INTENT-TEST",
		Version:     1,
		ArtifactRef: "openspec/changes/test-change/intent.md",
		Digest:      "sha256:" + hex.EncodeToString(sum[:]),
	}

	if err := (IntentResolver{}).Verify(root, reference); err != nil {
		t.Fatal(err)
	}
}

func TestIntentResolverRejectsInvalidContextBinding(t *testing.T) {
	tests := []struct {
		name         string
		intent       string
		writeContext func(*testing.T, string)
		want         string
	}{
		{
			name:   "missing context",
			intent: minimalFlowIntent("confirmed", "None.", true),
			want:   "OpenSpec context is required",
		},
		{
			name:   "mismatched context identity",
			intent: minimalFlowIntent("confirmed", "None.", true),
			writeContext: func(t *testing.T, changeDir string) {
				t.Helper()
				context := strings.Replace(
					minimalContext("sufficient", "None."),
					"CONTEXT-TEST",
					"CONTEXT-OTHER",
					1,
				)
				if err := os.WriteFile(
					filepath.Join(changeDir, "context.md"),
					[]byte(context),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
			},
			want: "does not match context.md",
		},
		{
			name:   "unknown context reference",
			intent: strings.ReplaceAll(minimalFlowIntent("confirmed", "None.", true), "CTX-1", "CTX-2"),
			writeContext: func(t *testing.T, changeDir string) {
				t.Helper()
				if err := os.WriteFile(
					filepath.Join(changeDir, "context.md"),
					[]byte(minimalContext("sufficient", "None.")),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
			},
			want: "context item reference does not exist",
		},
		{
			name:   "context symlink escapes repository",
			intent: minimalFlowIntent("confirmed", "None.", true),
			writeContext: func(t *testing.T, changeDir string) {
				t.Helper()
				outside := t.TempDir()
				outsidePath := filepath.Join(outside, "context.md")
				if err := os.WriteFile(
					outsidePath,
					[]byte(minimalContext("sufficient", "None.")),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outsidePath, filepath.Join(changeDir, "context.md")); err != nil {
					t.Fatal(err)
				}
			},
			want: "intent context escapes the repository",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			changeDir := filepath.Join(root, "openspec", "changes", "test-change")
			if err := os.MkdirAll(changeDir, 0o700); err != nil {
				t.Fatal(err)
			}
			raw := []byte(test.intent)
			if err := os.WriteFile(filepath.Join(changeDir, "intent.md"), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if test.writeContext != nil {
				test.writeContext(t, changeDir)
			}
			sum := sha256.Sum256(raw)
			reference := domain.WorkSpecIntentReference{
				ID:          "INTENT-TEST",
				Version:     1,
				ArtifactRef: "openspec/changes/test-change/intent.md",
				Digest:      "sha256:" + hex.EncodeToString(sum[:]),
			}

			err := (IntentResolver{}).Verify(root, reference)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Verify() error = %v, want containing %q", err, test.want)
			}
		})
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
