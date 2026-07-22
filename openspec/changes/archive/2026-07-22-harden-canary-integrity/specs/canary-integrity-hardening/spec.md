## ADDED Requirements

### Requirement: Evidence transactions are serialized across local processes
**Intent IDs:** OUT-1, OUT-4, SIG-1, SIG-4

For cooperating Goalrail processes that target the same absolute evidence path, the evidence store SHALL serialize the complete read, validation, sequence and digest calculation, append, and sync transaction. A successful append MUST leave one canonical record linked to the chain state observed while holding the process-shared lock. Reads and verification MUST NOT observe an in-progress cooperating append.

#### Scenario: Concurrent processes append distinct events
- **WHEN** two synchronized Goalrail processes append distinct valid events to the same evidence path
- **THEN** both events are retained exactly once with unique consecutive sequences, the digest links are canonical, and complete-chain verification succeeds

#### Scenario: Reader overlaps the first append
- **WHEN** a reader starts after the first writer has acquired the process-shared lock but before the evidence file exists
- **THEN** the reader waits for the writer transaction and observes the committed event instead of returning an empty store

#### Scenario: Required process lock is unavailable
- **WHEN** the current platform cannot provide the required process-shared evidence lock
- **THEN** the store fails the operation before appending an event instead of falling back to process-local locking

### Requirement: Terminal identity conflicts are wrong joins
**Intent IDs:** OUT-2, OUT-4, SIG-2, SIG-4

When immutable run context has already been bound to a provider-authoritative root session, a later `SESSION_CONFLICT` SHALL project to `wrong` lineage. One such projection MUST increment `wrong_joins`, set the wrong-join hard-stop signal, and produce `STOP` immediately.

#### Scenario: Verified root session changes
- **WHEN** a verified change/run binding receives a terminal `SESSION_CONFLICT` after its bounded resolution attempt
- **THEN** the operator report records one wrong join and stops the canary without waiting for the unresolved-link threshold

#### Scenario: Later retry cannot erase a wrong join
- **WHEN** a terminal `SESSION_CONFLICT` is followed by a later lineage event such as `RESOLUTION_ATTEMPT_EXHAUSTED`
- **THEN** the operator report retains the earlier wrong join and continues to produce `STOP`

#### Scenario: Lineage remains unresolved without a conflicting identity
- **WHEN** lineage is still unlinked after its bounded attempt because an authoritative identity is absent rather than conflicting
- **THEN** the operator report records an unresolved link and preserves the existing two-link stop threshold

### Requirement: Wording-only intent amendments preserve provenance
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4

A wording-only amendment SHALL preserve every source-evidence entry and the evidence-reference set attached to each stable semantic item. It MAY reorder unchanged evidence entries, references, or semantic items and MAY change statement wording without changing the snapshot version. Replacing evidence content, adding or removing evidence, or repointing an item to different evidence MUST fail validation.

#### Scenario: Source evidence changes in place
- **WHEN** a wording-only amendment changes the kind, statement, or reference of source evidence while retaining its stable ID
- **THEN** amendment validation rejects the provenance change

#### Scenario: Semantic item is repointed
- **WHEN** a wording-only amendment adds, removes, or replaces an evidence reference for a stable desired outcome, non-goal, or success signal
- **THEN** amendment validation rejects the provenance change

#### Scenario: Only wording and order change
- **WHEN** a wording-only amendment changes statement text or reorders unchanged items and evidence references while preserving their stable identities and values
- **THEN** amendment validation accepts the edit without creating a new intent version
