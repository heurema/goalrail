## MODIFIED Requirements

### Requirement: Session stop retains the question and binds it to intent
**Intent IDs:** OUT-2, OUT-4, SIG-4, SIG-5

At session stop in a connected directory, when the reserved path holds a
question, the hook SHALL retain its exact bytes append-only in the state root
outside the repository, applying the existing escalation hygiene. The retained
record SHALL carry the question digest and the session reference, and SHALL have
an identity of its own — one record per occurrence, so two sessions asking a
byte-identical question keep separate, individually answerable records. A
question that cannot be read still leaves a persisted record with an explicit
reason: the session tried to escalate, and the attempt must leave evidence.

The record SHALL be bound to the directory's single active confirmed intent —
its ID, version, and digest — only after active-intent discovery validates the
artifact form required by that change. A pair-backed change SHALL consume its
Context Pack and Intent Snapshot through the selected artifact contract or
pinned legacy mode. An existing lifecycle in which Context is not required and
no pair contract is declared SHALL retain its intent-only validation path. When
no single active confirmed intent is identifiable, the record SHALL be retained
unbound with an explicit reason naming why, and MUST NOT be bound by guessing.

If an active change contains an Intent artifact that explicitly claims
`confirmed` status but its required pair cannot select or satisfy the artifact
contract, discovery SHALL retain the shared structured artifact diagnostic as
binding evidence. That evidence SHALL name the invalid change while keeping
artifact content bounded and redacted. The question record MUST remain unbound,
even if another active intent would otherwise be the single valid confirmed
candidate; discovery MUST NOT hide the invalid confirmed claimant by skipping
it and binding the other one.

An absent, incomplete, or candidate work-in-progress artifact that does not
claim confirmed status SHALL NOT be promoted into invalid-confirmed evidence.
All discovery and diagnostic retention remains background work: it MUST NOT
surface into, block, break, or delay the user's session.

A background session produces a question record, not a run outcome: no run ID,
launch claim, terminal receipt, or `blocked` status is minted.

The answer to a recorded question is a new intent version carrying the existing
resolves-and-disposition record, citing the question record's own identifier,
and the next session works from that version. No second answering mechanism,
dialogue, acknowledgement, or resume exists.

#### Scenario: Two sessions ask the identical question
- **WHEN** two sessions produce byte-identical questions
- **THEN** the payload is retained once but each occurrence keeps its own record and identity, and neither collides with the other

#### Scenario: The question cannot be read
- **WHEN** the reserved path holds something oversized, non-regular, or otherwise unreadable at session stop
- **THEN** a record with an explicit reason is still persisted, because the escalation attempt itself is evidence

#### Scenario: Question is retained and bound
- **WHEN** a session ends with a question and the directory has exactly one active confirmed Context and Intent pair that satisfies its selected artifact contract
- **THEN** the exact bytes are retained append-only outside the repository and the record names that intent's ID, version, and digest

#### Scenario: A confirmed claimant is contract-invalid
- **WHEN** a session ends with a question and an active change's Intent artifact claims confirmed status but its required pair fails artifact contract selection or conformance
- **THEN** the question is retained unbound and its binding evidence carries the shared structured diagnostic naming that change

#### Scenario: A valid intent coexists with an invalid confirmed claimant
- **WHEN** one active pair is a valid confirmed intent and another active change claims confirmed status but fails artifact conformance
- **THEN** the question remains unbound with diagnostic evidence naming the invalid change instead of silently binding the valid intent

#### Scenario: Work in progress is incomplete
- **WHEN** an absent, incomplete, or candidate active artifact does not claim confirmed status
- **THEN** discovery does not represent it as an invalid confirmed claimant

#### Scenario: The active lifecycle does not require Context
- **WHEN** an existing active lifecycle's schema and Intent declare no pair contract and do not require a Context Pack
- **THEN** discovery validates its confirmed Intent through the existing intent-only path instead of inventing a missing pair

#### Scenario: No single active intent exists
- **WHEN** a session ends with a question and the directory has no active confirmed intent, several, or only unconfirmed ones
- **THEN** the question is retained unbound with an explicit reason and no binding is guessed

#### Scenario: The worktree file is deleted afterwards
- **WHEN** the question file is removed after the session ends
- **THEN** the retained bytes and the record are unchanged

#### Scenario: The loop closes through the intent gate
- **WHEN** the owner answers a recorded question
- **THEN** the answer is a new intent version carrying the resolves-and-disposition record, and recorded identifiers alone lead from the question to that version

#### Scenario: Diagnostic retention remains invisible
- **WHEN** ambient discovery retains an invalid-confirmed diagnostic or encounters another internal failure
- **THEN** the session starts, runs, and ends without receiving that diagnostic or waiting for Goalrail background work
