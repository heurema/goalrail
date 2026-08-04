# harness-doctor Specification

## Purpose

Define the single surface that answers "is this repository's harness intact, and
if not, what do I run": attachment state per scaffold, overlay drift reported per
file, currency decided by comparing digests rather than by trusting a recorded
version, the checking toolchain reported as a fact rather than a fault, and
observability reported once as optional without nagging or leaking. Every state
that is not working names an action the tool can actually perform, and the report
is readable by a machine as well as a person.
## Requirements
### Requirement: One diagnosis covers the whole harness
**Intent IDs:** OUT-6, OUT-11, SIG-7, SIG-13

Goalrail SHALL expose one diagnosis surface for a repository's harness, grown
from attachment health rather than replacing it, and it SHALL report at least:
the repository is not initialized; the overlay is missing; one or more overlay
files have drifted from the binary's canon; the overlay is behind that canon; the
attachment is not registered; the attachment is registered; a registration for
the scaffold exists in a scope this arrangement no longer uses; a registration
names an event this arrangement supersedes; the checking toolchain's presence;
observability's presence; and everything is working.

Every state that is not working SHALL name its next action. An optional facility
that is simply absent SHALL be stated without a next action and without
contributing a failure, because a report that nags about what the product
declares optional teaches the user to ignore it.

Drift SHALL be reported per file. "Something under the overlay differs" leaves
the user diffing the overlay by hand, file by file, which is exactly the work
the diagnosis exists to remove.

The diagnosis SHALL be readable by a machine as well as a person: structured
output alongside the human report, and an exit status that separates healthy from
not, so the check can run unattended.

Every next action the diagnosis prints SHALL be a command this tool accepts —
or, where the tool deliberately has no command for the act because the content
is the user's own to remove, the action SHALL say that plainly rather than
inventing a command. A report that prescribes a command which cannot be run, or
cannot perform the repair, is worse than no report: the user follows correct
advice and has no remaining move.

#### Scenario: The repository is not initialized
- **WHEN** diagnosis runs in a directory with no Goalrail marker
- **THEN** it reports that state and names the initialization command

#### Scenario: The overlay is missing
- **WHEN** diagnosis runs where the marker exists but the overlay does not
- **THEN** it reports the overlay as missing and names the command that installs it

#### Scenario: An overlay file has drifted
- **WHEN** one materialized overlay file differs from the binary's canonical copy
- **THEN** the diagnosis names that file specifically and reports drift, rather than reporting the overlay as generally divergent

#### Scenario: The overlay is behind the canon
- **WHEN** the repository's overlay matches an older canon than the installed binary carries
- **THEN** the diagnosis reports the overlay as behind and names the update command

#### Scenario: A registration exists in the superseded scope
- **WHEN** a registration for a repository-scope scaffold is found in user-level configuration
- **THEN** the diagnosis names it, explains that it duplicates the repository-scope attachment, and names the consented command that removes it — without modifying it

#### Scenario: A registration names a superseded event
- **WHEN** a registration is found naming a stop-like event that fires once per turn, from an earlier arrangement
- **THEN** the diagnosis reports it as needing repair, names the consequence of leaving it — one question retained again on every turn — and names the command that repairs it

#### Scenario: Everything is working
- **WHEN** the repository is initialized, the overlay matches the canon, and the attachment is registered
- **THEN** the diagnosis reports the harness as working and asks nothing further

#### Scenario: The diagnosis is consumed by a machine
- **WHEN** diagnosis is invoked for automated use
- **THEN** it emits structured output and an exit status that distinguishes healthy from not

#### Scenario: A next action names a command that does not exist
- **WHEN** any next action the diagnosis prints names a command this tool does not accept
- **THEN** that violates this requirement

### Requirement: Currency is judged by digests, not by a recorded version
**Intent IDs:** OUT-6, OUT-9, SIG-4

Whether a repository's overlay is current SHALL be decided by comparing its files
against the binary's canonical copies, and MUST NOT depend on a version recorded
inside the repository. The participation marker is per clone and is not shared
through the repository, so a recorded version would answer the same question
differently in two clones of one repository — and the question is about
committed content.

The `gr` version SHALL be reported as information about the binary. No verdict
about the repository may be derived from it.

#### Scenario: The version string alone changes
- **WHEN** the reported `gr` version differs while every overlay file still matches the binary's canon
- **THEN** the diagnosis reports the overlay as current

#### Scenario: The files differ while the version agrees
- **WHEN** an overlay file differs from the binary's canon while any recorded version would suggest currency
- **THEN** the diagnosis reports drift, because the files decide

### Requirement: The checking toolchain is reported as a fact, not a fault
**Intent IDs:** OUT-6, SIG-12

The diagnosis SHALL state whether the runtime the stock OpenSpec CLI requires is
present, and SHALL name what its absence costs — validation and archival of
changes — so the consequence is legible rather than guessed.

That absence MUST NOT be reported as a failure of the harness, MUST NOT
contribute to an unhealthy verdict, and MUST NOT produce a next action. The
harness stands without it; only the checks do not run.

#### Scenario: The runtime is absent
- **WHEN** diagnosis runs where the runtime the stock CLI needs is not on the executable lookup path
- **THEN** the report states the absence, names what cannot be run without it, produces no next action, and does not make the harness verdict unhealthy

#### Scenario: The runtime is present
- **WHEN** diagnosis runs where that runtime is available
- **THEN** the report states its presence as a fact

### Requirement: Observability absence is reported once, without nagging or leaking
**Intent IDs:** OUT-10, SIG-10

The diagnosis SHALL report observability in one line. Absence SHALL be stated as
optional, with no next action and no effect on the overall verdict; Goalrail
works fully with no observability backend, and the local state root remains the
source of truth.

No output SHALL contain key material. Where an endpoint is configured, the report
SHALL name at most the endpoint.

#### Scenario: No observability is configured
- **WHEN** diagnosis runs with no observability endpoint configured
- **THEN** it reports the harness as working, states observability as not configured and optional, and prints no next action for it

#### Scenario: Observability is configured
- **WHEN** an endpoint and keys are present in the environment or the state root
- **THEN** the report names the endpoint at most and no key material appears in any output

### Requirement: The diagnosis reports a newer release as a fact, never as a problem
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2

The diagnosis SHALL report whether a newer Goalrail release exists. It SHALL do
so in the voice it uses for the toolchain and for observability: a stated fact,
with no next action, contributing nothing to whether the harness is reported as
working and nothing to the exit code.

The reason is not stylistic. The tool has no command that replaces its own
binary, so a next action here would name a command the tool does not accept,
which a promoted requirement already forbids. A repository whose harness is
intact is working whether or not its binary is a release behind, and a caller
that reads the exit code learns about the repository, not about the binary.

Where the line rests on an answer from a network source, it SHALL name that
source, so the report discloses its counterparty where the user is rather than
only in documentation. Where there is no answer and no counterparty, it names
none.

The check MUST NOT compare anything when the running binary's own version is not
a release version — a build from a checkout, from a modified tree, or one
carrying no version at all. In those cases the diagnosis SHALL say that the
question does not apply rather than guess. No wording SHALL assert that the
installation is up to date: the source can lag a published release, so the
report states what was found and when it was looked for.

Where the check produced no answer — because it was declined, or because it
failed — the diagnosis SHALL say so in one line, and a declined check SHALL be
distinguishable from a failed one. It MUST NOT surface an error, a stack trace,
or any text the source produced.

An answer read from a cache that is not yet due for refresh is an answer, not an
absence: it is reported like any other, stating when it was obtained.

#### Scenario: A newer release exists
- **WHEN** the running binary is a release version older than the newest released one
- **THEN** the diagnosis states that a newer release exists, names the service it asked, prescribes no command, and leaves the working verdict and the exit code exactly as they would have been

#### Scenario: The binary is current
- **WHEN** the running binary's version matches the newest released one
- **THEN** the diagnosis states what it found and when, names the service it asked, and does not assert that the installation is up to date

#### Scenario: The running binary is not a release build
- **WHEN** the version the binary reports is a pseudo-version, carries a modification marker, is the toolchain's development marker, or is unknown
- **THEN** no comparison is made and the diagnosis says the question does not apply

#### Scenario: The check produced no answer
- **WHEN** the check was declined or failed
- **THEN** the diagnosis says so in one line, distinguishing a declined check from a failed one, with no error, no stack trace, and no text from the source

#### Scenario: The answer comes from a cache that is not yet due
- **WHEN** a diagnosis runs within the caching interval and a cached answer is present
- **THEN** no request is made, the answer is reported compared against the running binary's version, and the report states when it was obtained rather than that the check did not run

#### Scenario: A newer release would change the verdict
- **WHEN** the existence of a newer release would make the harness report as not working, add a problem, or change the exit code
- **THEN** that violates this requirement

#### Scenario: A next action would be prescribed
- **WHEN** the line about a newer release prescribes a command
- **THEN** that violates this requirement, because the tool has no command that replaces its own binary

### Requirement: The diagnosis reports whether the branch's review still describes it
**Intent IDs:** OUT-6, SIG-4

Where a receipt exists for the current branch, the diagnosis SHALL report in one
line whether that review still describes the branch: current where the branch's
present diff digest matches the receipt's, stale where it does not. Where no
receipt exists for the branch, it SHALL report that too. Where a receipt exists
but cannot be read or its digest cannot be recomputed — truncated, malformed,
oversized, an unknown schema, a pruned base commit — the diagnosis SHALL report
it as unreadable and name the cause, because presenting "never reviewed" for a
branch whose evidence is corrupt is the one wrong answer available here. A review that was run
and then outrun by three more commits is worse than no review, because it reads
as done.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. A branch with
no review is an ordinary state, and the review is evidence rather than a gate.

The state SHALL follow from recomputing the digest alone. Committing further
work makes a review stale by itself, and running a fresh review makes it current
by itself; there SHALL be no flag, no acknowledgement command, and no stored
dismissal. A condition that must be cleared by hand is one nobody reads.

The line MUST NOT reproduce the reviewer's report. The report is stored
verbatim in the receipt for whoever wants it; reprinting it on every diagnosis
would be noise.

#### Scenario: The review still describes the branch
- **WHEN** diagnosis runs on a branch whose receipt digest matches its present diff
- **THEN** it reports the review as current, naming the reviewer and the time, with no effect on the verdict or the exit status

#### Scenario: New work has outrun the review
- **WHEN** a commit is added after a review and diagnosis runs
- **THEN** it reports the review as stale, and no flag was needed to change that

#### Scenario: A fresh review restores currency
- **WHEN** the review is run again after that commit
- **THEN** the diagnosis reports it as current again, with no dismissal or acknowledgement anywhere

#### Scenario: The branch has never been reviewed
- **WHEN** diagnosis runs on a branch with no receipt
- **THEN** it reports the absence as a condition, and the overall verdict and exit status are what they would be without the line

#### Scenario: The report is not reprinted
- **WHEN** the line is reported for a branch whose receipt holds a long report
- **THEN** no part of the report text appears in the diagnosis output

#### Scenario: The receipt is unusable
- **WHEN** diagnosis runs where a receipt exists but cannot be read or recomputed
- **THEN** it reports the review as unreadable with the cause, and the overall verdict and exit status are unchanged

### Requirement: Review evidence preserves the public non-interactive command flow
**Intent IDs:** OUT-6, SIG-4

Initialization, review, and diagnosis SHALL remain usable through their public
command surfaces with no terminal attached. Each successful command SHALL emit
one parseable JSON result and MUST NOT prompt. Adding or changing review
evidence MUST NOT alter initialization semantics or the diagnosis verdict and
exit status that would otherwise follow from repository health.

#### Scenario: The public flow runs without a terminal
- **WHEN** initialization, review, and diagnosis run in sequence with valid inputs and no terminal attached
- **THEN** every command completes without prompting and each successful result parses as JSON

#### Scenario: Review state does not become a diagnosis gate
- **WHEN** diagnosis observes a current, stale, absent, or unreadable review receipt without any other repository-health change
- **THEN** its JSON names that review state while its verdict and exit status remain what repository health independently requires

### Requirement: The diagnosis reports rules that predate an adoption, and stops on its own
**Intent IDs:** OUT-5, SIG-4, SIG-5

Where the marker records an adoption of a configuration that carried at least one
rule, and that rules block still matches the digest recorded at the adoption, the
diagnosis SHALL report it in one line naming the replaced schema and when the
adoption happened. Initialization discloses the same fact once, to whoever ran
it; that is routinely an agent, and the person whose rules they are may never see
it. The diagnosis is where a standing condition belongs.

Where the configuration carried no rules at the adoption, the diagnosis SHALL
report nothing. There is nothing that predates the schema, and the advisory's
stop condition is an edit to rules that do not exist — a standing condition the
user could never clear by reviewing anything.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. Rules written
against a replaced schema are frequently still correct, and Goalrail cannot tell
which ones are not.

The report SHALL stop carrying the line once the rules block no longer matches
the recorded digest, with no flag, no acknowledgement command and no stored
dismissal. Editing the rules is the act of reviewing them, so the only honest
stop condition is the one the user's own edit produces. A condition that must be
dismissed by hand becomes a condition nobody reads.

Where the marker carries no adoption record, the diagnosis SHALL report nothing
about rules at all.

The line MUST NOT reproduce the rules themselves. Initialization discloses them
in full at the moment they can still be compared against what was replaced; a
diagnosis that reprinted them at every run would be noise.

#### Scenario: Rules are unchanged since an adoption
- **WHEN** diagnosis runs where the marker records an adoption and the rules block still matches the recorded digest
- **THEN** it reports one line naming the replaced schema and the adoption time, the overall verdict is unaffected, and the exit status is what it would have been without the line

#### Scenario: The rules have been edited since the adoption
- **WHEN** the rules block is edited after an adoption and diagnosis runs again
- **THEN** the line is absent, and no flag, command or stored acknowledgement was needed to remove it

#### Scenario: An unrelated part of the configuration is edited
- **WHEN** the configuration is edited outside the rules block after an adoption and diagnosis runs
- **THEN** the line is still reported, because the rules themselves are unchanged

#### Scenario: The repository never replaced a schema
- **WHEN** diagnosis runs where the marker carries no adoption record
- **THEN** it reports nothing about rules

#### Scenario: The adopted configuration carried no rules
- **WHEN** diagnosis runs where the marker records an adoption of a configuration that carried no rules
- **THEN** it reports nothing about rules, however many times it is run

#### Scenario: The line does not reprint the rules
- **WHEN** the line is reported for a configuration carrying several rules
- **THEN** no rule text appears in the diagnosis output
