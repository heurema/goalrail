# Work Spec Specification

## Purpose

Define the minimal provider-neutral contract that binds one trusted local run to confirmed intent, pinned repository state, bounded execution inputs, and explicit authority limits.

## Requirements
### Requirement: WorkSpec is minimal and provider-neutral
**Intent IDs:** OUT-5, OUT-6, SIG-5, SIG-6

The current canonical WorkSpec SHALL describe exactly one trusted repository-local run using only a stable WorkSpec ID and version, one work-unit ID, one managed-project declaration identity and digest, one exact Goalrail change identity, repository identity and pinned base revision, one exact confirmed Intent Snapshot version and digest, bounded task statement and path scope, non-empty required check references, explicit stop conditions, and the declared local execution and effect posture.

It MUST NOT contain provider names, provider launch arguments, provider session identity, approval UI state, pull-request metadata, mutable review or check results, raw lineage events, or OpenSpec runtime types. The Goalrail change reference is a provider-neutral stable artifact reference; OpenSpec MAY compile or materialize that artifact but MUST NOT become a canonical WorkSpec type.

#### Scenario: Current minimal WorkSpec is accepted
- **WHEN** a WorkSpec contains every current canonical field, no extra field, and only provider-neutral values
- **THEN** validation accepts it as the complete bounded input for one managed run

#### Scenario: Project or change binding is absent
- **WHEN** a new WorkSpec omits its work-unit, managed-project, exact confirmed intent, or exact Goalrail change binding
- **THEN** validation rejects it before freezing or preparing a run

#### Scenario: Provider or mutable integration details leak into WorkSpec
- **WHEN** a WorkSpec contains a Codex or Claude field, provider launch argument, provider session ID, pull-request or review state, approval UI state, or OpenSpec runtime object
- **THEN** validation rejects it before the WorkSpec can be frozen

### Requirement: WorkSpec binds confirmed intent and repository state
**Intent IDs:** OUT-5, OUT-6, OUT-10, SIG-5, SIG-6, SIG-12

A current WorkSpec SHALL resolve one valid committed project declaration and policy, one exact confirmed Intent Snapshot version, one current Goalrail change compiled from that intent, one open work-unit identity, and one pinned base revision in the declared repository. The intent and change SHALL belong to the same managed project and the change SHALL cite the exact intent identity and digest. The WorkSpec's task, checks, stop conditions, and bounded path scope MUST be present before freezing, and every path MUST resolve inside the declared worktree.

Validation SHALL fail closed on a candidate, missing, superseded-without-version, substituted, or project-mismatched intent; a missing, ambiguous, archived, parallel-source, or intent-mismatched change; a declaration or policy mismatch; a closed or conflicting work unit; an absent base revision; or a path escape. Failure MUST occur without preparing or launching a run.

#### Scenario: Managed project, intent, change, and repository state resolve
- **WHEN** all identities and digests match, intent is confirmed, the change is current and bound to that intent, the work unit is open, the base revision exists, and scope stays inside the worktree
- **THEN** the WorkSpec passes its project, intent, change, work-unit, and repository gates

#### Scenario: Intent is not confirmed
- **WHEN** the referenced intent is candidate, missing, substituted, or superseded without the exact version
- **THEN** WorkSpec validation fails without preparing or launching a run

#### Scenario: Parallel change source is supplied
- **WHEN** the referenced plan or change is not the current Goalrail change bound to the exact confirmed intent
- **THEN** validation fails with a categorical source-of-truth reason

#### Scenario: Project policy changed after WorkSpec construction
- **WHEN** the declaration or policy identity no longer matches the WorkSpec binding
- **THEN** validation fails and requires a new WorkSpec version rather than silently adopting the new policy

#### Scenario: Scope escapes the repository
- **WHEN** a declared path resolves outside the managed worktree root
- **THEN** WorkSpec validation fails and identifies the invalid scope entry

### Requirement: WorkSpec freezing produces deterministic identity
**Intent IDs:** OUT-5, OUT-6, SIG-6, SIG-10

Goalrail SHALL canonicalize a valid current WorkSpec and derive one deterministic digest from its semantic content. Fields that represent sets MUST have a stable canonical order. A frozen WorkSpec MUST remain immutable; any material change SHALL require a new WorkSpec version and digest.

Project declaration, policy, intent, change, work-unit, repository revision, task, path scope, checks, stop conditions, and posture bindings SHALL all contribute to the deterministic identity. Mutable run, session, commit, pull-request, review, check-result, exception, and owner-decision evidence SHALL be attached later through lineage and MUST NOT mutate the frozen WorkSpec.

#### Scenario: Equivalent current inputs produce one digest
- **WHEN** two valid WorkSpecs have identical semantic content but their set-valued fields arrive in different orders
- **THEN** canonicalization produces identical frozen content and the same digest

#### Scenario: Frozen authority is changed
- **WHEN** an actor attempts to change any project, policy, intent, change, work-unit, revision, task, scope, check, stop-condition, or posture binding
- **THEN** Goalrail rejects the mutation and requires a new WorkSpec version

#### Scenario: Late pull-request identity becomes known
- **WHEN** a pull request is created after the WorkSpec is frozen
- **THEN** lineage binds the pull request to the work unit and frozen WorkSpec without changing the WorkSpec digest

### Requirement: WorkSpec payload is bounded and safe to retain
**Intent IDs:** OUT-4, SIG-5

A WorkSpec SHALL retain a bounded task statement and stable references needed to execute and review the run. It MUST reject credentials, secrets, raw provider prompts, transcripts, raw source bodies, provider session identities, and unbounded content.

#### Scenario: Bounded task and references are supplied
- **WHEN** the task statement and all references satisfy the configured metadata bounds and contain no prohibited payload
- **THEN** Goalrail retains them in the frozen WorkSpec

#### Scenario: Sensitive or raw provider content is supplied
- **WHEN** a WorkSpec contains a credential, secret-shaped value, raw prompt, transcript, raw source body, provider session identity, or unbounded field
- **THEN** validation fails before the content is retained or passed to a provider

### Requirement: V0 posture remains trusted local and provider-enforced
**Intent IDs:** OUT-3, SIG-4, SIG-6

The v0 WorkSpec SHALL declare a trusted local repository run whose filesystem sandbox and interactive approvals remain enforced by the selected provider. The accepted posture MUST exclude credential injection, external effects, automatic Git lifecycle actions, background continuation, automatic retry, and additional agent launches.

#### Scenario: Trusted local posture is declared
- **WHEN** the WorkSpec declares the supported trusted-local and provider-enforced posture
- **THEN** Goalrail accepts the posture without claiming separate effect authority

#### Scenario: Unsupported authority is requested
- **WHEN** a WorkSpec requests credentials, external effects, automatic Git actions, provider-enforcement bypass, background work, retry, or additional agents
- **THEN** validation rejects the WorkSpec rather than silently broadening its authority

### Requirement: Legacy WorkSpecs remain readable but cannot mint current conformance
**Intent IDs:** OUT-9, SIG-5, SIG-11

Goalrail SHALL continue to decode and inspect retained legacy WorkSpecs and receipts under their original schemas. A legacy WorkSpec lacking current project, change, or work-unit bindings MUST NOT be used to prepare a new normally conformant run in a managed project. A committed bounded migration policy MAY classify exact retained legacy evidence as `BOOTSTRAP`, but MUST NOT rewrite it or report it as `VALID`.

#### Scenario: Historical receipt references a legacy WorkSpec
- **WHEN** an operator inspects retained v0 evidence after project migration
- **THEN** Goalrail displays the original immutable identities and labels the missing current bindings without rewriting history

#### Scenario: New run requests legacy authority
- **WHEN** a managed project attempts to prepare a new run from a legacy WorkSpec schema
- **THEN** preparation is refused unless an exact bounded bootstrap policy applies, and any permitted classification remains `BOOTSTRAP`
