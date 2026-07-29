## ADDED Requirements

### Requirement: Connection is explicit, consented, and reversible
**Intent IDs:** OUT-1, SIG-1

Goalrail SHALL attach to a scaffold through one explicit connection command
that registers persistent session hooks in the scaffold's user configuration.
Registration MUST NOT happen without the user's explicit consent, and a
matching disconnection command SHALL remove everything the connection added,
leaving no residue. Both commands SHALL be safe to repeat.

After connection, the user runs no Goalrail command per task: sessions are
started, driven, and ended entirely by the user in their own scaffold.

#### Scenario: User connects once
- **WHEN** the user runs the connection command and consents
- **THEN** the persistent session hooks are registered in the scaffold's user configuration and no further Goalrail command is needed for ordinary work

#### Scenario: User disconnects
- **WHEN** the user runs the disconnection command
- **THEN** every registration the connection added is removed and no Goalrail residue remains in the scaffold configuration

#### Scenario: Commands are repeated
- **WHEN** connection or disconnection runs a second time
- **THEN** the outcome is unchanged and nothing is duplicated or broken

#### Scenario: Configuration would change without consent
- **WHEN** any Goalrail path would modify scaffold user configuration outside the explicit consented connection command
- **THEN** that modification is refused

### Requirement: Attachment acts only in initialized directories
**Intent IDs:** OUT-2, SIG-2

The first act of every hook invocation SHALL be to determine whether the
session's directory is a Goalrail-initialized repository. Everywhere else the
hook MUST exit immediately, reading nothing beyond the initialization check,
writing nothing, and retaining no observation of the session. In particular,
the scaffold's event payload MUST NOT be read before the check passes: a
payload can carry prompts, transcript paths, and authorization fields, and
reading one in an unconnected directory observes the session regardless of
what happens afterwards.

Observing sessions outside connected directories is prohibited, not merely
avoided: a persistent hook fires for every session of the user, and acting
outside initialized directories would monitor unrelated work.

#### Scenario: Session in an unconnected directory
- **WHEN** a session starts or stops in a directory that is not Goalrail-initialized
- **THEN** the hook exits immediately with zero writes and zero retained observations

#### Scenario: Session in a connected directory
- **WHEN** a session starts in a Goalrail-initialized directory
- **THEN** the ambient behaviour of this capability applies

### Requirement: Session start announces the channel and clears stale questions
**Intent IDs:** OUT-3, SIG-3

At session start in a connected directory, the hook SHALL tell the agent the
escalation channel exists, using one fixed ambient announcement under the same
discipline as the launch announcement: it names the reserved path and its
conditions, names no provider, describes no work item, suggests no conflict,
and accompanies only the event that opens a session — never resumption,
clearing, or compaction.

When the reserved path already holds a question left by an earlier session,
the hook SHALL archive it into the state root outside the repository before
the session proceeds, so a stale question is never attributed to the new
session. The archival read SHALL apply the same bounded regular-file hygiene
as retention: a stale artifact that is oversized, non-regular, or a link
outside the repository is cleared and noted without being followed, blocking
on, or copied.

#### Scenario: Agent learns the channel exists
- **WHEN** a session opens in a connected directory
- **THEN** the agent's context carries the fixed ambient announcement and nothing about the task or the repository's content

#### Scenario: Recurring session events stay silent
- **WHEN** the scaffold re-signals session start for resumption, clearing, or compaction
- **THEN** no further announcement is injected

#### Scenario: Stale question is archived
- **WHEN** the reserved path holds a question from an earlier session at session start
- **THEN** the file is archived into the state root, the reserved path is clear for the new session, and the archived question is never attributed to it

### Requirement: Session stop retains the question and binds it to intent
**Intent IDs:** OUT-4, OUT-5, SIG-3, SIG-4

At session stop in a connected directory, when the reserved path holds a
question, the hook SHALL retain its exact bytes append-only in the state root
outside the repository, applying the existing escalation hygiene. The retained
record SHALL carry the question digest and the session reference, and SHALL
have an identity of its own — one record per occurrence, so two sessions
asking a byte-identical question keep separate, individually answerable
records. A question that cannot be read still leaves a persisted record with
an explicit reason: the session tried to escalate, and the attempt must leave
evidence.

The record SHALL be bound to the directory's single active confirmed intent —
its ID, version, and digest. When no single active confirmed intent is
identifiable, the record SHALL be retained unbound with an explicit reason
naming why, and MUST NOT be bound by guessing.

A background session produces a question record, not a run outcome: no run ID,
launch claim, terminal receipt, or `blocked` status is minted.

The answer to a recorded question is a new intent version carrying the
existing resolves-and-disposition record, citing the question record's own
identifier, and the next session works from that version. No second answering
mechanism, dialogue, acknowledgement, or resume exists.

#### Scenario: Two sessions ask the identical question
- **WHEN** two sessions produce byte-identical questions
- **THEN** the payload is retained once but each occurrence keeps its own record and identity, and neither collides with the other

#### Scenario: The question cannot be read
- **WHEN** the reserved path holds something oversized, non-regular, or otherwise unreadable at session stop
- **THEN** a record with an explicit reason is still persisted, because the escalation attempt itself is evidence

#### Scenario: Question is retained and bound
- **WHEN** a session ends with a question at the reserved path and the directory has one active confirmed intent
- **THEN** the exact bytes are retained append-only outside the repository and the record names that intent's ID, version, and digest

#### Scenario: No single active intent exists
- **WHEN** a session ends with a question and the directory has no active confirmed intent, several, or only unconfirmed ones
- **THEN** the question is retained unbound with an explicit reason and no binding is guessed

#### Scenario: The worktree file is deleted afterwards
- **WHEN** the question file is removed after the session ends
- **THEN** the retained bytes and the record are unchanged

#### Scenario: The loop closes through the intent gate
- **WHEN** the owner answers a recorded question
- **THEN** the answer is a new intent version carrying the resolves-and-disposition record, and recorded identifiers alone lead from the question to that version

### Requirement: Background failure is invisible to the session
**Intent IDs:** OUT-6, SIG-5

A Goalrail malfunction in background mode MUST NOT block, break, or delay the
user's session: the session proceeds exactly as if Goalrail were absent. The
helper SHALL treat every internal error as a silent no-op toward the scaffold.

This posture is deliberately opposite to the wrapper lifecycle, which remains
fail-closed by its promoted requirements: a wrapper launch that cannot announce
the channel is still refused. The two postures serve different guarantees — the
wrapper certifies a run, the background layer must first do no harm.

#### Scenario: Helper is broken
- **WHEN** the ambient helper fails for any internal reason
- **THEN** the user's session starts, runs, and ends normally, as if Goalrail were not installed

#### Scenario: Wrapper posture is unchanged
- **WHEN** the wrapper lifecycle launches a run
- **THEN** its promoted fail-closed behaviour applies exactly as before
