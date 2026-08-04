# Local Run Specification

## Purpose

Define the operator-assisted lifecycle for one trusted local run, from inspectable preparation through a one-shot provider launch and bounded terminal receipt, while preserving provider enforcement and a separate activation gate.
## Requirements
### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-1, OUT-2, OUT-4, SIG-1, SIG-3

The `gr` operator surface SHALL resolve and validate the requested WorkSpec,
freeze its canonical content, and display the exact snapshot and digest before
launch. Preparation MUST NOT launch a provider or mutate the target repository
worktree.

Before any prepared state is persisted, preparation SHALL verify the exact
referenced confirmed intent bytes and digest. When a Context Pack is present or
required, preparation SHALL additionally consume the Context Pack and Intent
Snapshot as a pair: it MUST select the pair's artifact contract or pinned legacy
mode before semantic interpretation, then parse the intent against the pack.
The pair's contract selection, normalized semantics, and failures SHALL be
obtained through `IntentResolver`'s direct production path rather than inferred
from compiler tests. The pack MUST resolve inside the canonical repository root.
The intent artifact, and every Context Pack that is read, MUST resolve inside the
canonical repository root, MUST be a regular file, and MUST remain within the
same bounded artifact size limit.

Rejection of a non-regular artifact MUST NOT depend on a check that a concurrent
local process can invalidate before the artifact is read. Reading SHALL
therefore inspect the same open descriptor it reads from, and SHALL open in a
way that neither follows a substituted symbolic link nor blocks on a
substituted pipe. Preparation fails with a bounded reason instead of blocking or
reading outside the verified boundary.

A Context Pack SHALL be required whenever the change loader would require one:
a current change under the project intent schema always requires it, an archived
change requires it only when its intent declares one, and a change under another
schema that declares none does not. When a Context Pack is required and absent,
unreadable, malformed, mismatched, oversized, non-regular, or resolving outside
the canonical repository root, when any context reference the intent uses is
absent from the pack, or when artifact contract selection or pair conformance
fails, preparation MUST fail before any prepared state is persisted and before
any run ID, launch claim, or provider invocation exists.

An artifact contract or format failure SHALL return the shared structured
artifact diagnostic. A partial, malformed, mismatched, or unsupported declared
contract MUST NOT be retried as legacy input. The diagnostic remains subject to
the same repository boundary and bounded-output protections as preparation; a
caller MUST NOT need to parse its rendered text to identify the failure.

Preparation SHALL additionally fail closed when the reserved escalation path is
already populated in the frozen baseline observation, so that a stale, foreign,
or manually authored question cannot attach itself to a new run. The gate
applies to that exact path and MUST NOT reject unrelated content that shares its
directory.

The reserved escalation path SHALL receive the same repository-boundary
treatment as a declared scoped path: its existing ancestors MUST resolve inside
the canonical repository root. A reserved directory that resolves outside the
repository MUST fail preparation, because a provider writing through it would
place the question where worktree observation can never see it and the channel
would be silently unusable.

This verification MUST NOT depend on proposal, specification, design, task,
provider, or activation artifacts, and MUST NOT add Context Pack fields or an
escalation path field to the canonical WorkSpec.

#### Scenario: Operator prepares a valid run
- **WHEN** the operator prepares a valid WorkSpec
- **THEN** `gr` displays the frozen snapshot and digest and records no provider launch

#### Scenario: Preparation fails validation
- **WHEN** the WorkSpec, confirmed intent reference, repository revision, scope, checks, stop conditions, or posture is invalid
- **THEN** preparation fails with a bounded reason and no run ID or provider launch is created

#### Scenario: Contract v1 confirmed intent is verified
- **WHEN** preparation reads an exact confirmed intent and Context Pack carrying the supported equal contract v1 declaration, both resolving inside the canonical repository root as regular files
- **THEN** `IntentResolver` selects contract v1, parses the intent against that pack, verifies every context reference, and persists no prepared state until all checks succeed

#### Scenario: Pinned legacy confirmed intent is verified
- **WHEN** preparation reads an exact confirmed intent and Context Pack with no contract declaration whose syntax belongs to the pinned legacy allowlist
- **THEN** `IntentResolver` selects the named legacy mode, verifies the pair semantics, and persists no prepared state until all checks succeed

#### Scenario: Declared artifact contract is invalid
- **WHEN** a required pair carries a partial, malformed, mismatched, or unsupported declared artifact contract
- **THEN** preparation returns the shared structured diagnostic without legacy fallback, prepared state, run ID, launch claim, or provider invocation

#### Scenario: Context-backed confirmed intent is verified
- **WHEN** preparation reads the exact confirmed intent and the Context Pack of its change, both resolving inside the canonical repository root as regular files
- **THEN** the intent is parsed against that pack, every context reference it uses is present, and no prepared state is persisted until both succeed

#### Scenario: Required Context Pack is absent
- **WHEN** a current change under the project intent schema supplies no Context Pack, or an intent declares a pack that does not resolve next to it
- **THEN** preparation fails before persisting prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Context Pack is invalid or escapes the repository
- **WHEN** the Context Pack is unreadable, malformed, oversized, identifies a different pack than the intent declares, omits a referenced context item, or resolves outside the canonical repository root
- **THEN** preparation fails before persisting prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Context Pack is not required
- **WHEN** an archived change declares no Context Pack, or a change under another schema declares none, and no sibling context artifact exists
- **THEN** preparation verifies the intent alone and continues without requiring a pack or selecting a pair contract

#### Scenario: Intent artifact or Context Pack is not a regular file
- **WHEN** either artifact is a FIFO, device, directory, or other non-regular file
- **THEN** preparation rejects it with a bounded reason and does not block waiting for the artifact

#### Scenario: An artifact is substituted after it is resolved
- **WHEN** a concurrent local process replaces a resolved artifact with a symbolic link or a pipe between resolution and reading
- **THEN** preparation rejects it rather than following the link outside the verified boundary or blocking on the pipe, because the regular-file check inspects the same descriptor that is read

#### Scenario: Reserved escalation path is already populated
- **WHEN** the reserved escalation path holds content at the moment the baseline observation is frozen
- **THEN** preparation fails with a bounded reason before any prepared state is persisted, while unrelated content in the same directory does not block preparation

#### Scenario: Reserved escalation directory escapes the repository
- **WHEN** an existing ancestor of the reserved escalation path resolves outside the canonical repository root
- **THEN** preparation fails with a bounded reason, rather than admitting a run whose question would be written where observation cannot see it

### Requirement: Start is explicit and one-shot
**Intent IDs:** OUT-2, SIG-2, SIG-3

Goalrail SHALL launch a prepared run only after a separate explicit operator start action. A successful start SHALL generate one run ID and permit at most one provider launch for the frozen WorkSpec instance. Goalrail MUST reject duplicate starts and MUST NOT retry, start another agent, or continue the run in the background.

#### Scenario: Explicit start launches once
- **WHEN** the operator explicitly starts one valid prepared WorkSpec
- **THEN** Goalrail generates one run ID and invokes the selected provider adapter exactly once

#### Scenario: Start is repeated
- **WHEN** start is requested again for a WorkSpec instance whose launch was already attempted
- **THEN** Goalrail rejects the request and does not invoke the provider again

#### Scenario: Provider launch fails
- **WHEN** the single provider launch returns an error or terminates before completing
- **THEN** Goalrail records a terminal failure without retrying or starting a replacement run

### Requirement: Provider adapter preserves enforcement boundaries
**Intent IDs:** OUT-2, OUT-3, SIG-2, SIG-3

The local-run lifecycle SHALL obtain launch behavior and provider-authoritative root session identity through a replaceable adapter boundary. The first v0 adapter SHALL support Codex, while canonical WorkSpec and run state remain provider-neutral. Goalrail MUST NOT auto-approve, bypass, weaken, or reinterpret provider sandbox and approval decisions.

The adapter boundary SHALL additionally carry the launch-time escalation
announcement in the terms of the provider's own contract. An adapter that cannot
deliver the announcement MUST fail the launch explicitly, rather than starting a
run whose escalation channel is unreachable. A silent degradation would return
the run to the guessing behaviour the channel exists to prevent, while still
recording an outcome that looks ordinary.

An adapter MUST NOT report delivery it does not itself perform. Where delivery
depends on a component the adapter only invokes — a launcher, a capsule, or any
other collaborator that completes the path into the session — the report SHALL
come from the component that knows whether that path exists. An adapter that
assumed delivery on a collaborator's behalf would defeat this check entirely,
because the launch would proceed while the channel remained unreachable.

Carrying the announcement MUST NOT add provider fields to the canonical WorkSpec
or the terminal receipt, and MUST NOT alter the provider's sandbox or approval
decisions.

#### Scenario: Codex requests operator approval
- **WHEN** Codex gates an action through its normal approval mechanism
- **THEN** the request remains with Codex and `gr` neither approves it automatically nor represents it as Goalrail authorization

#### Scenario: Provider denies an action
- **WHEN** the provider sandbox or approval boundary denies an action
- **THEN** the action remains denied and Goalrail records the bounded provider outcome

#### Scenario: Provider configuration is unavailable
- **WHEN** the requested adapter cannot supply its required launch contract
- **THEN** Goalrail fails the run without falling back to another provider or guessing provider state

#### Scenario: The adapter carries the announcement
- **WHEN** an adapter launches a run
- **THEN** it delivers the announcement in its own contract's terms, and the canonical WorkSpec and terminal receipt gain no provider field for it

#### Scenario: The adapter cannot deliver the announcement
- **WHEN** an adapter has no path for delivering the announcement to the run
- **THEN** the launch fails explicitly and no run is started whose escalation channel would be unreachable

#### Scenario: Delivery depends on a collaborator the adapter only invokes
- **WHEN** the announcement reaches the session only if a launcher or capsule completes the path
- **THEN** the report of whether delivery happens comes from that component rather than being assumed by the adapter, so a missing collaborator fails the launch instead of passing the check

### Requirement: Run lineage uses provider-authoritative identity
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, SIG-1, SIG-2, SIG-4

Each launched run SHALL bind its generated run ID and frozen WorkSpec digest to
the exact root session identity returned by the provider-authoritative launch
receipt. Missing, malformed, or conflicting identity MUST remain explicit and
MUST NOT be repaired through manual entry, prompt parsing, transcript parsing,
provider terminal text, or heuristic matching.

For the exact supported Codex contract, an invocation-local `SessionStart`
definition used for lineage SHALL contain exactly one matcher group with
exactly one nested synchronous command handler. The handler SHALL declare
`type="command"` and one safely quoted command string that invokes the absolute
local capsule executable with the fixed `hook` subcommand. Goalrail MUST reject
unsafe handler input and the spent flattened v1 shape instead of emitting or
accepting a definition that registers no handler.

The bounded provider lifecycle payload SHALL travel through private one-shot
IPC to the existing correlation path. Goalrail MUST NOT retain the raw payload
or add provider-specific hook fields to the WorkSpec. Deterministic contract
and correlation tests MUST prove this boundary without launching Codex or
creating another provider attempt.

#### Scenario: Exact Codex hook definition is rendered
- **WHEN** Goalrail receives a valid absolute local capsule executable for the exact supported Codex contract
- **THEN** it renders one `SessionStart` matcher group containing exactly one nested synchronous command handler with a safely quoted capsule command

#### Scenario: Spent flattened hook definition is rejected
- **WHEN** the old v1 shape places a command array directly on the matcher group and omits the nested handler list
- **THEN** deterministic contract validation identifies zero registered handlers and Goalrail does not treat the shape as usable lineage configuration

#### Scenario: Unsafe hook command input is supplied
- **WHEN** the capsule executable is non-absolute or contains input that cannot be safely represented by the bounded handler command
- **THEN** rendering fails without returning a partial hook definition or starting a provider

#### Scenario: Provider returns verified root identity
- **WHEN** one pinned-schema-compatible `SessionStart` payload reaches the private one-shot IPC for the immutable run context
- **THEN** Goalrail records the verified WorkSpec-to-run-to-session lineage and retains no raw hook payload

#### Scenario: Root identity is missing or conflicting
- **WHEN** the adapter returns no root identity, malformed identity, or identity that conflicts with the immutable run context
- **THEN** Goalrail records an unlinked terminal outcome and does not guess or accept a manual replacement

### Requirement: Terminal receipt is bounded and reviewable
**Intent IDs:** OUT-1, OUT-4, OUT-5, SIG-1, SIG-4, SIG-5

Goalrail SHALL produce a terminal receipt containing the WorkSpec ID, version, and digest; run ID; provider-authoritative root session reference when verified; pinned repository revision; effective path scope; exact selected check references and results; terminal status; and a bounded worktree-delta reference. The receipt MUST NOT contain raw prompts, transcripts, credentials, secrets, raw source bodies, or copied worktree content.

A receipt SHALL carry an explicit schema identifier. The current identifier is
`goalrail.terminal-receipt/v1`. A stored receipt that carries no schema
identifier SHALL remain readable as the unversioned predecessor and MUST NOT be
rewritten, re-signed, or invalidated by the introduction of the identifier.

A v1 receipt SHALL additionally carry the frozen intent reference — identifier,
version, and digest — taken from the frozen WorkSpec, so a run can be traced to
the confirmed intent it was launched from without a naming convention. Receipt
validation MUST reject a v1 receipt whose intent reference is absent or
malformed; the relaxed behaviour remains available only to receipts that predate
the schema identifier. A chain that silently disappears is worse than one that
was never promised.

When an escalation artifact was retained for the run, the receipt SHALL carry an
escalation block naming the reserved path, the digest of the retained bytes, and
a reference to the retained copy in the run store. The block SHALL reference the
retained bytes rather than the mutable worktree file, and MUST NOT embed the
payload itself.

#### Scenario: Run completes with all selected checks
- **WHEN** the launched run reaches a terminal state and every selected check has an explicit result
- **THEN** Goalrail emits a complete review receipt with the exact check set and bounded worktree-delta reference

#### Scenario: Required check result is missing
- **WHEN** a terminal receipt lacks a result for any selected check
- **THEN** Goalrail marks verification incomplete and does not report the run as fully passing

#### Scenario: Receipt contains prohibited payload
- **WHEN** receipt data contains raw prompts, transcripts, credentials, secrets, raw source bodies, or copied worktree content
- **THEN** receipt validation fails before the prohibited payload is retained

#### Scenario: Receipt is emitted under the current schema
- **WHEN** Goalrail emits any terminal receipt after this change
- **THEN** the receipt carries the `goalrail.terminal-receipt/v1` identifier and the frozen intent reference

#### Scenario: A receipt predating the schema identifier is read
- **WHEN** a stored receipt carries no schema identifier
- **THEN** it still validates as the unversioned predecessor and is neither rewritten nor rejected

#### Scenario: A v1 receipt lacks a usable intent reference
- **WHEN** a receipt carries the current schema identifier but its intent reference is absent, non-canonical, zero-versioned, or carries a malformed digest
- **THEN** receipt validation rejects it, while a receipt without a schema identifier is still accepted without one

#### Scenario: Blocked receipt records the escalation
- **WHEN** a run ends blocked
- **THEN** its receipt names the reserved path, the digest of the retained bytes, and the retained copy, and contains no copy of the payload itself

### Requirement: Real activation remains separately gated
**Intent IDs:** OUT-1, OUT-5, SIG-1, SIG-2, SIG-7

The v0 implementation SHALL continue to reject every non-fixture production
start before provider launch unless an explicit, owner-selected dogfood
authorization matches one prepared WorkSpec digest and its pinned repository
revision exactly. The matching authorization SHALL permit at most one provider
invocation and SHALL remain distinct from intent confirmation, planning
completion, or general effect authority. Context, intent, proposal, specs,
design, tasks, tests, or implementation MUST NOT otherwise activate a real
run.

#### Scenario: Planning and implementation are complete without authorization
- **WHEN** all planning artifacts and local implementation checks exist but no
  exact dogfood authorization exists
- **THEN** Goalrail records zero real provider launches and rejects every
  non-fixture start before provider invocation

#### Scenario: Exact dogfood authorization exists
- **WHEN** an operator starts the one prepared WorkSpec named by a matching
  owner-selected authorization at its pinned repository revision
- **THEN** Goalrail invokes the selected provider adapter at most once and
  continues to reject every other non-fixture start

#### Scenario: Fixture exercises the lifecycle
- **WHEN** a deterministic fixture starts the v0 lifecycle
- **THEN** Goalrail exercises prepare, one-shot launch simulation, lineage,
  checks, and receipt behavior without creating a real provider session

### Requirement: Gr performs no hidden follow-on actions
**Intent IDs:** OUT-5, SIG-6

The `gr` flow MUST stop after its terminal receipt. It MUST NOT automatically create or switch branches, stage files, commit, push, create or merge a pull request, rebase, deploy, publish, release, inject credentials, perform external effects, retry, launch another agent, or schedule later continuation.

#### Scenario: Terminal receipt is emitted
- **WHEN** a fixture or later authorized real run emits its terminal receipt
- **THEN** `gr` returns control to the operator without any automatic follow-on action

### Requirement: Blocked is a terminal escalation outcome
**Intent IDs:** OUT-1, OUT-2, OUT-3, SIG-1, SIG-2, SIG-3

The local-run lifecycle SHALL provide the terminal run state `blocked`: the
provider run ended without an implementation because the work item cannot be
completed as specified and an owner decision is required. `blocked` is terminal.
Goalrail MUST NOT resume, retry, continue, or reopen a blocked run, and MUST NOT
create a dialogue, control plane, or background continuation from it. The loop
closes only through a new intent version and a new one-shot run.

The escalation channel SHALL be one constant repository-relative path, fixed by
Goalrail and identical for every WorkSpec. It MUST NOT be declared, overridden,
or otherwise expressed in the canonical WorkSpec.

A prepared run MAY be started long after its baseline was frozen, and a prepared
run is reused rather than re-observed. Goalrail SHALL therefore reject a start
whose reserved path is already populated, before the provider is launched and
before any run ID or launch claim exists. An artifact that predates the launch
belongs to no provider and MUST NOT be attributed to one.

At provider observation, Goalrail SHALL capture the artifact at that path from
the same worktree observation that decides the run's delta, and SHALL retain its
exact bytes append-only in the run store before any terminal receipt is written.
The retained bytes are authoritative for the run; a later deletion, truncation,
or substitution of the worktree file MUST NOT change the recorded outcome.

The retained bytes MUST be the bytes the observation recorded. When the artifact
read for retention does not match the digest the observation recorded for it,
the artifact changed between the two reads, the single-observation guarantee
does not hold, and the record SHALL be marked invalid with an explicit reason
rather than bound to the receipt. Retention MUST complete before the observation
is persisted, and a run whose observation names the reserved path without a
retained record MUST NOT reach a terminal receipt.

The reserved path SHALL remain visible in the observed changed paths, MUST NOT be
recorded as a scope violation, and MUST NOT be counted as an in-scope edit,
whether or not it falls inside the declared scope.

Goalrail SHALL treat the artifact as opaque bytes and SHALL validate hygiene
only: a regular file, non-empty, within a bounded size limit, valid UTF-8, free
of unsupported control characters, and free of secret-shaped content. Goalrail
MUST NOT parse, interpret, execute, or otherwise semantically validate the
payload.

The terminal status SHALL follow these rules:

- `blocked` requires a valid retained artifact AND an empty in-scope delta.
- A valid retained artifact together with any in-scope edit SHALL yield `failed`
  with an explicit reason, and the artifact SHALL still be retained.
- A retained artifact that fails hygiene validation SHALL yield `failed` with an
  explicit reason and MUST NOT yield `blocked`.
- `unlinked`, `denied`, and `launch_failed` SHALL take precedence over `blocked`.
  Their receipts SHALL record the escalation digest as evidence and MUST NOT
  record the status `blocked`.
- Supplied check results SHALL be recorded verbatim and MUST NOT be replaced,
  suppressed, or forced to unavailable because an artifact exists.
- An explicit check failure SHALL keep the status `failed`. The escalation is
  recorded as evidence, but it MUST NOT convert a failing check into `blocked`.
- While a retained artifact exists, no combination of supplied inputs SHALL
  yield `passed`.
- `blocked` is a run outcome only. It MUST NOT become a per-check result state,
  and the `gr finish` result grammar is unchanged.

#### Scenario: Provider returns a question and leaves the scope clean
- **WHEN** a launched run populates the reserved escalation path with a valid artifact and produces no in-scope edit
- **THEN** Goalrail retains the exact bytes append-only and emits a terminal receipt with status `blocked` and no resume path

#### Scenario: Escalation artifact already exists at preparation
- **WHEN** the reserved escalation path is already populated in the frozen baseline observation
- **THEN** preparation fails closed with a bounded reason before any prepared state is persisted, and no run ID, launch claim, or provider invocation is created

#### Scenario: Escalation artifact appears between preparation and launch
- **WHEN** a prepared run is started and the reserved path was populated after its baseline was frozen
- **THEN** the start fails closed before the provider is launched, and no run ID or launch claim is created, so the artifact is never attributed to that run

#### Scenario: The artifact changes between observation and retention
- **WHEN** the bytes read for retention do not match the digest the deciding observation recorded for the reserved path
- **THEN** the record is marked invalid with an explicit reason, the mismatched bytes are not retained as the observed artifact, and the run does not end blocked

#### Scenario: An observed question was not retained
- **WHEN** a run's persisted observation names the reserved path but no retained escalation record exists
- **THEN** finishing fails closed rather than treating the question as absent, and the run cannot be recorded as passed

#### Scenario: Artifact is accompanied by in-scope edits
- **WHEN** a run populates the reserved escalation path and also edits paths inside the declared scope
- **THEN** the terminal status is `failed` with an explicit reason, the artifact is still retained, and the status is not `blocked`

#### Scenario: Artifact fails hygiene validation
- **WHEN** the artifact at the reserved path is non-regular, empty, oversized, invalid UTF-8, control-character bearing, or secret-shaped
- **THEN** the terminal status is `failed` with an explicit reason and is neither `blocked` nor `passed`

#### Scenario: An unattributable session leaves a question
- **WHEN** the provider outcome is unlinked, denied, or launch-failed and the reserved path holds an artifact
- **THEN** the receipt records that provider outcome as the status and records the escalation digest as evidence without minting a blocked status

#### Scenario: Missing or unavailable check results accompany an escalation
- **WHEN** a run whose reserved path holds a valid retained artifact reaches finish with missing or unavailable check results and a clean in-scope delta
- **THEN** the results are recorded verbatim and the status is `blocked` rather than `passed` or verification-incomplete

#### Scenario: A failing check accompanies an escalation
- **WHEN** the operator supplies an explicit failing check result for a run whose reserved path holds a valid retained artifact
- **THEN** the failing result is recorded verbatim, the status stays `failed`, and the escalation is recorded as evidence rather than converting the failure into `blocked`

#### Scenario: The worktree artifact is removed after the run
- **WHEN** the file at the reserved path is deleted, truncated, or replaced after provider observation
- **THEN** the recorded outcome and the retained bytes are unchanged and the receipt still resolves the exact question that produced the outcome

### Requirement: The escalation channel is announced at launch
**Intent IDs:** OUT-1, OUT-3, OUT-4, OUT-5, SIG-1, SIG-4, SIG-5

A launched run SHALL be told that the escalation channel exists. Admitting a
channel a run cannot learn about delivers no capability: an unannounced channel
is indistinguishable from an absent one, and the measured consequence of an
absent channel is that the work is guessed rather than questioned.

The announcement SHALL state the reserved path, that it applies when the work
item cannot be completed as specified from the repository alone, that the
declared scope must be left untouched for the escalation to stand, and where the
payload shape is published. It MUST NOT describe the work item, characterise the
repository, suggest that a conflict or ambiguity exists, or instruct the run to
look for one. The wording is fixed rather than composed per run.

The announcement SHALL name no provider, model, harness, or vendor. Transport is
the adapter's responsibility and may be provider-specific; the announced content
may not be.

The announcement SHALL be delivered exactly once, at launch, as a statement. It
MUST NOT create a reply path, acknowledgement, capability handshake, retry, or
any later delivery, and MUST NOT be repeated inside a run. Where a provider
signals session events that recur within one run — resumption, clearing, or
compaction — the announcement SHALL accompany only the event that opens the run.

Delivery SHALL be verified before any irreversible step of the launch,
including before a one-time activation authorization is consumed. An
authorization spent on a launch that cannot happen is discarded with nothing to
show for it, and no retry can use it afterwards.

Goalrail MUST NOT place the announcement in the canonical WorkSpec, the terminal
receipt, or the target repository worktree. Composing the work item itself
remains outside this boundary.

#### Scenario: A launched run learns the channel exists
- **WHEN** a run is launched
- **THEN** it is told the reserved path, when the channel applies, that the declared scope must stay untouched, and where the payload shape is published

#### Scenario: The reserved path appears in no artifact the run is given
- **WHEN** the announcement is delivered
- **THEN** the reserved path still appears in no WorkSpec field, no receipt field, and no file written into the target worktree

#### Scenario: The announcement would hint at the work
- **WHEN** announcement text would describe the work item, characterise the repository, or suggest that a conflict or ambiguity is present
- **THEN** it violates this requirement, because a hint would measure the hint rather than the environment

#### Scenario: The announcement is delivered once
- **WHEN** a run is launched and continues to completion
- **THEN** exactly one announcement exists, with no acknowledgement, retry, repetition, or reply path

#### Scenario: A provider session event recurs inside a run
- **WHEN** the provider signals a session event for resumption, clearing, or compaction after the run has started
- **THEN** no further announcement accompanies it, because one launch-time statement must not become a recurring instruction

#### Scenario: An undeliverable announcement meets a one-time authorization
- **WHEN** a launch cannot deliver the announcement and the run is authorized by a one-time activation record
- **THEN** the launch fails without consuming that authorization, leaving it usable once the delivery path exists
