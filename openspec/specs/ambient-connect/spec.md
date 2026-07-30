# ambient-connect Specification

## Purpose

Define how Goalrail attaches to sessions the user starts in their own scaffold,
rather than launching a provider itself: one consented reversible connection,
action confined to explicitly initialized repositories, a launch-time
announcement of the escalation channel, retention of a session's question bound
to the intent it belongs to, and a fail-quiet posture that never harms an
ordinary session.
## Requirements
### Requirement: Connection is explicit, consented, and reversible
**Intent IDs:** OUT-1, OUT-4, SIG-1, SIG-4

Goalrail SHALL attach to a scaffold through one explicit connection command
that registers persistent session hooks in the scaffold's user configuration.
Registration MUST NOT happen without the user's explicit consent, and a
matching disconnection command SHALL remove everything the connection added,
leaving no residue. Both commands SHALL be safe to repeat.

Registration alone does not make the attachment act. The scaffold requires the
user to review and trust the exact hook definition before it runs, and records
that trust against the definition's current form, so a changed command needs
review again. Connection SHALL therefore disclose this: it states that the
trust step is required, names the surface where the user performs it, and says
plainly that the attachment does nothing until then. Silence here is the worst
available outcome — the user connects, works, observes nothing, and reasonably
concludes the product is broken.

The disclosure SHALL match what was actually established for that scaffold.
Where the gate was observed, it is stated. Where it was not, the disclosure
SHALL say the behaviour is unverified instead of asserting a requirement:
inventing an obstacle sends the user hunting for a screen that may not exist.

Connection SHALL NOT describe an already-trusted attachment as inert. Repeating
the command on a working attachment reports it as working.

Attachment is registered at user scope, which means the hook is invoked for
every session the user starts anywhere, and confining action to initialized
repositories is enforced by Goalrail's own first act rather than by absence.
This is a recorded limitation, not a preference: registering inside the
repository would make the boundary structural, and that arrangement is
externally blocked — project-local hooks are silently skipped with no trust
prompt even in a trusted project, and the session-opening event has already
passed by the time a user could grant trust. Should that change, this
limitation is the first thing to revisit.

After connection and trust, the user runs no Goalrail command per task:
sessions are started, driven, and ended entirely by the user in their own
scaffold.

#### Scenario: User connects once
- **WHEN** the user runs the connection command and consents
- **THEN** the persistent session hooks are registered in the scaffold's user configuration and no further Goalrail command is needed for ordinary work

#### Scenario: Connection is repeated on a working attachment
- **WHEN** the connection command runs again after the user has completed the trust step
- **THEN** it reports the attachment as active rather than telling the user to grant trust again

#### Scenario: The scaffold's trust behaviour is unverified
- **WHEN** connection registers hooks in a scaffold whose trust gate has not been observed
- **THEN** the disclosure says the behaviour is unverified rather than asserting a required approval step

#### Scenario: Connection discloses the trust step
- **WHEN** connection registers the hooks
- **THEN** its output states that the scaffold requires the user to review and trust them, names the surface for doing so, and says the attachment does not act until then

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

### Requirement: Attachment health is reportable and trust is never forged
**Intent IDs:** OUT-2, OUT-3, SIG-2, SIG-3

A registered hook does not run until the scaffold's own trust step has been
completed by the user, so registration alone does not make an attachment work.
Goalrail SHALL therefore report attachment health on demand, distinguishing at
least: the scaffold is not connected; this repository is not initialized; the
hooks are registered but not yet trusted; the attachment is working. Each state
that is not working SHALL name the next action.

Goalrail MUST NOT write, compute, or reproduce the scaffold's hook-trust
records, and MUST NOT automate, pre-approve, or simulate the user's approval by
any means. Trust is a standing consent to run a command in every one of the
user's sessions; it belongs to the user, granted through the scaffold's own
surface. That this is technically possible, and is practised by other
integrations, does not make it permissible here: it would depend on private
implementation details and would convert an explicit consent into an assumed
one.

Reporting trust state is observation and is permitted. Reading is not writing.

A report SHALL NOT claim more than it verified. Where the scaffold records
trust against a form Goalrail may not reproduce, the report SHALL say that a
record exists without claiming it still matches the current definition, and
SHALL name that gap. Where a scaffold's trust behaviour has not been observed
at all, the report SHALL say so rather than assert either answer.

Health SHALL be reported against everything a working attachment requires, not
a proxy for it: every registered event, not one; the registered executable
still present and runnable, since a registration pointing at a moved binary
satisfies every configuration check and still cannot run; and a scaffold
configuration that cannot be read or parsed reported as its own failure with
its own action, because recommending connection would repeat the same failure.

Where a health query does not name a scaffold, Goalrail SHALL report every
supported scaffold rather than assume one. A confident diagnosis about a
scaffold the user never chose is worse than no answer.

#### Scenario: Attachment is not yet trusted
- **WHEN** the user asks for attachment health after connecting but before trusting the hooks
- **THEN** the report says the hooks are registered but not trusted, and names the next action

#### Scenario: Repository is not initialized
- **WHEN** the user asks for attachment health in a directory Goalrail was never initialized in
- **THEN** the report says so and names the next action, without treating the directory as participating

#### Scenario: Only part of the registration survives
- **WHEN** one of the registered events is removed while the rest of the registration remains
- **THEN** the attachment is not reported as connected or working, because the missing event silently removes either the announcement or the retention of questions

#### Scenario: The registered executable is gone
- **WHEN** the binary a registration points at has been moved or removed
- **THEN** the attachment is not reported as working, and the report names that cause

#### Scenario: Scaffold configuration cannot be read
- **WHEN** the scaffold configuration is unreadable or malformed
- **THEN** the report names that as the failure with its own next action, rather than reporting an ordinary disconnected state

#### Scenario: No scaffold is named
- **WHEN** a health query names no scaffold
- **THEN** every supported scaffold is reported, rather than one being assumed

#### Scenario: Trust freshness cannot be established
- **WHEN** a trust record exists but Goalrail cannot confirm it still matches the current hook definition
- **THEN** the report states that a record exists, names the unverified part, and does not claim the definition is trusted as it now stands

#### Scenario: Attachment is working
- **WHEN** the scaffold is connected, the repository initialized, and the hooks trusted
- **THEN** the report says the attachment is working and asks nothing further of the user

#### Scenario: Trust would be written by Goalrail
- **WHEN** any Goalrail path would write, compute, or reproduce a scaffold trust record, or otherwise approve hooks on the user's behalf
- **THEN** that is prohibited, regardless of technical feasibility

