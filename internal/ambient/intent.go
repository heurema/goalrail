package ambient

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"

	"github.com/heurema/goalrail/internal/adapters/openspec"
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
	ActiveConfirmedIntent(repositoryRoot string) (*IntentRef, string)
}

// OpenSpecIntents resolves through the OpenSpec change layout: current changes
// live in openspec/changes, archived ones under openspec/changes/archive.
type OpenSpecIntents struct{}

const (
	reasonNoChanges    = "NO_ACTIVE_CHANGE"
	reasonNoConfirmed  = "NO_CONFIRMED_INTENT"
	reasonAmbiguous    = "SEVERAL_CONFIRMED_INTENTS"
	changesRelativeDir = "openspec/changes"
)

// ActiveConfirmedIntent binds only when exactly one current change carries a
// confirmed intent. Zero, several, or only unconfirmed candidates return no
// reference and a reason: the record stays honest rather than guessing which
// work a question belonged to.
func (OpenSpecIntents) ActiveConfirmedIntent(repositoryRoot string) (*IntentRef, string) {
	changesRoot := filepath.Join(repositoryRoot, filepath.FromSlash(changesRelativeDir))
	entries, err := os.ReadDir(changesRoot)
	if err != nil {
		return nil, reasonNoChanges
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
		return nil, reasonNoChanges
	}

	found := make([]IntentRef, 0, 2)
	for _, name := range names {
		artifact := filepath.Join(changesRoot, name, "intent.md")
		raw, readErr := os.ReadFile(artifact)
		if readErr != nil {
			continue
		}
		snapshot, parseErr := openspec.ReadIntent(bytes.NewReader(raw))
		if parseErr != nil || snapshot.Status != domain.IntentConfirmed {
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
	switch len(found) {
	case 0:
		return nil, reasonNoConfirmed
	case 1:
		reference := found[0]
		return &reference, ""
	default:
		return nil, reasonAmbiguous
	}
}
