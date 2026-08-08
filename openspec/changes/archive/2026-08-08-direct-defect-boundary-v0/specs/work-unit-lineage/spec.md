## MODIFIED Requirements

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
