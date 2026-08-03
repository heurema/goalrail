## MODIFIED Requirements

### Requirement: Local drift stops an update rather than being overwritten
**Intent IDs:** OUT-1, SIG-1

An update SHALL refuse to replace overlay files that differ from every canon
this binary knows, and SHALL name the differing files and the flag that
discards local edits. A user's edit is theirs, and an update that silently
overwrites it turns a customization into a loss.

An overlay file byte-identical to a **previous** canon SHALL be classified as
behind, never as edited, and the update SHALL bring it forward without any
flag. The first canon change in this project's history is exactly this case for
every adopter at once: their files match what an earlier binary materialized,
they edited nothing, and demanding `--discard-local-edits` from all of them
would misname a routine upgrade as a conflict. A migration test SHALL pin the
transition from the actual previous canon, by digest, so the safety is proven
against the bytes that shipped rather than against a fixture.

#### Scenario: An edited file stops the update
- **WHEN** an overlay file matches no canon this binary knows
- **THEN** the update refuses, names the file, and proceeds only with the explicit discard flag

Before each replacement, update SHALL perform a just-in-time optimistic digest
check against the state it inspected. A mismatch SHALL fail closed unless the
owner supplied the explicit discard flag. This narrows the portable
compare/write window; it MUST NOT be described as atomic protection against a
non-cooperating writer that changes a path after the final check. If a later
path fails its final check after earlier paths were updated, the report SHALL
retain the completed outcomes and recovery point so the partial update is
visible and recoverable.

A bounded discard retry SHALL retain one cumulative recovery set: paths
replaced by an earlier partial attempt stay present, paths observed again are
refreshed to the latest bytes, and the manifest digest describes the bytes
actually copied rather than the stale inspection. If a copied digest differs
from that attempt's inspection, materialization SHALL NOT begin and the same
bounded retry policy SHALL apply, preventing an ABA sequence from validating
against bytes the recovery set does not hold. The report SHALL accumulate
completed outcomes and actually discarded edits across attempts. The CLI SHALL
emit that report on an error whenever a file was created or updated, even when
all changed inputs began missing and no backup was needed.

#### Scenario: A canon-defined file changes during update
- **WHEN** a current, behind, edited, or missing overlay file changes after inspection and the final digest check observes the change
- **THEN** the update refuses without the explicit discard flag, leaves that path's latest bytes untouched, and reports any earlier replacements

#### Scenario: An explicitly discarded file changes during update
- **WHEN** the same race occurs with the explicit discard flag
- **THEN** a bounded retry backs up the latest bytes before replacing them and the report names the discarded file

#### Scenario: A retry follows a partial update
- **WHEN** an earlier path was replaced before a later path failed its final check
- **THEN** the retry preserves one cumulative recovery set and the final report retains both the earlier replacement and every actually discarded edit

#### Scenario: Backup bytes changed after inspection
- **WHEN** the file differs between inspection and the backup read
- **THEN** the recovery manifest records the digest and classification of the bytes actually copied, and update retries before materialization rather than accepting an ABA return to the stale digest

#### Scenario: A partial creation needs no backup
- **WHEN** one missing path was created before a later missing path appears concurrently and stops the update
- **THEN** the CLI emits the partial report even though the backup field is empty

#### Scenario: A previous canon's file is behind, not edited
- **WHEN** an overlay file is byte-identical to a canon a previous binary shipped
- **THEN** the diagnosis reports it as behind and the update replaces it without any flag

#### Scenario: The real transition is pinned
- **WHEN** the migration test runs
- **THEN** it materializes the previous canon by its recorded digest and proves the update crosses to the current one clean
