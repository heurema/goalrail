# Operator CLI Help Specification

## Purpose

Define concise built-in lifecycle and command help for the `gr` operator
surface without changing machine-readable command results.
## Requirements
### Requirement: Gr exposes a concise lifecycle help surface
**Intent IDs:** OUT-1, OUT-2

The `gr help` command SHALL describe the `prepare → inspect → start → finish`
lifecycle, state that start remains explicit and owner-assisted, and identify
how to obtain help for each command. The help text MUST NOT expose fixture-only
or broader activation controls.

Help SHALL additionally present background attachment as its own surface,
distinct from that lifecycle: connecting a scaffold, initializing a repository,
and disconnecting. It SHALL make plain that after connection ordinary work
needs no Goalrail command, so an operator cannot mistake the wrapper lifecycle
for something they must run per task.

The hook entry point SHALL NOT appear in help. It is invoked by the scaffold,
never by a person, and listing it would invite manual invocation of a
fail-quiet path whose silence would then read as breakage.

#### Scenario: Operator requests top-level help
- **WHEN** an operator invokes `gr help`
- **THEN** the command prints the concise lifecycle and points to help for
  `prepare`, `inspect`, `start`, and `finish`

#### Scenario: Operator looks for the background surface
- **WHEN** an operator invokes `gr help`
- **THEN** connection, initialization, and disconnection are presented as a
  separate surface, and the text states that ordinary work needs no per-task
  Goalrail command

#### Scenario: Help would advertise the hook entry point
- **WHEN** help text would list the scaffold-invoked hook entry point as an
  operator command
- **THEN** that violates this requirement, because the hook is never invoked by
  a person

### Requirement: Each Gr command exposes built-in flag guidance
**Intent IDs:** OUT-1, OUT-2

Each supported `gr` command SHALL expose built-in help for its flags and
required arguments. Adding help MUST NOT change the JSON output or execution
semantics of successful `prepare`, `inspect`, `start`, or `finish` commands,
nor of the background attachment commands.

#### Scenario: Operator requests command help
- **WHEN** an operator invokes a supported command with `--help`
- **THEN** the command prints its flag guidance without invoking the lifecycle

#### Scenario: Existing machine-readable command succeeds
- **WHEN** an operator invokes a supported command with valid non-help input
- **THEN** its existing JSON result and execution semantics remain unchanged

