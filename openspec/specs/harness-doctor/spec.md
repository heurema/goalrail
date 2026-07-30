# harness-doctor Specification

## Purpose
TBD - created by archiving change harness-init-v0. Update Purpose after archive.
## Requirements
### Requirement: One diagnosis covers the whole harness
**Intent IDs:** OUT-6, SIG-7

Goalrail SHALL expose one diagnosis surface for a repository's harness, grown
from attachment health rather than replacing it, and it SHALL report at least:
the repository is not initialized; the overlay is missing; one or more overlay
files have drifted from the binary's canon; the overlay is behind that canon; the
attachment is not registered; the attachment is registered; a registration for
the scaffold exists in a scope this arrangement no longer uses; a registration
names an event this arrangement supersedes; the checking toolchain's presence;
observability's presence; and everything is working.

Every state that is not working SHALL name its next action. An optional facility
that is simply absent SHALL be stated without a next action and without
contributing a failure, because a report that nags about what the product
declares optional teaches the user to ignore it.

Drift SHALL be reported per file. "Something under the overlay differs" leaves
the user diffing the overlay by hand, file by file, which is exactly the work
the diagnosis exists to remove.

The diagnosis SHALL be readable by a machine as well as a person: structured
output alongside the human report, and an exit status that separates healthy from
not, so the check can run unattended.

Every next action the diagnosis prints SHALL be a command this tool accepts —
or, where the tool deliberately has no command for the act because the content
is the user's own to remove, the action SHALL say that plainly rather than
inventing a command. A report that prescribes a command which cannot be run, or
cannot perform the repair, is worse than no report: the user follows correct
advice and has no remaining move.

#### Scenario: The repository is not initialized
- **WHEN** diagnosis runs in a directory with no Goalrail marker
- **THEN** it reports that state and names the initialization command

#### Scenario: The overlay is missing
- **WHEN** diagnosis runs where the marker exists but the overlay does not
- **THEN** it reports the overlay as missing and names the command that installs it

#### Scenario: An overlay file has drifted
- **WHEN** one materialized overlay file differs from the binary's canonical copy
- **THEN** the diagnosis names that file specifically and reports drift, rather than reporting the overlay as generally divergent

#### Scenario: The overlay is behind the canon
- **WHEN** the repository's overlay matches an older canon than the installed binary carries
- **THEN** the diagnosis reports the overlay as behind and names the update command

#### Scenario: A registration exists in the superseded scope
- **WHEN** a registration for a repository-scope scaffold is found in user-level configuration
- **THEN** the diagnosis names it, explains that it duplicates the repository-scope attachment, and names the consented command that removes it — without modifying it

#### Scenario: A registration names a superseded event
- **WHEN** a registration is found naming a stop-like event that fires once per turn, from an earlier arrangement
- **THEN** the diagnosis reports it as needing repair, names the consequence of leaving it — one question retained again on every turn — and names the command that repairs it

#### Scenario: Everything is working
- **WHEN** the repository is initialized, the overlay matches the canon, and the attachment is registered
- **THEN** the diagnosis reports the harness as working and asks nothing further

#### Scenario: The diagnosis is consumed by a machine
- **WHEN** diagnosis is invoked for automated use
- **THEN** it emits structured output and an exit status that distinguishes healthy from not

#### Scenario: A next action names a command that does not exist
- **WHEN** any next action the diagnosis prints names a command this tool does not accept
- **THEN** that violates this requirement

### Requirement: Currency is judged by digests, not by a recorded version
**Intent IDs:** OUT-6, OUT-9, SIG-4

Whether a repository's overlay is current SHALL be decided by comparing its files
against the binary's canonical copies, and MUST NOT depend on a version recorded
inside the repository. The participation marker is per clone and is not shared
through the repository, so a recorded version would answer the same question
differently in two clones of one repository — and the question is about
committed content.

The `gr` version SHALL be reported as information about the binary. No verdict
about the repository may be derived from it.

#### Scenario: The version string alone changes
- **WHEN** the reported `gr` version differs while every overlay file still matches the binary's canon
- **THEN** the diagnosis reports the overlay as current

#### Scenario: The files differ while the version agrees
- **WHEN** an overlay file differs from the binary's canon while any recorded version would suggest currency
- **THEN** the diagnosis reports drift, because the files decide

### Requirement: The checking toolchain is reported as a fact, not a fault
**Intent IDs:** OUT-6, SIG-12

The diagnosis SHALL state whether the runtime the stock OpenSpec CLI requires is
present, and SHALL name what its absence costs — validation and archival of
changes — so the consequence is legible rather than guessed.

That absence MUST NOT be reported as a failure of the harness, MUST NOT
contribute to an unhealthy verdict, and MUST NOT produce a next action. The
harness stands without it; only the checks do not run.

#### Scenario: The runtime is absent
- **WHEN** diagnosis runs where the runtime the stock CLI needs is not on the executable lookup path
- **THEN** the report states the absence, names what cannot be run without it, produces no next action, and does not make the harness verdict unhealthy

#### Scenario: The runtime is present
- **WHEN** diagnosis runs where that runtime is available
- **THEN** the report states its presence as a fact

### Requirement: Observability absence is reported once, without nagging or leaking
**Intent IDs:** OUT-10, SIG-10

The diagnosis SHALL report observability in one line. Absence SHALL be stated as
optional, with no next action and no effect on the overall verdict; Goalrail
works fully with no observability backend, and the local state root remains the
source of truth.

No output SHALL contain key material. Where an endpoint is configured, the report
SHALL name at most the endpoint.

#### Scenario: No observability is configured
- **WHEN** diagnosis runs with no observability endpoint configured
- **THEN** it reports the harness as working, states observability as not configured and optional, and prints no next action for it

#### Scenario: Observability is configured
- **WHEN** an endpoint and keys are present in the environment or the state root
- **THEN** the report names the endpoint at most and no key material appears in any output

