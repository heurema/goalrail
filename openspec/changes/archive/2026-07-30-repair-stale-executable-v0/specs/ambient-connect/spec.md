## MODIFIED Requirements

### Requirement: Connection is explicit, consented, and reversible
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, OUT-5, OUT-6, SIG-1, SIG-2, SIG-3, SIG-4, SIG-5, SIG-6

Goalrail SHALL attach to a scaffold through one explicit connection command
that registers persistent session hooks in the scaffold's user configuration.
Registration MUST NOT happen without the user's explicit consent, and a
matching disconnection command SHALL remove everything the connection added,
leaving no residue. Both commands SHALL be safe to repeat.

Where a scaffold distinguishes which session-start occurrence a hook applies
to, registration SHALL name only the occurrences that open a session, and
SHALL NOT register a form that fires on every occurrence. The single-delivery
rule is a property of the announcement, so it MUST hold on every supported
scaffold — including one whose transport does not inspect the occurrence and
therefore cannot enforce it after the fact. Removal SHALL cover every
occurrence the connection registered, and MUST leave entries it did not add
untouched.

Whether a connection is present SHALL be judged by what its registration
names, not only by the registration being recognisable as Goalrail's. A
registration whose handler names an executable other than the one connection
was invoked with is not the registration this connection would write, and may
name a binary that no longer exists, so it SHALL be treated as needing repair
rather than reported as already present. Where that executable is still
runnable, the repair happens anyway and costs the user the review step the
disclosure names. That cost is accepted deliberately: the alternative is a
remedy a health report prescribes which sometimes does nothing, and a
connection command that cannot make the attachment use the binary it was
invoked from.

Repair SHALL replace rather than accompany. The connection SHALL NOT leave two
registrations for the same event, because a second registration duplicates
every hook invocation and leaves a removal that finds only one of them. A
registration that already names the current executable SHALL be left exactly as
it is, and repair MUST NOT alter a handler the connection did not add, change a
foreign entry's occurrence, or widen a repaired session-start entry beyond the
occurrence that opens a session.

Where a health report names re-running connection as the remedy for a defect,
connection SHALL be able to repair that defect. A report that prescribes a
command which cannot perform the repair is worse than no report: the user
follows correct advice, observes no change, and has no remaining move.

Registration alone does not make the attachment act. The scaffold requires the
user to review and trust the exact hook definition before it runs, and records
that trust against the definition's current form, so a changed command needs
review again. Connection SHALL therefore disclose this: it states that the
trust step is required, names the surface where the user performs it, and says
plainly that the attachment does nothing until then. Silence here is the worst
available outcome — the user connects, works, observes nothing, and reasonably
concludes the product is broken.

A repair changes the hook definition, so it SHALL carry that same disclosure and
SHALL NOT report the repaired attachment as active on the strength of a trust
record that was made against the command just replaced. A repair is the only
moment at which Goalrail knows a trust record no longer matches, because it
made the change itself; reading the record cannot establish this, since
reproducing the form it is recorded against is prohibited.

The disclosure SHALL match what was actually established for that scaffold.
Where the gate was observed, it is stated. Where it was not, the disclosure
SHALL say the behaviour is unverified instead of asserting a requirement:
inventing an obstacle sends the user hunting for a screen that may not exist.
This applies to the repair disclosure exactly as it applies to the first
connection.

Connection SHALL NOT describe an already-trusted attachment as inert. Repeating
the command on a working attachment reports it as working.

Attachment is registered at user scope, which means the hook is invoked for
every session the user starts anywhere, and confining action to initialized
repositories is enforced by Goalrail's own first act rather than by absence.
This is a recorded limitation rather than a preference, and it SHALL be
attributed to the scaffold whose evidence produced it: on the first supported
scaffold, registering inside the repository is externally blocked, because
project-local hooks are silently skipped with no trust prompt even in a trusted
project and the session-opening event has already passed by the time a user
could grant trust. The second supported scaffold does not share that
constraint — its project-level hooks are an ordinary settings layer that merges
with user settings — so a scaffold-specific route remains open there and is the
first thing to revisit. A true observation about one provider MUST NOT be
recorded as a property of attachment in general.

The second supported scaffold's end-to-end behaviour has not been observed in a
live session: its announcement delivery, question retention, and whether it
prompts for any approval remain unverified, and no requirement here may be read
as claiming otherwise. Its registration shape rests on the provider's published
matcher contract together with an existing working configuration that agrees
with it.

After connection and trust, the user runs no Goalrail command per task:
sessions are started, driven, and ended entirely by the user in their own
scaffold.

#### Scenario: User connects once
- **WHEN** the user runs the connection command and consents
- **THEN** the persistent session hooks are registered in the scaffold's user configuration and no further Goalrail command is needed for ordinary work

#### Scenario: A scaffold distinguishes session-start occurrences
- **WHEN** connection registers session-start hooks on a scaffold whose registration names which occurrence applies
- **THEN** it registers only the occurrences that open a session, and registers no form that fires on every occurrence

#### Scenario: An earlier registration fires on every occurrence
- **WHEN** connection finds its own session-start registration present but not scoped to the opening occurrence
- **THEN** it replaces that registration with the scoped form rather than reporting the attachment as already present, so a user who connected with an earlier version can repair it

#### Scenario: Removal spans every registered occurrence
- **WHEN** the user disconnects a scaffold whose session-start hooks were registered per occurrence
- **THEN** every entry the connection added is removed, and any entry it did not add for the same event survives

#### Scenario: The registration names a moved executable
- **WHEN** connection finds its own registration present but naming an executable other than the one it was invoked with
- **THEN** it reports the attachment as needing repair rather than as already present

#### Scenario: A repair replaces the stale registration
- **WHEN** connection repairs a registration that names a moved executable
- **THEN** exactly one registration per event remains, naming the current executable, and no second registration or removal residue is left behind

#### Scenario: The registration already names the current executable
- **WHEN** the user reruns the consented connection command and every registered handler already names the executable it was invoked with
- **THEN** the scaffold configuration is left byte-identical

#### Scenario: A repair meets a handler it did not add
- **WHEN** a repair replaces a stale registration for an event that also carries a handler the connection did not add
- **THEN** the foreign handler survives unchanged, including its occurrence, and only the stale registration is replaced

#### Scenario: A repair meets an event that is already current
- **WHEN** a repair replaces a stale registration for one event while another registered event already names the current executable
- **THEN** the other event's handlers are left unchanged, so a part of the registration that is correct does not lose its review record

#### Scenario: A repaired session-start entry stays scoped
- **WHEN** a repair replaces a session-start registration on a scaffold that distinguishes occurrences
- **THEN** the replacement names only the occurrence that opens a session, whatever shape the stale registration had

#### Scenario: A health remedy names connection
- **WHEN** a health report tells the user to re-run connection to repair a defect it detected
- **THEN** connection performs that repair, rather than reporting the attachment as already present and changing nothing

#### Scenario: A limitation is recorded from one scaffold's evidence
- **WHEN** a scope limitation is recorded because one scaffold blocks a stronger arrangement
- **THEN** the record names that scaffold rather than stating the limitation as a property of attachment in general

#### Scenario: A scaffold has not been exercised live
- **WHEN** a supported scaffold's end-to-end behaviour has not been observed in a live session
- **THEN** that is recorded, and no requirement is read as claiming the scaffold is verified

#### Scenario: Connection discloses the trust step
- **WHEN** connection registers the hooks
- **THEN** its output states that the scaffold requires the user to review and trust them, names the surface for doing so, and says the attachment does not act until then

#### Scenario: A repair discloses that review applies again
- **WHEN** connection replaces a stale registration
- **THEN** its output states that the hook definition changed and needs the scaffold's review step again, names the surface for it, and does not report the attachment as active on the strength of the existing trust record

#### Scenario: The scaffold's trust behaviour is unverified
- **WHEN** connection registers or repairs hooks on a scaffold whose trust gate has not been observed
- **THEN** the disclosure says the behaviour is unverified rather than asserting a mandatory approval step

#### Scenario: Connection is repeated on a working attachment
- **WHEN** the user reruns the connection command on an attachment that is already registered and trusted
- **THEN** the outcome reports it as active rather than describing it as not yet active

#### Scenario: User disconnects
- **WHEN** the user runs the disconnection command
- **THEN** every registration the connection added is removed and no Goalrail residue remains in the scaffold configuration

#### Scenario: Commands are repeated
- **WHEN** connection or disconnection runs a second time
- **THEN** the outcome is unchanged and nothing is duplicated or broken

#### Scenario: Configuration would change without consent
- **WHEN** any Goalrail path would modify scaffold user configuration outside the explicit consented connection command
- **THEN** that modification is refused
