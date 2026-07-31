package harness

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/updatecheck"
)

func answering(version string, at time.Time) updatecheck.Check {
	return func(context.Context) (string, time.Time, error) { return version, at, nil }
}

func failing(err error) updatecheck.Check {
	return func(context.Context) (string, time.Time, error) { return "", time.Time{}, err }
}

// TestAvailabilityReportsEachStateAsAFact walks every state the diagnosis can
// report and pins that each one is a statement rather than a demand.
func TestAvailabilityReportsEachStateAsAFact(t *testing.T) {
	checkedAt := time.Date(2026, 7, 30, 19, 44, 12, 0, time.UTC)
	for _, testCase := range []struct {
		name       string
		check      updatecheck.Check
		local      string
		wantState  string
		wantInNote []string
	}{
		{
			name: "a newer release exists", check: answering("v0.2.0", checkedAt), local: "v0.1.1",
			wantState: updateNewer, wantInNote: []string{"v0.2.0", "v0.1.1", updatecheck.Source, "2026-07-30T19:44:12Z"},
		},
		{
			name: "nothing newer was found", check: answering("v0.1.1", checkedAt), local: "v0.1.1",
			wantState: updateCurrent, wantInNote: []string{"nothing newer", updatecheck.Source, "2026-07-30T19:44:12Z"},
		},
		{
			name: "the source is behind this binary", check: answering("v0.1.0", checkedAt), local: "v0.1.1",
			wantState: updateCurrent, wantInNote: []string{"nothing newer"},
		},
		{
			name: "this build is not a release", check: answering("v0.2.0", checkedAt), local: "v0.1.2-0.2026073018-abc+dirty",
			wantState: updateNotApplicable, wantInNote: []string{"not a release"},
		},
		{
			name: "the user declined", check: failing(updatecheck.DeclinedError{Reason: "GOPROXY does not start with the public proxy"}), local: "v0.1.1",
			wantState: updateDeclined, wantInNote: []string{"not checked", "GOPROXY"},
		},
		{
			name: "the check failed", check: failing(context.DeadlineExceeded), local: "v0.1.1",
			wantState: updateUnavailable, wantInNote: []string{"did not run"},
		},
		{
			name: "no check was supplied", check: nil, local: "v0.1.1",
			wantState: updateNotChecked, wantInNote: []string{"not checked"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := availability(testCase.check, testCase.local)
			if state.State != testCase.wantState {
				t.Errorf("state = %q, want %q", state.State, testCase.wantState)
			}
			for _, want := range testCase.wantInNote {
				if !strings.Contains(state.Note, want) {
					t.Errorf("note %q does not mention %q", state.Note, want)
				}
			}
			// However it turns out, it never tells the user to run something.
			for _, forbidden := range []string{"run `", "gr update", "gr init"} {
				if strings.Contains(state.Note, forbidden) {
					t.Errorf("note %q prescribes a command; this is a fact, not a problem", state.Note)
				}
			}
		})
	}
}

// TestAFailingCheckSaysNothingTheSourceSaid pins that the source's own words —
// which carry its internal paths and git errors — never reach a user.
func TestAFailingCheckSaysNothingTheSourceSaid(t *testing.T) {
	leaky := failing(&leakyError{"/tmp/gopath/pkg/mod/cache: fatal: could not read Username for 'https://github.com'"})
	state := availability(leaky, "v0.1.1")
	if strings.Contains(state.Note, "gopath") || strings.Contains(state.Note, "Username") {
		t.Errorf("the source's own text reached the report: %q", state.Note)
	}
}

type leakyError struct{ text string }

func (e *leakyError) Error() string { return e.text }

// TestADeclinedCheckReadsDifferentlyFromAFailedOne pins the distinction the
// promoted requirement asks for: obeying and not getting through are not the
// same event, and a user who turned the check off should not think it broke.
func TestADeclinedCheckReadsDifferentlyFromAFailedOne(t *testing.T) {
	declined := availability(failing(updatecheck.DeclinedError{Reason: "GR_NO_UPDATE_CHECK is set"}), "v0.1.1")
	failed := availability(failing(context.DeadlineExceeded), "v0.1.1")
	if declined.State == failed.State || declined.Note == failed.Note {
		t.Errorf("a declined check reads as a failed one: %q vs %q", declined.Note, failed.Note)
	}
}

// TestAnAnswerFromTheCacheIsStillAnAnswer pins what three independent reviewers
// caught in the plan: at a daily interval a cache hit is the ordinary path, and
// reporting it as "the check did not run" would hide the answer almost always.
func TestAnAnswerFromTheCacheIsStillAnAnswer(t *testing.T) {
	yesterday := time.Now().UTC().Add(-20 * time.Hour).Truncate(time.Second)
	state := availability(answering("v0.2.0", yesterday), "v0.1.1")
	if state.State != updateNewer {
		t.Fatalf("a cached answer reported as %q", state.State)
	}
	if !strings.Contains(state.Note, yesterday.Format(time.RFC3339)) {
		t.Errorf("note %q does not say when the answer was obtained", state.Note)
	}
}

// TestTheCheckNeverClaimsCurrency pins the wording rule that the source's own
// lag forces: the report says what was found and when, and never that the user
// is up to date.
func TestTheCheckNeverClaimsCurrency(t *testing.T) {
	state := availability(answering("v0.1.1", time.Now().UTC()), "v0.1.1")
	for _, forbidden := range []string{"up to date", "up-to-date", "latest version", "you are current"} {
		if strings.Contains(strings.ToLower(state.Note), forbidden) {
			t.Errorf("note %q claims currency the source cannot support", state.Note)
		}
	}
}

// TestNoCheckIsMadeWhenThisBuildIsNotARelease pins that the short-circuit comes
// before the request, not after it: there is nothing to learn.
func TestNoCheckIsMadeWhenThisBuildIsNotARelease(t *testing.T) {
	asked := false
	check := func(context.Context) (string, time.Time, error) {
		asked = true
		return "v0.2.0", time.Now(), nil
	}
	for _, local := range []string{"(devel)", "unknown", "v0.1.1+dirty", "v0.1.2-0.2026073018-abc"} {
		asked = false
		if state := availability(check, local); state.State != updateNotApplicable {
			t.Errorf("%s reported as %q", local, state.State)
		}
		if asked {
			t.Errorf("%s still caused a request", local)
		}
	}
}

// TestTheUpdateLineChangesNoVerdict is the promise every unattended caller
// depends on: a repository whose harness is intact is working, and exits zero,
// whether or not the binary running the check is a release behind.
//
// It is written against the summary rather than a live diagnosis for a reason
// worth keeping: a test binary reports the toolchain's development marker in
// every version-control state, so no test can reach the "newer" state through
// Diagnose. What carries the invariant is that the summary never reads the field
// at all, and that is what is asserted.
func TestTheUpdateLineChangesNoVerdict(t *testing.T) {
	base := Diagnosis{Initialized: true, Overlay: OverlayState{Present: true, Complete: true, Current: true}}
	baseProblems, baseActions := summarize(base)

	for _, state := range []UpdateAvailability{
		{State: updateNewer, Latest: "v99.0.0", Source: updatecheck.Source, Note: "update: v99.0.0 is released"},
		{State: updateCurrent, Note: "update: nothing newer found"},
		{State: updateNotApplicable, Note: "update: not checked"},
		{State: updateDeclined, Note: "update: not checked (declined)"},
		{State: updateUnavailable, Note: "update: the check did not run"},
		{State: updateNotChecked, Note: "update: not checked"},
	} {
		withUpdate := base
		withUpdate.Update = state
		problems, actions := summarize(withUpdate)
		if len(problems) != len(baseProblems) || len(actions) != len(baseActions) {
			t.Errorf("state %q changed the problem count: %d, want %d", state.State, len(problems), len(baseProblems))
		}
		for _, problem := range problems {
			if strings.Contains(strings.ToLower(problem), "update") || strings.Contains(strings.ToLower(problem), "release") {
				t.Errorf("state %q became a problem: %q", state.State, problem)
			}
		}
	}
}

// TestTheReportCarriesOneUpdateLine pins where the fact appears: once, beside the
// toolchain and observability lines, and never among the problems.
func TestTheReportCarriesOneUpdateLine(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	diagnosis, err := Diagnose(DiagnoseInput{
		RepositoryRoot: root,
		Home:           t.TempDir(),
		StateRoot:      t.TempDir(),
		LatestRelease:  answering("v99.0.0", time.Now().UTC()),
	})
	if err != nil {
		t.Fatal(err)
	}
	report := Describe(diagnosis)
	if count := strings.Count(report, "update: "); count != 1 {
		t.Errorf("the report carries %d update lines, want 1:\n%s", count, report)
	}
	if strings.Contains(report, "problem: update") {
		t.Errorf("the update line was reported as a problem:\n%s", report)
	}
}
