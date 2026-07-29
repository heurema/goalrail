## ADDED Requirements

### Requirement: An answering intent version records the escalation it resolves
**Intent IDs:** OUT-5, SIG-5

An Intent Snapshot version created in response to a blocked run MAY record the
escalation it answers as lifecycle provenance: the run identifier and the digest
of the retained escalation bytes, together with one disposition of `answered`,
`spurious`, or `withdrawn`. The record is optional, is provenance rather than a
fourth semantic intent group, and MUST NOT be added to a previously confirmed
version — answering a blocked run creates a new version.

When the record is present, validation SHALL reject a malformed run identifier,
a malformed digest, an unknown disposition, or a duplicate record within one
version. An absent record MUST NOT block confirmation, and its absence MUST NOT
be inferred as any particular disposition.

Adding, removing, or changing the record is a material change. A wording-only
amendment SHALL preserve it exactly, so a version that is already confirmed
cannot acquire or alter a disposition without returning to candidate and being
confirmed again.

The record SHALL NOT grant effect authority. It expresses which question a
version answers; it does not authorize a run, a launch, or any external effect.

#### Scenario: Owner answers a blocked run
- **WHEN** a new intent version is created to resolve a blocked run
- **THEN** it may record that run identifier, the retained escalation digest, and the disposition `answered`, while the prior confirmed version remains unmodified

#### Scenario: The question should not have been asked
- **WHEN** the owner judges a blocked run's question unnecessary or already answered
- **THEN** the answering version records the disposition `spurious` or `withdrawn`, making a low-value escalation visible and countable rather than invisible

#### Scenario: The record is malformed
- **WHEN** a recorded escalation reference carries a malformed run identifier, a malformed digest, an unknown disposition, or a duplicate entry
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
