## ADDED Requirements

### Requirement: One provider-neutral work unit connects existing authoritative artifacts
**Intent IDs:** OUT-5, OUT-6, SIG-5, SIG-6

Goalrail SHALL assign one stable work-unit identifier before the first managed run and retain a provider-neutral lineage record for that unit. The record SHALL reference, rather than copy or reinterpret, the exact committed project declaration and policy identity, confirmed Intent Snapshot version and digest, Goalrail change identity, frozen WorkSpec identity and digest, provider-authoritative run and root-session identities, material commits, pull request, comment/review index, selected checks, terminal receipt, and owner decision.

Every reference SHALL name its artifact kind, stable identity, immutable version or digest where the source exposes one, source location or adapter, and binding time. Canonical lineage MUST NOT contain provider prompts, transcripts, comment bodies, source bodies, credentials, secret-shaped data, or provider-specific fields outside bounded adapter references.

#### Scenario: Complete normal work unit is resolved
- **WHEN** every required normal-flow reference exists and its identity binding verifies
- **THEN** Goalrail resolves one connected graph from project declaration through owner decision without copying semantic artifact content

#### Scenario: Existing artifact is referenced
- **WHEN** a confirmed intent, frozen WorkSpec, terminal receipt, commit, or review is added to lineage
- **THEN** the record stores its stable reference and integrity identity rather than a duplicate semantic representation

#### Scenario: Provider content is supplied as lineage
- **WHEN** an adapter attempts to retain a raw prompt, transcript, review body, source body, credential, or unbounded provider payload
- **THEN** lineage validation fails before that content is retained

### Requirement: Lineage grows through immutable backward-reference events
**Intent IDs:** OUT-6, SIG-6, SIG-10

Lineage SHALL grow as an ordered set of immutable, digest-identified events or equivalent append-only records. Creating a later binding MUST NOT rewrite confirmed intent, a frozen WorkSpec, an earlier run receipt, or a previously retained lineage event. An event SHALL identify the work unit, relation, source and target references, actor or adapter, observation time, and deterministic semantic digest.

References unavailable at run time, including commit, pull-request, review, check, and owner-decision identities, SHALL remain explicitly missing until a later bounded event attaches them. Repeating an identical event SHALL be idempotent. Two non-equivalent bindings for a single-valued relation SHALL remain visible as a conflict and MUST NOT be resolved by last-write-wins.

#### Scenario: Pull request is created after the run
- **WHEN** a run receipt exists before a pull-request identifier is known
- **THEN** the work unit remains incomplete until a later immutable event binds the pull request without modifying the run receipt

#### Scenario: Identical event is observed twice
- **WHEN** the same semantic lineage event is ingested more than once
- **THEN** canonicalization yields one event identity and no duplicate edge

#### Scenario: Two commits claim one single-valued slot
- **WHEN** incompatible bindings are supplied for a relation that policy declares single-valued
- **THEN** both claims remain inspectable and lineage is classified ambiguous rather than silently choosing the latest

### Requirement: Lineage exposes explicit lifecycle completeness
**Intent IDs:** OUT-6, OUT-8, SIG-6, SIG-8

Goalrail SHALL distinguish at least an open work unit, a unit ready for shared admission, and a closed work unit. Open lineage MAY omit future identities but MUST name every missing required relation. Admission-ready lineage SHALL contain all evidence the governing policy requires before integration, including the bounded owner decision or provider-authenticated approval required for that boundary. Closed lineage SHALL additionally bind the integration or rejection outcome and the terminal owner-visible closure evidence.

An empty comment set, absent external review, or non-integration outcome MUST be represented explicitly where policy permits it; absence MUST NOT be inferred as success. A unit whose required late evidence never arrives remains incomplete and cannot acquire normal valid status merely because its code was committed or merged elsewhere.

#### Scenario: Work is still local
- **WHEN** intent, change, WorkSpec, run, and receipt exist but no commit or pull request exists
- **THEN** lineage is open and reports the exact late relations still missing

#### Scenario: Owner approves admission
- **WHEN** provider-authenticated owner approval, required reviews, checks, and every other pre-integration relation are bound
- **THEN** lineage may become admission-ready while remaining distinct from the later integration outcome

#### Scenario: Pull request has no comments
- **WHEN** policy permits a pull request with zero comments and the adapter verifies that empty set
- **THEN** lineage binds an explicit empty comment index rather than omitting the relation

#### Scenario: Integration outcome is retained
- **WHEN** an admitted pull request is merged or rejected
- **THEN** one closure event binds that outcome and the unit can become closed without rewriting earlier evidence

### Requirement: Exceptions are first-class lineage evidence, never normal validity
**Intent IDs:** OUT-8, SIG-8, SIG-9

Any standing exemption, emergency break-glass action, or bounded bootstrap/migration path SHALL be represented by an immutable exception event bound to one project policy identity and work unit. It SHALL include a categorical class, actor, reason reference, exact path and effect scope, start, expiry or closure condition, and required owner decision. Free-form justification alone MUST NOT widen scope or override committed policy.

Exception evidence SHALL remain visible after closure and MUST NOT be normalized into ordinary valid lineage. Missing, expired, conflicting, or out-of-scope exception evidence SHALL make the work unit non-admissible.

#### Scenario: Standing policy exemption applies
- **WHEN** a committed narrow exemption matches the exact work-unit scope and remains current
- **THEN** lineage records the exemption identity and classification instead of reporting the unit as ordinary valid work

#### Scenario: Emergency break-glass is invoked
- **WHEN** an authorized actor invokes the committed emergency path for a bounded material change
- **THEN** lineage retains the actor, reason, scope, expiry or closure, and owner decision as break-glass evidence

#### Scenario: Small diff claims an exception
- **WHEN** a change cites only its small size or favored tool without a matching committed rule or explicit bounded exception
- **THEN** no exception event is valid and the missing normal lineage remains visible

#### Scenario: Exception expires before admission
- **WHEN** an exception's time or event boundary has passed before shared admission
- **THEN** the exception cannot authorize admission and remains retained as expired evidence

### Requirement: Stable indexes resolve lineage from any durable bead
**Intent IDs:** OUT-6, SIG-6

Goalrail SHALL provide deterministic indexes or resolvers from every durable identifier available to the core or a registered adapter, including work unit, intent, change, WorkSpec, run, provider root session, commit, pull request, terminal receipt, and owner decision. An index SHALL contain references and digests only and MUST NOT become an alternate source of semantic truth.

Missing provider access SHALL be reported as unavailable rather than causing a fabricated empty result. Conflicting reverse mappings SHALL produce ambiguity and retain every claimant.

#### Scenario: Operator starts from a commit
- **WHEN** an operator asks for lineage using a bound commit identity
- **THEN** Goalrail resolves the work unit and every currently reachable bead with its verification status

#### Scenario: Provider is unavailable
- **WHEN** resolving a pull request or review requires an unavailable provider adapter
- **THEN** the resolver reports that edge unavailable and does not treat it as absent or complete

#### Scenario: One receipt claims two work units
- **WHEN** reverse indexes contain incompatible work-unit bindings for the same terminal receipt
- **THEN** resolution reports ambiguity and neither claimant receives normal valid lineage
