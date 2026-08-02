## ADDED Requirements

### Requirement: One command reviews the current branch in a fresh session
**Intent IDs:** OUT-1, OUT-3, SIG-1

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

The reviewer SHALL be invoked through the vendor's own documented
non-interactive interface, so the user's existing subscription is used as its
vendor intends. Goalrail MUST NOT vendor, pin, or wrap a reviewer, MUST NOT
require any provider's plugin, and MUST NOT depend on a third-party review
tool. A vendor's refusal surfaces to the caller unchanged, and being runnable —
the executable resolving, not a configuration directory existing — is what
makes a provider a candidate.

The command SHALL refuse in exactly one case: no reviewer can be run at all.

#### Scenario: One provider is installed
- **WHEN** the command runs where only the author's own provider can be run
- **THEN** it reviews in fresh mode with that provider in a clean session, writes a receipt naming the mode, and asks nothing

#### Scenario: Two providers are installed
- **WHEN** the command runs where the provider that did not author the change can be run
- **THEN** it reviews in cross mode with that provider

#### Scenario: Cross falls back to fresh, loudly
- **WHEN** two providers are installed but the non-author's command does not resolve
- **THEN** the review runs in fresh mode and the receipt records the fallback and its reason

#### Scenario: Nothing can review
- **WHEN** no reviewer executable resolves at all
- **THEN** the command refuses and names what is missing, and that is its only refusal besides the gate

#### Scenario: The reviewer cannot write
- **WHEN** a reviewer is invoked
- **THEN** it is given no editing tool and no write-capable sandbox, whatever the machine's own configuration says

#### Scenario: The vendor CLI refuses
- **WHEN** the reviewer's own command exits non-zero or reports an unusable invocation
- **THEN** the failure is reported as the reviewer's own, no receipt is written, and Goalrail does not retry with altered arguments

### Requirement: Authorship is inferred, and not knowing it refuses nothing
**Intent IDs:** OUT-1, OUT-5

The command SHALL infer the author's provider from the invoking environment,
matching each provider's primary session marker only: `CLAUDECODE` for a Claude
Code session, the Codex session identifier for a Codex one. A companion
variable of one tool inside another's session is an ordinary configuration and
MUST NOT count as authorship — a prefix match would have handed the review back
to its own author while reporting success.

An explicit override SHALL take precedence over detection unconditionally.

Where no primary marker is present, or more than one is, the command SHALL
proceed and record the author as `unknown`. The fresh session provides the
independence either way; detection only improves reviewer choice, so its
failure is information for the receipt, not a reason to stop. With authorship
unknown and several runnable providers, selection SHALL be deterministic.

#### Scenario: A single session marker decides
- **WHEN** the environment carries exactly one provider's primary session marker
- **THEN** that provider is the author and cross mode prefers the other

#### Scenario: Authorship cannot be determined
- **WHEN** the environment carries no primary marker, or two
- **THEN** the review proceeds, the receipt records the author as unknown, and the reviewer choice is deterministic

#### Scenario: The override wins
- **WHEN** an authorship override is given
- **THEN** it decides the author regardless of the environment

### Requirement: A budget gate is the only other refusal
**Intent IDs:** OUT-2, SIG-2

Where configuration names a gate command, the command SHALL run it before
invoking any reviewer — including a refute round — and SHALL refuse on a
non-zero exit, naming the gate as the reason. Every gate invocation SHALL run
under the review's own deadline, bounded across its whole process tree, because
a gate is routinely a pipeline whose descendant can outlive the shell. A gate
the deadline stopped SHALL be reported as a timeout carrying its own output, and
never as a refusal: it returned no verdict, and telling automation the budget
denied the review is a different fact with a different response. The gate SHALL be a command named
in configuration and MUST NOT be a path, provider, or budget service built into
Goalrail. Where no gate is configured, the review SHALL proceed, and nothing
SHALL be reported as missing.

#### Scenario: A configured gate refuses
- **WHEN** the gate command exits non-zero
- **THEN** no reviewer is invoked, no receipt is written, and the refusal names the gate

#### Scenario: A gate outlives the deadline
- **WHEN** a gate, or any process it spawned, is still running at the deadline
- **THEN** the review ends within the bound, reports a timeout rather than a refusal, and carries whatever the gate had produced

#### Scenario: No gate is configured
- **WHEN** configuration names no gate command
- **THEN** the review proceeds without one

### Requirement: A re-review covers what changed, and the chain proves the rest
**Intent IDs:** OUT-3, SIG-4

Where a receipt already exists for the branch, a new review SHALL by default
cover only the range from that receipt's head to the present head, so the cost
of a round is proportional to what the round is about. Reviewing the whole
branch again on every round was measured into the failure this prevents: each
round repaid the full price of every previous one.

The receipt SHALL record both the range this round actually reviewed and the
digest of the branch's full diff, so the chain of receipts proves cumulative
coverage while staleness stays a single comparison. A full re-review SHALL
remain available by explicit flag.

The reviewer's read access is not narrowed by the range: it may read any file
it needs. Only the diff under review is scoped. The accepted limit is stated
rather than hidden: a later round's change can break what an earlier round
approved, and an incremental diff will not show it — the same limit human
incremental review carries.

#### Scenario: The second round is incremental
- **WHEN** a review runs on a branch that already has a receipt
- **THEN** the reviewed range starts at the previous receipt's head, and the receipt records both that range and the full branch digest

#### Scenario: A full pass is available
- **WHEN** the caller asks for a full review by flag
- **THEN** the whole branch diff is reviewed regardless of previous receipts

#### Scenario: The first round is the whole branch
- **WHEN** no receipt exists for the branch
- **THEN** the reviewed range is the branch's full diff against the base

### Requirement: A refute round challenges findings without Goalrail judging them
**Intent IDs:** OUT-8, SIG-3

The command SHALL offer a refute round in every mode: a fresh session receives
the previous report and the reviewed diff and is instructed to refute the
findings rather than add new ones. The refuter SHALL be the other runnable
provider where one exists and the same provider in a clean session otherwise —
the value of the round is a fresh attempt to kill the findings before they are
acted on, and that value does not require a second vendor.

The round SHALL run only when the caller asks for it. The rule the caller
applies is stated here so every loop applies the same one: findings exist and
the caller is about to act on them; zero findings end the round. Goalrail MUST
NOT trigger the round itself, MUST NOT assess risk, and MUST NOT read either
report to decide anything. Passing the report verbatim to the refuter is not
reading it.

The receipt for a refuted review SHALL carry both reports as verbatim bytes,
each with its own digest. Which findings survived is the reader's judgement.

#### Scenario: A refute round with two providers
- **WHEN** the caller triggers refute where the other provider is runnable
- **THEN** that provider receives the report and the diff in a fresh session, and the receipt carries both reports verbatim

#### Scenario: A refute round with one provider
- **WHEN** the caller triggers refute where only one provider is runnable
- **THEN** the same provider refutes in a clean session, and the receipt says so

#### Scenario: Goalrail never decides
- **WHEN** a refuted receipt is inspected
- **THEN** no field states which findings survived, were confirmed, or were dismissed

### Requirement: A round that changed nothing is measured, not judged
**Intent IDs:** OUT-3, OUT-5

Where a round reviews the same head as the previous receipt and the working
tree carries no change of the author's, the receipt SHALL record it as a
consecutive unchanged round, counting up across such rounds and resetting to
zero the moment work moves. The previous round produced findings the author
acted on nothing about, and further rounds spend without converging — a loop
needs a stop signal that does not require reading any report, and this is one
that is measured.

A round counts only where the previous round was also over identical state:
both rounds at the same head, and neither carrying work of the author's. Work
that was written and then discarded moved between the rounds even though
nothing landed, and a stop signal that cannot tell that apart is one nobody
should stop on — so the receipt SHALL record whether the tree was clean when
each round ran, and SHALL distinguish "not known" from "not clean". A receipt
written before that field existed, or a round whose tree could not be measured,
knows nothing about that state; recording either as unclean would state an
unknown as a measurement, and a round it cannot vouch for SHALL never count. The measurement SHALL be taken regardless of the reviewed
scope: a full pass changes what is read, not whether anything moved.

Goalrail SHALL report the count and MUST NOT act on it: it neither refuses the
review nor ends any loop, because the loop belongs to the caller. The
**untracked** instructions file Goalrail itself materialized MUST NOT count as
the author's work, or Goalrail's own artifact would suppress its own signal in
every repository that has not committed it yet — but once that file is
committed, an edit or deletion of it is ordinary work and MUST count, because
changing the review rules is a change.

#### Scenario: Consecutive rounds that changed nothing
- **WHEN** a review runs twice against the same head with no change of the author's in the tree
- **THEN** the second receipt records one consecutive unchanged round, and a third records two

#### Scenario: Acting on the findings resets it
- **WHEN** a commit lands after an unchanged round and the review runs again
- **THEN** the count is zero

#### Scenario: Work in progress is not a stalemate
- **WHEN** the tree carries the author's uncommitted changes at the same head
- **THEN** the count is zero

#### Scenario: Goalrail's own artifact does not count
- **WHEN** the instructions file was materialized and never committed
- **THEN** it does not make the tree count as changed

#### Scenario: An edit to committed instructions is work
- **WHEN** the instructions file is committed and then modified or deleted by the author
- **THEN** the tree counts as changed

#### Scenario: Discarded work is not a stalemate
- **WHEN** a round ran with the author's work in the tree and that work is discarded before the next round at the same head
- **THEN** the count is zero, and only the round after that can begin counting

#### Scenario: An unknown tree state never counts
- **WHEN** the previous receipt carries no tree state, or this round's tree cannot be measured
- **THEN** the count is zero and the receipt records the state as unknown rather than as unclean

#### Scenario: A full pass is measured too
- **WHEN** an unchanged round is repeated with the full-review flag
- **THEN** the count still rises, because scope is not movement

### Requirement: The receipt is bound to what was reviewed, and to how
**Intent IDs:** OUT-5, SIG-3

A completed review SHALL leave a receipt carrying: the base and head commits of
the reviewed range, the digest of that range's canonical diff, the digest of
the branch's full canonical diff, the mode and the reason it was selected, the
reviewer's identity, the author or `unknown`, the measured duration, the time,
and every report as verbatim bytes with a digest per report.

No field SHALL be derived by reading, parsing, scoring, or summarizing a
report. The canonical diff SHALL be rendered with the reader's own git
configuration taken out of the answer, so the same range digests identically on
any machine and staleness describes the branch rather than whoever asked.

The receipt SHALL be written to Goalrail's per-clone state with the same
protection the rest of the local state store carries, MUST NOT be written into
the repository, and MUST NOT be written at all when the reviewer did not
complete or when it could never be read back within the receipt bound.

#### Scenario: A receipt describes its own review
- **WHEN** a review completes
- **THEN** recomputing the range digest from the receipt's own commits reproduces it, the reports are byte-identical, and the mode, reason, author and duration are present

#### Scenario: The digest ignores the reader's configuration
- **WHEN** a diff-affecting git setting changes after a review
- **THEN** the recorded digest still reproduces, and the branch does not read as stale for it

#### Scenario: No receipt without a review
- **WHEN** the reviewer fails, times out, or emits more than the receipt bound
- **THEN** nothing is stored, and the failure names its cause

### Requirement: A review is bounded, whole tree included
**Intent IDs:** OUT-3, SIG-1

Every review SHALL run under a deadline, and the deadline SHALL bound the whole
process tree the reviewer spawns, not its direct child alone. The reviewers are
wrappers whose descendants inherit the pipes; a deadline that kills the parent
and then waits on the pipes was measured letting a twenty-minute limit run to
fifty-three. A review that exceeds its deadline SHALL be reported as such,
write no receipt, and leave no reviewer processes behind.

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
`openspec/changes/pre-pr-review-v0/evidence/effort-experiment-2026-08-01.md`. The deadline travels with the
effort because raising one without the other only moves the failure from a false
clean verdict to an unfinished review.

#### Scenario: A full pass is thorough by default
- **WHEN** a full pass runs with no effort and no deadline named by the caller
- **THEN** it uses the full-pass effort and the full-pass deadline, both above the loop round's

#### Scenario: The caller's values win
- **WHEN** a full pass runs with an effort or a deadline named explicitly
- **THEN** those are used, however cheap or short

#### Scenario: A reviewer that outlives its child is still bounded
- **WHEN** the reviewer's descendant keeps the pipes open past the deadline
- **THEN** the review ends within the deadline plus a short grace, reports the deadline as the cause, and writes no receipt

#### Scenario: The model is stated, per provider
- **WHEN** a review runs with no model named by the caller
- **THEN** a provider with an evidenced default runs on it, and a provider without one keeps its own configuration untouched

#### Scenario: A model name does not cross providers
- **WHEN** a refute round runs with a refuter of a different provider than the reviewer
- **THEN** the refuter resolves its own provider's model rather than receiving the reviewer's

#### Scenario: Effort is stated
- **WHEN** an ordinary loop round runs with no effort named by the caller
- **THEN** the reviewer is invoked at the moderate default, not at whatever the machine's interactive configuration names

### Requirement: Reviewer instructions are repository content
**Intent IDs:** OUT-7

The instructions handed to the reviewer SHALL be read from a committed file in
the repository, materialized with a default when none exists and never
overwritten. The file MUST NOT live at a path Goalrail's own ignore rules
exclude from version control. The default SHALL carry what the findings ratchet
has promoted, and SHALL include the refuter's instruction text, so both rounds
draw from the same committed source.

#### Scenario: The default is materialized once
- **WHEN** the command runs in a repository with no instructions file
- **THEN** the default is written, is committable, and the review uses it

#### Scenario: Edited instructions are used
- **WHEN** the instructions file has been edited and the command runs again
- **THEN** the reviewer receives the edited instructions and the file is unmodified afterwards
