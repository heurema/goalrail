package harness

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/heurema/goalrail/internal/updatecheck"
)

// UpdateAvailability is what the diagnosis knows about newer releases.
//
// It is a fact, never a problem. The tool has no command that replaces its own
// binary, so a next action here would name a command the tool does not accept;
// and a repository whose harness is intact is working whether or not the binary
// running the check is a release behind.
type UpdateAvailability struct {
	// State is one of: newer, nothing_newer_found, not_applicable, declined,
	// unavailable, not_checked. The second is deliberately not "current": this
	// lookup can only say what it found, and a machine consumer reading
	// "current" as authoritative would be trusting a source that measurably
	// lags a fresh release.
	State string `json:"state"`

	// Latest is the newest released version, where one was learned.
	Latest string `json:"latest,omitempty"`

	// Source is the counterparty that answered, named only where an answer rests
	// on one. The request itself identifies nothing, so this is where the report
	// discloses whom it asked.
	Source string `json:"source,omitempty"`

	// CheckedAt is when that answer was obtained, which may be earlier than now:
	// an answer reused from the cache is still an answer, and saying when it was
	// learned is what keeps it honest.
	CheckedAt string `json:"checked_at,omitempty"`

	// Note is the one line a report prints.
	Note string `json:"note"`
}

const (
	updateNewer         = "newer"
	updateCurrent       = "nothing_newer_found"
	updateNotApplicable = "not_applicable"
	updateDeclined      = "declined"
	updateUnavailable   = "unavailable"
	updateNotChecked    = "not_checked"
)

// availability compares what the source says with what this binary reports about
// itself.
//
// It never claims currency. The source discovers a release late when nobody asks
// it — measured on this repository at minutes to a quarter of an hour — so the
// most that can honestly be said is what was found and when it was looked for.
func availability(check updatecheck.Check, local string) UpdateAvailability {
	if check == nil {
		return UpdateAvailability{State: updateNotChecked, Note: "update: not checked"}
	}
	if !updatecheck.IsRelease(local) {
		// Asking would be pointless: nothing can be concluded from comparing a
		// release against a build that is not one.
		return UpdateAvailability{
			State: updateNotApplicable,
			Note:  fmt.Sprintf("update: not checked (this build reports %s, which is not a release)", local),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), updatecheck.Timeout)
	defer cancel()
	latest, checkedAt, err := check(ctx)
	if err != nil {
		var declined updatecheck.DeclinedError
		if errors.As(err, &declined) {
			return UpdateAvailability{
				State: updateDeclined,
				Note:  "update: not checked (" + declined.Reason + ")",
			}
		}
		// Every failure reads the same to the user, and none of the source's own
		// words reach them.
		return UpdateAvailability{State: updateUnavailable, Note: "update: the check did not run"}
	}

	state := UpdateAvailability{
		Latest:    latest,
		Source:    updatecheck.Source,
		CheckedAt: checkedAt.UTC().Format(time.RFC3339),
	}
	if updatecheck.Newer(latest, local) {
		state.State = updateNewer
		state.Note = fmt.Sprintf("update: %s is released, this is %s (asked %s at %s)",
			latest, local, updatecheck.Source, state.CheckedAt)
		return state
	}
	state.State = updateCurrent
	state.Note = fmt.Sprintf("update: nothing newer than %s found as of %s (asked %s)",
		local, state.CheckedAt, updatecheck.Source)
	return state
}

// InspectUpdateAvailability exposes the existing bounded, optional release
// fact to the project-aware doctor without moving network authority into that
// package. A nil check remains a read-free "not checked" result.
func InspectUpdateAvailability(
	check func(context.Context) (version string, checkedAt time.Time, err error),
	local string,
) UpdateAvailability {
	if check == nil {
		return availability(nil, local)
	}
	return availability(updatecheck.Check(check), local)
}
