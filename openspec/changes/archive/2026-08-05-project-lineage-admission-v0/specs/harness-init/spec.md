## MODIFIED Requirements

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
