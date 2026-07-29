## ADDED Requirements

### Requirement: Attachment health is reportable and trust is never forged
**Intent IDs:** OUT-2, OUT-3, SIG-2, SIG-3

A registered hook does not run until the scaffold's own trust step has been
completed by the user, so registration alone does not make an attachment work.
Goalrail SHALL therefore report attachment health on demand, distinguishing at
least: the scaffold is not connected; this repository is not initialized; the
hooks are registered but not yet trusted; the attachment is working. Each state
that is not working SHALL name the next action.

Goalrail MUST NOT write, compute, or reproduce the scaffold's hook-trust
records, and MUST NOT automate, pre-approve, or simulate the user's approval by
any means. Trust is a standing consent to run a command in every one of the
user's sessions; it belongs to the user, granted through the scaffold's own
surface. That this is technically possible, and is practised by other
integrations, does not make it permissible here: it would depend on private
implementation details and would convert an explicit consent into an assumed
one.

Reporting trust state is observation and is permitted. Reading is not writing.

#### Scenario: Attachment is not yet trusted
- **WHEN** the user asks for attachment health after connecting but before trusting the hooks
- **THEN** the report says the hooks are registered but not trusted, and names the next action

#### Scenario: Repository is not initialized
- **WHEN** the user asks for attachment health in a directory Goalrail was never initialized in
- **THEN** the report says so and names the next action, without treating the directory as participating

#### Scenario: Attachment is working
- **WHEN** the scaffold is connected, the repository initialized, and the hooks trusted
- **THEN** the report says the attachment is working and asks nothing further of the user

#### Scenario: Trust would be written by Goalrail
- **WHEN** any Goalrail path would write, compute, or reproduce a scaffold trust record, or otherwise approve hooks on the user's behalf
- **THEN** that is prohibited, regardless of technical feasibility

## MODIFIED Requirements

### Requirement: Connection is explicit, consented, and reversible
**Intent IDs:** OUT-1, OUT-4, SIG-1, SIG-4

Goalrail SHALL attach to a scaffold through one explicit connection command
that registers persistent session hooks in the scaffold's user configuration.
Registration MUST NOT happen without the user's explicit consent, and a
matching disconnection command SHALL remove everything the connection added,
leaving no residue. Both commands SHALL be safe to repeat.

Registration alone does not make the attachment act. The scaffold requires the
user to review and trust the exact hook definition before it runs, and records
that trust against the definition's current form, so a changed command needs
review again. Connection SHALL therefore disclose this: it states that the
trust step is required, names the surface where the user performs it, and says
plainly that the attachment does nothing until then. Silence here is the worst
available outcome — the user connects, works, observes nothing, and reasonably
concludes the product is broken.

Attachment is registered at user scope, which means the hook is invoked for
every session the user starts anywhere, and confining action to initialized
repositories is enforced by Goalrail's own first act rather than by absence.
This is a recorded limitation, not a preference: registering inside the
repository would make the boundary structural, and that arrangement is
externally blocked — project-local hooks are silently skipped with no trust
prompt even in a trusted project, and the session-opening event has already
passed by the time a user could grant trust. Should that change, this
limitation is the first thing to revisit.

After connection and trust, the user runs no Goalrail command per task:
sessions are started, driven, and ended entirely by the user in their own
scaffold.

#### Scenario: User connects once
- **WHEN** the user runs the connection command and consents
- **THEN** the persistent session hooks are registered in the scaffold's user configuration and no further Goalrail command is needed for ordinary work

#### Scenario: Connection discloses the trust step
- **WHEN** connection registers the hooks
- **THEN** its output states that the scaffold requires the user to review and trust them, names the surface for doing so, and says the attachment does not act until then

#### Scenario: User disconnects
- **WHEN** the user runs the disconnection command
- **THEN** every registration the connection added is removed and no Goalrail residue remains in the scaffold configuration

#### Scenario: Commands are repeated
- **WHEN** connection or disconnection runs a second time
- **THEN** the outcome is unchanged and nothing is duplicated or broken

#### Scenario: Configuration would change without consent
- **WHEN** any Goalrail path would modify scaffold user configuration outside the explicit consented connection command
- **THEN** that modification is refused
