## ADDED Requirements

### Requirement: Context evidence remains separate from semantic intent
**Intent IDs:** OUT-1, SIG-1, SIG-2

A flow change SHALL create a versioned Context Pack before intent confirmation. The pack SHALL contain bounded evidence items and collection lifecycle metadata; it MUST NOT add a fourth semantic intent group or promote an external claim into owner intent.

#### Scenario: Local evidence informs a candidate interpretation
- **WHEN** repository evidence can change how a request is understood
- **THEN** the flow records a concise context item with stable identity, source reference, observation time, and relevance before the candidate Intent Snapshot references it

#### Scenario: External evidence informs a candidate interpretation
- **WHEN** a material unknown requires an external source
- **THEN** the flow records the concise claim, stable source reference, access time, and why the evidence can affect the current change without treating the claim as owner confirmation

#### Scenario: Context would create another intent group
- **WHEN** a Context Pack is compiled into an Intent Snapshot
- **THEN** context remains provenance referenced by semantic items and the snapshot still contains exactly desired outcomes, non-goals, and observable success signals

### Requirement: Context collection is bounded by material uncertainty
**Intent IDs:** OUT-2, SIG-3

The context workflow SHALL begin with the repository anchors required by the applicable project rules and SHALL perform external research only for an explicit material unknown. It MUST record a terminal collection outcome of `sufficient`, `material_unknown`, or `budget_exhausted`, and MUST stop when further evidence cannot change current intent or the current decision.

#### Scenario: Repository evidence is sufficient
- **WHEN** required repository anchors resolve every material ambiguity
- **THEN** the pack completes as `sufficient` without an external lookup

#### Scenario: External evidence resolves a material unknown
- **WHEN** one or more bounded external probes make the unknown immaterial to intent and the current decision
- **THEN** the pack records the evidence and completes as `sufficient` without continuing to collect adjacent information

#### Scenario: Material uncertainty remains
- **WHEN** a bounded probe cannot resolve a fact that could change intent or the selected decision
- **THEN** the pack completes as `material_unknown` or `budget_exhausted`, identifies the unknown, and does not pretend the evidence is sufficient

### Requirement: Context payload is concise and safe to retain
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2

A canonical Context Pack SHALL retain only bounded claims, identifiers, timestamps, relevance statements, and source references. It MUST reject raw source bodies, prompts, transcripts, credentials, secrets, and values that exceed the defined metadata bounds.

#### Scenario: Concise external claim is recorded
- **WHEN** an external source is relevant to a material unknown
- **THEN** the pack accepts a bounded claim and reference without copying the source body

#### Scenario: Raw or sensitive material is supplied
- **WHEN** a context item contains a raw page, transcript, credential, secret-shaped value, or an unbounded field
- **THEN** validation fails before the pack can be completed or referenced by confirmed intent

### Requirement: Development lifecycle exposes sources of truth and owner gates
**Intent IDs:** OUT-7, SIG-10

Goalrail SHALL document the lifecycle from source evidence and Context Pack through confirmed Intent Snapshot, OpenSpec proposal/spec/design/tasks, local implementation and verification, canonical runtime evidence, Langfuse telemetry references, canonical capability specs, decision records, and archived changes. The document MUST identify the source of truth and separate owner gate for apply, commit, merge, activation, and external effects.

#### Scenario: New change begins
- **WHEN** an operator starts a significant Goalrail change
- **THEN** the lifecycle document identifies the required artifact order and the location that owns each durable fact

#### Scenario: Planning artifacts are complete
- **WHEN** OpenSpec reports apply-ready
- **THEN** the document makes clear that planning completion does not authorize implementation, commit, merge, canary activation, or an external effect
