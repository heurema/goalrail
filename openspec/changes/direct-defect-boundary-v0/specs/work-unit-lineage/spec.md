## MODIFIED Requirements

### Requirement: Exceptions are first-class lineage evidence, never normal validity
**Intent IDs:** OUT-8, SIG-8, SIG-9, OUT-1, OUT-2, OUT-4, SIG-1, SIG-2, SIG-3

Any standing exemption, emergency break-glass action, bounded bootstrap/migration path, or claimed restoration of an existing requirement SHALL be represented by an immutable exception event bound to one project policy identity and work unit. It SHALL include a categorical class, actor, reason reference, exact path and effect scope, start, expiry or closure condition, and required owner decision. Free-form justification alone MUST NOT widen scope or override committed policy.

A `restoration` exception is the claim that a change alters no requirement but returns behaviour to one already recorded. It SHALL additionally bind a digest-addressable requirement artifact — a retained confirmed Intent Snapshot, or a committed specification file at a recorded revision — and its effect scope SHALL be the paths over which that claim is made. A claim naming no such artifact has nothing to bind and MUST NOT produce a valid exception event, whatever its prose says.

A `restoration` exception SHALL be anchored in an ancestor of the first commit that touches a material path within its effect scope. An observation timestamp MUST NOT substitute for that ancestry. Recording the claim after the work it excuses makes it out-of-scope evidence rather than a late repair, because the claim exists to constrain what an author decided before implementing, and a claim written afterwards is a description of the diff instead of a condition on it.

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
- **WHEN** a `restoration` exception binds a digest-addressable requirement artifact and is anchored in an ancestor of the first material commit in its effect scope
- **THEN** the exception event is valid, and the work unit is classified by that exception rather than reported as ordinary valid work

#### Scenario: Restoration is claimed after the work
- **WHEN** the claim is recorded only after a material path within its effect scope has already been committed
- **THEN** the exception is out-of-scope evidence, the work unit is non-admissible, and the ordering failure remains visible rather than being settled by the later record

#### Scenario: Restoration names a requirement it cannot bind
- **WHEN** a claim names a requirement in prose but binds no retained snapshot digest and no committed specification file at a recorded revision
- **THEN** no valid exception event exists and the missing normal lineage remains visible

#### Scenario: A new normative statement is needed to justify the change
- **WHEN** the same work unit both claims restoration and amends or adds a requirement within its effect scope
- **THEN** the claim has no unchanged prior requirement to bind, no valid `restoration` exception exists, and the normal lineage for a material change remains required
