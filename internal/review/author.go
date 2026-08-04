// Package review runs one independent review of a branch and records what was
// reviewed.
//
// An author cannot review their own artifact: the same context produces the
// same blind spots. So the review is performed by a provider the author did not
// write with, in a process that never sees the author's session — and the whole
// point collapses if the author is guessed wrong, because the failure is
// silent. Detection therefore uses explicit primary markers and records an
// unknown author rather than guessing when its caller cannot decide.
package review

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
)

// ErrAuthorUndetectable reports an environment carrying no provider's primary
// session marker.
var ErrAuthorUndetectable = errors.New("no agent session was detected in the environment")

// ErrAuthorAmbiguous reports an environment carrying more than one.
var ErrAuthorAmbiguous = errors.New("more than one agent session was detected in the environment")

// primaryMarkers is the registered set of variables whose presence means "a
// session of this scaffold is running this process".
//
// Deliberately a registry, and deliberately not a prefix match. A Claude Code
// session on this machine also carries CODEX_COMPANION_SESSION_ID, because a
// companion is reachable from it — matching any CODEX_* variable would have
// read that as Codex authorship and handed the review straight back to the
// author. Codex desktop currently exposes CODEX_THREAD_ID; CODEX_SESSION_ID is
// retained as the primary marker accepted by earlier Goalrail releases. Two
// markers of one provider are one identity, not an ambiguity.
var primaryMarkers = map[ambient.Scaffold][]string{
	ambient.ScaffoldClaudeCode: {"CLAUDECODE"},
	ambient.ScaffoldCodex:      {"CODEX_THREAD_ID", "CODEX_SESSION_ID"},
}

// Lookup reads one environment variable, reporting whether it was set at all.
// It is os.LookupEnv in production and a map in tests; nothing here reads the
// process environment directly, because a detector that cannot be given an
// environment cannot be tested against the environments that matter.
type Lookup func(name string) (string, bool)

// MarkerNames lists the variables detection consults, so a refusal can name
// them without restating them from memory.
func MarkerNames() []string {
	var names []string
	for _, markers := range primaryMarkers {
		names = append(names, markers...)
	}
	sort.Strings(names)
	return names
}

// DetectAuthor reports which scaffold's session is running this process.
//
// A marker counts only when it is set to something other than the empty string:
// an exported-but-empty variable is how a wrapper says "not this", and reading
// it as presence would be the same silent misattribution the narrow matching
// above exists to prevent.
func DetectAuthor(lookup Lookup) (ambient.Scaffold, error) {
	var found []ambient.Scaffold
	detected := make(map[ambient.Scaffold][]string)
	for _, scaffold := range ambient.SupportedScaffolds() {
		markers, known := primaryMarkers[scaffold]
		if !known {
			continue
		}
		for _, marker := range markers {
			if value, set := lookup(marker); set && strings.TrimSpace(value) != "" {
				detected[scaffold] = append(detected[scaffold], marker)
			}
		}
		if len(detected[scaffold]) > 0 {
			found = append(found, scaffold)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("%w (looked for %s); state the authoring agent explicitly",
			ErrAuthorUndetectable, strings.Join(MarkerNames(), ", "))
	default:
		names := make([]string, 0, len(found))
		for _, scaffold := range found {
			names = append(names, fmt.Sprintf("%s (%s)", scaffold, strings.Join(detected[scaffold], ", ")))
		}
		return "", fmt.Errorf("%w: %s; state the authoring agent explicitly",
			ErrAuthorAmbiguous, strings.Join(names, " and "))
	}
}

// AuthorMarkers lists every variable that identifies an agent session, primary
// or otherwise, so a spawned reviewer can be started without inheriting the
// author's identity.
//
// A reviewer launched from inside the author's session inherits its whole
// environment. This repository already met that: the live verification of the
// Claude Code path had to unset the CLAUDE* family by hand to make a spawned
// session look like a fresh one rather than a continuation of the parent. A
// reviewer that believes it is the author's own session is not an independent
// reviewer.
func AuthorMarkers() []string {
	return []string{
		"CLAUDECODE",
		"CLAUDE_CODE_ENTRYPOINT",
		"CLAUDE_CODE_SESSION_ID",
		"CLAUDE_CODE_HOST_SESSION_ID",
		"CLAUDE_CODE_CHILD_SESSION",
		"CLAUDE_PID",
		"CLAUDE_AGENT_SDK_VERSION",
		"CLAUDE_CODE_EXECPATH",
		"CLAUDE_EFFORT",
		"CODEX_THREAD_ID",
		"CODEX_SESSION_ID",
		"CODEX_COMPANION_SESSION_ID",
	}
}

// executables names the command each provider is reviewed with.
var executables = map[ambient.Scaffold]string{
	ambient.ScaffoldCodex:      "codex",
	ambient.ScaffoldClaudeCode: "claude",
}

// Executable reports the command one provider is invoked as.
func Executable(scaffold ambient.Scaffold) (string, bool) {
	name, known := executables[scaffold]
	return name, known
}

// SelectReviewer picks the reviewer for one author out of the providers whose
// command can actually be run.
//
// Runnability is checked rather than assumed. A configuration directory left
// behind by an uninstall would otherwise select a provider whose binary is
// gone, and the review would fail somewhere further along with a message about
// the wrong thing. The candidates are the ones a caller offers; the check is
// whether each can be executed at all.
// AuthorUnknown records that no authorship could be determined. The fresh
// session provides the independence either way; detection only improves the
// choice, so its failure is information for the receipt, not a reason to stop.
const AuthorUnknown ambient.Scaffold = "unknown"

// Selection is a reviewer together with how it was chosen — because "a
// different vendor reviewed this" has to stay a provable claim, a cross that
// quietly became fresh would be exactly the unprovable kind.
type Selection struct {
	Reviewer ambient.Scaffold
	// Mode is "cross" where the reviewer is a different provider from the
	// author, "fresh" where it is the same provider (or the author is unknown)
	// in a clean session.
	Mode string
	// Reason states why this mode, in one sentence.
	Reason string
}

func SelectReviewer(author ambient.Scaffold, candidates []ambient.Scaffold, lookPath func(string) (string, error)) (Selection, error) {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	runnable := make([]ambient.Scaffold, 0, len(candidates))
	var missing []string
	for _, scaffold := range candidates {
		name, defined := Executable(scaffold)
		if !defined {
			continue
		}
		if _, err := lookPath(name); err != nil {
			missing = append(missing, fmt.Sprintf("%s (%s is not on PATH)", scaffold, name))
			continue
		}
		runnable = append(runnable, scaffold)
	}
	if len(runnable) == 0 {
		return Selection{}, fmt.Errorf("no reviewer can be run at all: %s", strings.Join(missing, ", "))
	}
	// Cross where possible: a provider that did not author the change adds
	// different weights on top of the fresh context.
	for _, scaffold := range runnable {
		if scaffold != author {
			mode, reason := "cross", fmt.Sprintf("%s did not author this change", scaffold)
			if author == AuthorUnknown || author == "" {
				// With no author there is no "other side"; the choice is fresh
				// by definition and deterministic by candidate order.
				mode, reason = "fresh", "the author is unknown, so the first runnable provider reviews in a clean session"
			}
			return Selection{Reviewer: scaffold, Mode: mode, Reason: reason}, nil
		}
	}
	// Only the author's own provider runs. A clean session of the same provider
	// is the ordinary single-tool case, not a degraded one — the mechanism is
	// the fresh context, and a second vendor only strengthens it.
	reason := fmt.Sprintf("only %s is runnable, so it reviews in a clean session", author)
	if len(missing) > 0 {
		reason = fmt.Sprintf("cross review was unavailable — %s — so %s reviews in a clean session",
			strings.Join(missing, ", "), author)
	}
	return Selection{Reviewer: runnable[0], Mode: "fresh", Reason: reason}, nil
}
