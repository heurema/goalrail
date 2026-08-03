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

const MaxResolvedIntentBytes = MaxPairArtifactBytes

// IntentResolver verifies the current project intent artifact and exposes only
// the provider-neutral verification result to the local-run service.
type IntentResolver struct{}

type ResolvedIntent struct {
	Intent domain.IntentSnapshot
	Pair   *ConformedPair
}

func (IntentResolver) Verify(
	repositoryRoot string,
	reference domain.WorkSpecIntentReference,
) error {
	_, err := (IntentResolver{}).Resolve(repositoryRoot, reference)
	return err
}

func (IntentResolver) Resolve(
	repositoryRoot string,
	reference domain.WorkSpecIntentReference,
) (ResolvedIntent, error) {
	root, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return ResolvedIntent{}, fmt.Errorf("resolve repository root: %w", err)
	}
	root = filepath.Clean(root)
	artifactPath := filepath.Join(root, filepath.FromSlash(reference.ArtifactRef))
	resolvedArtifact, err := filepath.EvalSymlinks(artifactPath)
	if err != nil {
		return ResolvedIntent{}, inputUnavailableDiagnostic(
			safeReferencePath(reference.ArtifactRef, "intent.md"), ArtifactKindIntent, "unavailable input", err,
		)
	}
	resolvedArtifact = filepath.Clean(resolvedArtifact)
	if !pathWithin(root, resolvedArtifact) {
		return ResolvedIntent{}, fmt.Errorf("intent artifact escapes the repository")
	}

	raw, err := readBoundedRegularFile(resolvedArtifact, "intent artifact")
	if err != nil {
		return ResolvedIntent{}, inputUnavailableDiagnostic(
			safeReferencePath(reference.ArtifactRef, "intent.md"), ArtifactKindIntent, "unavailable or non-regular input", err,
		)
	}

	sum := sha256.Sum256(raw)
	observedDigest := "sha256:" + hex.EncodeToString(sum[:])
	if observedDigest != reference.Digest {
		return ResolvedIntent{}, fmt.Errorf("intent digest mismatch: expected %s, got %s", reference.Digest, observedDigest)
	}
	resolved, err := readResolvedIntent(root, resolvedArtifact, raw)
	if err != nil {
		var diagnostic *ArtifactDiagnostic
		if errors.As(err, &diagnostic) {
			return ResolvedIntent{}, err
		}
		return ResolvedIntent{}, fmt.Errorf("read confirmed intent: %w", err)
	}
	if resolved.Intent.Status != domain.IntentConfirmed {
		return ResolvedIntent{}, fmt.Errorf("intent status is %q, not confirmed", resolved.Intent.Status)
	}
	if resolved.Intent.ID != reference.ID || resolved.Intent.Version != reference.Version {
		return ResolvedIntent{}, fmt.Errorf(
			"intent identity mismatch: expected %s version %d, got %s version %d",
			reference.ID,
			reference.Version,
			resolved.Intent.ID,
			resolved.Intent.Version,
		)
	}
	return resolved, nil
}

func readResolvedIntent(
	repositoryRoot string,
	resolvedArtifact string,
	rawIntent []byte,
) (ResolvedIntent, error) {
	changeDir := filepath.Dir(resolvedArtifact)
	contextPath := filepath.Join(changeDir, "context.md")
	resolvedContext, err := filepath.EvalSymlinks(contextPath)
	if errors.Is(err, os.ErrNotExist) {
		snapshot, readErr := readIntentWithoutContext(changeDir, rawIntent)
		return ResolvedIntent{Intent: snapshot}, readErr
	}
	if err != nil {
		return ResolvedIntent{}, inputUnavailableDiagnostic("context.md", ArtifactKindContext, "unavailable input", err)
	}
	resolvedContext = filepath.Clean(resolvedContext)
	if !pathWithin(repositoryRoot, resolvedContext) {
		return ResolvedIntent{}, fmt.Errorf("intent context escapes the repository")
	}

	rawContext, err := readBoundedRegularFile(resolvedContext, "intent context")
	if err != nil {
		return ResolvedIntent{}, inputUnavailableDiagnostic(
			safeRelativePath(repositoryRoot, resolvedContext, "context.md"),
			ArtifactKindContext,
			"unavailable or non-regular input",
			err,
		)
	}

	pair, err := ConformPair(
		rawContext,
		rawIntent,
		safeRelativePath(repositoryRoot, resolvedContext, "context.md"),
		safeRelativePath(repositoryRoot, resolvedArtifact, "intent.md"),
	)
	if err != nil {
		return ResolvedIntent{}, err
	}
	return ResolvedIntent{Intent: pair.Intent, Pair: &pair}, nil
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
		return domain.IntentSnapshot{}, inputUnavailableDiagnostic(
			"context.md", ArtifactKindContext, "missing required input", ErrContextRequired,
		)
	}

	declaresContext, err := intentDeclaresContext(rawIntent)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	if declaresContext {
		return domain.IntentSnapshot{}, inputUnavailableDiagnostic(
			"context.md", ArtifactKindContext, "missing declared input", ErrContextRequired,
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

func safeReferencePath(reference string, fallback string) string {
	if normalized, ok := normalizeLogicalPath(reference); ok {
		return normalized
	}
	return fallback
}

func safeRelativePath(root string, candidate string, fallback string) string {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return fallback
	}
	return safeReferencePath(filepath.ToSlash(relative), fallback)
}
