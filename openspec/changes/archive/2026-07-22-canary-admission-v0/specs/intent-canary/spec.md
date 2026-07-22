## ADDED Requirements

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

### Requirement: Pre-activation admission remains synthetic-only
**Intent IDs:** OUT-5, SIG-6, SIG-7

While manifest version 1 remains `frozen_not_activated`, the operator and evidence validation boundaries MUST reject every non-synthetic admission or assignment before append. Synthetic excluded and eligible decisions MAY exercise the complete admission and check-freeze lifecycle without authorizing real execution.

#### Scenario: Real candidate is presented before activation
- **WHEN** an operator or direct evidence writer attempts a non-synthetic admission or assignment
- **THEN** the operation returns the explicit not-activated failure, appends no event, and consumes no ordinal

#### Scenario: Synthetic lifecycle is rehearsed
- **WHEN** synthetic candidates are excluded or accepted and accepted work freezes checks before delivery
- **THEN** the harness records and verifies the complete sequence without changing the manifest activation state

## MODIFIED Requirements

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
