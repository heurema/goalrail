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
**Intent IDs:** OUT-1, OUT-2, OUT-3, SIG-1, SIG-2, SIG-3, SIG-5

Where a repository already carries an OpenSpec root, initialization SHALL add
the Goalrail schema and switch the configuration to it, and MUST NOT touch any
existing spec or change file. Adopting the harness is not a reason to disturb
work that is already in flight.

Where the configuration already names a custom schema that is not Goalrail's,
initialization SHALL stop and ask rather than switch. Silently rewiring another
workflow's schema would change how every subsequent change in that repository is
compiled, and the user would discover it from the results rather than from the
act.

Where a switch did replace another schema, the report SHALL carry an adoption
section describing what the switch changed. Confirming a switch answers whether
to adopt; it does not tell the user what they have adopted, and the difference
between those two is where a repository silently ends up running rules written
against a schema that is gone.

Where both schemas are files in the repository, that section SHALL state the
artifact-level difference between them: which artifacts exist only in one, whose
dependencies differ, and whose instructions differ. It SHALL report that an
instruction differs and MUST NOT characterise what the difference means. Their
structure is comparable; their prose is English written for another project's
purpose, and a mechanical reading of it would produce confident nonsense.

The adopted side of that comparison SHALL be the schema the repository will
actually compile against — the overlay file on disk once initialization has
finished with it — and not the copy embedded in the binary. Initialization
leaves an edited overlay file alone by design, and OpenSpec then uses that file;
describing the embedded copy instead would report differences for a schema the
switch did not adopt.

Where the replaced schema is not a file in the repository, the section SHALL
state that the comparison could not be made and why, and the rules disclosure
and the pin count SHALL still be produced. A schema shipped with the stock CLI
resolves from the installed package rather than from the repository, and
reaching into that package would make the harness depend on the Node runtime it
is required to work without. Partial disclosure is the honest outcome there; a
comparison against something the repository does not contain is not.

That section SHALL also report the rules the configuration carries: how many
there are, each of them verbatim, and a plain statement that they were present
when the schema was replaced and that Goalrail neither interprets nor edits
them. Their presence at that moment is what the repository proves; when they
were written, and against which schema, it does not.
Rules are the one part of the configuration Goalrail deliberately does not own —
which is why the schema key is rewritten a line at a time — so disclosing them
is the whole of what it may do. Nothing else in the toolchain validates a rule
against the active schema, so a rule that outlived its schema is reported here
or nowhere.

That section SHALL also state how many changes still name the replaced schema,
counted from the changes present in the repository, and SHALL say that its
directory must remain while that count is above zero. Every change records the
schema it was written under, and removing a schema directory those changes still
name would break them; whether removal is safe is therefore a count, not an
opinion, and initialization MUST NOT remove, move or archive that directory
either way.

The adoption section SHALL be absent where no schema was replaced, and its
presence MUST NOT make initialization interactive or change its exit status.

#### Scenario: A stock OpenSpec root is adopted
- **WHEN** initialization runs where an OpenSpec root exists with the default schema and existing specs and changes
- **THEN** the Goalrail schema is added, the configuration names it, every existing spec and change file is byte-identical afterwards, and the report states what changed

#### Scenario: The configuration names a foreign custom schema
- **WHEN** initialization runs where the configuration names a custom schema that is not Goalrail's
- **THEN** initialization stops, names that schema, and switches nothing until the user explicitly confirms

#### Scenario: The configuration already names the Goalrail schema
- **WHEN** initialization runs where the configuration already names the Goalrail schema
- **THEN** the configuration is left exactly as it is and the report says so

#### Scenario: A confirmed switch reports what the adopted schema changed
- **WHEN** initialization switches a repository from another schema to Goalrail's
- **THEN** the report names every artifact present in only one of the two schemas, every artifact whose dependencies differ, and every artifact whose instructions differ

#### Scenario: The replaced schema is not a file in the repository
- **WHEN** a switch replaced a schema that ships with the stock CLI rather than living in the repository
- **THEN** the report states that the schemas could not be compared and why, and still reports the rules and the count of changes naming the replaced schema

#### Scenario: The overlay on disk differs from the embedded copy
- **WHEN** a switch happens in a repository whose overlay schema file was edited and is therefore left alone
- **THEN** the comparison describes that file rather than the copy embedded in the binary

#### Scenario: The rules are disclosed without being judged
- **WHEN** a switch replaced a schema in a repository whose configuration carries rules
- **THEN** the report states their count, reproduces each rule verbatim, states that they were present when the schema was replaced, and states that Goalrail neither interprets nor edits them
- **AND** the report renders no verdict on any individual rule and the configuration's rules are byte-identical afterwards

#### Scenario: Whether the replaced schema may be removed is counted
- **WHEN** a switch replaced a schema that changes in the repository still name
- **THEN** the report states how many changes name it and that its directory must remain, and the directory is present and unmodified afterwards

#### Scenario: Nothing still names the replaced schema
- **WHEN** a switch replaced a schema that no change in the repository names
- **THEN** the report states a count of zero and that the directory may be removed, and initialization removes nothing

#### Scenario: No schema was replaced
- **WHEN** initialization creates a configuration where none existed, or leaves one that already names the Goalrail schema
- **THEN** the report carries no adoption section

### Requirement: An adoption is recorded where the harness keeps its own state
**Intent IDs:** OUT-4, OUT-5, SIG-4, SIG-5

Where initialization replaced another schema, it SHALL record the adoption in the
marker it already owns: the name of the replaced schema, when the adoption
happened, and a digest of the configuration's rules block as it stood at that
moment. The knowledge that rules predate a schema is worth nothing if it lives
only in the output of the command that performed the switch, because that command
is routinely run by an agent and its output is routinely read by nobody.

The record SHALL be written to Goalrail's own state and MUST NOT be written into
the user's configuration. Recording a fact about someone's rules is not a licence
to edit the file that holds them.

A marker carrying no adoption record SHALL be valid and SHALL read as a
repository that never replaced a schema. Every marker written before this
requirement lacks the record, and treating those as damaged would turn a working
installation into a reported fault.

The digest SHALL cover the rules block alone, so that editing the rules changes
it and editing unrelated parts of the configuration does not.

Where a valid marker already exists and adding the adoption record fails,
initialization SHALL keep that marker, report the adoption and the recording
failure, and complete normally. This degraded path applies only to additive
evidence: where no valid marker exists, a failure to create one SHALL still fail
initialization because the repository was not initialized.

#### Scenario: A switch is recorded
- **WHEN** initialization replaces another schema
- **THEN** the marker afterwards names the replaced schema, the time of the adoption, and a digest of the rules block

#### Scenario: An installation with no switch records no adoption
- **WHEN** initialization creates a configuration where none existed
- **THEN** the marker carries no adoption record

#### Scenario: A marker predating this requirement stays valid
- **WHEN** a diagnosis reads a marker that carries no adoption record
- **THEN** it is read as a repository that never replaced a schema, and no fault is reported

#### Scenario: Additive adoption evidence cannot be written
- **WHEN** initialization switches a schema where a valid marker exists but cannot be extended with the adoption record
- **THEN** initialization succeeds, the existing marker remains unchanged, and the adoption report names the recording failure

#### Scenario: The record stays out of the user's configuration
- **WHEN** initialization records an adoption
- **THEN** the configuration differs from its prior contents only in the schema key

### Requirement: Initialization registers the attachment where the scaffold allows it
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, OUT-5, OUT-6, SIG-1, SIG-2, SIG-3, SIG-4, SIG-5, SIG-6, SIG-7

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
version control system, and initialization SHALL make it so by the same act,
using a rule that belongs to the clone alone and that no commit can carry. That
rule is not repository content and needs nobody else's agreement, so writing it
belongs to registering rather than to a separate thing the user must be asked
for. The entries it adds SHALL be reported among initialization's changes,
because a write the user cannot see is one they can neither audit nor undo.

Initialization MUST NOT modify ignore rules the version control system would
share with anyone else unasked, and MUST NOT register in the hope that nobody
commits the file. Where the path cannot be made unshareable by the clone's own rule —
because it is already tracked, or because a shared rule outranks the clone's —
initialization SHALL install the rest of the harness, refuse the registration,
and name the reason, the exact ignore entry that would make it possible, and the
flag that adds it. An explicit flag SHALL add the missing entries to the shared
rules and proceed, reporting them among its changes, because there they are
repository content exactly as the overlay is, and because a shared rule that
outranks the clone's is the one condition only shared content can answer.

The clone's rule SHALL be located by asking the version control system where it
belongs rather than by assuming a path, because the assumed path is not a
directory in a linked worktree or a submodule and its parent is not guaranteed to
exist anywhere. Every such question SHALL be asked about the directory
initialization was given, with any repository the caller's environment selects
disregarded: an answer about another repository would put the rule there and then
read it back as though it governed this one. Writing the rule SHALL preserve what
the user already put there, SHALL NOT follow a link out of the repository, and
SHALL express each entry so that it covers the path from wherever that rule is
read — which is not always the directory the user named. Re-running SHALL add
nothing it has already added.

Where the rule cannot be written — the directory is not a repository, the version
control system is unavailable or unintelligible, the target is not a file
initialization may write, or the write fails — initialization SHALL complete
without it and let the refusal above answer for the consequence. Making a path
unshareable is a service initialization performs where it can, never a
precondition it imposes: the harness is inert repository content, and installing
it has never required a repository or the tool that manages one.

A repository with no work tree is the one case that is not merely skipped. It has
nowhere for repository content to live, so no path that installs or maintains the
harness SHALL write into it — not the overlay, not the marker, and not an ignore
rule under any flag. That SHALL be refused with the reason named, before the
first write.

Where the directory is not a repository, no rule is needed and none can be
written, but the registration becomes committable the moment the directory
becomes one. Initialization SHALL state that and complete, because the exposure
that earns a refusal elsewhere must not arrive here in silence.

The participation marker carries the same exposure for a weaker reason: it is per
clone, and committing it would make one user's initialization a shared repository
fact. Its entry SHALL be written by the same act as the registration's. Where it
cannot be, initialization SHALL still complete and SHALL say so, because refusing
to install the harness over an ignore rule would be a disproportionate response
to a recoverable condition.

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
including removing a registration that belongs to an earlier arrangement. It MUST
NOT write an ignore rule outside the repository it was invoked on, including into
configuration that would govern every other repository the user clones: consent
to initialize one repository is not consent to configure the machine.

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
- **WHEN** initialization would register hooks in a per-user project settings file the version control system does not yet ignore, and the clone's own rule can cover it
- **THEN** that rule is written, the registration proceeds, the report names the entries among its changes, and no rule the repository could supply to someone else is modified

#### Scenario: A shared rule outranks the clone's rule
- **WHEN** initialization would register hooks in a path that a rule the repository shares keeps from being ignored, so the clone's own rule cannot cover it
- **THEN** the registration is refused, the rest of the harness is installed, and the report names the reason, the exact ignore entry, and the flag that adds it

#### Scenario: The project settings path is already tracked
- **WHEN** initialization would register hooks in a per-user project settings file the repository already tracks
- **THEN** the registration is refused with the reason named, because no ignore rule protects a tracked file

#### Scenario: The user asks for the ignore entries to be added
- **WHEN** initialization runs with the explicit flag that manages ignore rules
- **THEN** the missing entries are added to the rules the repository shares, the registration proceeds, and the report names the entries among its changes

#### Scenario: The repository's layout is not the assumed one
- **WHEN** initialization runs in a linked worktree or a submodule, where the assumed location of the clone's rule is not a directory
- **THEN** the rule is written where the version control system says it belongs, and the registration proceeds

#### Scenario: The clone's rule file already has content
- **WHEN** initialization writes the clone's rule into a file the user has already written in, including one whose last line has no terminator
- **THEN** every line the user wrote still means what it meant, both entries take effect, and a second initialization changes no byte

#### Scenario: The directory is not a repository
- **WHEN** initialization runs where the version control system is available and the directory is not under version control
- **THEN** the harness is installed, nothing is refused, no rule is written, and the report states that the registration becomes committable if the directory becomes a repository

#### Scenario: The version control system is unavailable
- **WHEN** initialization runs on a machine where the version control system cannot be executed
- **THEN** initialization completes and installs the harness, because making a path unshareable is a service it performs where it can and never a precondition

#### Scenario: The repository has no work tree
- **WHEN** initialization runs against a repository that has no work tree, whether because it is bare or because the directory named is the repository's own
- **THEN** it is refused with the reason named before the first write, and no ignore rule and no harness content are written into it, by any path including the explicit flag

#### Scenario: The environment selects another repository
- **WHEN** initialization runs where the caller's environment names a repository other than the directory it was given
- **THEN** every question is asked and every rule written about the directory it was given, and the registration is refused unless that directory's own rules cover the path

#### Scenario: The directory named is below the top of the work tree
- **WHEN** initialization runs on a directory inside a repository rather than at the top of its work tree
- **THEN** the entries written cover the paths under that directory, and no rule the repository shares is modified

#### Scenario: The clone's rule cannot be written
- **WHEN** the rule this clone would be given cannot be written, because the location is not a file initialization may write or the write fails
- **THEN** the harness is still installed, the report says so, and the registration is refused with the reason named rather than the command failing

#### Scenario: The marker's ignore entry is missing
- **WHEN** initialization writes the participation marker where the clone's rule cannot cover it
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
- **WHEN** any initialization path would write to user-level scaffold configuration, or would write an ignore rule outside the repository it was invoked on
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
