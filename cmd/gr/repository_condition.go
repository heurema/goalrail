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
// notARepositoryError carries the exit status the condition deserves.
//
// A path outside version control is a state the command found; a discovery that
// could not run is the command failing to look. A caller reading only the status
// must be able to tell them apart, so the condition takes the status the command
// surface already assigns to "unmanaged" rather than the generic failure the
// broken discovery keeps.
type notARepositoryError struct{ message string }

func (err notARepositoryError) Error() string { return err.message }
func (notARepositoryError) ExitCode() int     { return exitUnmanaged }

// exitUnmanaged is the status a diagnosis already returns for a repository that
// declares no Goalrail project; a path that is not a repository at all is the
// same kind of answer and shares it.
const exitUnmanaged = 3

func statedRepositoryCondition(err error, path string) error {
	switch {
	case errors.Is(err, projectstate.ErrNotRepository):
	case errors.Is(err, projectstate.ErrDiscoveryRefused):
		// Git ran and said no for a reason of its own — disputed ownership is
		// the one users meet. The condition is named and the remedy is Git's to
		// give, so the reader is sent to Git rather than handed its sentence
		// second-hand along with a command to paste.
		return fmt.Errorf("Git refused to resolve %s, so its repository could not be identified; "+
			"run a Git command there yourself to see what it objects to", absolutePath(path))
	case errors.Is(err, projectstate.ErrDiscoveryUnavailable):
		return fmt.Errorf("Git could not be run, so whether %s is inside a repository was never established", absolutePath(path))
	default:
		return err
	}
	return notARepositoryError{message: fmt.Sprintf(
		"%s is not inside a Git repository, so there is no worktree root to resolve a Goalrail project from", absolutePath(path))}
}

// absolutePath resolves what the caller passed, because a sentence naming `.`
// tells a reader nothing about which directory was meant.
func absolutePath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		return absolute
	}
	return path
}
