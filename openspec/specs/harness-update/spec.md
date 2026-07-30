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
**Intent IDs:** OUT-8, SIG-9

Updating a repository's harness SHALL be one command. It SHALL re-materialize the
overlay from the binary's canon, verify the result by comparing digests rather
than by assuming the write succeeded, and report every file it rewrote together
with what the repository moved from and to.

The update MUST NOT run as a side effect of another command. A zero-action update
could change strict-validation behaviour underneath a working agent, which is the
class of silent breakage this project exists to remove; seamless means one
command, not zero.

The previous state SHALL remain recoverable after an update, so a user who
dislikes the result can return to what they had.

Updating MUST NOT require a Node runtime, network access, or the stock OpenSpec
CLI: the canon is already inside the binary.

#### Scenario: The overlay is behind the canon
- **WHEN** the user runs the update in a repository whose overlay matches an older canon
- **THEN** the overlay is re-materialized, the result is verified by digest, and the report names every rewritten file and the move from the old canon to the new one

#### Scenario: The overlay is already current
- **WHEN** the user runs the update where every overlay file already matches the canon
- **THEN** nothing is rewritten and the report says the repository is already current

#### Scenario: An update would run implicitly
- **WHEN** any other command would apply an update as a side effect
- **THEN** that violates this requirement

#### Scenario: The previous state is wanted back
- **WHEN** a user wants the pre-update overlay after an update
- **THEN** what the command left behind is sufficient to restore it

#### Scenario: No runtime or network is available
- **WHEN** the update runs with no Node runtime and no network access
- **THEN** it completes normally

### Requirement: Local drift stops an update rather than being overwritten
**Intent IDs:** OUT-8, SIG-4

Where any overlay file differs from the canon it was materialized from, the update
SHALL stop, name the drifted files, and change nothing. A user who edited a
template did so deliberately, and a command whose stated purpose is refreshing
files must not be the thing that destroys the only copy of that edit.

An explicit flag SHALL discard local edits and proceed. The report SHALL state
that local edits were discarded, so the loss is recorded at the moment it
happens rather than discovered later.

#### Scenario: A template was edited locally
- **WHEN** the update runs where one materialized template differs from the canon it came from
- **THEN** the update stops, names that file, and rewrites nothing

#### Scenario: The user chooses to discard local edits
- **WHEN** the user runs the update with the explicit discard flag on a drifted overlay
- **THEN** the canonical content is restored, the digests match the canon, and the report states that local edits were discarded

#### Scenario: Drift and behind-ness coincide
- **WHEN** the overlay is both behind the canon and locally edited
- **THEN** the update stops on the drift rather than treating the pending upgrade as permission to overwrite

### Requirement: The update does not update the binary
**Intent IDs:** OUT-6, SIG-7

The command SHALL make plain, in its help and in its report, that it updates a
repository's harness to what the installed binary carries and does not replace
the binary itself. The word invites the other expectation, and a user who
believes they upgraded Goalrail when they did not would misread every later
version statement.

The command MUST NOT attempt a network lookup for a newer release. A release
channel now exists, so the prohibition rests on the boundary rather than on the
absence of anything to query: creating the channel and asking it a question are
separate changes, and the second is not made here. The revisit condition recorded
when no channel existed is satisfied by that channel's existence; whether `gr`
should consult it remains an open decision with its own intent.

#### Scenario: The user reads the command's help
- **WHEN** the user asks for the update command's help
- **THEN** it states that the binary itself is not updated by this command

#### Scenario: A release lookup would be attempted
- **WHEN** the update would query a network location for a newer Goalrail release
- **THEN** that violates this requirement

#### Scenario: A release channel exists
- **WHEN** published releases are available to query
- **THEN** the command still performs no lookup, because consulting the channel is a separate change with its own confirmed intent

