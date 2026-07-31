## MODIFIED Requirements

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
