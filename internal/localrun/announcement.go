package localrun

import "errors"

// ErrAnnouncementUndeliverable is returned when an adapter has no path for
// telling a run that the escalation channel exists.
//
// The launch fails rather than degrading: a run that cannot escalate guesses,
// and its receipt looks like any other. That is worse than the world before the
// channel existed, because the evidence would imply a capability that was not
// present.
var ErrAnnouncementUndeliverable = errors.New("escalation announcement cannot be delivered")

// EscalationAnnouncement is the exact text a launched run is told. It is fixed
// rather than composed per run, and it is deliberately narrow.
//
// It names the channel and its conditions. It says nothing about the work item,
// does not characterise the repository, does not suggest that a conflict or an
// ambiguity exists, and never tells the run to look for one. The measurement
// that motivates the channel recorded that its treatment branch was loud —
// the channel occupied about a third of the task statement — and flagged its own
// result as possibly inflated. A loud announcement would measure the
// announcement instead of the environment.
//
// It also names no provider: transport is the adapter's business, content is
// not.
const EscalationAnnouncement = `This run has one escalation channel.

If the work item cannot be completed as specified from this repository alone,
write the question to ` + ReservedEscalationPath + ` and change nothing else.
The run then ends as blocked, and the question goes to the owner for a decision.
Answering it starts a new run; this one does not resume.

The escalation only stands on an otherwise untouched scope. A question that
arrives together with edits inside the declared scope is recorded as a failure,
not as a question.

The payload format is goalrail.escalation/v0.`
