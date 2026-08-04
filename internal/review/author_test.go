package review

import (
	"errors"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// environment turns a map into the lookup detection reads, so every legal
// invoking environment can be built by hand rather than by arranging one.
func environment(values map[string]string) Lookup {
	return func(name string) (string, bool) {
		value, set := values[name]
		return value, set
	}
}

func TestDetectAuthorReadsEachSingleProviderSession(t *testing.T) {
	claude, err := DetectAuthor(environment(map[string]string{"CLAUDECODE": "1"}))
	if err != nil || claude != ambient.ScaffoldClaudeCode {
		t.Fatalf("a Claude Code session detected as %q (%v)", claude, err)
	}

	codex, err := DetectAuthor(environment(map[string]string{"CODEX_SESSION_ID": "abc"}))
	if err != nil || codex != ambient.ScaffoldCodex {
		t.Fatalf("a Codex session detected as %q (%v)", codex, err)
	}

	desktop, err := DetectAuthor(environment(map[string]string{"CODEX_THREAD_ID": "thread-1"}))
	if err != nil || desktop != ambient.ScaffoldCodex {
		t.Fatalf("a Codex desktop session detected as %q (%v)", desktop, err)
	}
}

func TestDetectAuthorCollapsesMarkersFromTheSameProvider(t *testing.T) {
	author, err := DetectAuthor(environment(map[string]string{
		"CODEX_THREAD_ID":  "thread-1",
		"CODEX_SESSION_ID": "session-1",
	}))
	if err != nil || author != ambient.ScaffoldCodex {
		t.Fatalf("two Codex primary markers produced %q (%v)", author, err)
	}
}

// The case that makes the narrow matching necessary. This environment was
// measured on a real machine: a Claude Code session carries a Codex companion
// variable because a companion is reachable from it. Reading any CODEX_*
// variable as authorship would hand the review to the author's own provider and
// report success.
func TestDetectAuthorIgnoresACompanionOfTheOtherProvider(t *testing.T) {
	author, err := DetectAuthor(environment(map[string]string{
		"CLAUDECODE":                 "1",
		"CLAUDE_CODE_SESSION_ID":     "s-1",
		"CODEX_COMPANION_SESSION_ID": "c-1",
	}))
	if err != nil {
		t.Fatalf("a Claude Code session with a Codex companion refused: %v", err)
	}
	if author != ambient.ScaffoldClaudeCode {
		t.Fatalf("a companion variable decided authorship: got %q", author)
	}
}

func TestDetectAuthorRefusesWhenTwoSessionsClaimTheProcess(t *testing.T) {
	_, err := DetectAuthor(environment(map[string]string{
		"CLAUDECODE":      "1",
		"CODEX_THREAD_ID": "thread-1",
	}))
	if !errors.Is(err, ErrAuthorAmbiguous) {
		t.Fatalf("two primary markers produced %v", err)
	}
	for _, name := range []string{"CLAUDECODE", "CODEX_THREAD_ID"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("the refusal does not name %s: %v", name, err)
		}
	}
}

func TestDetectAuthorRefusesAnEmptyEnvironment(t *testing.T) {
	_, err := DetectAuthor(environment(nil))
	if !errors.Is(err, ErrAuthorUndetectable) {
		t.Fatalf("an empty environment produced %v", err)
	}
	for _, name := range MarkerNames() {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("the refusal does not name %s: %v", name, err)
		}
	}
}

// An exported-but-empty variable is how a wrapper says "not this". Reading it
// as presence is the same silent misattribution the narrow matching prevents.
func TestDetectAuthorTreatsAnEmptyMarkerAsAbsent(t *testing.T) {
	author, err := DetectAuthor(environment(map[string]string{
		"CLAUDECODE":      "",
		"CODEX_THREAD_ID": "thread-1",
	}))
	if err != nil || author != ambient.ScaffoldCodex {
		t.Fatalf("an empty marker was read as presence: %q (%v)", author, err)
	}

	if _, err := DetectAuthor(environment(map[string]string{"CLAUDECODE": "   "})); !errors.Is(err, ErrAuthorUndetectable) {
		t.Fatalf("a whitespace-only marker was read as presence: %v", err)
	}
}

func TestSelectReviewerPrefersCrossWhereBothRun(t *testing.T) {
	both := []ambient.Scaffold{ambient.ScaffoldCodex, ambient.ScaffoldClaudeCode}
	selection, err := SelectReviewer(ambient.ScaffoldClaudeCode, both, alwaysRunnable)
	if err != nil || selection.Reviewer != ambient.ScaffoldCodex || selection.Mode != "cross" {
		t.Fatalf("selected %+v for a Claude Code author (%v)", selection, err)
	}
	selection, err = SelectReviewer(ambient.ScaffoldCodex, both, alwaysRunnable)
	if err != nil || selection.Reviewer != ambient.ScaffoldClaudeCode || selection.Mode != "cross" {
		t.Fatalf("selected %+v for a Codex author (%v)", selection, err)
	}
}

// The single-tool machine is the ordinary case, not a degraded one. The
// mechanism is the fresh session; a second vendor only strengthens it.
func TestSelectReviewerFallsToFreshWithOnlyTheAuthorsProvider(t *testing.T) {
	selection, err := SelectReviewer(ambient.ScaffoldClaudeCode,
		[]ambient.Scaffold{ambient.ScaffoldClaudeCode}, alwaysRunnable)
	if err != nil {
		t.Fatalf("a single-provider machine refused: %v", err)
	}
	if selection.Reviewer != ambient.ScaffoldClaudeCode || selection.Mode != "fresh" {
		t.Fatalf("expected fresh mode with the author's provider, got %+v", selection)
	}
	if selection.Reason == "" {
		t.Fatal("a fresh selection carries no reason")
	}
}

// An unknown author still reviews; the choice is deterministic.
func TestSelectReviewerHandlesAnUnknownAuthor(t *testing.T) {
	both := []ambient.Scaffold{ambient.ScaffoldCodex, ambient.ScaffoldClaudeCode}
	selection, err := SelectReviewer(AuthorUnknown, both, alwaysRunnable)
	if err != nil || selection.Mode != "fresh" || selection.Reviewer != ambient.ScaffoldCodex {
		t.Fatalf("unknown author produced %+v (%v)", selection, err)
	}
}

// The one refusal left: nothing runs at all.
func TestSelectReviewerRefusesOnlyWhenNothingRuns(t *testing.T) {
	_, err := SelectReviewer(ambient.ScaffoldCodex,
		[]ambient.Scaffold{ambient.ScaffoldCodex, ambient.ScaffoldClaudeCode},
		func(string) (string, error) { return "", errors.New("not found") })
	if err == nil {
		t.Fatal("a machine with no runnable reviewer did not refuse")
	}
	if _, err := SelectReviewer(ambient.ScaffoldCodex, nil, alwaysRunnable); err == nil {
		t.Fatal("an empty candidate list did not refuse")
	}
}

// A cross that quietly became fresh is an unprovable claim; the reason says it.
func TestFreshFallbackNamesWhatWasUnavailable(t *testing.T) {
	selection, err := SelectReviewer(ambient.ScaffoldClaudeCode,
		[]ambient.Scaffold{ambient.ScaffoldCodex, ambient.ScaffoldClaudeCode},
		func(name string) (string, error) {
			if name == "codex" {
				return "", errors.New("not found")
			}
			return "/usr/bin/claude", nil
		})
	if err != nil || selection.Mode != "fresh" {
		t.Fatalf("expected a fresh fallback, got %+v (%v)", selection, err)
	}
	if !strings.Contains(selection.Reason, "codex") {
		t.Fatalf("the fallback reason does not name what was unavailable: %q", selection.Reason)
	}
}

// The reviewer must not inherit the session identity of the author. This
// repository already met that hazard: the live verification of the Claude Code
// path had to unset the CLAUDE* family by hand so a spawned session looked like
// a fresh one rather than a continuation of its parent.
func TestAuthorMarkersCoverEveryPrimaryMarker(t *testing.T) {
	stripped := make(map[string]struct{}, len(AuthorMarkers()))
	for _, name := range AuthorMarkers() {
		stripped[name] = struct{}{}
	}
	for _, name := range MarkerNames() {
		if _, present := stripped[name]; !present {
			t.Fatalf("a spawned reviewer would inherit %s", name)
		}
	}
	if _, present := stripped["CODEX_COMPANION_SESSION_ID"]; !present {
		t.Fatal("a spawned reviewer would inherit the companion marker")
	}
}

func TestStrippedEnvironmentRemovesOnlyWhatItWasGiven(t *testing.T) {
	kept := strippedEnvironment([]string{"KEEP=1", "DROP=2", "MALFORMED", "KEEP2=3"}, []string{"DROP"})
	joined := strings.Join(kept, ",")
	if strings.Contains(joined, "DROP=") {
		t.Fatalf("the named variable survived: %v", kept)
	}
	for _, expected := range []string{"KEEP=1", "KEEP2=3", "MALFORMED"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("%s was removed unasked: %v", expected, kept)
		}
	}
}

// alwaysRunnable stands in for a PATH where every reviewer exists.
func alwaysRunnable(string) (string, error) { return "/usr/bin/stub", nil }
