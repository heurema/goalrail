# Intent Snapshot Specification

## Purpose

Define how Goalrail separates source evidence from interpretation, confirms bounded semantic intent, preserves provenance across versions, and prevents intent confirmation from granting effect authority.
## Requirements
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

### Requirement: Flow intent confirmation requires completed context
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-3

A flow Intent Snapshot SHALL reference one valid Context Pack completed before the confirmation timestamp. Confirmation MUST fail when the pack is absent, still collecting, completed after confirmation, or retains an unresolved material unknown.

#### Scenario: Context is sufficient before owner review
- **WHEN** a valid Context Pack completed as `sufficient` before the owner confirms the candidate snapshot
- **THEN** confirmation validation accepts the context gate while retaining the owner's separate active verification requirement

#### Scenario: Context is missing or late
- **WHEN** a flow snapshot has no pack or the pack completion time follows the claimed confirmation time
- **THEN** confirmation validation fails and proposal compilation remains blocked

#### Scenario: Material unknown remains
- **WHEN** the referenced pack completed as `material_unknown` or `budget_exhausted` with an unresolved material unknown
- **THEN** the snapshot remains candidate until the owner resolves the meaning and a sufficient Context Pack version is recorded

### Requirement: Intent items retain context provenance without absorbing claims
**Intent IDs:** OUT-1, SIG-1, SIG-2

Every semantic intent item derived from context evidence SHALL reference stable Context Pack item IDs in addition to applicable owner or repository source evidence. The workflow MUST reject missing, duplicate, or cross-pack context references and MUST preserve those references across wording-only amendments.

#### Scenario: Candidate item uses context evidence
- **WHEN** context evidence contributes to a desired outcome, non-goal, or observable success signal
- **THEN** the semantic item references the exact context item while its confirmed wording remains an owner-verified interpretation

#### Scenario: Wording-only amendment changes context provenance
- **WHEN** a wording-only amendment adds, removes, or repoints a semantic item's context reference
- **THEN** validation rejects the amendment and requires a material intent version when the meaning or evidence basis changed

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

### Requirement: An answering intent version records the escalation it resolves
**Intent IDs:** OUT-5

An Intent Snapshot version created in response to a blocked run MAY record the
escalation it answers as lifecycle provenance: the canonical identifier of what
is being resolved — a blocked run or a background question record — and the
digest of the retained escalation bytes, together with one disposition of
`answered`, `spurious`, or `withdrawn`. The record is optional, is provenance
rather than a fourth semantic intent group, and MUST NOT be added to a
previously confirmed version — answering creates a new version.

When the record is present, validation SHALL reject a malformed resolved
reference, a malformed digest, an unknown disposition, or a duplicate record
within one version. An absent record MUST NOT block confirmation, and its
absence MUST NOT be inferred as any particular disposition.

Adding, removing, or changing the record is a material change. A wording-only
amendment SHALL preserve it exactly, so a version that is already confirmed
cannot acquire or alter a disposition without returning to candidate and being
confirmed again.

The record SHALL NOT grant effect authority. It expresses which question a
version answers; it does not authorize a run, a launch, or any external effect.

#### Scenario: Owner answers a blocked run
- **WHEN** a new intent version is created to resolve a blocked run
- **THEN** it may record that run identifier, the retained escalation digest, and the disposition `answered`, while the prior confirmed version remains unmodified

#### Scenario: Owner answers a background question
- **WHEN** a new intent version is created to resolve a question recorded by a background session
- **THEN** it records the question record's own identifier through the same field, because a background session has no run to name

#### Scenario: The question should not have been asked
- **WHEN** the owner judges a recorded question unnecessary or already answered
- **THEN** the answering version records the disposition `spurious` or `withdrawn`, making a low-value escalation visible and countable rather than invisible

#### Scenario: The record is malformed
- **WHEN** a recorded escalation reference carries a malformed resolved reference, a malformed digest, an unknown disposition, or a duplicate entry
- **THEN** validation rejects the version and does not silently drop the record

#### Scenario: A wording-only edit would acquire or alter the record
- **WHEN** a wording-only amendment adds an escalation record to a confirmed version, removes one, or changes its disposition
- **THEN** validation rejects the amendment and requires a material next version with fresh owner confirmation

#### Scenario: A version answers nothing
- **WHEN** an intent version records no escalation reference
- **THEN** confirmation proceeds unaffected and no disposition is inferred

#### Scenario: The record is read as authority
- **WHEN** an answering version carries a resolved escalation reference
- **THEN** launching any run from it still requires the separate applicable gates, because the record grants no effect authority

