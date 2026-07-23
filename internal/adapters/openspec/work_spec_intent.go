package openspec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const MaxResolvedIntentBytes = 1 << 20

// IntentResolver verifies the current project intent artifact and exposes only
// the provider-neutral verification result to the local-run service.
type IntentResolver struct{}

func (IntentResolver) Verify(
	repositoryRoot string,
	reference domain.WorkSpecIntentReference,
) error {
	root, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	root = filepath.Clean(root)
	artifactPath := filepath.Join(root, filepath.FromSlash(reference.ArtifactRef))
	resolvedArtifact, err := filepath.EvalSymlinks(artifactPath)
	if err != nil {
		return fmt.Errorf("resolve intent artifact: %w", err)
	}
	resolvedArtifact = filepath.Clean(resolvedArtifact)
	if !pathWithin(root, resolvedArtifact) {
		return fmt.Errorf("intent artifact escapes the repository")
	}

	file, err := os.Open(resolvedArtifact)
	if err != nil {
		return fmt.Errorf("open intent artifact: %w", err)
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, MaxResolvedIntentBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("read intent artifact: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close intent artifact: %w", closeErr)
	}
	if len(raw) > MaxResolvedIntentBytes {
		return fmt.Errorf("intent artifact exceeds %d bytes", MaxResolvedIntentBytes)
	}

	sum := sha256.Sum256(raw)
	observedDigest := "sha256:" + hex.EncodeToString(sum[:])
	if observedDigest != reference.Digest {
		return fmt.Errorf("intent digest mismatch: expected %s, got %s", reference.Digest, observedDigest)
	}
	snapshot, err := ReadIntent(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("read confirmed intent: %w", err)
	}
	if snapshot.Status != domain.IntentConfirmed {
		return fmt.Errorf("intent status is %q, not confirmed", snapshot.Status)
	}
	if snapshot.ID != reference.ID || snapshot.Version != reference.Version {
		return fmt.Errorf(
			"intent identity mismatch: expected %s version %d, got %s version %d",
			reference.ID,
			reference.Version,
			snapshot.ID,
			snapshot.Version,
		)
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil &&
		relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(relative)
}
