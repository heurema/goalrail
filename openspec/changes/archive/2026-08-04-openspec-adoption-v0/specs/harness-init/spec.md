## MODIFIED Requirements

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

## ADDED Requirements

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
