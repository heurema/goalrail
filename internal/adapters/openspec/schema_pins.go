package openspec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SchemaPinCounts reports the two kinds of change that keep a repository-owned
// schema directory in use.
type SchemaPinCounts struct {
	Active   int `json:"active"`
	Archived int `json:"archived"`
}

func (counts SchemaPinCounts) Total() int {
	return counts.Active + counts.Archived
}

// CountSchemaPins reads the same .openspec.yaml metadata as change loading.
// A missing changes directory is an ordinary zero; malformed metadata is not,
// because a false zero could tell a user that a still-needed schema may go.
func CountSchemaPins(repositoryRoot, schema string) (SchemaPinCounts, error) {
	changesRoot := filepath.Join(repositoryRoot, "openspec", "changes")
	active, err := countSchemaPinsIn(changesRoot, schema, "archive")
	if err != nil {
		return SchemaPinCounts{}, err
	}
	archived, err := countSchemaPinsIn(filepath.Join(changesRoot, "archive"), schema, "")
	if err != nil {
		return SchemaPinCounts{}, err
	}
	return SchemaPinCounts{Active: active, Archived: archived}, nil
}

func countSchemaPinsIn(root, schema, excludedDirectory string) (int, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read OpenSpec changes at %s: %w", root, err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == excludedDirectory {
			continue
		}
		metadata := filepath.Join(root, entry.Name(), ".openspec.yaml")
		named, readErr := readChangeSchema(metadata)
		if readErr != nil {
			return 0, fmt.Errorf("read schema pin at %s: %w", metadata, readErr)
		}
		if named == schema {
			count++
		}
	}
	return count, nil
}
