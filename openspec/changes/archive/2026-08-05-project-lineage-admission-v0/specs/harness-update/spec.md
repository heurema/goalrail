## MODIFIED Requirements

### Requirement: One command brings a repository's harness to the installed binary's canon
**Intent IDs:** OUT-10, SIG-12

Updating a managed repository's harness SHALL be one explicit command. It SHALL re-materialize only canon-owned overlay files, bootstrap files or managed instruction blocks, and prepared admission adapters from the installed binary's known canon, then verify the result by digest and report every file, block, and contract version moved from and to. It SHALL preserve the immutable project identifier and every repository-owned policy value, owner-authored instruction byte, semantic intent, change, WorkSpec, lineage, run, review, and receipt artifact.

Update MUST NOT run as a side effect of diagnosis, setup, hooks, verification, or ordinary work. A governance schema migration that changes repository-owned meaning SHALL remain a separate explicit migration or policy amendment rather than being hidden inside canon update.

The previous canon-owned state SHALL remain recoverable through the existing bounded backup discipline. A repository with no work tree SHALL be refused before any write. Update MUST NOT require a planning runtime, network access, release lookup, or the stock OpenSpec CLI, and MUST NOT update the `gr` binary or activate an external shared check.

#### Scenario: Overlay and bootstrap are behind
- **WHEN** canon-owned overlay and managed bootstrap paths match a previous retained canon
- **THEN** update materializes the current canon, verifies every changed digest, preserves repository-owned values, and reports each version transition

#### Scenario: Harness is already current
- **WHEN** every canon-owned path and managed block matches the installed binary's current canon
- **THEN** nothing is rewritten and the report says the harness is current

#### Scenario: Owner-authored instruction surrounds a managed block
- **WHEN** update replaces a Goalrail-owned instruction block inside an otherwise owner-authored file
- **THEN** bytes outside the proven managed block remain identical and the updated block verifies against canon

#### Scenario: Update would change project policy
- **WHEN** the current canon differs in a way that would require changing repository-owned identity, materiality, exception, or owner-prose content
- **THEN** update refuses that semantic migration and names the separate explicit action

#### Scenario: Update would run implicitly
- **WHEN** another command attempts to apply canon changes as a side effect
- **THEN** that violates this requirement

#### Scenario: Previous state is wanted back
- **WHEN** an owner needs the pre-update canon-owned bytes after a completed or partial update
- **THEN** the bounded recovery set and report are sufficient to restore them without rewriting repository-owned policy

#### Scenario: No runtime or network is available
- **WHEN** update runs with no external planning runtime and no network
- **THEN** it completes using only the canon embedded in the installed binary

#### Scenario: Repository has no work tree
- **WHEN** update targets a repository with no work tree
- **THEN** it is refused with zero project writes

## ADDED Requirements

### Requirement: Update reconciles managed blocks without claiming whole-file ownership
**Intent IDs:** OUT-10, SIG-12

A canon-owned adapter embedded in an owner-editable instruction or integration file SHALL be delimited by a stable Goalrail block identity and digest. Update SHALL replace a block only when the current bytes match a retained known canon or the owner supplies the existing explicit discard authority for that exact block. Missing, duplicated, nested, moved, or locally edited block boundaries SHALL be drift and MUST NOT be repaired by guessing.

#### Scenario: Managed block matches a previous canon
- **WHEN** one uniquely delimited block matches a retained previous Goalrail canon
- **THEN** update replaces only that block and leaves the rest of the file byte-identical

#### Scenario: Managed block was edited
- **WHEN** a delimited block matches no known canon
- **THEN** update names the block as drift and changes nothing without exact discard authority

#### Scenario: Block markers are ambiguous
- **WHEN** block markers are duplicated, nested, malformed, or identify more than one candidate
- **THEN** update fails closed for that file and does not choose a block heuristically
