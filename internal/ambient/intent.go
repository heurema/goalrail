package ambient

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

// IntentRef names the intent a question belongs to, by the identifiers the
// answering version will carry back.
type IntentRef struct {
	ID      domain.IntentID `json:"id"`
	Version uint32          `json:"version"`
	Digest  string          `json:"digest"`
	Change  string          `json:"change"`
}

// IntentResolver finds the repository's active confirmed intent, or explains
// why there is not exactly one.
type IntentResolver interface {
	ActiveConfirmedIntent(repositoryRoot string) IntentResolution
}

type BindingDiagnostic struct {
	Change     string                      `json:"change"`
	Diagnostic openspec.ArtifactDiagnostic `json:"diagnostic"`
}

type IntentResolution struct {
	Reference          *IntentRef          `json:"reference,omitempty"`
	UnboundReason      string              `json:"unbound_reason,omitempty"`
	BindingDiagnostics []BindingDiagnostic `json:"binding_diagnostics,omitempty"`
}

// OpenSpecIntents resolves through the OpenSpec change layout: current changes
// live in openspec/changes, archived ones under openspec/changes/archive.
type OpenSpecIntents struct{}

const (
	reasonNoChanges        = "NO_ACTIVE_CHANGE"
	reasonNoConfirmed      = "NO_CONFIRMED_INTENT"
	reasonAmbiguous        = "SEVERAL_CONFIRMED_INTENTS"
	reasonInvalidConfirmed = "INVALID_CONFIRMED_INTENT"
	changesRelativeDir     = "openspec/changes"
)

// ActiveConfirmedIntent binds only when exactly one current change carries a
// confirmed intent. Zero, several, or only unconfirmed candidates return no
// reference and a reason: the record stays honest rather than guessing which
// work a question belonged to.
func (OpenSpecIntents) ActiveConfirmedIntent(repositoryRoot string) IntentResolution {
	changesRoot := filepath.Join(repositoryRoot, filepath.FromSlash(changesRelativeDir))
	entries, err := os.ReadDir(changesRoot)
	if err != nil {
		return IntentResolution{UnboundReason: reasonNoChanges}
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archive" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return IntentResolution{UnboundReason: reasonNoChanges}
	}

	found := make([]IntentRef, 0, 2)
	diagnostics := make([]BindingDiagnostic, 0)
	for _, name := range names {
		changeDir := filepath.Join(changesRoot, name)
		artifact := filepath.Join(changeDir, "intent.md")
		raw, readErr := boundedio.ReadRegularFile(
			artifact,
			"ambient intent artifact",
			openspec.MaxPairArtifactBytes,
		)
		if readErr != nil {
			continue
		}
		inspection, inspectErr := openspec.InspectIntentEnvelope(raw)
		if inspectErr != nil {
			continue
		}
		contextPath := filepath.Join(changeDir, "context.md")
		_, contextErr := os.Lstat(contextPath)
		contextPresent := contextErr == nil || !errors.Is(contextErr, os.ErrNotExist)
		pairRequired, requirementErr := openspec.PairRequired(changeDir, inspection, contextPresent)
		if requirementErr != nil {
			continue
		}

		var snapshot domain.IntentSnapshot
		if pairRequired {
			var rawContext []byte
			if contextPresent {
				rawContext, readErr = boundedio.ReadRegularFile(
					contextPath,
					"ambient context artifact",
					openspec.MaxPairArtifactBytes,
				)
			}
			pair, conformErr := openspec.ConformPair(
				rawContext,
				raw,
				filepath.ToSlash(filepath.Join(changesRelativeDir, name, "context.md")),
				filepath.ToSlash(filepath.Join(changesRelativeDir, name, "intent.md")),
			)
			if conformErr != nil {
				if inspection.ClaimsConfirmed {
					appendBindingDiagnostic(&diagnostics, name, conformErr)
				}
				continue
			}
			snapshot = pair.Intent
		} else {
			var parseErr error
			snapshot, parseErr = openspec.ReadIntentArtifact(
				raw,
				filepath.ToSlash(filepath.Join(changesRelativeDir, name, "intent.md")),
			)
			if parseErr != nil {
				if inspection.ClaimsConfirmed {
					appendBindingDiagnostic(&diagnostics, name, parseErr)
				}
				continue
			}
		}
		if snapshot.Status != domain.IntentConfirmed {
			continue
		}
		sum := sha256.Sum256(raw)
		found = append(found, IntentRef{
			ID:      snapshot.ID,
			Version: snapshot.Version,
			Digest:  "sha256:" + hex.EncodeToString(sum[:]),
			Change:  name,
		})
	}
	if len(diagnostics) > 0 {
		return IntentResolution{
			UnboundReason:      reasonInvalidConfirmed,
			BindingDiagnostics: diagnostics,
		}
	}
	switch len(found) {
	case 0:
		return IntentResolution{UnboundReason: reasonNoConfirmed}
	case 1:
		reference := found[0]
		return IntentResolution{Reference: &reference}
	default:
		return IntentResolution{UnboundReason: reasonAmbiguous}
	}
}

func appendBindingDiagnostic(result *[]BindingDiagnostic, change string, err error) {
	var diagnostic *openspec.ArtifactDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic == nil {
		return
	}
	*result = append(*result, BindingDiagnostic{Change: change, Diagnostic: *diagnostic})
}
