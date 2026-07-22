# Intent Canary Specification

## Purpose

Define the frozen intent-first canary lifecycle, automatic execution lineage, append-only evidence, owner assessment, measurement rules, and exact pass and stop signals.

## Requirements

### Requirement: Every candidate has one append-only admission decision
**Intent IDs:** OUT-1, OUT-2, OUT-4, SIG-1, SIG-2, SIG-5

Before variant reveal, the canary SHALL record exactly one admission decision for each considered candidate under its existing immutable `change_id`. The decision MUST be `eligible` or `excluded` and MUST include actor, timestamp, bounded source reference, and categorical reason. A duplicate or conflicting decision for the same `change_id` MUST fail without changing evidence.

#### Scenario: Candidate is excluded
- **WHEN** an operator records an `excluded` decision before assignment
- **THEN** the append-only evidence contains the exclusion and its reason without a run, ordinal, or variant, and the next assignment ordinal remains unchanged

#### Scenario: Candidate is eligible
- **WHEN** an operator records an `eligible` decision
- **THEN** the same serialized evidence transaction records the next immutable ordinal, its manifest-defined variant, and the new run identity

#### Scenario: Decision is replayed or contradicted
- **WHEN** the same `change_id` receives a second admission decision, whether identical or different
- **THEN** the operation fails closed and the evidence chain remains byte-for-byte unchanged

### Requirement: Canary assignments are fixed before outcomes are known
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2

The canary SHALL accept 15 sequential real Goalrail changes after a separate activation decision and assign variants using the repeating order `flow`, `flow`, `baseline`, yielding 10 flow changes and 5 baseline changes. Eligibility MUST be decided without revealing the next variant. An eligible admission and assignment MUST be one serialized append-only transaction before execution outcome is known. An excluded admission MUST contain no ordinal or variant and MUST NOT advance the sequence. Assignment MUST NOT change based on difficulty or result.

#### Scenario: Eligible change enters the canary
- **WHEN** the next eligible real change is accepted after activation
- **THEN** the admission event atomically assigns the variant from its immutable ordinal in the fixed rotation

#### Scenario: Candidate is excluded before assignment
- **WHEN** a candidate fails the frozen eligibility rule
- **THEN** its reason is retained without consuming an ordinal or revealing the next variant

#### Scenario: Assigned change is abandoned
- **WHEN** an assigned change terminates without delivery
- **THEN** it remains in its original ordinal and counts toward the 15 assigned changes with an abandonment reason

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

### Requirement: Pre-activation admission remains synthetic-only
**Intent IDs:** OUT-5, SIG-6, SIG-7

While manifest version 1 remains `frozen_not_activated`, the operator and evidence validation boundaries MUST reject every non-synthetic admission or assignment before append. Synthetic excluded and eligible decisions MAY exercise the complete admission and check-freeze lifecycle without authorizing real execution.

#### Scenario: Real candidate is presented before activation
- **WHEN** an operator or direct evidence writer attempts a non-synthetic admission or assignment
- **THEN** the operation returns the explicit not-activated failure, appends no event, and consumes no ordinal

#### Scenario: Synthetic lifecycle is rehearsed
- **WHEN** synthetic candidates are excluded or accepted and accepted work freezes checks before delivery
- **THEN** the harness records and verifies the complete sequence without changing the manifest activation state

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

### Requirement: Runtime-owned canary artifacts survive development lifecycle transitions
**Intent IDs:** OUT-1, SIG-1, SIG-2, SIG-3

The canary SHALL keep its current frozen manifest and default append-only evidence location under a stable provider-neutral runtime path. Runtime code MUST NOT discover or address those artifacts through an active or dated archive path owned by a replaceable development provider.

#### Scenario: Completed OpenSpec change is archived
- **WHEN** the OpenSpec change that introduced the canary is moved from the active change set into a dated archive
- **THEN** the command continues to resolve the same current frozen manifest and default evidence location without reading the active or archived change directory

#### Scenario: Repository has no OpenSpec directory
- **WHEN** the canary command uses its repository-local default store in a repository root with no `openspec` directory
- **THEN** it writes the append-only evidence chain under the stable canary runtime path and does not recreate a development-provider path

### Requirement: Current operator guidance matches executable defaults
**Intent IDs:** OUT-2, OUT-3, SIG-1, SIG-5

Current operator guidance SHALL live outside historical planning archives and SHALL name the exact default evidence path used by the canary command. Archived guidance MAY preserve historical wording but MUST NOT be presented as the current operator surface.

#### Scenario: Operator follows current guidance without a store override
- **WHEN** an operator runs the documented command without `--store`
- **THEN** the documented evidence path and the executable default resolve to `canary/intent-canary-v0/events.jsonl`

#### Scenario: Current guidance drifts from the command
- **WHEN** a code or documentation change makes the two default paths differ
- **THEN** deterministic regression validation fails before the change is accepted

### Requirement: Runtime-boundary maintenance does not activate the canary
**Intent IDs:** OUT-1, SIG-4

A development-provider archive or runtime-artifact relocation SHALL preserve the frozen canary rules and activation state. It MUST NOT create an evidence event, consume an assignment ordinal, or bypass the separate owner instruction required for real activation.

#### Scenario: Runtime artifacts are relocated before activation
- **WHEN** the frozen manifest and current guidance are moved to their stable runtime path
- **THEN** the manifest remains `frozen_not_activated`, no repository evidence chain is created, and a real assignment remains rejected

### Requirement: Each change has an explicit lifecycle and stable identity
**Intent IDs:** OUT-4

A canary `change` SHALL mean one intended modification with a stable `change_id`, from assignment through a terminal owner assessment. Its terminal delivery state MUST be either `delivered` or `abandoned`: `delivered` means the result was handed to the owner or review with its recorded checks and evidence, without requiring merge or deploy; `abandoned` means work stopped and a reason was recorded.

#### Scenario: Work is handed to review without merge
- **WHEN** the result and recorded verification evidence are handed to the owner or reviewer
- **THEN** the change may be marked `delivered` even if it is not merged or deployed

#### Scenario: Work stops before handoff
- **WHEN** execution stops without a reviewable result
- **THEN** the change is marked `abandoned` with a reason and is not silently removed from the canary

### Requirement: Execution lineage is captured automatically
**Intent IDs:** OUT-3, SIG-1, SIG-6

Every canary change SHALL automatically record a verified lineage `change_id → run_id → root Codex session_id`. The root identity MUST come from a provider-authoritative lifecycle hook or launch receipt. In v0, one change SHALL use one root Codex session; resumed work retains that session identity, child agents inherit the root association, and concurrent changes MUST NOT share mutable current-change state.

#### Scenario: Root Codex session starts for an assigned change
- **WHEN** execution begins with valid change context
- **THEN** the adapter records the change, run, and session join without asking a person to copy an identifier

#### Scenario: Goalrail launches a Codex App task
- **WHEN** Goalrail creates an App task for an assigned change and the provider returns its immutable `threadId`
- **THEN** the adapter records that launch receipt as the root session identity for the existing immutable change and run context

#### Scenario: App task lacks a Goalrail launch receipt
- **WHEN** an App task was created manually or its provider-authoritative launch receipt is unavailable
- **THEN** the adapter records `UNLINKED`, does not infer identity from task text or transcripts, and excludes the task from v0 canary execution

#### Scenario: Root session resumes or starts a child agent
- **WHEN** the root session resumes or delegates to a child agent
- **THEN** emitted evidence remains associated with the original root session and canary change

#### Scenario: Two changes execute concurrently
- **WHEN** separate assigned changes run at the same time
- **THEN** each session resolves to its own immutable change and run context

#### Scenario: Correlation context is absent or conflicting
- **WHEN** the adapter cannot verify exactly one change/run/session join
- **THEN** it records `UNLINKED`, performs at most one bounded resolution attempt, and never guesses or writes a potentially wrong join

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
- **THEN** the workflow appends a contemporaneous material-correction event independent of the final `match`, `partial`, or `miss` assessment

#### Scenario: Only wording or a typo changes
- **WHEN** a correction does not change the meaning of an outcome, non-goal, or success signal
- **THEN** it is excluded from the material-correction count

### Requirement: Check selection is frozen before terminal verification
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4, SIG-5

An assigned change SHALL append a non-empty set of unique stable check references after scope is understood and before its first terminal verification result or owner assessment. Delivery MUST reference exactly the effective frozen set. The initial freeze MUST NOT be rewritten; a legitimate pre-result correction MUST append a reasoned event that supersedes the latest check-set event and MUST NOT remove a previously selected check. No check-set correction is permitted after terminal evidence.

#### Scenario: Delivery has no prior frozen check set
- **WHEN** an assigned change attempts to record delivery before freezing checks
- **THEN** the operation fails without appending terminal evidence

#### Scenario: Delivery references a different check set
- **WHEN** terminal verification evidence omits, adds, or replaces a reference relative to the effective frozen set
- **THEN** the operation fails without changing the evidence chain

#### Scenario: Check selection is corrected before a result
- **WHEN** a reasoned correction supersedes the latest check-set event before terminal evidence and retains every previously selected reference
- **THEN** the correction appends successfully and becomes the effective frozen set while preserving the original event

#### Scenario: Check selection is reduced or changed after a result
- **WHEN** a correction removes a selected reference or occurs after terminal evidence
- **THEN** the operation fails closed and earlier evidence remains unchanged

### Requirement: Evidence is append-only and source-addressed
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, SIG-1, SIG-2, SIG-3, SIG-4, SIG-5

Canary evidence SHALL identify admission decisions and reasons, the assigned change and variant when eligible, intent version when applicable, execution lineage, recorded source references, frozen check set and terminal result, terminal state, owner assessment, material corrections, overhead, and repeat opt-in. Inspect SHALL expose a candidate's admission, effective frozen checks, and later lifecycle state. Report SHALL count exclusions and their categorical reasons without adding them to assigned-change denominators. Later corrections MUST append a new event that references the superseded event rather than rewriting history.

#### Scenario: Assessment needs correction
- **WHEN** the owner corrects a previously recorded assessment
- **THEN** the workflow appends a correction event with actor, timestamp, reason, and superseded-event reference while preserving the original event

#### Scenario: Check set passes at a source reference
- **WHEN** all checks in the effective frozen set pass against the recorded source reference
- **THEN** the record marks the change `green` while preserving the independent intent outcome and original check-selection evidence

#### Scenario: Exclusions are reported
- **WHEN** one or more candidates are excluded before assignment
- **THEN** the report exposes their count and categorical reason counts while assigned, delivered, and outcome-rate denominators remain based only on assigned changes

### Requirement: Canary measurement is declared before execution
**Intent IDs:** OUT-4, SIG-2, SIG-3, SIG-4

Before the first assignment, the canary manifest SHALL define eligible-change rules, the fixed assignment rotation, the check-recording rule, the overhead measurement method, owner assessment timing, and formulas for all reported rates. The measurement rules MUST remain unchanged for the 15 assigned changes unless the canary is stopped and restarted as a new version.

#### Scenario: First change is ready to enter an incomplete manifest
- **WHEN** any required measurement rule is absent
- **THEN** the workflow blocks assignment and identifies the missing rule

#### Scenario: Operator wants to change a metric mid-canary
- **WHEN** a proposed measurement change would alter interpretation of existing observations
- **THEN** the current canary stops and any replacement begins under a new version without rewriting prior evidence

### Requirement: Pass and stop signals are evaluated exactly
**Intent IDs:** SIG-1, SIG-2, SIG-3, SIG-4, SIG-5, SIG-6, SIG-7, SIG-8

At 15 assigned changes, the canary SHALL report lineage integrity, delivered-change non-match rates by variant, contemporaneous prevented misunderstandings, flow overhead, repeat opt-in, abandonments, and evidence-integrity violations. A non-match rate SHALL equal `(partial + miss) / delivered assessed changes` for that variant, and comparisons SHALL use rates rather than raw counts.

The current flow implementation passes only if at least 13 of 15 lineage joins are verified with zero wrong joins, the flow non-match rate is directionally lower than baseline, at least one material misunderstanding was contemporaneously prevented before delivery, median flow overhead is at most 30 minutes, and at least two-thirds of flow changes have `repeat_optin=yes`.

The workflow SHALL stop or reshape immediately if median flow overhead exceeds 30 minutes, at least 3 flow changes are abandoned because of the process, any wrong join occurs, at least 2 links remain unresolved after the bounded resolution attempt, or evidence is retroactively rewritten. At completion it SHALL also stop or reshape if there were no material preventions, the flow non-match rate was not lower than baseline, and flow cost more.

#### Scenario: All pass signals hold at completion
- **WHEN** 15 changes have terminal records and every stated pass condition holds
- **THEN** the report marks this canary version `PASS` and preserves the evidence for owner review

#### Scenario: Hard stop signal occurs before completion
- **WHEN** a wrong join, evidence rewrite, excessive overhead, process-caused abandonment threshold, or unresolved-link threshold is observed
- **THEN** the workflow marks this canary version `STOP`, accepts no further assignments, and preserves all existing records

#### Scenario: Results show no useful movement
- **WHEN** the completed canary has no material prevention, flow non-match is not lower than baseline, and measured flow cost is higher
- **THEN** the report marks the current flow `RESHAPE` without claiming that intent itself is unimportant
