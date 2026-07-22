## ADDED Requirements

### Requirement: Intent evidence remains distinct from interpretation
**Intent IDs:** OUT-1

The intent workflow SHALL preserve owner statements and verified repository facts as source evidence separately from agent-produced interpretations. Unsupported or conflicting interpretations MUST be recorded as ambiguities rather than silently promoted to intent.

#### Scenario: Agent extracts a candidate interpretation
- **WHEN** an agent derives desired outcomes, non-goals, or success signals from source evidence
- **THEN** the workflow stores the derived fields as candidate interpretation and retains references to the source evidence without rewriting that evidence

#### Scenario: Evidence does not resolve a material meaning
- **WHEN** two plausible interpretations would produce materially different changes
- **THEN** the workflow records the ambiguity and blocks confirmation until the owner resolves it

### Requirement: Intent uses stable, bounded semantic groups
**Intent IDs:** OUT-1, OUT-5

An Intent Snapshot SHALL express semantic intent through exactly three groups: desired outcomes, non-goals, and observable success signals. Every item MUST have a stable ID within the intent lineage.

#### Scenario: Candidate snapshot is created
- **WHEN** the workflow creates a candidate Intent Snapshot
- **THEN** each semantic item is classified into one of the three groups and receives a stable ID

#### Scenario: Provenance metadata is recorded
- **WHEN** the workflow records sources, ambiguities, confirmation, or run references
- **THEN** it treats those fields as provenance or lifecycle metadata rather than additional semantic intent groups

### Requirement: Owner confirmation gates proposal compilation
**Intent IDs:** OUT-2

The workflow SHALL prevent proposal compilation from a candidate Intent Snapshot. Confirmation MUST record the owner, timestamp, and active verification action; absence of an objection alone SHALL NOT count as confirmation.

#### Scenario: Candidate intent has not been confirmed
- **WHEN** proposal compilation is requested for an Intent Snapshot whose status is `candidate`
- **THEN** the workflow refuses compilation and reports that owner confirmation is required

#### Scenario: Owner actively verifies intent
- **WHEN** the owner reviews the desired outcomes, non-goals, and success signals and explicitly confirms them
- **THEN** the workflow records the confirmation receipt and allows the confirmed version to compile into a proposal

### Requirement: Proposal preserves confirmed intent coverage
**Intent IDs:** OUT-2, OUT-5

A compiled proposal SHALL trace every proposed change to at least one confirmed intent item and SHALL preserve all applicable confirmed non-goals. The compiler MUST NOT add new intent.

#### Scenario: Proposal contains an untraced change
- **WHEN** a proposed change has no confirmed intent ID
- **THEN** validation fails and identifies the untraced change

#### Scenario: Proposal would violate a non-goal
- **WHEN** compilation introduces work excluded by a confirmed non-goal
- **THEN** the workflow blocks the proposal and requires a new owner-confirmed intent version before proceeding

### Requirement: Material amendments create a new version
**Intent IDs:** OUT-1, OUT-2, SIG-8

A material change to a desired outcome, non-goal, or observable success signal SHALL create a new Intent Snapshot version and preserve the prior version. A wording-only edit that preserves meaning MAY retain the current version. Corrections to recorded history MUST be appended as new events rather than overwriting prior evidence.

#### Scenario: Owner changes the desired result
- **WHEN** the owner changes the meaning of a desired outcome after confirmation
- **THEN** the workflow creates a new candidate version linked to the confirmed predecessor and requires confirmation again

#### Scenario: Owner fixes a typo
- **WHEN** the owner changes wording without changing meaning
- **THEN** the workflow may retain the version and records the wording edit without replacing historical evidence

### Requirement: Intent does not grant effect authority
**Intent IDs:** OUT-2, OUT-5

Confirmation of an Intent Snapshot SHALL express what the owner wants but SHALL NOT itself authorize tools, credentials, writes, deployments, publications, or other effects.

#### Scenario: Confirmed intent requests an external effect
- **WHEN** a confirmed Intent Snapshot includes an outcome that would require an external effect
- **THEN** downstream execution still requires a separate applicable task grant and enforcement check before that effect
