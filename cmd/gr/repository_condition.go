package main

import (
	"errors"
	"fmt"
	"path/filepath"

	projectstate "github.com/heurema/goalrail/internal/project"
)

// statedRepositoryCondition turns a discovery failure into what this command
// says about it.
//
// A user handed `fatal: not a git repository (or any of the parent
// directories): .git` has been given another tool's vocabulary for a situation
// Goalrail understands perfectly well, and an unattended caller reading that
// tool's exit status learns about the wrong process. Every command that resolves
// a repository translates through here rather than carrying its own sentence:
// three copies of a rule is how this one came to be missing from all but one
// command in the first place.
//
// A failure that is not the named condition keeps its own outcome. Discovery
// classifies only Git's own verdict as an absent repository, so an absent Git or
// a refused directory arrives here as something else and is passed through as
// the failure it is.
func statedRepositoryCondition(err error, path string) error {
	if !errors.Is(err, projectstate.ErrNotRepository) {
		return err
	}
	// The absolute path, because the caller may have passed `.` and a sentence
	// naming a dot tells a reader nothing about which directory was meant.
	if absolute, absErr := filepath.Abs(path); absErr == nil {
		path = absolute
	}
	return fmt.Errorf("%s is not inside a Git repository, so there is no worktree root to resolve a Goalrail project from", path)
}
