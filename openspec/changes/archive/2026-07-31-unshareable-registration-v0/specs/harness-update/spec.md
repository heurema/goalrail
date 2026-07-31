## MODIFIED Requirements

### Requirement: One command brings a repository's harness to the installed binary's canon
**Intent IDs:** OUT-3, SIG-3

Updating a repository's harness SHALL be one command. It SHALL re-materialize the
overlay from the binary's canon, verify the result by comparing digests rather
than by assuming the write succeeded, and report every file it rewrote together
with what the repository moved from and to.

The update MUST NOT run as a side effect of another command. A zero-action update
could change strict-validation behaviour underneath a working agent, which is the
class of silent breakage this project exists to remove; seamless means one
command, not zero.

A repository with no work tree has nowhere for the overlay to live, so updating
SHALL refuse it with the reason named rather than writing beside the repository's
own contents. That boundary belongs to the harness rather than to one command:
initialization and update install the same files, and a rule enforced in one of
them is a manner rather than a guarantee.

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

#### Scenario: The repository has no work tree
- **WHEN** the update runs against a repository that has no work tree
- **THEN** it is refused with the reason named, and nothing is written into it
