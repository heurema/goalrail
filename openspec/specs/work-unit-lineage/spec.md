# work-unit-lineage Specification

## Purpose

Define provider-neutral, append-only work-unit lineage, lifecycle completeness, exception evidence, and deterministic reverse resolution.
## Requirements
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

### Requirement: Exceptions are first-class lineage evidence, never normal validity
**Intent IDs:** OUT-8, SIG-8, SIG-9, OUT-1, OUT-2, OUT-4, SIG-1, SIG-2, SIG-3

Any standing exemption, emergency break-glass action, bounded bootstrap/migration path, or claimed restoration of an existing requirement SHALL be represented by an immutable exception event bound to one project policy identity and work unit. It SHALL include a categorical class, actor, reason reference, exact path and effect scope, start, expiry or closure condition, and required owner decision. Free-form justification alone MUST NOT widen scope or override committed policy.

A `restoration` exception is the claim that a change alters no requirement but returns behaviour to one already recorded. It SHALL additionally name a requirement artifact that the same work unit's lineage already records, by a reference and digest that appear together on one recorded target. A claim naming no such artifact, or naming a reference and a digest that the lineage does not record together, has nothing to bind and MUST NOT produce a valid exception event, whatever its prose says.

A `restoration` exception SHALL precede every commit that touches a material path within its effect scope. Precedence SHALL be established from where the claim's own artifact entered the history, and MUST NOT be read from any field the claim carries about itself. A claim that declares its own position is a self-report, and the claim exists to replace exactly that: evidence is read at the head of the range, so a claim recorded afterwards could otherwise name an earlier commit and stand.

Where the claim's artifact is not touched within the range under admission, it was committed before that range began and therefore precedes every commit in it. Where it is touched, the commit that first touched it SHALL be an ancestor of every commit touching a material path in the claim's effect scope. Ancestry SHALL be established by reachability, not by position in an ordering: a range containing parallel branches admits an ordering that is not ancestry, and preceding one branch is not preceding another.

A `restoration` exception MUST NOT stand where the same work unit adds or amends a normative artifact within the claim's effect scope. Which paths are normative SHALL be declared by the committed policy that already classifies paths, so that no second authority decides it. This is the diagnostic of the rule itself: a change that needs a governing statement to move in the same act is not a restoration, and this holds for any requirement in scope rather than only the one the claim names.

This ordering condition is specific to the `restoration` class. A break-glass action is invoked during an emergency and a bootstrap path exists precisely where prior lineage does not, so requiring either to precede its own work would describe something that cannot happen; their timing is inherent to what they are.

Whether a diff moved the bound requirement's boundary rather than restoring it is a review question and MUST NOT be inferred from the exception event. A valid `restoration` exception establishes that the claim was made in the required form and at the required time; it does not establish that the claim is true.

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

#### Scenario: Restoration is claimed before the work
- **WHEN** a `restoration` exception names a requirement artifact the lineage records, and its own artifact entered the history before every material commit in its effect scope
- **THEN** the exception event is valid, and the work unit is classified by that exception rather than reported as ordinary valid work

#### Scenario: Restoration is claimed after the work
- **WHEN** the claim's own artifact entered the history only after a material path within its effect scope had already been committed
- **THEN** the exception is out-of-scope evidence, the work unit is non-admissible, and the ordering failure remains visible rather than being settled by the later record

#### Scenario: A claim states its own position
- **WHEN** an exception event carries a field asserting when or where the claim was made
- **THEN** that field does not establish precedence, and the claim is judged by where its artifact entered the history

#### Scenario: The claim precedes one branch but not another
- **WHEN** a range contains parallel branches touching material paths in the claim's effect scope, and the claim precedes one of them only
- **THEN** precedence is not established and the work unit is non-admissible

#### Scenario: Restoration names a requirement it cannot bind
- **WHEN** a claim names a requirement in prose, or names a reference and a digest the lineage does not record together
- **THEN** no valid exception event exists and the missing normal lineage remains visible

#### Scenario: A new normative statement is needed to justify the change
- **WHEN** the same work unit claims restoration and adds or amends any normative artifact within the claim's effect scope, whether or not it is the artifact the claim names
- **THEN** the claim has no unchanged prior requirement to rest on, no valid `restoration` exception exists, and the normal lineage for a material change remains required

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

