## ADDED Requirements

### Requirement: The escalation payload has a provider-neutral published shape
**Intent IDs:** OUT-6, SIG-1

The escalation payload SHALL have one published provider-neutral shape,
`goalrail.escalation/v0`, describing what a run writes to the reserved path: a
block class, the sources the block was derived from, exactly one question, and —
where the block class admits them — an optional separating example and optional
per-source interpretations. The shape MUST accommodate block classes that have
neither two sources nor a separating example, such as a missing dependency or a
broken environment, without requiring absent fields.

The shape SHALL name no provider, model, harness, or vendor, and SHALL depend on
no Goalrail runtime type, so a payload remains valid evidence independently of
the tool that produced it.

#### Scenario: A conflicting-requirements block is expressed
- **WHEN** a run cannot proceed because two requirements in its own work item conflict
- **THEN** the payload names the block class, the conflicting sources, one question, and may carry the example that separates the readings

#### Scenario: A block class without two sources is expressed
- **WHEN** the block is a missing dependency or a broken environment rather than a conflict
- **THEN** the payload remains valid without a separating example or per-source interpretations

#### Scenario: The payload names a provider
- **WHEN** a payload embeds a provider, model, harness, or vendor identifier as part of its shape
- **THEN** the shape is violated, because the format is provider-neutral

### Requirement: Escalation payload semantics are validated outside Goalrail
**Intent IDs:** OUT-6, SIG-1

Semantic validation of the payload SHALL be performed by downstream tooling that
consumes the retained bytes — for example a kata oracle or a review surface.
Goalrail MUST NOT parse, interpret, execute, or grade the payload, consistent
with Goalrail never running checks; its responsibility ends at hygiene and
append-only retention.

A payload that fails downstream semantic validation SHALL NOT retroactively
change the recorded terminal status of the run that produced it. The receipt
records what was asked; the judgement of whether it was a good question is made
separately and recorded separately.

#### Scenario: Downstream tooling validates a retained payload
- **WHEN** a consumer reads the retained escalation bytes for a blocked run
- **THEN** it validates them against the published shape without depending on Goalrail to have done so

#### Scenario: Goalrail encounters a semantically empty payload
- **WHEN** the retained bytes pass hygiene but say nothing useful
- **THEN** Goalrail still records the outcome it observed, and the judgement is left to the downstream consumer and the owner's disposition on the answering intent version

#### Scenario: A payload is judged invalid after the run
- **WHEN** downstream validation rejects a retained payload
- **THEN** the recorded terminal status is unchanged and the judgement is recorded separately from the receipt
