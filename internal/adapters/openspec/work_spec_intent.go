package openspec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
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

	raw, err := readBoundedRegularFile(resolvedArtifact, "intent artifact")
	if err != nil {
		return err
	}

	sum := sha256.Sum256(raw)
	observedDigest := "sha256:" + hex.EncodeToString(sum[:])
	if observedDigest != reference.Digest {
		return fmt.Errorf("intent digest mismatch: expected %s, got %s", reference.Digest, observedDigest)
	}
	snapshot, err := readResolvedIntent(root, resolvedArtifact, raw)
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

func readResolvedIntent(
	repositoryRoot string,
	resolvedArtifact string,
	rawIntent []byte,
) (domain.IntentSnapshot, error) {
	changeDir := filepath.Dir(resolvedArtifact)
	contextPath := filepath.Join(changeDir, "context.md")
	resolvedContext, err := filepath.EvalSymlinks(contextPath)
	if errors.Is(err, os.ErrNotExist) {
		return readIntentWithoutContext(changeDir, rawIntent)
	}
	if err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("resolve intent context: %w", err)
	}
	resolvedContext = filepath.Clean(resolvedContext)
	if !pathWithin(repositoryRoot, resolvedContext) {
		return domain.IntentSnapshot{}, fmt.Errorf("intent context escapes the repository")
	}

	rawContext, err := readBoundedRegularFile(resolvedContext, "intent context")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}

	contextPack, err := ReadContext(bytes.NewReader(rawContext))
	if err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("read intent context: %w", err)
	}
	snapshot, err := readIntent(bytes.NewReader(rawIntent), &contextPack)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	if err := domain.ValidateFlowIntentSnapshot(snapshot); err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("validate OpenSpec flow intent: %w", err)
	}
	return snapshot, nil
}

// readIntentWithoutContext handles an intent whose sibling context.md is
// absent. It applies the same schema-aware distinction as changeRequiresContext
// so that omitting both the Context Pack declaration and the artifact cannot
// bypass verification for a current change under the project intent schema.
func readIntentWithoutContext(
	changeDir string,
	rawIntent []byte,
) (domain.IntentSnapshot, error) {
	required, err := changeRequiresContext(changeDir)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	if required {
		return domain.IntentSnapshot{}, fmt.Errorf(
			"%w: change requires a sibling context.md",
			ErrContextRequired,
		)
	}

	declaresContext, err := intentDeclaresContext(rawIntent)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	if declaresContext {
		return domain.IntentSnapshot{}, fmt.Errorf(
			"%w: intent declares sibling context.md",
			ErrContextRequired,
		)
	}
	return ReadIntent(bytes.NewReader(rawIntent))
}

// readBoundedRegularFile reads an already resolved artifact under the shared
// intent size bound. The descriptor-level guarantees live in boundedio so the
// escalation artifact and the intent artifact cannot drift apart.
func readBoundedRegularFile(path string, label string) ([]byte, error) {
	return boundedio.ReadRegularFile(path, label, MaxResolvedIntentBytes)
}

func intentDeclaresContext(rawIntent []byte) (bool, error) {
	document, err := readMarkdownDocument(bytes.NewReader(rawIntent))
	if err != nil {
		return false, err
	}
	metadata, err := parseBoldMetadata(document.preamble)
	if err != nil {
		return false, err
	}
	return cleanInline(metadata["context pack"]) != "", nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil &&
		relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(relative)
}
