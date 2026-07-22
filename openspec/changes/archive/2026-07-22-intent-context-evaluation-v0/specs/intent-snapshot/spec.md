## ADDED Requirements

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
