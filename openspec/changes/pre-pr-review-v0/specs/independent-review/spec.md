## ADDED Requirements

### Requirement: One command reviews the current branch inside the author's loop
**Intent IDs:** OUT-1, OUT-3, SIG-1

Goalrail SHALL provide a command that reviews the current branch's changes
against a base ref by running an installed reviewer, and SHALL do so without
asking anyone anything. An author cannot review their own artifact, and a review
that begins when the work is already public arrives after the cost it exists to
prevent; the review is therefore a step of the loop that produced the work.

The command SHALL determine the author's provider from the invoking
environment, SHALL select as reviewer an installed provider that is not the
author's, and SHALL run that reviewer over the branch's diff with the assembled
instructions. It MUST NOT prompt, and it MUST NOT require a flag as a consent
step; the caller invoking it is the decision.

The reviewer SHALL be invoked through the vendor's own documented
non-interactive interface, so that the user's existing subscription is used as
its vendor intends. Goalrail MUST NOT vendor, pin, wrap, or reimplement a
reviewer, and MUST NOT introduce a third-party review tool. The resulting
exposure to vendor interface drift is accepted: a refusal from the vendor CLI
surfaces to the caller unchanged rather than being absorbed.

Where no reviewer other than the author's provider is installed, the command
SHALL refuse and name what is missing. Reviewing with the author's own provider
in the author's own context would reproduce the author's blind spots, which is
the one thing this exists to avoid.

#### Scenario: A review runs with no flags and no questions
- **WHEN** the command is invoked on a branch from a session whose provider is detectable and another provider is installed
- **THEN** it selects the other provider, runs the review, writes a receipt, and asks nothing at any point

#### Scenario: Only the author's own provider is installed
- **WHEN** the command is invoked and no installed provider differs from the author's
- **THEN** it refuses, names the missing side, and runs no review

#### Scenario: The vendor CLI refuses
- **WHEN** the reviewer's own command exits non-zero or reports an unusable invocation
- **THEN** the failure is reported to the caller as the reviewer's own, no receipt is written, and Goalrail does not retry with altered arguments

### Requirement: Authorship is inferred, and undecidable authorship is refused
**Intent IDs:** OUT-1, SIG-1

The command SHALL infer the author's provider from the environment of the
process that invoked it: a Claude Code session is identified by `CLAUDECODE`,
and a Codex session by its own `CODEX_*` session marker. The invoking process
knows what it is, so asking the caller to state it on the ordinary path would be
a question with a knowable answer.

An explicit override SHALL exist and SHALL take precedence over detection.

Where the environment carries the primary markers of more than one provider, or
none, the command SHALL refuse, state what it saw, and name the override. One
tool's companion variables can be present inside another tool's session — this
is an ordinary configuration, not a corruption — so a marker that is merely
present is not evidence of authorship. Choosing silently under that ambiguity
would pick the author as its own reviewer, which fails invisibly rather than
loudly.

#### Scenario: A Claude Code session is detected
- **WHEN** the command runs in an environment carrying `CLAUDECODE`, and no other provider's primary session marker
- **THEN** the author is Claude Code and the reviewer is another installed provider

#### Scenario: Two providers' primary markers are present
- **WHEN** the environment carries the primary session markers of two providers
- **THEN** the command refuses, states both, names the override, and runs no review

#### Scenario: No provider is detectable
- **WHEN** the environment carries no provider's primary session marker and no override is given
- **THEN** the command refuses and names the override rather than defaulting to one

#### Scenario: The override wins
- **WHEN** an authorship override is given in an environment that would detect a different provider
- **THEN** the override decides the author and the review proceeds

### Requirement: A budget gate is the only other refusal
**Intent IDs:** OUT-2, SIG-2

Where configuration names a gate command, the command SHALL run it before
invoking any reviewer and SHALL refuse the review on a non-zero exit, naming the
gate as the reason. Every review spends a metered subscription, and this is the
first thing Goalrail does that costs money without a separate human act.

The gate SHALL be a command named in configuration and MUST NOT be a path,
provider, or budget service built into Goalrail. A personal budget tool belongs
in one user's configuration, never in a released binary.

Where no gate is configured, the review SHALL proceed. Absence of a gate is a
choice the user already made by not making one, and turning it into a refusal
would make the ordinary path ask for consent.

#### Scenario: A configured gate refuses
- **WHEN** the gate command exits non-zero
- **THEN** no reviewer is invoked, no receipt is written, and the refusal names the gate

#### Scenario: A configured gate permits
- **WHEN** the gate command exits zero
- **THEN** the review proceeds normally

#### Scenario: No gate is configured
- **WHEN** configuration names no gate command
- **THEN** the review proceeds without one, and nothing is reported as missing

### Requirement: The receipt is bound to what was reviewed
**Intent IDs:** OUT-5, SIG-3

A completed review SHALL leave a receipt carrying the base commit, the head
commit, a digest of the reviewed diff, the reviewer's identity, the time, and
the reviewer's report as verbatim bytes together with a digest of those bytes.

No field of the receipt SHALL be derived by reading, parsing, scoring, or
summarizing the report. Goalrail stores what the reviewer said and what was
reviewed; deciding which findings are real belongs to the loop's author.
Building a prose reader into the feature that exists to catch confident misreads
of foreign formats would be the defect class it was built to catch.

The diff digest SHALL be computed from a canonical diff of the reviewed range,
so that the same range yields the same digest on any machine and a different
range never yields the same one.

The receipt SHALL be written to Goalrail's per-clone state and MUST NOT be
written into the repository. A receipt is evidence about one clone's branch at
one moment, not repository content someone else should receive.

#### Scenario: A receipt describes its own review
- **WHEN** a review completes
- **THEN** recomputing the diff digest from the receipt's own base and head reproduces the recorded digest, and the stored report is byte-identical to what the reviewer emitted

#### Scenario: The receipt stays out of the repository
- **WHEN** a review completes
- **THEN** the repository's tracked and untracked files are unchanged apart from an instructions file materialized on first use

### Requirement: Reviewer instructions are repository content
**Intent IDs:** OUT-7

The instructions handed to the reviewer SHALL be read from a committed file in
the repository, and Goalrail SHALL materialize a default there when none
exists. What a reviewer is told to look for is the accumulated result of what
has previously been missed, so it has to be shared, versioned, and reviewable
like any other repository content rather than embedded in a binary.

The file MUST NOT be written to a path Goalrail's own ignore rules exclude from
version control, because instructions that cannot be committed cannot be shared.

Materializing the default MUST NOT overwrite an existing file.

#### Scenario: The default is materialized once
- **WHEN** the command runs in a repository with no instructions file
- **THEN** the default is written, is committable, and the review uses it

#### Scenario: Edited instructions are used
- **WHEN** the instructions file has been edited and the command runs again
- **THEN** the reviewer receives the edited instructions and the file is unmodified afterwards
