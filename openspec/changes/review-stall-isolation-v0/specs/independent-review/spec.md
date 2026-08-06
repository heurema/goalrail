## MODIFIED Requirements

### Requirement: One command reviews the current branch in a fresh session
**Intent IDs:** OUT-1, OUT-3, SIG-1, SIG-3, SIG-4

Goalrail SHALL provide a command that reviews the current branch's changes
against a base ref by running a reviewer in a fresh session that never sees the
author's context, and SHALL do so without asking anyone anything. Context
asymmetry is the mechanism: an author cannot review their own artifact, and a
reviewer that shares the author's reasoning reproduces the author's blind
spots. A different provider strengthens that independence; it is not its
precondition.

The mode SHALL follow from what is installed and runnable, never from a
judgement. Where one provider can be run, the reviewer is that provider in a
clean session — **fresh**, an ordinary mode rather than a degraded one, because
most users have one tool. Where the provider that did not author the change can
be run, it is preferred — **cross**. A cross selection that falls back to fresh
because the other provider's command would not run SHALL be recorded with its
reason, never silent: "a different vendor reviewed this" must stay a provable
claim.

The reviewer SHALL be invoked without the capability to modify the repository,
and MUST NOT be merely instructed to refrain. A reviewer that can edit the work
is no longer reviewing it, and asking is not the same as removing the
permission: this boundary was defeated twice while it lived in an allowance —
first by a shell wildcard, then by a write-capable flag on a read-only-looking
subcommand. Where a provider's sandbox is configurable, the invocation SHALL set
it to read-only and override whatever the machine configures; where tools are
enumerated, no editing tool SHALL appear.

A fresh session is fresh in what it inherits, and the machine's provider
integrations are inherited. Stripping the author's environment does not reach
them: they are named in the provider's own configuration, and one of them was
measured blocking a reviewer indefinitely while the environment was already
stripped and the sandbox already read-only. The command SHALL therefore accept
integration names from the caller and remove each named integration from the
reviewer's session. The caller supplies only a name; Goalrail renders that name
into the provider's own documented removal syntax and adds nothing else. An
arbitrary argument passthrough SHALL NOT be offered: it would turn the reviewer's
read-only boundary back into a filtering problem, and that boundary has already
been defeated twice while it lived in an allowance. Because the rendering is
fixed and the only caller input is a bounded identifier, no caller input can
reach the sandbox mode, the model, the effort, or the reviewer identity recorded
in the receipt.

Goalrail MUST NOT name, enumerate, detect, or special-case any provider
integration, server, or tool: what to remove is the caller's knowledge of their
own machine, not a list this repository maintains and ages. Isolation SHALL be
off by default, because an integration is ordinarily an asset. A provider whose
interface cannot express the removal of one named integration SHALL refuse the
request rather than accept it and do nothing, because a silently ignored
isolation request reads as isolation that happened.

The reviewer SHALL be invoked through the vendor's own documented
non-interactive interface, so the user's existing subscription is used as its
vendor intends. Goalrail MUST NOT vendor, pin, or wrap a reviewer, MUST NOT
require any provider's plugin, and MUST NOT depend on a third-party review
tool. A vendor's refusal surfaces to the caller unchanged, and being runnable —
the executable resolving, not a configuration directory existing — is what
makes a provider a candidate.

#### Scenario: An integration the provider cannot switch off blocks the reviewer
- **WHEN** a caller names an integration to remove and the reviewer's provider can express that removal
- **THEN** the reviewer runs without it and the review completes, while Goalrail names no integration of its own

#### Scenario: Isolation is not the default
- **WHEN** a review runs with no integration named
- **THEN** the reviewer keeps the machine's ordinary integrations

#### Scenario: A name cannot reach another setting
- **WHEN** a caller supplies a name that is empty, oversized, or carries characters that could alter the invocation's structure
- **THEN** the review refuses before any reviewer starts, and the sandbox, model, effort, and reviewer identity arguments are what they would have been without isolation

#### Scenario: A provider that cannot express the removal
- **WHEN** a caller names an integration and the reviewer's provider offers no per-integration removal
- **THEN** the review refuses and says so, rather than running a reviewer that kept the integration

### Requirement: A review is bounded, whole tree included
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2, SIG-5

Every review SHALL run under a deadline, and the deadline SHALL bound the whole
process tree the reviewer spawns, not its direct child alone. The reviewers are
wrappers whose descendants inherit the pipes; a deadline that kills the parent
and then waits on the pipes was measured letting a twenty-minute limit run to
fifty-three. A review that exceeds its deadline SHALL be reported as such,
write no receipt, and leave no reviewer processes behind.

A deadline bounds total cost; it does not bound waste. A reviewer that stops
entirely pays the whole deadline and returns nothing, and the caller cannot tell
it apart from one still working — measured twice on one branch, at 25 and then
50 minutes, each run recording no reviewer event whatever for the remainder after
a single blocking call. Every review SHALL therefore also run under a progress
bound: where the reviewer produces no observable output for a bounded period, the
review SHALL stop, report a stalled reviewer as an outcome distinct from an
exceeded deadline, write no receipt, and leave no reviewer processes behind. The
report SHALL name the reviewer's last observed activity, taken from output
already captured, because the alternative is reading provider session files to
learn what a command already knew. The progress bound SHALL be shorter than the
deadline it accompanies and SHALL be nameable by the caller; doubling a deadline
does not diagnose a stall, which is what the second measured run established.

A stalled review MUST NOT be recorded, reported, or counted as a completed
review, a finding-free review, or a passing one. It reviewed nothing.

The reviewer's model SHALL be stated on the invocation rather than inherited
from the authoring session, which chose its model for writing code and not for
checking it. A default SHALL be set per provider and only where there is
evidence for one; where there is none, the provider's own configuration is left
alone rather than overridden by a pinned vendor identifier that would age. A
model named by the caller SHALL apply to the reviewer it was named for and MUST
NOT be carried to a refuter of another provider, because a model name belongs to
one vendor and the second invocation would fail after the first has been paid
for.

The receipt SHALL record the resolved effort and model, and the refuter's model
separately where a refute round ran, and stored receipts SHALL remain
distinguishable by those settings rather than one overwriting another. Where
Goalrail pinned no model, the receipt SHALL say that the provider used its own
configuration rather than leaving the field empty: an empty field reads as "no
model" and the provider certainly used one. What that configuration named is
outside what the receipt can prove, and stating the limit is required rather
than implying completeness.

The reviewer's reasoning effort SHALL be stated on the invocation rather than
inherited from the machine's interactive configuration, with a moderate
default: a review is a step inside a loop that runs again after every fix, so
its cost is paid on every round.

A full pass SHALL default to a higher effort and a longer deadline than a loop
round, and the caller's explicit values SHALL always win over both. A full pass
is the thoroughness pass by definition, and running it at the loop's cheap
default produces a confident wrong verdict — the most expensive output this
command has. Measured on one range with one set of instructions, effort the only
variable: the moderate default reviewed clean and missed three real defects, two
of them P1, which the higher effort reported — recorded with its design, ranges,
durations and reproduction command in
`openspec/changes/archive/2026-08-04-pre-pr-review-v0/evidence/effort-experiment-2026-08-01.md`. The deadline travels with the
effort because raising one without the other only moves the failure from a false
clean verdict to an unfinished review.

#### Scenario: A reviewer stops producing output
- **WHEN** a reviewer emits nothing for longer than the progress bound while its deadline still has time left
- **THEN** the review stops within that bound, reports a stalled reviewer distinctly from a deadline overrun, names its last observed activity, writes no receipt, and leaves no reviewer processes behind

#### Scenario: A slow reviewer is not a stalled one
- **WHEN** a reviewer keeps producing output but is still working as the deadline approaches
- **THEN** the progress bound does not stop it, and exceeding the deadline is reported as a deadline overrun

#### Scenario: A stalled review is not a clean one
- **WHEN** a review stopped for lack of progress is read back by any caller or receipt consumer
- **THEN** it is not presented as completed, finding-free, or passing

#### Scenario: A full pass is thorough by default
- **WHEN** a full pass runs with no effort and no deadline named by the caller
- **THEN** it uses the full-pass effort and the full-pass deadline, both above the loop round's

#### Scenario: The caller's values win
- **WHEN** a full pass runs with an effort or a deadline named explicitly
- **THEN** those are used, however cheap or short

#### Scenario: A reviewer that outlives its child is still bounded
- **WHEN** the reviewer's descendant keeps the pipes open past the deadline
- **THEN** the review ends within the deadline plus a short grace, reports the deadline as the cause, and writes no receipt

#### Scenario: Canonical diff rendering reaches the deadline
- **WHEN** rendering the canonical reviewed range has not completed by the review deadline
- **THEN** its Git process is stopped, no instructions, gate, or reviewer side effect occurs, and no receipt is written

#### Scenario: The model is stated, per provider
- **WHEN** a review runs with no model named by the caller
- **THEN** a provider with an evidenced default runs on it, and a provider without one keeps its own configuration untouched

#### Scenario: A model name does not cross providers
- **WHEN** a refute round runs with a refuter of a different provider than the reviewer
- **THEN** the refuter resolves its own provider's model rather than receiving the reviewer's

#### Scenario: Effort is stated
- **WHEN** an ordinary loop round runs with no effort named by the caller
- **THEN** the reviewer is invoked at the moderate default, not at whatever the machine's interactive configuration names
