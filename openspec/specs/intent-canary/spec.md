# Intent Canary Specification

## Purpose

Define the frozen intent-first canary lifecycle, automatic execution lineage, append-only evidence, owner assessment, measurement rules, and exact pass and stop signals.

## Requirements

### Requirement: Canary assignments are fixed before outcomes are known
**Intent IDs:** OUT-4, SIG-2

The canary SHALL accept 15 sequential real Goalrail changes and assign variants using the repeating order `flow`, `flow`, `baseline`, yielding 10 flow changes and 5 baseline changes. Assignment MUST occur when a change enters the canary, MUST be recorded before execution outcome is known, and MUST NOT be changed based on difficulty or result.

#### Scenario: Eligible change enters the canary
- **WHEN** the next eligible real change is accepted
- **THEN** the workflow assigns the variant from its immutable ordinal in the fixed rotation

#### Scenario: Assigned change is abandoned
- **WHEN** an assigned change terminates without delivery
- **THEN** it remains in its original ordinal and counts toward the 15 assigned changes with an abandonment reason

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

### Requirement: Evidence is append-only and source-addressed
**Intent IDs:** OUT-4, SIG-1, SIG-3, SIG-8

Canary evidence SHALL identify the change, variant, intent version when applicable, execution lineage, recorded source reference, check set and result, terminal state, owner assessment, material corrections, overhead, and repeat opt-in. Later corrections MUST append a new event that references the superseded event rather than rewriting history.

#### Scenario: Assessment needs correction
- **WHEN** the owner corrects a previously recorded assessment
- **THEN** the workflow appends a correction event with actor, timestamp, reason, and superseded-event reference while preserving the original event

#### Scenario: Check set passes at a source reference
- **WHEN** all checks selected before assessment pass against the recorded source reference
- **THEN** the record marks the change `green` while preserving the independent intent outcome

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
