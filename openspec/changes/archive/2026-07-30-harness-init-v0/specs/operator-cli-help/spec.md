## MODIFIED Requirements

### Requirement: Gr exposes a concise lifecycle help surface
**Intent IDs:** OUT-1, OUT-2, OUT-7, SIG-8

The `gr help` command SHALL describe the `prepare → inspect → start → finish`
lifecycle, state that start remains explicit and owner-assisted, and identify
how to obtain help for each command. The help text MUST NOT expose fixture-only
or broader activation controls.

Help SHALL additionally present the harness as its own surface, distinct from
that lifecycle: installing the harness in a repository, diagnosing it, updating
it to what the installed binary carries, connecting a scaffold that registers
only at user scope, and disconnecting. It SHALL make plain that after
installation ordinary work needs no Goalrail command, so an operator cannot
mistake the wrapper lifecycle for something they must run per task.

The diagnosis command SHALL be discoverable from help, because the states it
reports — hooks registered but not yet trusted, or an overlay that has drifted
from the canonical one — are otherwise indistinguishable from a broken
installation.

A command name that is superseded SHALL keep working and SHALL name its
successor, and every remedy the tool prints SHALL name a command the tool
accepts. Renaming a surface while printed advice still names the old one leaves
the user following instructions that fail.

The hook entry point SHALL NOT appear in help. It is invoked by the scaffold,
never by a person, and listing it would invite manual invocation of a
fail-quiet path whose silence would then read as breakage.

#### Scenario: Operator requests top-level help
- **WHEN** an operator invokes `gr help`
- **THEN** the command prints the concise lifecycle and points to help for
  `prepare`, `inspect`, `start`, and `finish`

#### Scenario: Operator looks for the background surface
- **WHEN** an operator invokes `gr help`
- **THEN** installation, diagnosis, updating, connection, and disconnection are
  presented as a separate surface, and the text states that ordinary work needs
  no per-task Goalrail command

#### Scenario: Operator cannot tell untrusted from broken
- **WHEN** an operator's attachment is registered but not yet trusted, or the
  repository's overlay has drifted from the canonical one
- **THEN** help points at the diagnosis command that names that exact state

#### Scenario: A superseded command name is used
- **WHEN** an operator invokes a command name this change supersedes
- **THEN** it still runs and its output names the command that replaces it

#### Scenario: Help would advertise the hook entry point
- **WHEN** help text would list the scaffold-invoked hook entry point as an
  operator command
- **THEN** that violates this requirement, because the hook is never invoked by
  a person
