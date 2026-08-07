## MODIFIED Requirements

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
