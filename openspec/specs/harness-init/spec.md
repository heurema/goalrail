# harness-init Specification

## Purpose

Define what one initialization installs in one repository: the OpenSpec overlay
carrying the Goalrail schema, materialized from a canon the binary holds; the
participation marker; and the session hooks, registered where the scaffold's
settings layer allows it and only where no commit could hand them to someone
else. Also define what initialization refuses — a foreign custom schema, a
shareable registration path — and that installing or maintaining the harness
never depends on the runtime the checking toolchain needs.

## Requirements
### Requirement: One command installs the whole harness in one repository
**Intent IDs:** OUT-1, SIG-1, SIG-2

Initialization SHALL install everything a repository needs to be driven as a
Goalrail repository, not merely mark it. That payload is the canonical overlay —
the OpenSpec configuration naming the Goalrail schema, the schema itself, and its
templates — materialized from copies carried inside the `gr` binary, together
with the existing participation marker.

The canonical copy SHALL live in the binary and be materialized per repository.
A repository's files are therefore a copy that can drift, never the source of
truth, and every later comparison is made against the binary's canon rather than
against the repository's own history.

The canon covers the schema and its templates. The OpenSpec configuration is
repository-owned content that carries the project's own description alongside the
schema name, so initialization SHALL create it when absent and SHALL manage only
the key that names the schema. Comparing the whole configuration against a canon
would report every project's own description as drift, which would train the user
to ignore the report that matters.

Initialization SHALL report what it created, what it changed, and what it left
alone. The report SHALL name the exact pinned invocation the repository is now
driven by, including the explicit schema argument the pinned version requires,
because a repository holding only the files still fails on its first change
otherwise. The report SHALL also state that the materialized files are the
user's to commit: initialization leaves uncommitted repository content, and a
user who expects a self-contained act would otherwise not know that.

Repeating initialization SHALL leave the repository byte-identical. Nothing in
the payload is generated per run, so a second run has nothing legitimate to
change.

#### Scenario: A fresh repository is initialized
- **WHEN** the user runs initialization in a directory that has no Goalrail marker and no OpenSpec root
- **THEN** the overlay and the marker are materialized from the binary's canon, and the report names what was created, the pinned invocation with its explicit schema argument, and that committing the new files is the user's act

#### Scenario: Initialization is repeated
- **WHEN** the user runs initialization a second time in a repository it already initialized
- **THEN** every file is byte-identical to what the first run produced

#### Scenario: The materialized overlay is the one the checking CLI accepts
- **WHEN** the overlay carried in the binary is materialized into a repository
- **THEN** the stock OpenSpec CLI validates that overlay and creates a change with the Goalrail schema, verified on the development side rather than in the user's repository

### Requirement: An existing OpenSpec root survives initialization
**Intent IDs:** OUT-2, SIG-3

Where a repository already carries an OpenSpec root, initialization SHALL add
the Goalrail schema and switch the configuration to it, and MUST NOT touch any
existing spec or change file. Adopting the harness is not a reason to disturb
work that is already in flight.

Where the configuration already names a custom schema that is not Goalrail's,
initialization SHALL stop and ask rather than switch. Silently rewiring another
workflow's schema would change how every subsequent change in that repository is
compiled, and the user would discover it from the results rather than from the
act.

#### Scenario: A stock OpenSpec root is adopted
- **WHEN** initialization runs where an OpenSpec root exists with the default schema and existing specs and changes
- **THEN** the Goalrail schema is added, the configuration names it, every existing spec and change file is byte-identical afterwards, and the report states what changed

#### Scenario: The configuration names a foreign custom schema
- **WHEN** initialization runs where the configuration names a custom schema that is not Goalrail's
- **THEN** initialization stops, names that schema, and switches nothing until the user explicitly confirms

#### Scenario: The configuration already names the Goalrail schema
- **WHEN** initialization runs where the configuration already names the Goalrail schema
- **THEN** the configuration is left exactly as it is and the report says so

### Requirement: Initialization registers the attachment where the scaffold allows it
**Intent IDs:** OUT-3, SIG-5, SIG-6

Initialization SHALL register the session hooks for a scaffold whose settings
layer permits registration inside the repository, because marking a repository
and attaching to its sessions are one act, and a registration that lives in the
repository is never invoked in unrelated sessions at all.

The scaffold SHALL be selected by detection, with an explicit flag override.
Where no supported scaffold is detected, initialization SHALL still install the
overlay and the marker, and the report SHALL name the command that registers the
attachment later; installing the harness must not depend on which agent
environment happens to be present. Where a supported scaffold registers only at
user scope, the report SHALL name that separate consented command rather than
implying the attachment is complete.

The registration SHALL be written to the scaffold's per-user project settings
file, and MUST NOT be written to a settings file the repository could supply to
someone else. A registration a repository could carry would run in every
teammate's session on the strength of one user's consent, and consent to run a
command in one's own sessions is not transferable.

Registration SHALL therefore happen only where that path is ignored by the
version control system. Where it is not, initialization SHALL install the rest of
the harness, refuse the registration, and name the reason, the exact ignore entry
that would make it possible, and the flag that adds it. Initialization MUST NOT
edit the repository's ignore rules unasked, and MUST NOT register in the hope
that nobody commits the file. An explicit flag SHALL add the missing entries and
proceed, reporting them among its changes, because they are repository content
exactly as the overlay is.

The participation marker carries the same exposure for a weaker reason: it is per
clone, and committing it would make one user's initialization a shared repository
fact. Where its ignore entry is missing, initialization SHALL still complete and
SHALL say so, because refusing to install the harness over an ignore rule would
be a disproportionate response to a recoverable condition.

The events registered SHALL be the ones whose cadence matches the meaning they
carry: the occurrence that opens a session, and the event that fires when a session
ends. Where a scaffold's stop-like event fires once per turn instead, it MUST NOT
be the one registered — a question left at the reserved path would be retained
again on every turn, minting a record per turn out of one session's single
question.

Re-running initialization SHALL repair a registration of ours that is stale,
unscoped, or naming a superseded event, under the same discipline connection
follows: replace rather than accompany, leave a correct registration
byte-identical, and leave a handler it did not add untouched.

Initialization MUST NOT modify user-level scaffold configuration for any reason,
including removing a registration that belongs to an earlier arrangement.

#### Scenario: A scaffold is detected
- **WHEN** initialization runs where exactly one supported repository-scope scaffold is present
- **THEN** the hooks are registered for that scaffold in its per-user project settings file

#### Scenario: The scaffold is named explicitly
- **WHEN** the user names a scaffold with the override flag
- **THEN** that scaffold is used regardless of what detection would have concluded

#### Scenario: No scaffold is detected
- **WHEN** initialization runs where no supported scaffold is present
- **THEN** the overlay and the marker are still installed and the report names the command that registers the attachment later

#### Scenario: The scaffold registers only at user scope
- **WHEN** initialization runs for a scaffold whose repository-scope route is externally blocked
- **THEN** the report names the separate consented connection command instead of implying the attachment is complete

#### Scenario: The project settings path is not ignored
- **WHEN** initialization would register hooks in a per-user project settings file the version control system does not ignore
- **THEN** the registration is refused, the rest of the harness is installed, and the report names the reason, the exact ignore entry, and the flag that adds it

#### Scenario: The user asks for the ignore entries to be added
- **WHEN** initialization runs with the explicit flag that manages ignore rules
- **THEN** the missing entries are added, the registration proceeds, and the report names the entries among its changes

#### Scenario: The marker's ignore entry is missing
- **WHEN** initialization writes the participation marker in a repository whose ignore rules do not cover it
- **THEN** initialization completes and the report states that the marker is committable, so one user's initialization does not silently become a shared repository fact

#### Scenario: The scaffold's stop-like event fires once per turn
- **WHEN** initialization registers hooks on a scaffold whose stop-like event fires once per turn and which also exposes an event that fires when a session ends
- **THEN** the session-ending event is registered and the per-turn event is not

#### Scenario: Re-initialization meets a registration naming a per-turn event
- **WHEN** initialization runs where its own registration names the per-turn event from an earlier arrangement
- **THEN** that registration is replaced with the session-ending event, leaving exactly one registration per event and no residue

#### Scenario: Re-initialization meets a stale registration
- **WHEN** initialization runs where its own registration names an executable other than the one it was invoked with
- **THEN** exactly one registration per event remains, naming the current executable, with no duplicate and no residue

#### Scenario: Re-initialization meets a correct registration
- **WHEN** initialization runs where its own registration already names the current executable and the correct occurrence
- **THEN** the settings file is left byte-identical

#### Scenario: A foreign handler shares the event
- **WHEN** initialization registers or repairs an event that also carries a handler it did not add
- **THEN** the foreign handler survives unchanged, including its occurrence

#### Scenario: User-level configuration would be modified
- **WHEN** any initialization path would write to user-level scaffold configuration
- **THEN** that modification is refused, including when it would remove a registration from an earlier arrangement

### Requirement: Installing and maintaining the harness needs no Node runtime
**Intent IDs:** OUT-1, OUT-6, SIG-12

Initialization, update, and diagnosis MUST NOT execute Node, a package runner,
or the stock OpenSpec CLI. Those commands SHALL complete on a machine where no
Node runtime exists, because the harness they manage is inert repository content
and the tool that manages it is a single binary.

Whether the materialized overlay is one the stock CLI accepts SHALL be
established on the development side, against the pinned CLI, before the canon is
embedded. Verification belongs where the canon is authored, not in every user's
repository.

The stock CLI remains what validates and archives changes. That dependency is
the agent's, not `gr`'s, and it SHALL be reported as a fact rather than enforced
as a precondition.

#### Scenario: No Node runtime is present
- **WHEN** initialization, update, or diagnosis runs on a machine with no Node runtime on the executable lookup path
- **THEN** each command completes normally, and the absence appears only where diagnosis reports the environment

#### Scenario: A Goalrail command would invoke the checking CLI
- **WHEN** any `gr` command would execute Node, a package runner, or the stock OpenSpec CLI
- **THEN** that violates this requirement

### Requirement: Initialization writes no credential into the repository
**Intent IDs:** OUT-10

Initialization MUST NOT write an observability endpoint's credentials, tokens,
or keys into repository content. Observability configuration is read from the
environment or from the state root outside the repository, and absence of it is a
fully working configuration.

#### Scenario: Observability is already configured
- **WHEN** initialization runs where an observability endpoint and keys are present in the environment or the state root
- **THEN** no credential is written into repository content

#### Scenario: Observability is absent
- **WHEN** initialization runs with no observability configured
- **THEN** initialization completes normally and nothing about the missing backend is treated as a failure

