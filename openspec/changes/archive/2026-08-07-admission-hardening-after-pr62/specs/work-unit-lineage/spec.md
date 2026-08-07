## MODIFIED Requirements

### Requirement: One provider-neutral work unit connects existing authoritative artifacts
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-8, SIG-1, SIG-2, SIG-3, SIG-9

Goalrail SHALL assign one stable work-unit identifier before the first managed run and retain a provider-neutral lineage record for that unit. The record SHALL reference, rather than copy or reinterpret, the exact committed project declaration and policy identity, confirmed Intent Snapshot version and digest, Goalrail change identity, frozen WorkSpec identity and digest, provider-authoritative run and root-session identities, material commits, pull request, comment/review index, selected checks, terminal receipt, and owner decision.

Every reference SHALL name its artifact kind, stable identity, immutable version or digest where the source exposes one, source location or adapter, and binding time. A relation whose semantics affect admission SHALL not be satisfied by reference shape alone: the exact retained intent snapshot SHALL validate as actively confirmed, the exact terminal receipt SHALL validate with status `passed`, and an external owner decision SHALL carry authenticated authority evidence matching committed policy. Canonical lineage MUST NOT contain provider prompts, transcripts, comment bodies, source bodies, credentials, secret-shaped data, or provider-specific fields outside bounded adapter references.

#### Scenario: Complete normal work unit is resolved
- **WHEN** every required normal-flow reference exists and its identity, integrity, authority, and semantic binding verifies
- **THEN** Goalrail resolves one connected graph from project declaration through owner decision without copying semantic artifact content

#### Scenario: Existing artifact is referenced
- **WHEN** a confirmed intent, frozen WorkSpec, terminal receipt, commit, or review is added to lineage
- **THEN** the record stores its stable reference and integrity identity rather than a duplicate semantic representation

#### Scenario: Candidate intent is labeled confirmed by an event
- **WHEN** a lineage event targets an intent artifact whose retained snapshot is candidate or malformed
- **THEN** the confirmed-intent relation remains non-satisfying regardless of the event relation label

#### Scenario: Failed receipt is labeled terminal
- **WHEN** a lineage event targets a structurally valid terminal receipt whose status is not `passed`
- **THEN** the terminal-receipt relation remains non-satisfying regardless of the event relation label

#### Scenario: Provider content is supplied as lineage
- **WHEN** an adapter attempts to retain a raw prompt, transcript, review body, source body, credential, or unbounded provider payload
- **THEN** lineage validation fails before that content is retained

### Requirement: Lineage exposes explicit lifecycle completeness
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-8, SIG-1, SIG-2, SIG-3, SIG-9, SIG-10

Goalrail SHALL distinguish at least an open work unit, a unit ready for shared admission, and a closed work unit. Open lineage MAY omit future identities but MUST name every missing required relation. The admission verifier's effective lifecycle view MAY combine immutable committed work-unit evidence with exact bounded provider observations from the current authenticated admission packet, but those observations MUST remain attributable packet evidence and MUST NOT be persisted or presented as branch-authored canonical lineage. Admission-ready lineage SHALL contain all evidence the governing policy requires before integration, including semantically confirmed intent, a successful terminal receipt, and the bounded provider-authenticated owner decision required for that boundary. Closed lineage SHALL additionally bind the integration or rejection outcome and the terminal owner-visible closure evidence.

An empty comment set, absent external review, failed or incomplete terminal execution, unauthenticated owner claim, or non-integration outcome MUST be represented explicitly where policy permits it; absence or relation labels alone MUST NOT be inferred as success. A unit whose required late evidence never arrives remains incomplete and cannot acquire normal valid status merely because its code was committed or merged elsewhere.

#### Scenario: Work is still local
- **WHEN** intent, change, WorkSpec, run, and a passed receipt exist but no commit or pull request exists
- **THEN** lineage is open and reports the exact late relations still missing

#### Scenario: Authenticated provider evidence completes late relations
- **WHEN** the current packet supplies authenticated pull-request, review, check, and owner-decision evidence for the exact committed range and every repository-owned relation verifies
- **THEN** the effective lifecycle view may become admission-ready without requiring those provider observations to have been precommitted

#### Scenario: Owner approves admission
- **WHEN** provider-authenticated owner approval, required reviews, checks, a confirmed intent, a passed receipt, and every other pre-integration relation are bound
- **THEN** lineage may become admission-ready while remaining distinct from the later integration outcome

#### Scenario: Owner claim is branch controlled
- **WHEN** an event actor label or repository target claims owner approval without authenticated authority evidence
- **THEN** the owner-decision relation remains missing or invalid and the work unit cannot become admission-ready

#### Scenario: Pull request has no comments
- **WHEN** policy permits a pull request with zero comments and the adapter verifies that empty set
- **THEN** lineage binds an explicit empty comment index rather than omitting the relation

#### Scenario: Terminal execution did not pass
- **WHEN** a required terminal receipt exists but its validated status is not `passed`
- **THEN** the work unit remains non-admission-ready and reports the receipt relation as semantically unsatisfied

#### Scenario: Integration outcome is retained
- **WHEN** an admitted pull request is merged or rejected
- **THEN** one closure event binds that outcome and the unit can become closed without rewriting earlier evidence
