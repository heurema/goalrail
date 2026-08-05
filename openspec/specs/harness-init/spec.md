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
**Intent IDs:** OUT-1, OUT-2, OUT-9, SIG-1, SIG-2, SIG-11

Initialization SHALL establish everything a repository needs to declare Goalrail governance: the OpenSpec overlay selected by the project, the versioned Goalrail project declaration, the initial repository-owned governance policy, canonical human guidance, supported-agent bootstrap adapters, and the prepared shared-admission integration. Canon-owned content SHALL be materialized from copies carried inside the `gr` binary; repository-owned project identity and policy values SHALL be created once and thereafter preserved as owner content.

The declaration, policy, overlay, and bootstrap/admission files SHALL be repository content that the owner reviews and commits. Initialization SHALL report every created or changed path, the exact pinned OpenSpec invocation, which content is canon-owned or repository-owned, and that committing the project declaration is the act that makes future clones and worktrees managed. It MUST NOT use a per-clone or per-worktree marker as project identity.

The canonical overlay SHALL remain a materialized copy that can drift. OpenSpec configuration is repository-owned except for the selected schema key, so initialization SHALL preserve the project description and manage only the schema selection. Initialization MUST NOT run the checking CLI, require its runtime, create a commit, enable a remote required check, or claim protected enforcement.

Repeating initialization against identical content SHALL leave every project file byte-identical. A reserved declaration path with invalid or conflicting content SHALL fail before the first project write rather than be replaced or treated as a fresh unmanaged repository.

#### Scenario: A fresh repository is initialized
- **WHEN** the owner runs initialization in a worktree with no Goalrail declaration and no conflicting OpenSpec custom schema
- **THEN** the overlay, declaration, policy, human guidance, supported-agent adapters, and prepared admission integration are materialized, and the report names every path, ownership boundary, pinned planning invocation, and required commit

#### Scenario: Initialization is repeated
- **WHEN** initialization runs again against the exact project content it previously created
- **THEN** every project file is byte-identical and the report says no project initialization change is required

#### Scenario: The declaration is cloned
- **WHEN** initialized project content is committed and cloned or opened through a linked worktree
- **THEN** the checkout resolves the committed managed-project identity without another initialization command or marker

#### Scenario: The materialized overlay is checked in development
- **WHEN** the overlay carried in the binary is exercised by Goalrail's conformance suite
- **THEN** the pinned stock OpenSpec CLI validates the overlay and creates a change using the Goalrail schema without requiring that CLI during repository initialization

#### Scenario: Declaration path already conflicts
- **WHEN** initialization finds malformed, unsupported, linked, or owner-authored content at the reserved declaration path
- **THEN** it names the conflict and performs zero project writes rather than overwriting the governance claim

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
counted from the changes present in the repository. Where a repository-local
directory for that schema exists, the section SHALL say that it must remain
while the count is above zero and may be removed only when the count is zero.
Every change records the schema it was written under, and removing a schema
directory those changes still name would break them; whether removal is safe is
therefore a count, not an opinion. Where no repository-local directory exists,
as with a schema supplied only by the stock package, the section SHALL state
that no removal action is available and MUST NOT direct the user toward files in
the installed package. Initialization MUST NOT remove, move or archive a schema
directory in any case.

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
- **AND** the report states that no repository-local schema directory exists and offers no removal action

#### Scenario: The overlay on disk differs from the embedded copy
- **WHEN** a switch happens in a repository whose overlay schema file was edited and is therefore left alone
- **THEN** the comparison describes that file rather than the copy embedded in the binary

#### Scenario: The rules are disclosed without being judged
- **WHEN** a switch replaced a schema in a repository whose configuration carries rules
- **THEN** the report states their count, reproduces each rule verbatim, states that they were present when the schema was replaced, and states that Goalrail neither interprets nor edits them
- **AND** the report renders no verdict on any individual rule and the configuration's rules are byte-identical afterwards

#### Scenario: A multiline rules value is disclosed completely
- **WHEN** a top-level `rules:` value begins with `{`, `[`, `|` or `>` and continues on later lines before another top-level key
- **THEN** the report reproduces every source line belonging to that value and excludes the following top-level key
- **AND** editing retained rule content changes the recorded digest even when the rules shape cannot be counted confidently

#### Scenario: Whether the replaced schema may be removed is counted
- **WHEN** a switch replaced a schema that changes in the repository still name
- **THEN** the report states how many changes name it and that its directory must remain, and the directory is present and unmodified afterwards

#### Scenario: Nothing still names the replaced schema
- **WHEN** a switch replaced a schema whose repository-local directory exists and no change in the repository names it
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

Extending an existing marker SHALL publish the complete replacement atomically.
Any failure before publication SHALL leave the previous marker byte-identical
and valid; initialization MUST NOT truncate or rewrite the live marker in place.

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

#### Scenario: Existing marker replacement fails during publication
- **WHEN** replacement marker bytes have been written but cannot be published over an existing valid marker
- **THEN** initialization succeeds through the degraded path, and the previous marker remains byte-identical and valid

#### Scenario: The record stays out of the user's configuration
- **WHEN** initialization records an adoption
- **THEN** the configuration differs from its prior contents only in the schema key

### Requirement: Initialization registers the attachment where the scaffold allows it
**Intent IDs:** OUT-2, OUT-3, OUT-4, OUT-10, SIG-2, SIG-3, SIG-12

Initialization MAY attach a detected or explicitly selected supported scaffold only within the exact local mutations disclosed by the invoking setup or initialization action. Project identity SHALL remain established by committed content whether attachment succeeds, is declined, requires a scaffold trust step, or no supported scaffold is present.

Where a scaffold permits repository-scope registration in a non-executable or user-approved settings surface, Goalrail SHALL use that supported scope. Where registration is user-local, Goalrail SHALL write it only under explicit authorization that enumerates the exact path, executable, events, rollback, and any clone-local ignore entries. It MUST NOT write or simulate provider trust records, silently change user-wide or system-wide configuration, or make one user's executable registration committable for teammates.

Clone-local ignore entries and repository-relative settings paths SHALL be resolved through Git for the exact target worktree. Writes SHALL preserve unrelated content, refuse unsafe links and path escapes, and remain idempotent. A shared ignore or instruction change MUST be part of the enumerated repository-content plan; it MUST NOT be smuggled into local attachment consent.

The registered events SHALL match session-open and session-end semantics rather than a stop-like event that fires every turn. Re-running SHALL repair only stale registrations Goalrail can prove it owns, leave correct registrations byte-identical, and leave foreign handlers untouched. A repository with no work tree SHALL be refused before any project or attachment write.

#### Scenario: A scaffold is detected during authorized setup
- **WHEN** the exact setup plan authorizes attachment for a detected supported scaffold
- **THEN** Goalrail applies only the enumerated local registration, reports any required provider trust action, and keeps project identity independent of attachment health

#### Scenario: No scaffold is detected
- **WHEN** initialization creates project content on a machine with no supported scaffold
- **THEN** project initialization completes, the report names local setup as pending, and no scaffold configuration is guessed or changed

#### Scenario: User-local configuration was not authorized
- **WHEN** initialization would need to modify a user-local registration absent an exact consented setup plan
- **THEN** it leaves user configuration unchanged and reports the separate setup action

#### Scenario: Registration path could be committed accidentally
- **WHEN** a local executable registration would land in tracked or otherwise shareable project content without being an explicit safe bootstrap adapter
- **THEN** Goalrail refuses that registration and names the exact conflict rather than relying on the user not to commit it

#### Scenario: Existing foreign handler shares an event
- **WHEN** the target scaffold event already contains a handler Goalrail did not create
- **THEN** Goalrail preserves the foreign handler and adds, repairs, or refuses only its own bounded registration according to the disclosed plan

#### Scenario: Re-initialization meets a correct registration
- **WHEN** the exact Goalrail registration already points to the planned durable executable and current events
- **THEN** re-running leaves it byte-identical

#### Scenario: Provider trust is required
- **WHEN** the scaffold does not run the registration until the user completes its own trust step
- **THEN** Goalrail reports that state and next action without writing, reproducing, or pre-approving the trust record

#### Scenario: Repository has no work tree
- **WHEN** initialization targets a bare repository or another Git repository with no work tree
- **THEN** it performs zero project and attachment writes and names the unsupported condition

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
