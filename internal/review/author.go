// Package review runs one independent review of a branch and records what was
// reviewed.
//
// An author cannot review their own artifact: the same context produces the
// same blind spots. So the review is performed by a provider the author did not
// write with, in a process that never sees the author's session — and the whole
// point collapses if the author is guessed wrong, because the failure is
// silent. Everything here is built around refusing rather than guessing.
package review

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
)

// ErrAuthorUndetectable reports an environment carrying no provider's primary
// session marker.
var ErrAuthorUndetectable = errors.New("no agent session was detected in the environment")

// ErrAuthorAmbiguous reports an environment carrying more than one.
var ErrAuthorAmbiguous = errors.New("more than one agent session was detected in the environment")

// primaryMarkers is the one variable per scaffold whose presence means "a
// session of this scaffold is running this process".
//
// Deliberately one variable each, and deliberately not a prefix match. A Claude
// Code session on this machine also carries CODEX_COMPANION_SESSION_ID, because
// a companion is reachable from it — matching any CODEX_* variable would have
// read that as Codex authorship and handed the review straight back to the
// author. `CODEX_SESSION_ID` is the same variable the hook path already treats
// as the Codex session reference.
var primaryMarkers = map[ambient.Scaffold]string{
	ambient.ScaffoldClaudeCode: "CLAUDECODE",
	ambient.ScaffoldCodex:      "CODEX_SESSION_ID",
}

// Lookup reads one environment variable, reporting whether it was set at all.
// It is os.LookupEnv in production and a map in tests; nothing here reads the
// process environment directly, because a detector that cannot be given an
// environment cannot be tested against the environments that matter.
type Lookup func(name string) (string, bool)

// MarkerNames lists the variables detection consults, so a refusal can name
// them without restating them from memory.
func MarkerNames() []string {
	names := make([]string, 0, len(primaryMarkers))
	for _, name := range primaryMarkers {
		names = append(names, name)
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
	for _, scaffold := range ambient.SupportedScaffolds() {
		marker, known := primaryMarkers[scaffold]
		if !known {
			continue
		}
		if value, set := lookup(marker); set && strings.TrimSpace(value) != "" {
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
			names = append(names, fmt.Sprintf("%s (%s)", scaffold, primaryMarkers[scaffold]))
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
		"CODEX_SESSION_ID",
		"CODEX_COMPANION_SESSION_ID",
	}
}

// SelectReviewer picks the reviewer for one author out of what is installed.
//
// The reviewer is any installed provider that is not the author's. Where none
// exists the answer is a refusal rather than a fallback to the author's own
// provider: reviewing with the author's provider would reproduce the author's
// blind spots, which is the one outcome this exists to avoid, and it would do
// so while reporting success.
func SelectReviewer(author ambient.Scaffold, installed []ambient.Scaffold) (ambient.Scaffold, error) {
	for _, scaffold := range installed {
		if scaffold != author {
			return scaffold, nil
		}
	}
	return "", fmt.Errorf("no reviewer is installed other than the author's own provider (%s); "+
		"install a second agent scaffold, because reviewing with the author's provider reproduces the author's blind spots", author)
}
