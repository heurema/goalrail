## ADDED Requirements

### Requirement: The diagnosis reports a newer release as a fact, never as a problem
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2

The diagnosis SHALL report whether a newer Goalrail release exists. It SHALL do
so in the voice it uses for the toolchain and for observability: a stated fact,
with no next action, contributing nothing to whether the harness is reported as
working and nothing to the exit code.

The reason is not stylistic. The tool has no command that replaces its own
binary, so a next action here would name a command the tool does not accept,
which a promoted requirement already forbids. A repository whose harness is
intact is working whether or not its binary is a release behind, and a caller
that reads the exit code learns about the repository, not about the binary.

Where the line rests on an answer from a network source, it SHALL name that
source, so the report discloses its counterparty where the user is rather than
only in documentation. Where there is no answer and no counterparty, it names
none.

The check MUST NOT compare anything when the running binary's own version is not
a release version — a build from a checkout, from a modified tree, or one
carrying no version at all. In those cases the diagnosis SHALL say that the
question does not apply rather than guess. No wording SHALL assert that the
installation is up to date: the source can lag a published release, so the
report states what was found and when it was looked for.

Where the check produced no answer — because it was declined, or because it
failed — the diagnosis SHALL say so in one line, and a declined check SHALL be
distinguishable from a failed one. It MUST NOT surface an error, a stack trace,
or any text the source produced.

An answer read from a cache that is not yet due for refresh is an answer, not an
absence: it is reported like any other, stating when it was obtained.

#### Scenario: A newer release exists
- **WHEN** the running binary is a release version older than the newest released one
- **THEN** the diagnosis states that a newer release exists, names the service it asked, prescribes no command, and leaves the working verdict and the exit code exactly as they would have been

#### Scenario: The binary is current
- **WHEN** the running binary's version matches the newest released one
- **THEN** the diagnosis states what it found and when, names the service it asked, and does not assert that the installation is up to date

#### Scenario: The running binary is not a release build
- **WHEN** the version the binary reports is a pseudo-version, carries a modification marker, is the toolchain's development marker, or is unknown
- **THEN** no comparison is made and the diagnosis says the question does not apply

#### Scenario: The check produced no answer
- **WHEN** the check was declined or failed
- **THEN** the diagnosis says so in one line, distinguishing a declined check from a failed one, with no error, no stack trace, and no text from the source

#### Scenario: The answer comes from a cache that is not yet due
- **WHEN** a diagnosis runs within the caching interval and a cached answer is present
- **THEN** no request is made, the answer is reported compared against the running binary's version, and the report states when it was obtained rather than that the check did not run

#### Scenario: A newer release would change the verdict
- **WHEN** the existence of a newer release would make the harness report as not working, add a problem, or change the exit code
- **THEN** that violates this requirement

#### Scenario: A next action would be prescribed
- **WHEN** the line about a newer release prescribes a command
- **THEN** that violates this requirement, because the tool has no command that replaces its own binary
