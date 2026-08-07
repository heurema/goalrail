## ADDED Requirements

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
