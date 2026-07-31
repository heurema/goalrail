package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/localrun"
	"github.com/heurema/goalrail/internal/updatecheck"
)

// exitDiagnosis is the status a diagnosis returns when it found something the
// user can act on. It is separate from a usage or internal failure, so an
// unattended check can tell "your harness needs attention" from "the command did
// not run".
const exitDiagnosis = 1

// diagnosisError carries a non-zero exit for an otherwise successful report.
type diagnosisError struct{ problems int }

func (err diagnosisError) Error() string {
	return fmt.Sprintf("%d harness problem(s) reported", err.problems)
}

func (err diagnosisError) ExitCode() int { return exitDiagnosis }

// doctorFailure marks a diagnosis that did not run — bad usage or an internal
// failure — as distinct from one that ran and found problems. A script watching
// the exit status must be able to tell "the harness needs attention" (1) from
// "the check itself broke" (2), or a broken cron job reads as a healthy repo.
type doctorFailure struct{ cause error }

func (err doctorFailure) Error() string { return err.cause.Error() }
func (err doctorFailure) Unwrap() error { return err.cause }
func (err doctorFailure) ExitCode() int { return 2 }

func runDoctor(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("doctor", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "repository to diagnose")
	scaffold := set.String("scaffold", "", "codex or claude-code; omit to report every supported scaffold")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	asJSON := set.Bool("json", false, "emit the diagnosis as JSON")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return doctorFailure{cause: err}
	}
	if set.NArg() != 0 {
		return doctorFailure{cause: fmt.Errorf("doctor accepts no positional arguments")}
	}
	diagnosis, err := diagnose(*repository, *scaffold, *stateDirectory)
	if err != nil {
		return doctorFailure{cause: err}
	}
	if *asJSON {
		if err := writeJSON(stdout, diagnosis); err != nil {
			return doctorFailure{cause: err}
		}
	} else if _, err := fmt.Fprint(stdout, harness.Describe(diagnosis)); err != nil {
		// A report that could not be written is a failed check, not a failed
		// harness: an unattended caller reading the exit status must not confuse
		// a broken pipe with an unhealthy repository.
		return doctorFailure{cause: err}
	}
	if !diagnosis.Working {
		return diagnosisError{problems: len(diagnosis.Problems)}
	}
	return nil
}

func diagnose(repository, scaffold, stateDirectory string) (harness.Diagnosis, error) {
	root, err := filepath.Abs(repository)
	if err != nil {
		return harness.Diagnosis{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return harness.Diagnosis{}, err
	}
	input := harness.DiagnoseInput{RepositoryRoot: root, Home: home}
	if scaffold != "" {
		selected, parseErr := parseScaffold(scaffold)
		if parseErr != nil {
			return harness.Diagnosis{}, parseErr
		}
		input.Scaffolds = append(input.Scaffolds, selected)
	}
	// The state root is where observability may be configured, outside the
	// repository. Resolving it must not create it: a diagnosis writes nothing.
	if resolved, resolveErr := localrun.ResolveStateRoot(stateDirectory); resolveErr == nil {
		input.StateRoot = resolved
	}
	// The one place the real check is constructed. The diagnosis is handed a
	// function and never builds one, which is what keeps the network out of every
	// other command — and it is also why the diagnosis itself still writes
	// nothing: remembering the answer happens in here, not in there.
	input.LatestRelease = updatecheck.New(input.StateRoot, updatecheck.Environment{})
	return harness.Diagnose(input)
}

// runHealth is the superseded name. It keeps its exact stdout, because a script
// parsing its JSON must not break on a deprecation line, and names its successor
// on stderr instead.
func runHealthAlias(args []string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stderr, "gr health is now gr doctor, which reports the overlay and the "+
		"toolchain as well as the attachment; this name still works.")
	return runHealth(args, stdout, stderr)
}

func runUpdate(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("update", flag.ContinueOnError)
	set.SetOutput(stderr)
	// The word invites the other expectation, so the help says plainly what this
	// does not do: a user who believes they upgraded Goalrail when they did not
	// would misread every later version statement.
	set.Usage = func() {
		fmt.Fprint(stderr, "usage: gr update [flags]\n\n"+
			"Brings this repository's harness up to what the installed gr carries.\n"+
			"It does not update the gr binary itself, and makes no network request.\n\n")
		set.PrintDefaults()
	}
	repository := set.String("repo", ".", "repository whose harness to update")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	discard := set.Bool("discard-local-edits", false,
		"replace overlay files that differ from the canon, discarding local edits")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("update accepts no positional arguments")
	}
	root, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	stateRoot, err := localrun.ResolveStateRoot(*stateDirectory)
	if err != nil {
		return err
	}
	report, err := harness.Update(harness.UpdateInput{
		RepositoryRoot:    root,
		StateRoot:         stateRoot,
		DiscardLocalEdits: *discard,
	})
	// A failed verification still made a backup and rewrote files. Discarding the
	// report there would lose the backup path at exactly the moment the user needs
	// it, so a report that names one is printed even on the error path.
	if report.Backup != "" {
		if writeErr := writeJSON(stdout, report); writeErr != nil && err == nil {
			return writeErr
		}
		return err
	}
	if err != nil {
		return err
	}
	return writeJSON(stdout, report)
}

func runVersion(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("version", flag.ContinueOnError)
	set.SetOutput(stderr)
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	canon, err := harness.CurrentCanon()
	if err != nil {
		return err
	}
	// The canon identity is reported beside the version because it is what
	// decides anything about a repository; the version is information about the
	// binary.
	return writeJSON(stdout, map[string]any{
		"version": harness.Version,
		"canon":   canon.ID,
	})
}
