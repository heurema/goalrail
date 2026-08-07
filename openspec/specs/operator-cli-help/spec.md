# Operator CLI Help Specification

## Purpose

Define concise built-in lifecycle and command help for the `gr` operator
surface without changing machine-readable command results.
## Requirements
### Requirement: Gr exposes a concise lifecycle help surface
**Intent IDs:** OUT-2, OUT-4, OUT-5, OUT-7, OUT-10, SIG-2, SIG-3, SIG-5, SIG-12

The `gr help` command SHALL describe the existing `prepare → inspect → start → finish` run lifecycle, state that start remains explicit and owner-assisted, and identify built-in help for each command. It SHALL also describe the managed-project lifecycle in its dependency order: initialize or migrate project governance; diagnose project and local readiness; plan and explicitly authorize missing setup; confirm intent and select one current Goalrail change; prepare and run bounded work; inspect lineage; verify the immutable base/head range; and use the same verifier at shared admission.

Help MUST NOT state that material ordinary work needs no per-task Goalrail flow. It SHALL instead say that supported agents perform the flow through committed bootstrap guidance, while the operator can inspect each artifact and verdict. It SHALL distinguish project identity, local setup, advisory local checks, and protected shared admission, and MUST NOT call local hooks or `prek` unbypassable.

Initialization, migration, setup planning, setup execution, diagnosis, update, connection, disconnection, lineage inspection, and lineage verification SHALL be discoverable as separate operations with their authority boundaries. Help SHALL state that setup permission covers only an exact plan, that Goalrail does not automate provider trust, and that external required-check activation is separate owner-authorized work.

A superseded command name SHALL keep working during its declared migration window and name its successor. Every printed remedy SHALL name an accepted command or an exact documented user/provider action. Internal hook entry points, fixture-only controls, and broader activation controls MUST NOT appear as operator commands.

#### Scenario: Operator requests top-level help
- **WHEN** an operator invokes `gr help`
- **THEN** help presents both the bounded run lifecycle and the managed-project setup, intent/change, lineage, and admission lifecycle in dependency order

#### Scenario: Operator looks for ordinary feature flow
- **WHEN** an operator reads top-level help after cloning a managed project
- **THEN** help directs supported agents through diagnosis and exact setup when needed, then confirmed intent and one current change before material code work

#### Scenario: Operator compares local and shared checks
- **WHEN** an operator reads help for lineage verification
- **THEN** help states that local integrations are advisory and protected shared admission is authoritative only after verified activation

#### Scenario: Operator encounters an untrusted scaffold
- **WHEN** help describes setup or connection for a scaffold with its own trust step
- **THEN** it names the user's provider action and does not imply Goalrail can pre-approve trust

#### Scenario: Superseded command name is used
- **WHEN** an operator invokes a command retained for migration compatibility
- **THEN** it still runs within the declared window and its output names the current command and changed semantics

#### Scenario: Help would advertise a hook entry point
- **WHEN** generated help would list a scaffold- or Git-invoked internal hook as an operator command
- **THEN** that violates this requirement

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

### Requirement: A command's failure is stated in Goalrail's own words
**Intent IDs:** OUT-1, OUT-2, OUT-3, SIG-1, SIG-2, SIG-3

Where a `gr` command cannot proceed because of a condition it can name, it SHALL
say so itself. It MUST NOT relay the message, the exit status, or any other
output of a program it invoked on the user's behalf. A user who reads
`fatal: not a git repository (or any of the parent directories): .git` has been
handed another tool's vocabulary for a situation Goalrail understands perfectly
well, and an unattended caller reading the exit status of that other tool learns
about the wrong process.

This is a property of the command surface rather than of any one command. A
command corrected in isolation carries nothing to the next one, and the
requirement exists so that a command written later inherits it.

Stating the condition MUST NOT hide a failure. Where the invoked program did not
run at all, or failed for a reason that is not the named condition, that remains
a distinct outcome in wording and in exit status: a check that did not run and a
state the check found are different answers, and a caller already reads the
difference.

The subject is any program Goalrail invokes, not one of them. Git is the program
this reaches today.

#### Scenario: A command meets a path outside version control
- **WHEN** any `gr` command that resolves a repository is run where there is none
- **THEN** it names that condition in its own words, and its output carries no message or exit status produced by the program it asked

#### Scenario: A command would relay what it was told
- **WHEN** a command's output repeats the text or the exit status of a program it invoked
- **THEN** that violates this requirement

#### Scenario: The invoked program could not run
- **WHEN** the program a command depends on is absent, or fails for a reason other than the condition being named
- **THEN** the command reports that as a distinct outcome, distinguishable in wording and exit status from the condition it would otherwise have named

#### Scenario: A command is added later
- **WHEN** a new command resolves a repository
- **THEN** it is subject to this requirement without needing its own correction

