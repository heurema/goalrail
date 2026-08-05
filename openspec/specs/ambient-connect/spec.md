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
**Intent IDs:** OUT-1, OUT-2, OUT-4, OUT-11, SIG-1, SIG-2, SIG-4, SIG-13

Goalrail SHALL attach to a scaffold through one explicit consented command that
registers persistent session hooks, and SHALL remove everything it added through
a matching command. Registration MUST NOT happen without the user's explicit
consent. Both commands SHALL be safe to repeat.

Which command that is, and which scope it writes, is a property of the scaffold
rather than of attachment in general. Where a scaffold's settings layer permits
registration inside the repository, the consented command is initialization, and
the registration is written there. Where it does not, the consented command is
connection, and the registration is written at user scope. For a scaffold that
registers per repository, a separate connection step has no reason to exist:
connection SHALL write nothing for it and SHALL name initialization instead.

Where a scaffold distinguishes which session-start occurrence a hook applies
to, registration SHALL name only the occurrences that open a session, and
SHALL NOT register a form that fires on every occurrence. The single-delivery
rule is a property of the announcement, so it MUST hold on every supported
scaffold — including one whose transport does not inspect the occurrence and
therefore cannot enforce it after the fact. Removal SHALL cover every
occurrence the connection registered, and MUST leave entries it did not add
untouched.

Whether a connection is present SHALL be judged by what its registration
names, not only by the registration being recognisable as Goalrail's. A
registration whose handler names an executable other than the one connection
was invoked with is not the registration this connection would write, and may
name a binary that no longer exists, so it SHALL be treated as needing repair
rather than reported as already present. Where that executable is still
runnable, the repair happens anyway and costs the user the review step the
disclosure names. That cost is accepted deliberately: the alternative is a
remedy a health report prescribes which sometimes does nothing, and a
connection command that cannot make the attachment use the binary it was
invoked from.

Repair SHALL replace rather than accompany. The connection SHALL NOT leave two
registrations for the same event, because a second registration duplicates
every hook invocation and leaves a removal that finds only one of them. A
registration that already names the current executable SHALL be left exactly as
it is, and repair MUST NOT alter a handler the connection did not add, change a
foreign entry's occurrence, or widen a repaired session-start entry beyond the
occurrence that opens a session. This holds wherever the registration lives, so
the command that registers per repository repairs on re-run under the same rule.

The events a registration names SHALL carry the meaning the retention requirement
already has. Retention belongs to a session ending, so where a scaffold
distinguishes an event that fires once per turn from one that fires when the
session ends, the registration MUST NOT name the per-turn event: a question left
at the reserved path would be retained again on every turn, minting a record per
turn from one session's single question, and per-occurrence identity — which is
correct for two sessions asking the same thing — would then multiply one
escalation instead of separating two. A registration naming such an
event SHALL be treated as needing repair, through the consented command that
owns its scope: at repository scope, re-running initialization replaces it in
place with the session-scoped form; at user scope, where only the earlier
arrangement wrote it, it is reported and removed through the consented
disconnection command rather than rewritten by initialization. Removal SHALL
cover whichever event was actually registered rather than only the event the
current arrangement writes.

Where a health report names re-running connection as the remedy for a defect,
connection SHALL be able to repair that defect. A report that prescribes a
command which cannot perform the repair is worse than no report: the user
follows correct advice, observes no change, and has no remaining move.

Consent MUST NOT be given on another person's behalf. A repository-scope
registration SHALL be written only to a settings file the repository cannot
supply to someone else, and only where that path is unshareable — ignored by the
version control system, made so by the same act where that is possible. Trust is
a standing consent to run a command in one's own sessions; a registration a commit
could carry would install that command in every teammate's sessions on the
strength of one user's decision, which is not the user's to give. Where the path
cannot be made unshareable, the registration SHALL be refused with the reason
named rather than written in the hope that nobody commits it.

The scope a scaffold uses SHALL be recorded with the evidence that produced it,
and a true observation about one provider MUST NOT be recorded as a property of
attachment in general. On the first supported scaffold, registering inside the
repository is externally blocked, because project-local hooks are silently
skipped with no trust prompt even in a trusted project and the session-opening
event has already passed by the time a user could grant trust; there, user scope
is forced, the hook is invoked for every session the user starts anywhere, and
confining action to initialized repositories is enforced by Goalrail's own first
act rather than by absence. The second supported scaffold does not share that
constraint — its project-level settings are an ordinary layer that merges with
user settings — so there the registration lives in the repository, is never
invoked in unrelated sessions at all, and the boundary is a property of where it
is installed rather than of what Goalrail does first. A registration for that
scaffold that remains at user scope from the earlier arrangement SHALL be
reported rather than silently modified, and its removal SHALL pass through the
consented command that owns user-level configuration.

Registration alone does not make the attachment act where a scaffold gates it.
Where the scaffold requires the user to review and trust the exact hook
definition before it runs, and records that trust against the definition's
current form, a changed command needs review again. Connection SHALL therefore
disclose what is actually established for that scaffold, distinguishing four
states: a gate observed in a live session is stated as required; a gate observed
absent in a live session — a registered hook ran with no approval step — is
stated as observed, and the disclosure sends the user nowhere to approve
anything; a gate that the provider's own documentation records as absent without
an observation is stated as documented, with the documented scope of any
adjacent consent surface named so the user is not sent to a screen that gates
something else; and a scaffold whose behaviour is neither observed nor
documented is stated as unverified. Silence is the worst available outcome — the
user connects, works, observes nothing, and reasonably concludes the product is
broken — and inventing an obstacle is its own kind of misinformation, because it
sends the user hunting for a screen that may not exist.

A repair changes the hook definition, so it SHALL carry that same disclosure and
SHALL NOT report the repaired attachment as active on the strength of a trust
record that was made against the command just replaced. A repair is the only
moment at which Goalrail knows a trust record no longer matches, because it
made the change itself; reading the record cannot establish this, since
reproducing the form it is recorded against is prohibited.

A working attachment SHALL NOT be described as inert. Repeating the consented
command that registers a scaffold — connection where that is the one that writes,
initialization where the registration lives in the repository — reports the
attachment as working. The rule follows the command that owns the registration,
not the word "connect": for a repository-scope scaffold, connection writes
nothing and names initialization, so requiring connection itself to report a
working attachment would demand an outcome from a command that deliberately
inspects nothing.

The second supported scaffold has now been exercised in a live session, and the
record SHALL distinguish what that run established from what it did not. Observed:
the announcement reaches the agent, and a question left at the reserved path is
retained outside the repository with its own identity and an explicit unbound
reason. The agent named the reserved path and the payload schema although the task
mentioned neither, which is how delivery was established rather than assumed.

The approval question that run left open has since been answered by observation,
and the record SHALL say so at its own strength: in three interactive sessions
with the hooks registered in the repository's per-user project settings file —
the settings layer a real user's registration lives in, not a per-run supply —
the registered hook ran with no approval step, and the announcement reached the
agent. The disclosure for this scaffold therefore states the absence as
observed. What those sessions did not capture MUST stay outside the claim: what
the scaffold's startup screen displayed was not recorded, and the documented
facts about its trust surfaces remain documentation — the provider records no
approval step for a hook configured in a settings file, describes its hook
browser as read-only, and gates its workspace trust dialog on permission rules
and additional directories rather than on hooks. The registration shape itself
still rests on the provider's published matcher contract together with an
existing working configuration that agrees with it.

An earlier attempt at live verification was abandoned on the belief that this
scaffold's login state is tied to the home directory, which left only routes that
wrote into the owner's working configuration or copied an account file holding
unrelated history. That belief was wrong: credentials live in the operating
system keychain, and the provider's CLI accepts settings for a single run, so
observation needed neither. No requirement here may rest on the withdrawn
blocker.

After connection and trust, the user runs no Goalrail command per task:
sessions are started, driven, and ended entirely by the user in their own
scaffold.

#### Scenario: User connects once
- **WHEN** the user runs the consented command for a scaffold and consents
- **THEN** the persistent session hooks are registered in the scope that scaffold's settings layer allows, and no further Goalrail command is needed for ordinary work

#### Scenario: Registration scope follows the scaffold
- **WHEN** a scaffold's settings layer permits registration inside the repository
- **THEN** initialization is the consented command that registers it there, and the registration is not written at user scope

#### Scenario: Connection is invoked for a scaffold that registers per repository
- **WHEN** the user runs the connection command for a scaffold whose registration belongs in the repository
- **THEN** nothing is written and the outcome names initialization as the command that registers it

#### Scenario: Consent would be given on someone else's behalf
- **WHEN** a repository-scope registration would be written to a settings file the repository could supply to a teammate, and the path cannot be made unshareable
- **THEN** the registration is refused and the reason is named

#### Scenario: A scaffold distinguishes session-start occurrences
- **WHEN** registration writes session-start hooks on a scaffold whose registration names which occurrence applies
- **THEN** it registers only the occurrences that open a session, and registers no form that fires on every occurrence

#### Scenario: An earlier registration fires on every occurrence
- **WHEN** the consented command finds its own session-start registration present but not scoped to the opening occurrence
- **THEN** it replaces that registration with the scoped form rather than reporting the attachment as already present, so a user who connected with an earlier version can repair it

#### Scenario: Removal spans every registered occurrence
- **WHEN** the user disconnects a scaffold whose session-start hooks were registered per occurrence
- **THEN** every entry the registration added is removed, and any entry it did not add for the same event survives

#### Scenario: Removal spans a repository-scope registration
- **WHEN** the user disconnects a scaffold whose hooks were registered inside the repository
- **THEN** every entry that registration added is removed with no residue, and any entry it did not add survives unchanged

#### Scenario: A registration names an event that fires once per turn
- **WHEN** a registration of ours names a stop-like event that fires once per turn on a scaffold that also exposes a session-ending event
- **THEN** it is treated as needing repair through the consented command that owns its scope — replaced in place at repository scope, reported and removed through consented disconnection at user scope — so one session's single question is retained once

#### Scenario: Removal covers the event that was registered
- **WHEN** the user disconnects a scaffold whose handlers were registered against an event the current arrangement no longer writes
- **THEN** those handlers are removed too, so a superseded registration cannot survive a disconnection

#### Scenario: The registration names a moved executable
- **WHEN** the consented command finds its own registration present but naming an executable other than the one it was invoked with
- **THEN** it reports the attachment as needing repair rather than as already present

#### Scenario: A repair replaces the stale registration
- **WHEN** the consented command repairs a registration that names a moved executable
- **THEN** exactly one registration per event remains, naming the current executable, and no second registration or removal residue is left behind

#### Scenario: The registration already names the current executable
- **WHEN** the user reruns the consented command and every registered handler already names the executable it was invoked with
- **THEN** the scaffold configuration is left byte-identical

#### Scenario: A repair meets a handler it did not add
- **WHEN** a repair replaces a stale registration for an event that also carries a handler the registration did not add
- **THEN** the foreign handler survives unchanged, including its occurrence, and only the stale registration is replaced

#### Scenario: A repair meets an event that is already current
- **WHEN** a repair replaces a stale registration for one event while another registered event already names the current executable
- **THEN** the other event's handlers are left unchanged, so a part of the registration that is correct does not lose its review record

#### Scenario: A repaired session-start entry stays scoped
- **WHEN** a repair replaces a session-start registration on a scaffold that distinguishes occurrences
- **THEN** the replacement names only the occurrence that opens a session, whatever shape the stale registration had

#### Scenario: A health remedy names connection
- **WHEN** a health report tells the user to re-run a consented command to repair a defect it detected
- **THEN** that command performs the repair, rather than reporting the attachment as already present and changing nothing

#### Scenario: A limitation is recorded from one scaffold's evidence
- **WHEN** a scope limitation is recorded because one scaffold blocks a stronger arrangement
- **THEN** the record names that scaffold rather than stating the limitation as a property of attachment in general

#### Scenario: A user-scope registration remains from the earlier arrangement
- **WHEN** a registration for a repository-scope scaffold is found at user scope from the earlier arrangement
- **THEN** it is reported with the consented command that removes it, and initialization does not modify it

#### Scenario: A scaffold has not been exercised live
- **WHEN** a supported scaffold's end-to-end behaviour has not been observed in a live session
- **THEN** that is recorded, and no requirement is read as claiming the scaffold is verified

#### Scenario: Part of a scaffold's behaviour has been observed
- **WHEN** a live run establishes some of a scaffold's behaviour and leaves the rest unobserved
- **THEN** the record states which part was observed and which was not, rather than describing the scaffold as verified or as unverified as a whole

#### Scenario: An observation was made under conditions that do not settle a question
- **WHEN** a live run produces no evidence of a behaviour under conditions where that behaviour would not appear anyway
- **THEN** the absence is not recorded as evidence, and the record names the conditions that make it inconclusive

#### Scenario: Connection discloses the trust step
- **WHEN** the consented command registers the hooks on a scaffold whose trust gate was observed
- **THEN** its output states that the scaffold requires the user to review and trust them, names the surface for doing so, and says the attachment does not act until then

#### Scenario: A repair discloses that review applies again
- **WHEN** the consented command replaces a stale registration
- **THEN** its output states that the hook definition changed and needs the scaffold's review step again where one applies, names the surface for it, and does not report the attachment as active on the strength of the existing trust record

#### Scenario: The scaffold's trust behaviour is unverified
- **WHEN** registration or repair happens on a scaffold whose trust gate is neither observed nor documented
- **THEN** the disclosure says the behaviour is unverified rather than asserting a mandatory approval step

#### Scenario: The scaffold was observed to require no approval
- **WHEN** registration or repair happens on a scaffold where a live session showed a registered hook running with no approval step
- **THEN** the disclosure states that as observed, asserts no review step, and sends the user nowhere to approve anything

#### Scenario: The scaffold documents no approval step
- **WHEN** registration or repair happens on a scaffold whose documentation records no approval step for a hook configured in a settings file
- **THEN** the disclosure states that as documented rather than observed, and names what the scaffold's adjacent consent surface actually gates instead of implying it gates hooks

#### Scenario: Connection is repeated on a working attachment
- **WHEN** the user reruns the consented command that registers this scaffold — connection at user scope, initialization at repository scope — on an attachment that is already registered and trusted
- **THEN** the outcome reports it as active rather than describing it as not yet active

#### Scenario: User disconnects
- **WHEN** the user runs the disconnection command
- **THEN** every registration the consented command added is removed and no Goalrail residue remains in the scaffold configuration

#### Scenario: Commands are repeated
- **WHEN** registration or disconnection runs a second time
- **THEN** the outcome is unchanged and nothing is duplicated or broken

#### Scenario: Configuration would change without consent
- **WHEN** any Goalrail path would modify scaffold configuration outside an explicit consented command
- **THEN** that modification is refused

### Requirement: Attachment acts only in initialized directories
**Intent IDs:** OUT-1, OUT-2, OUT-10, SIG-1, SIG-2, SIG-12

The first act of every persistent hook invocation SHALL be to resolve the session's Git worktree root and inspect only the canonical Goalrail project declaration path. In a repository with no declaration claim, the hook MUST exit immediately, reading no event payload, writing nothing, and retaining no observation. A valid declaration SHALL make ambient behavior eligible in every clone and linked worktree, independent of any legacy marker; local attachment health remains a separate fact.

If the reserved declaration path exists but is invalid, the hook MUST fail closed before reading the event payload or retaining session data. It MAY emit one fixed bounded diagnostic directing the supported agent to `gr doctor`, but MUST NOT infer unmanaged status or inspect prompt-, transcript-, credential-, or authorization-bearing payload fields.

Observing sessions outside declared projects remains prohibited. A persistent user-scope hook can fire for every session, so declaration discovery MUST precede payload parsing and MUST NOT follow a symlink or path outside the resolved worktree.

#### Scenario: Session in an unmanaged directory
- **WHEN** a session starts or stops in a repository with no Goalrail declaration claim
- **THEN** the hook exits immediately with zero payload reads, writes, and retained observations

#### Scenario: Session in a managed clone without a legacy marker
- **WHEN** a session starts in a clone carrying a valid committed declaration and no checkout-local marker
- **THEN** the ambient behavior of this capability applies subject to local attachment health

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path exists but cannot be safely validated
- **THEN** the hook reads no event payload, retains nothing, and emits at most the fixed diagnosis direction

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
**Intent IDs:** OUT-2, OUT-4, SIG-4, SIG-5

At session stop in a connected directory, when the reserved path holds a
question, the hook SHALL retain its exact bytes append-only in the state root
outside the repository, applying the existing escalation hygiene. The retained
record SHALL carry the question digest and the session reference, and SHALL have
an identity of its own — one record per occurrence, so two sessions asking a
byte-identical question keep separate, individually answerable records. A
question that cannot be read still leaves a persisted record with an explicit
reason: the session tried to escalate, and the attempt must leave evidence.

The record SHALL be bound to the directory's single active confirmed intent —
its ID, version, and digest — only after active-intent discovery validates the
artifact form required by that change. A pair-backed change SHALL consume its
Context Pack and Intent Snapshot through the selected artifact contract or
pinned legacy mode. An existing lifecycle in which Context is not required and
no pair contract is declared SHALL retain its intent-only validation path. When
no single active confirmed intent is identifiable, the record SHALL be retained
unbound with an explicit reason naming why, and MUST NOT be bound by guessing.

If an active change contains an Intent artifact that explicitly claims
`confirmed` status but its required pair cannot select or satisfy the artifact
contract, discovery SHALL retain the shared structured artifact diagnostic as
binding evidence. That evidence SHALL name the invalid change while keeping
artifact content bounded and redacted. The question record MUST remain unbound,
even if another active intent would otherwise be the single valid confirmed
candidate; discovery MUST NOT hide the invalid confirmed claimant by skipping
it and binding the other one.

An absent, incomplete, or candidate work-in-progress artifact that does not
claim confirmed status SHALL NOT be promoted into invalid-confirmed evidence.
All discovery and diagnostic retention remains background work: it MUST NOT
surface into, block, break, or delay the user's session.

A background session produces a question record, not a run outcome: no run ID,
launch claim, terminal receipt, or `blocked` status is minted.

The answer to a recorded question is a new intent version carrying the existing
resolves-and-disposition record, citing the question record's own identifier,
and the next session works from that version. No second answering mechanism,
dialogue, acknowledgement, or resume exists.

#### Scenario: Two sessions ask the identical question
- **WHEN** two sessions produce byte-identical questions
- **THEN** the payload is retained once but each occurrence keeps its own record and identity, and neither collides with the other

#### Scenario: The question cannot be read
- **WHEN** the reserved path holds something oversized, non-regular, or otherwise unreadable at session stop
- **THEN** a record with an explicit reason is still persisted, because the escalation attempt itself is evidence

#### Scenario: Question is retained and bound
- **WHEN** a session ends with a question and the directory has exactly one active confirmed Context and Intent pair that satisfies its selected artifact contract
- **THEN** the exact bytes are retained append-only outside the repository and the record names that intent's ID, version, and digest

#### Scenario: A confirmed claimant is contract-invalid
- **WHEN** a session ends with a question and an active change's Intent artifact claims confirmed status but its required pair fails artifact contract selection or conformance
- **THEN** the question is retained unbound and its binding evidence carries the shared structured diagnostic naming that change

#### Scenario: A valid intent coexists with an invalid confirmed claimant
- **WHEN** one active pair is a valid confirmed intent and another active change claims confirmed status but fails artifact conformance
- **THEN** the question remains unbound with diagnostic evidence naming the invalid change instead of silently binding the valid intent

#### Scenario: Work in progress is incomplete
- **WHEN** an absent, incomplete, or candidate active artifact does not claim confirmed status
- **THEN** discovery does not represent it as an invalid confirmed claimant

#### Scenario: The active lifecycle does not require Context
- **WHEN** an existing active lifecycle's schema and Intent declare no pair contract and do not require a Context Pack
- **THEN** discovery validates its confirmed Intent through the existing intent-only path instead of inventing a missing pair

#### Scenario: No single active intent exists
- **WHEN** a session ends with a question and the directory has no active confirmed intent, several, or only unconfirmed ones
- **THEN** the question is retained unbound with an explicit reason and no binding is guessed

#### Scenario: The worktree file is deleted afterwards
- **WHEN** the question file is removed after the session ends
- **THEN** the retained bytes and the record are unchanged

#### Scenario: The loop closes through the intent gate
- **WHEN** the owner answers a recorded question
- **THEN** the answer is a new intent version carrying the resolves-and-disposition record, and recorded identifiers alone lead from the question to that version

#### Scenario: Diagnostic retention remains invisible
- **WHEN** ambient discovery retains an invalid-confirmed diagnostic or encounters another internal failure
- **THEN** the session starts, runs, and ends without receiving that diagnostic or waiting for Goalrail background work

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
**Intent IDs:** OUT-2, OUT-3, OUT-4, OUT-10, SIG-2, SIG-3, SIG-12

A registered hook does not run until the scaffold's own trust step has been completed where the scaffold has one. Goalrail SHALL report attachment health on demand, distinguishing at least: repository unmanaged; project declaration invalid; managed project not locally connected; hooks registered but trust pending or unverifiable; executable missing; configuration unreadable; and attachment locally working. Project identity SHALL come from the committed declaration and MUST NOT depend on the attachment verdict.

Health SHALL be judged in the scope where each scaffold's registration belongs. A surviving registration in a superseded scope SHALL be named with the consented removal action and MUST NOT be modified by diagnosis. Goalrail MUST NOT write, compute, reproduce, pre-approve, or simulate scaffold trust records. Reporting an observed trust state is permitted; absence of verifiable trust freshness SHALL remain explicit.

Health SHALL verify every required registered event, the exact durable executable, safe configuration parsing, and the current Goalrail-owned handler identity. Where no scaffold is named, every supported scaffold SHALL be reported. Local attachment working MUST NOT be described as full project enforcement; shared admission remains a separate diagnosis layer.

#### Scenario: Managed project is not connected locally
- **WHEN** attachment health runs in a valid declared project with no registration for the selected scaffold
- **THEN** it reports managed identity, local disconnection, and the exact consented setup action

#### Scenario: Repository is unmanaged
- **WHEN** attachment health runs where no declaration claim exists
- **THEN** it reports unmanaged and does not treat the directory as a participant

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path is present but invalid
- **THEN** attachment health reports declared-invalid and does not recommend creating an unrelated local marker

#### Scenario: Attachment is not yet trusted
- **WHEN** hooks are registered but the scaffold's required trust step is incomplete or unverified
- **THEN** the report names the exact user action or uncertainty without writing a trust record

#### Scenario: Health is judged in the scaffold scope
- **WHEN** a scaffold's registration belongs inside the repository or at user scope
- **THEN** health reads that declared scope and does not infer state from another scope

#### Scenario: Only part of registration survives
- **WHEN** one required event is missing while another remains
- **THEN** the attachment is not locally working and the missing event is named

#### Scenario: Registered executable is gone
- **WHEN** the configured Goalrail executable is missing or not runnable
- **THEN** health reports the exact failure and routes through consented setup planning

#### Scenario: Scaffold configuration cannot be read
- **WHEN** scaffold configuration is unreadable, unsafe, or malformed
- **THEN** health names that failure rather than reporting ordinary disconnection

#### Scenario: No scaffold is named
- **WHEN** a health query names no scaffold
- **THEN** every supported scaffold is reported without assuming one

#### Scenario: Attachment is locally working
- **WHEN** the project declaration is valid, every required event points to the current durable executable, configuration is readable, and trust verifies where required
- **THEN** health reports the attachment locally working and makes no claim about shared admission

#### Scenario: Trust would be written by Goalrail
- **WHEN** any Goalrail path would write, compute, reproduce, or otherwise approve scaffold trust on the user's behalf
- **THEN** that action is prohibited regardless of setup authorization
