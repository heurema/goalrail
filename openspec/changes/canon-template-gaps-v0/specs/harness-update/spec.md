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

#### Scenario: A previous canon's file is behind, not edited
- **WHEN** an overlay file is byte-identical to a canon a previous binary shipped
- **THEN** the diagnosis reports it as behind and the update replaces it without any flag

#### Scenario: The real transition is pinned
- **WHEN** the migration test runs
- **THEN** it materializes the previous canon by its recorded digest and proves the update crosses to the current one clean
