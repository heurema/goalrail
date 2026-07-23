## ADDED Requirements

### Requirement: WorkSpec is minimal and provider-neutral
**Intent IDs:** OUT-1, SIG-1

The canonical WorkSpec SHALL describe exactly one trusted repository-local run using only a stable WorkSpec ID and version, repository identity and pinned base revision, confirmed intent or change reference, bounded task statement and path scope, non-empty required check references, explicit stop conditions, and the declared local execution and effect posture. It MUST NOT contain provider names, provider launch arguments, provider session identity, approval UI state, or OpenSpec runtime types.

#### Scenario: Minimal WorkSpec is accepted
- **WHEN** a WorkSpec contains every canonical field, no extra field, and only provider-neutral values
- **THEN** validation accepts it as the complete input for one run

#### Scenario: Provider details leak into WorkSpec
- **WHEN** a WorkSpec contains a Codex field, provider launch argument, provider session ID, approval state, or OpenSpec-specific runtime value
- **THEN** validation rejects it before the WorkSpec can be frozen

### Requirement: WorkSpec binds confirmed intent and repository state
**Intent IDs:** OUT-1, SIG-2

A WorkSpec SHALL reference one exact confirmed Intent Snapshot version and one pinned repository base revision. Its bounded path scope MUST resolve inside the declared repository, and its task, checks, and stop conditions MUST be present before freezing.

#### Scenario: Confirmed intent and repository state resolve
- **WHEN** the referenced intent is confirmed, the pinned revision exists in the declared repository, and every scoped path remains inside that repository
- **THEN** the WorkSpec passes its intent and repository gates

#### Scenario: Intent is not confirmed
- **WHEN** the referenced intent is candidate, missing, superseded without an exact version, or otherwise invalid
- **THEN** WorkSpec validation fails without preparing or launching a run

#### Scenario: Scope escapes the repository
- **WHEN** a declared path resolves outside the repository root
- **THEN** WorkSpec validation fails and identifies the invalid scope entry

### Requirement: WorkSpec freezing produces deterministic identity
**Intent IDs:** OUT-1, SIG-1, SIG-2

Goalrail SHALL canonicalize a valid WorkSpec and derive one deterministic digest from its semantic content. Fields that represent sets MUST have a stable canonical order. A frozen WorkSpec MUST remain immutable; any material change SHALL require a new WorkSpec version and digest.

#### Scenario: Equivalent inputs produce one digest
- **WHEN** two valid WorkSpecs have identical semantic content but their set-valued fields arrive in different orders
- **THEN** canonicalization produces identical frozen content and the same digest

#### Scenario: Frozen content is changed
- **WHEN** an actor attempts to change the task, scope, checks, stop conditions, posture, intent reference, or pinned revision of a frozen WorkSpec
- **THEN** Goalrail rejects the mutation and requires a new WorkSpec version

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
