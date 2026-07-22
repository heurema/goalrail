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

### Requirement: Pre-activation admission remains synthetic-only
**Intent IDs:** OUT-5, SIG-6, SIG-7

While manifest version 1 remains `frozen_not_activated`, the operator and evidence validation boundaries MUST reject every non-synthetic admission or assignment before append. Synthetic excluded and eligible decisions MAY exercise the complete admission and check-freeze lifecycle without authorizing real execution.

#### Scenario: Real candidate is presented before activation
- **WHEN** an operator or direct evidence writer attempts a non-synthetic admission or assignment
- **THEN** the operation returns the explicit not-activated failure, appends no event, and consumes no ordinal

#### Scenario: Synthetic lifecycle is rehearsed
- **WHEN** synthetic candidates are excluded or accepted and accepted work freezes checks before delivery
- **THEN** the harness records and verifies the complete sequence without changing the manifest activation state

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

### Requirement: Outcome assessment is owner-recorded and independent of checks
**Intent IDs:** OUT-4, SIG-2, SIG-3

After delivery, the owner SHALL assess intent outcome as `match`, `partial`, or `miss`: `match` means current intent is met and non-goals are respected; `partial` means the result is useful but a material part of intent is unmet; `miss` means substantial rework is required or the result is unintended. The record SHALL separately capture whether a material intent correction occurred before delivery.

#### Scenario: Checks pass but intent is only partially met
- **WHEN** the recorded check set passes at its recorded source reference but the owner assesses `partial` or `miss`
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
