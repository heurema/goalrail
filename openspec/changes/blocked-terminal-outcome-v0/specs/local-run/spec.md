## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-2, OUT-3, SIG-3

The `gr` operator surface SHALL resolve and validate the requested WorkSpec, freeze its canonical content, and display the exact snapshot and digest before launch. Preparation MUST NOT launch a provider or mutate the target repository worktree.

Before any prepared state is persisted, preparation SHALL verify the exact
referenced confirmed intent bytes and digest. When a Context Pack is present or
required, preparation SHALL additionally parse that intent against the pack,
and the pack MUST resolve inside the canonical repository root. The intent
artifact, and every Context Pack that is read, MUST resolve inside the
canonical repository root, MUST be a regular file, and MUST remain within the
same bounded artifact size limit.

Rejection of a non-regular artifact MUST NOT depend on a check that a
concurrent local process can invalidate before the artifact is read. Reading
SHALL therefore inspect the same open descriptor it reads from, and SHALL open
in a way that neither follows a substituted symbolic link nor blocks on a
substituted pipe. Preparation fails with a bounded reason instead of blocking
or reading outside the verified boundary.

A Context Pack SHALL be required whenever the change loader would require one:
a current change under the project intent schema always requires it, an
archived change requires it only when its intent declares one, and a change
under another schema that declares none does not. When a Context Pack is
required and absent, unreadable, malformed, mismatched, oversized, non-regular,
or resolving outside the canonical repository root, and when any context
reference the intent uses is absent from the pack, preparation MUST fail before
any prepared state is persisted and before any run ID, launch claim, or
provider invocation exists.

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
- **THEN** preparation verifies the intent alone and continues without requiring a pack

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
