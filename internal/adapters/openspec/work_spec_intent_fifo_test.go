//go:build unix

package openspec

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

// TestIntentResolverRejectsNonRegularArtifacts proves that a FIFO supplied as
// the intent artifact or as the Context Pack fails with a bounded reason. Each
// case runs under a deadline because the defect this covers is a hang: opening
// a FIFO blocks until a writer connects, so a regression would never return.
func TestIntentResolverRejectsNonRegularArtifacts(t *testing.T) {
	tests := []struct {
		name string
		// fifoName is created as a FIFO inside the change directory instead of
		// being written as a regular file.
		fifoName string
	}{
		{name: "intent artifact is a FIFO", fifoName: "intent.md"},
		{name: "context pack is a FIFO", fifoName: "context.md"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			changeDir := filepath.Join(root, "openspec", "changes", "test-change")
			if err := os.MkdirAll(changeDir, 0o700); err != nil {
				t.Fatal(err)
			}

			raw := []byte(minimalFlowIntent("confirmed", "None.", true))
			reference := domain.WorkSpecIntentReference{
				ID:          "INTENT-TEST",
				Version:     1,
				ArtifactRef: "openspec/changes/test-change/intent.md",
				Digest:      "sha256:" + hex.EncodeToString(digestOf(raw)),
			}

			if test.fifoName != "intent.md" {
				if err := os.WriteFile(filepath.Join(changeDir, "intent.md"), raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.fifoName != "context.md" {
				context := []byte(minimalContext("sufficient", "None."))
				if err := os.WriteFile(filepath.Join(changeDir, "context.md"), context, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := syscall.Mkfifo(filepath.Join(changeDir, test.fifoName), 0o600); err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() {
				done <- (IntentResolver{}).Verify(root, reference)
			}()

			select {
			case err := <-done:
				if err == nil {
					t.Fatal("expected rejection for a non-regular artifact")
				}
				if !strings.Contains(err.Error(), "not a regular file") {
					t.Fatalf("error = %v, want a bounded non-regular-file reason", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("verification blocked on a FIFO instead of failing with a bounded reason")
			}
		})
	}
}

func digestOf(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}
