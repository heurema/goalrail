## MODIFIED Requirements

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
