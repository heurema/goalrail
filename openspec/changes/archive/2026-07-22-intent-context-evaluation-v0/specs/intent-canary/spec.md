## ADDED Requirements

### Requirement: Manifest version 2 freezes the complete measured flow
**Intent IDs:** OUT-6, SIG-8

The canary SHALL preserve manifest version 1 as an immutable, never-activated historical contract and SHALL freeze the Context-Pack flow under manifest version 2 before any real assignment. Version 2 SHALL retain the fixed 15-position `flow`, `flow`, `baseline` rotation and existing pass and stop thresholds, use a distinct evidence stream, and remain `frozen_not_activated` until a separate explicit owner action.

#### Scenario: Version 2 becomes the current rehearsal target
- **WHEN** this change completes locally
- **THEN** manifest version 2 and its operator guidance exist without changing version 1 content or admitting a real candidate

#### Scenario: Evidence targets the wrong manifest version
- **WHEN** an event or operator action attempts to mix version 1 and version 2 rules or evidence streams
- **THEN** validation fails before append and neither history is rewritten

#### Scenario: Real work is presented before activation
- **WHEN** a non-synthetic candidate targets manifest version 2
- **THEN** the operation returns the explicit not-activated failure and consumes no ordinal

### Requirement: Langfuse telemetry is reconciled through verified lineage
**Intent IDs:** OUT-4, SIG-5, SIG-6

For either variant, the canary SHALL query a read-only telemetry source using only verified root session identity and SHALL validate that every returned trace belongs to that session. It SHALL deduplicate trace identities, calculate each root-turn envelope from observation timestamps, and emit provider-neutral trace references and timing intervals. It MUST NOT infer a join from text, tags, ordering, or repository-wide mutable state.

#### Scenario: Session has multiple valid root-turn traces
- **WHEN** the telemetry source returns observations for distinct traces under the verified root session
- **THEN** reconciliation emits each trace reference once with the envelope from its earliest start to latest end

#### Scenario: Telemetry contains duplicates or nested observations
- **WHEN** multiple observations or repeated result pages refer to the same trace
- **THEN** reconciliation deduplicates by trace identity and does not sum nested observation durations separately

#### Scenario: Telemetry conflicts with lineage
- **WHEN** a returned trace names another session or contains a malformed or reversed interval
- **THEN** reconciliation fails without attaching the trace or producing an overhead value

#### Scenario: Telemetry is missing or delayed
- **WHEN** the bounded read returns no usable trace envelope
- **THEN** the canary retains the verified session lookup, records machine telemetry as unavailable, does not impute zero, and applies the manifest's completion-readiness rule

### Requirement: Flow overhead preserves machine and owner components
**Intent IDs:** OUT-5, SIG-6, SIG-7

For a flow assignment, the canary SHALL retain distinct reconciled agent-turn intervals and one explicit owner-review interval and SHALL derive `flow_overhead_minutes` from their exact total. Baseline telemetry MAY be reconciled for lineage and operational comparison but MUST NOT receive a flow-overhead total. Missing required flow evidence SHALL leave the total unavailable.

#### Scenario: Complete flow timing evidence exists
- **WHEN** distinct flow-only trace envelopes and the required owner-review interval are valid
- **THEN** the measurement exposes agent seconds, owner-review seconds, trace count, and the unrounded combined minutes

#### Scenario: Owner-review timing is missing
- **WHEN** owner review is required but has no complete interval
- **THEN** machine seconds remain inspectable while the combined flow-overhead total remains unavailable

#### Scenario: Baseline traces are reconciled
- **WHEN** a baseline assignment has verified Langfuse trace envelopes
- **THEN** the evidence may retain the trace references but rejects `flow_overhead_minutes` and flow-only owner-review cost

### Requirement: Synthetic pair proves measurement before activation
**Intent IDs:** OUT-3, OUT-4, OUT-5, OUT-6, SIG-9

Before any activation decision, one synthetic flow case and one synthetic baseline case SHALL traverse manifest version 2 admission, verified lineage, telemetry reconciliation, frozen checks, terminal evidence, and reporting. The flow case SHALL also traverse sufficient context, confirmed intent, item-level owner assessment, and complete overhead measurement.

#### Scenario: Synthetic flow rehearsal completes
- **WHEN** the flow fixture reaches delivery and assessment
- **THEN** its append-only projection exposes Context Pack identity, intent item judgments, trace references, separate overhead components, aggregate outcome, and report contribution

#### Scenario: Synthetic baseline rehearsal completes
- **WHEN** the baseline fixture reaches delivery and assessment
- **THEN** its projection exposes verified trace references and item-level assessment without a Context Pack gate or flow-overhead total

#### Scenario: Rehearsal evidence is incomplete
- **WHEN** either case cannot calculate every metric required by its variant
- **THEN** the rehearsal fails and manifest version 2 remains not ready for an activation decision

## MODIFIED Requirements

### Requirement: Outcome assessment is owner-recorded and independent of checks
**Intent IDs:** OUT-3, SIG-4

After delivery and before rework, the owner SHALL judge every stable desired outcome, non-goal, and observable success signal from the applicable confirmed Intent Snapshot. Desired outcomes use `achieved`, `partial`, or `missed`; non-goals use `preserved` or `violated`; success signals use `observed` or `missing`. The assessment SHALL also record the aggregate intent outcome as `match`, `partial`, or `miss`, and validation MUST reject an aggregate that is inconsistent with the item judgments. The record SHALL separately capture whether a material intent correction occurred before delivery, and checks SHALL remain independent evidence.

#### Scenario: All intent items are satisfied
- **WHEN** every desired outcome is `achieved`, every non-goal is `preserved`, and every success signal is `observed`
- **THEN** the only valid aggregate owner outcome is `match`

#### Scenario: A bounded material part is unmet
- **WHEN** no non-goal is violated, no desired outcome is `missed`, and at least one desired outcome is `partial` or success signal is `missing`
- **THEN** the only valid aggregate owner outcome is `partial`

#### Scenario: Result substantially contradicts intent
- **WHEN** any desired outcome is `missed` or any non-goal is `violated`
- **THEN** the only valid aggregate owner outcome is `miss`

#### Scenario: Item judgment is absent or duplicated
- **WHEN** an assessment omits a stable intent item, includes it more than once, or judges an ID outside the confirmed snapshot
- **THEN** validation fails before append and no aggregate is inferred for the owner

#### Scenario: Checks pass but intent is not fully met
- **WHEN** the recorded check set passes at its recorded source reference but the valid aggregate is `partial` or `miss`
- **THEN** the change is recorded as `wrong-but-green` without changing either the check result or owner assessment

#### Scenario: Material misunderstanding is corrected before delivery
- **WHEN** an outcome, non-goal, or success-signal meaning is corrected before delivery and the correction changes the work
- **THEN** the workflow appends a contemporaneous material-correction event independent of the final item judgments and aggregate assessment

#### Scenario: Only wording or a typo changes
- **WHEN** a correction does not change the meaning of an outcome, non-goal, or success signal
- **THEN** it is excluded from the material-correction count
