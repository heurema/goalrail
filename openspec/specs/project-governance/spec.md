# project-governance Specification

## Purpose

Define the committed declaration, policy, bootstrap, drift, and migration contracts that make Goalrail governance portable across clones and worktrees.
## Requirements
### Requirement: A committed declaration establishes managed-project identity
**Intent IDs:** OUT-4, OUT-8, SIG-4, SIG-5, SIG-9

A Goalrail-managed repository SHALL carry one versioned declaration at the canonical repository-root path. The declaration SHALL have a stable schema identifier, one immutable project identifier, the governance contract version, and stable path-and-digest references to the committed policy, supported bootstrap contract, and setup profile. It MUST contain no credential, user identity, provider session state, absolute machine path, or checkout-local attachment state.

Project discovery SHALL begin at the Git worktree root and SHALL use the committed declaration rather than `.git`, a user directory, or a per-worktree marker. The same bytes SHALL therefore identify the same project in every clone and linked worktree. Deleting or changing a local legacy marker MUST NOT make a declared project unmanaged.

Presence of the canonical declaration path is a governance claim even when its bytes are malformed, drifted, or unsupported. Such a repository SHALL be classified as declared-invalid and fail closed; it MUST NOT fall through to the unmanaged behavior used for repositories that never declared Goalrail. For every evaluated revision, the policy, bootstrap, and setup-profile bytes SHALL exist at their declared repository-relative paths, be canonical where their artifact contract requires canonical encoding, and match the declaration's exact digests and project identity bindings.

Outside the explicit bootstrap or migration state where the base has no valid declaration, the head declaration's project identifier SHALL equal the base declaration's project identifier. A self-consistent replacement declaration and policy with a different identifier MUST NOT migrate or rename the project implicitly.

#### Scenario: Declaration reaches clones and worktrees
- **WHEN** a valid Goalrail project declaration and every bound governance artifact are committed and the repository is cloned or given a linked worktree
- **THEN** every checkout resolves the same project identifier and reports the repository as managed without a local initialization marker

#### Scenario: Declaration is malformed
- **WHEN** the canonical declaration path exists but its content is malformed, unsupported, substituted, or not a regular file inside the worktree
- **THEN** project discovery reports declared-invalid and managed work fails closed rather than treating the repository as unmanaged

#### Scenario: Bound governance artifact differs
- **WHEN** the policy, bootstrap, or setup-profile bytes at an evaluated revision are missing or do not match the path, digest, schema, or project binding recorded by the declaration
- **THEN** that revision's governance is invalid and admission fails closed

#### Scenario: Non-bootstrap range changes project identity
- **WHEN** a range with valid base governance contains a self-consistent head declaration and policy with a different project identifier
- **THEN** admission rejects the range as invalid before evaluating it under either identity

#### Scenario: Explicit bootstrap establishes identity
- **WHEN** the base has no project declaration and the head introduces one complete valid governance set through the explicit bootstrap or migration path
- **THEN** the head project identifier may establish the new managed-project identity under the distinct bootstrap classification

#### Scenario: Legacy marker is absent
- **WHEN** a valid declared project is opened in a checkout with no legacy ambient marker
- **THEN** managed-project identity remains true and only local attachment readiness may be absent

#### Scenario: Unrelated repository is inspected
- **WHEN** neither a valid declaration nor the reserved declaration path exists at the repository root
- **THEN** Goalrail treats the repository as unmanaged and performs no managed-project observation or write

### Requirement: The repository carries one bootstrap contract for humans and supported agents
**Intent IDs:** OUT-2, OUT-5, OUT-10, SIG-2, SIG-5, SIG-12

A managed repository SHALL carry concise canonical bootstrap guidance and a versioned list of supported agent adapters. Each supported adapter SHALL cause the agent to inspect the Goalrail declaration and local readiness before material project work. If required tooling or attachment is absent, the adapter SHALL emit the standardized `GOALRAIL_SETUP_REQUIRED` state, perform zero project code writes, and direct the agent to the consented setup flow.

Agent-specific instruction surfaces SHALL be thin adapters to the same canonical guidance. They MUST NOT copy semantic intent, define a second materiality policy, select a parallel spec authority, or claim that their provider owns the project flow. A human entrypoint SHALL describe the same setup and admission boundary.

Initialization or update MUST preserve unrelated owner-authored instruction content. Where a supported instruction surface cannot be created or reconciled without overwriting or ambiguously competing with existing instructions, the operation SHALL stop for that path, name the conflict, and leave the existing bytes unchanged.

#### Scenario: Supported agent receives a feature request on a clean machine
- **WHEN** a supported adapter starts in a declared project whose required local setup is absent
- **THEN** it reports `GOALRAIL_SETUP_REQUIRED`, changes no project code, and presents the setup planning action before interpreting the feature as implementation authority

#### Scenario: Two supported agents enter one project
- **WHEN** Codex and Claude adapters read the same managed repository
- **THEN** both resolve the same declaration, intent authority, setup contract, and admission command without creating provider-specific canonical state

#### Scenario: Existing instructions conflict
- **WHEN** installing or updating a thin adapter would overwrite or create an ambiguous competing rule in an owner-authored instruction file
- **THEN** Goalrail leaves that file byte-identical and reports the exact conflict for owner resolution

#### Scenario: Human opens the project without a supported agent
- **WHEN** a person follows the repository's canonical bootstrap entrypoint
- **THEN** the documented flow names managed identity, setup consent, confirmed intent, lineage verification, and shared admission consistently with supported adapters

### Requirement: Governing project content is versioned and drift-aware
**Intent IDs:** OUT-4, OUT-8, SIG-4, SIG-5, SIG-9, SIG-10

The declaration SHALL distinguish canon-owned bootstrap material from repository-owned governance values. Canon-owned files and managed blocks SHALL have deterministic expected digests for their declared contract version. Repository-owned project identity, materiality rules, exception policy, setup choices, and owner-maintained prose SHALL never be silently replaced by initialization, diagnosis, setup, or update.

A material governance amendment SHALL change the governing contract version or the exact declaration reference and digest for every amended policy, bootstrap, or setup-profile artifact and SHALL itself require lineage admission. Admission-time Git collection SHALL read and verify every declaration-bound artifact from the immutable base and head revisions before either governance state is accepted. A checkout-local flag, environment variable, missing hook, or removed legacy marker MUST NOT suspend project governance or mint conformant evidence.

#### Scenario: Canon-owned bootstrap drifts
- **WHEN** a managed bootstrap file or block differs from every canon known to the installed binary or from the digest referenced by the evaluated declaration
- **THEN** diagnosis reports the exact drift, admission treats the governance state as invalid, and no automatic command overwrites it without the existing explicit discard authority

#### Scenario: Repository-owned policy changes
- **WHEN** project policy bytes differ from the version referenced by the declaration
- **THEN** diagnosis reports a governance mismatch and admission fails until an authorized policy amendment binds the new identity

#### Scenario: Setup profile changes without declaration update
- **WHEN** setup-profile bytes differ from the path or digest referenced by the declaration
- **THEN** admission reports invalid governance rather than accepting the stale declaration

#### Scenario: Bootstrap changes without declaration update
- **WHEN** bootstrap bytes differ from the path or digest referenced by the declaration
- **THEN** admission reports invalid governance rather than accepting the stale declaration

#### Scenario: Local opt-out is attempted
- **WHEN** a user removes local hooks, sets a local flag, or deletes a legacy marker while the committed declaration remains
- **THEN** the project remains managed and no conformant verdict can be derived from the local opt-out

### Requirement: Existing checkout-local projects migrate once at project scope
**Intent IDs:** OUT-9, SIG-11

Goalrail SHALL detect a v0.1.8-style overlay or local participation marker without a committed declaration and report an explicit project-migration action. Migration SHALL create the declaration, policy, and bootstrap artifacts without rewriting existing intent, OpenSpec, WorkSpec, run, review, or receipt evidence. It SHALL report which new files are repository content that must be reviewed and committed.

Migration MUST NOT silently run from diagnosis, hooks, update, or an ordinary feature request. Repeating the migration against identical project content SHALL be byte-stable. After the declaration is committed, future clones and worktrees SHALL need only local setup, never project migration or `gr init` to discover identity.

#### Scenario: Current v0.1.8 project is migrated
- **WHEN** the owner explicitly migrates a project with the current overlay and legacy local marker but no declaration
- **THEN** Goalrail creates the project-scoped artifacts, preserves all existing evidence bytes, and reports the commit boundary

#### Scenario: Migration is repeated
- **WHEN** the explicit migration runs again without an intervening governance change
- **THEN** no project file changes and the report says the project declaration already represents the migration

#### Scenario: Fresh clone follows migration
- **WHEN** the migrated artifacts are committed and cloned to a machine with no legacy marker
- **THEN** the clone is recognized as managed and enters setup readiness rather than project initialization

