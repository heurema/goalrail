# harness-update Specification

## Purpose

Define how a repository's harness is brought to the canon the installed binary
carries: one command, verified by comparing digests rather than by assuming the
writes succeeded, with the replaced files kept outside the repository so recovery
does not depend on them ever having been committed. A local edit stops the update
rather than being overwritten, and the command never updates the binary itself or
consults a release channel.
## Requirements
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

### Requirement: Local drift stops an update rather than being overwritten
**Intent IDs:** OUT-1, SIG-1, SIG-4

An update SHALL refuse to replace overlay files that differ from every canon
this binary knows, and SHALL name the differing files and the flag that
discards local edits. A user's edit is theirs, and an update that silently
overwrites it turns a customization into a loss.

An overlay file whose path remains defined by the **current** canon and whose
bytes are identical to a previous canon SHALL be classified as behind, never
as edited, and the update SHALL bring it forward without any flag. A path that
only a previous canon defined is superseded instead and remains untouched. The
first canon change in this project's history is exactly the behind case for
every adopter at once: their files match what an earlier binary materialized,
they edited nothing, and demanding `--discard-local-edits` from all of them
would misname a routine upgrade as a conflict. A migration test SHALL pin the
transition from the actual previous canon, by digest, so the safety is proven
against the bytes that shipped rather than against a synthetic fixture.

#### Scenario: A template was edited locally
- **WHEN** an overlay file matches no canon this binary knows
- **THEN** the update refuses, names the file, and rewrites nothing without the explicit discard flag

#### Scenario: The user chooses to discard local edits
- **WHEN** the user runs the update with the explicit discard flag on a drifted overlay
- **THEN** the canonical content is restored, the digests match the canon, and the report states that local edits were discarded

#### Scenario: Drift and behind-ness coincide
- **WHEN** an overlay contains both a previous-canon file and a file that differs from every canon this binary knows
- **THEN** the update stops on the edited file rather than treating the pending upgrade as permission to overwrite it

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
- **WHEN** a path remains defined by the current canon and its overlay file is byte-identical to a canon a previous binary shipped
- **THEN** the diagnosis reports it as behind and the update replaces it without any flag

#### Scenario: The real transition is pinned
- **WHEN** the migration test runs
- **THEN** it reads every retained previous-canon file, validates every recorded digest and the derived canon ID, reconstructs the old overlay, and proves the update crosses to the current one clean

### Requirement: The update does not update the binary
**Intent IDs:** OUT-5

The command SHALL make plain, in its help and in its report, that it updates a
repository's harness to what the installed binary carries and does not replace
the binary itself. The word invites the other expectation, and a user who
believes they upgraded Goalrail when they did not would misread every later
version statement.

The command MUST NOT attempt a network lookup for a newer release. The
prohibition rests on the boundary rather than on the absence of anything to
query: a release channel exists, and the diagnosis now asks it whether a newer
release has been published. This command still asks nothing, because updating a
repository's harness and learning about the binary are separate acts, and a
command that quietly did both would make its own name a lie. The separation is
enforced by a check that fails if the transport becomes reachable from here,
rather than by discipline.

#### Scenario: The user reads the command's help
- **WHEN** the user asks for the update command's help
- **THEN** it states that the binary itself is not updated by this command

#### Scenario: A release lookup would be attempted
- **WHEN** the update would query a network location for a newer Goalrail release
- **THEN** that violates this requirement

#### Scenario: A release channel exists
- **WHEN** published releases are available to query
- **THEN** this command still performs no lookup, and the question is answered by the diagnosis instead

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
