# ADR-0025 — Provider credential storage boundary

Status: accepted
Date: 2026-05-05

## Context

ADR-0024 accepts `VcsConnection` as the future provider-neutral VCS connection
and metadata-discovery boundary. It also keeps raw provider tokens out of
`VcsConnection`, keeps `RepoBinding` separate from checkout permission, keeps
`RepositoryRecord` deferred, and preserves customer-hosted runner
compatibility.

`docs/ops/VCS_BACKEND_IMPLEMENTATION_SEQUENCING.md` makes provider
credential/token storage the next required boundary before any VCS schema, API,
OAuth, provider client, repository metadata listing, checkout, runner, gate, or
proof work starts.

GitLab is the first provider candidate, but advisory GitLab research found a
security mismatch: `read_api` appears useful for private metadata discovery,
while GitLab also documents repository file access through `read_api` on the
Repository Files API. Goalrail therefore needs a credential boundary that is
stricter than provider OAuth scopes alone.

Goalrail currently has self-hosted Installation / Organization / Project /
RepoBinding persistence, password credential/session auth, CLI login, and
metadata-only repository-context init. It does not have provider OAuth,
provider tokens, provider credential storage, `VcsConnection` schema/API,
provider clients, repository metadata APIs, checkout, runner, gate, or proof for
this boundary.

This ADR is documentation-only.

## Decision

Goalrail accepts a provider credential storage boundary for future
provider-backed VCS metadata discovery.

Raw provider credentials are secrets. They are not `VcsConnection`,
`RepoBinding`, `RepositoryRecord`, checkout eligibility, or proof evidence by
themselves.

For future provider-mediated metadata discovery, provider credentials may be
stored only in a server-side credential store for the running Goalrail
Installation, and only after a later implementation slice provides encrypted
storage, redaction, refresh, revocation, deletion, retention, and audit
behavior that conforms to this ADR.

Self-hosted deployments own their provider credential encryption keying and
operational custody. Future SaaS deployments may use service-owned keying for
service-managed provider connections. Customer-hosted runners and
customer-owned runtime setups may keep repository credentials entirely outside
the Goalrail control plane.

Goalrail cloud or a Goalrail-hosted control plane must not be required to hold
repository credentials for all repository work. Customer-hosted runner
compatibility remains first-class.

Future code may create credentialless `pending_setup` `VcsConnection` records
as non-secret setup state. Such records are not active, cannot list provider
metadata, cannot authorize checkout, and must expire or be cleanable. Active
provider metadata discovery requires successful provider authorization and an
accepted credential storage implementation.

## Definitions

### Provider credential

A provider credential is any secret or provider-issued authorization artifact
that can authenticate to, refresh access to, or grant repository/provider
permissions.

Provider credentials include:

- OAuth grants and authorization codes
- OAuth access tokens
- OAuth refresh tokens
- provider app/client credentials, such as OAuth client secrets
- provider installation tokens, such as future app installation access tokens
- deploy keys, SSH private keys, and key material
- checkout credentials for a bounded runner job
- runtime/customer-owned repository credentials used outside the Goalrail
  server

Provider credentials do not include non-secret display metadata such as
provider kind, redacted account reference, normalized provider instance
reference, connection state, or repository full name. Some of that metadata can
still be sensitive customer information and must be handled carefully.

### VcsConnection

`VcsConnection` is a provider-neutral domain relationship and metadata-discovery
boundary. It may reference credential posture, but it is not a token, not a
credential, not a provider installation token, not a deploy key, and not a
checkout credential.

### CheckoutCredential

`CheckoutCredential` remains a future runner/provider adapter artifact. It is
not accepted as a core MVP object by this ADR. Any future checkout credential
must be scoped to a bounded runner job where possible and must follow
ADR-0008's runner boundary.

## Credential Ownership and Storage Boundary

Goalrail uses a split custody model:

- provider-mediated metadata discovery can use server-side encrypted credential
  storage when an Organization explicitly authorizes that connection
- customer-hosted runner checkout can use customer-owned credentials inside
  customer infrastructure without storing those credentials in the Goalrail
  server
- future Goalrail-hosted runner checkout may use short-lived checkout
  credentials only through a separate runner/checkout boundary
- provider app/client credentials are deployment secrets, not domain objects
  and not repository files

Self-hosted mode:

- the running self-hosted Installation owns operational custody of provider
  credentials it stores
- encryption keys are deployment-owned
- operators are responsible for backup, restore, rotation, and deletion
  practice once a later implementation exists
- self-hosted code paths must remain organization-aware and must not collapse
  `Organization` into `Installation`

Future SaaS mode:

- service-managed provider credentials may be protected by service-owned keying,
  such as a managed KMS or equivalent later decision
- service operator visibility must remain redacted by default
- SaaS tenancy must not make provider accounts equivalent to Goalrail
  Organizations

Customer-hosted runner / customer-owned credential mode:

- repository credentials may remain entirely outside the Goalrail server
- Goalrail may receive bounded metadata, receipts, and artifacts
- this path must remain possible even when no provider credential is stored in
  Goalrail

This ADR does not choose a database schema, key-management provider, config
surface, or environment variable name.

## Non-Storage Surfaces

Provider credentials must not be stored in:

- browser `localStorage`
- browser `sessionStorage`
- cookies
- IndexedDB
- frontend bundles
- repo files
- local project markers such as `.goalrail/project.yml`
- docs
- generated fixtures
- logs
- unredacted traces
- screenshots
- PR bodies
- Git history, except explicit encrypted server-side operational storage if a
  later implementation accepts it; committed fixtures may contain only
  synthetic non-secret values

Provider credentials must also not be embedded in issue bodies, review comments,
deployment manifests, workflow files, sample configs, screenshots, copied curl
commands, local marker repair commands, or docs-check reports.

## Encryption and Key Boundary

Before any provider token persistence exists, a later implementation must
define:

- encryption-at-rest mechanism for provider credential values
- authenticated encryption or equivalent integrity protection
- key source and key-loading boundary
- key rotation behavior
- backup and restore implications
- deletion behavior for ciphertext and key material
- development/test behavior that does not introduce real secrets
- how encrypted credential rows are associated with `Installation`,
  `Organization`, and future `VcsConnection` state without making credentials
  domain objects

Minimum expectation:

- raw access tokens, refresh tokens, OAuth grants, app secrets, deploy keys, and
  checkout credentials are never persisted as plaintext
- plaintext exists only in process memory for the minimum operation needed
- plaintext is never logged, returned in API responses, rendered in HTML, or
  written to repo files
- credential metadata is stored separately from encrypted secret material where
  practical

Self-hosted keying is deployment-owned. Future SaaS keying is service-owned.
Customer-hosted runner credentials can remain customer-owned and outside the
Goalrail server.

This ADR does not implement encryption.

## Redaction and Logging

Safe to log, subject to customer metadata sensitivity:

- Goalrail connection id
- Goalrail Organization id
- provider kind
- redacted provider account reference
- redacted provider instance reference
- provider connection state transition
- adapter operation name from an allowlist, such as `list_projects`
- HTTP status class or provider error category
- rate-limit category
- correlation id
- timestamp

Must never be logged or emitted unredacted:

- OAuth access tokens
- OAuth refresh tokens
- OAuth grants or authorization codes
- OAuth client secrets
- PKCE code verifiers
- provider installation tokens
- deploy key private material
- SSH private keys
- Authorization headers
- Set-Cookie headers
- full provider response bodies
- repository file contents
- raw provider request URLs if they contain credentials or sensitive query
  parameters
- unredacted stack traces that include request headers or provider payloads

Token identifiers:

- future code may use one-way token fingerprints for equality checks or audit
  correlation
- token fingerprints must not be usable as bearer credentials
- token hashes must use a boundary chosen by a later implementation, such as a
  keyed hash or equivalent server-side secret context
- last-four-style display is not required and should be avoided by default; if a
  later implementation allows it, it must be derived only for operator display
  and never treated as a credential verifier

Provider errors and fixtures:

- provider errors must be normalized before user display
- provider response snippets must be redacted before logs, traces, tests, docs,
  or PR output
- generated fixtures must contain metadata only and no token-shaped real
  secrets
- tests must assert redaction for token-bearing paths before live provider
  credential handling is merged

## Refresh, Rotation, and Concurrency

GitLab advisory research states that GitLab OAuth access tokens expire after
two hours and refresh returns new tokens while invalidating the existing access
token and refresh token.

Any future provider token refresh implementation must therefore be
single-writer or transactionally safe.

Required future behavior:

- one refresh operation owns a connection/token version at a time
- refresh compares the stored token version or equivalent concurrency guard
  before replacing credentials
- storing the new token set and invalidating the old local token set must be
  atomic from Goalrail's point of view
- concurrent refresh attempts must not overwrite newer refresh tokens with stale
  ones
- refresh errors must not leak raw provider response bodies or credentials

Failure handling:

- stale refresh token: reload current credential state; if a newer successful
  refresh exists, use the new state; otherwise move the connection toward
  `needs_reconnect` or equivalent blocked state
- concurrent refresh conflict: retry only if the retry policy is explicit and
  bounded; otherwise report a safe retryable error
- provider-revoked token: stop metadata refresh and move connection toward
  `revoked` or `needs_reconnect`
- expired access token with valid refresh token: refresh through the
  single-writer path
- unavailable provider: preserve current credential state, report
  `unavailable` or equivalent transient state, and avoid destructive deletion
- repeated refresh failure: stop automatic retries according to later policy and
  require reconnect

This ADR does not implement refresh or token rotation.

## Revocation, Reconnect, Disable, Deletion, and Retention

Revocation and disconnect affect provider metadata discovery and future
provider-mediated checkout eligibility only.

If provider credentials are revoked, disconnected, disabled, expired, or
deleted:

- metadata refresh must stop or report unavailable/stale state
- future provider-mediated checkout eligibility must be blocked or require
  reconnect
- existing `RepoBinding` records remain reviewable as historical or declared
  metadata unless policy explicitly disables them
- customer-hosted runner flows may remain possible if policy allows them and
  they do not require Goalrail server-side provider authorization
- Goalrail Projects, Goals, Contracts, WorkItems, Decisions, and Proofs must
  not be silently deleted
- historical proof remains historical evidence and is not rewritten

Reconnect:

- reconnect creates or refreshes provider authorization for the same Goalrail
  Organization context when policy allows
- reconnect does not mutate historical proof
- reconnect may update provider account metadata and future connection state
  through server-owned transitions

Disabled by policy:

- `disabled` means Goalrail policy or an admin blocks connection use
- disabled state must not require provider revocation
- disabled credentials should not be used for refresh, metadata discovery, or
  checkout eligibility

Deletion and retention:

- credential deletion must delete or cryptographically destroy token material
  according to the later storage implementation
- redacted audit records may remain for security and operational history
- retained audit records must not contain raw secrets
- repository metadata retention after disconnect must be separately documented
  before provider metadata snapshots are persisted
- backup retention and deletion limitations must be documented before
  production token persistence

## Audit and Event Posture

Future audit events are conceptual in this ADR. Exact event names can be
accepted by a later implementation slice if repository conventions require it.

Minimum future audit concepts:

- `provider_connection.setup_started`
- `provider_connection.authorized`
- `provider_connection.refresh_started`
- `provider_connection.refresh_succeeded`
- `provider_connection.refresh_failed`
- `provider_connection.needs_reconnect`
- `provider_connection.revoked`
- `provider_connection.disabled`
- `provider_connection.reconnected`
- `provider_credential.deleted`

Audit payloads may include:

- Goalrail `installation_id`
- Goalrail `organization_id`
- future `vcs_connection_id`
- actor kind and actor id
- provider kind
- redacted provider instance reference
- redacted provider account reference
- requested scope names
- granted scope summary
- connection state transition
- credential fingerprint id, if a later implementation defines one safely
- provider error category
- provider HTTP status code
- adapter allowlist version
- timestamp and correlation id

Audit payloads must not include:

- raw tokens
- refresh tokens
- OAuth authorization codes
- OAuth client secrets
- PKCE code verifiers
- deploy keys or SSH private material
- Authorization headers
- full provider response bodies
- repository file contents

Operator visibility should show enough state to diagnose connect, refresh,
reconnect, revoke, disable, deletion, permission, and rate-limit issues without
exposing credentials.

## GitLab `read_api` and Adapter Allowlist

GitLab `read_api` is not narrow enough to prove metadata-only behavior by
scope alone. Advisory research found that GitLab also documents Repository
Files API access with `read_api`.

Goalrail therefore accepts this boundary:

- GitLab metadata discovery may request `read_user` and `read_api` only if a
  later implementation includes user-facing consent and a strict adapter
  allowlist
- `read_repository`, `write_repository`, and `api` scopes are out of the first
  metadata-only provider connection unless a later ADR explicitly changes that
  boundary
- metadata discovery must not call GitLab Repository Files API endpoints
- metadata discovery must not use Git-over-HTTP
- metadata discovery must not read blobs, file trees, raw files, package
  contents, CI logs, or source contents
- clone URLs returned by GitLab are metadata only and do not authorize checkout
- user-facing consent must explain that provider scopes may be broader than
  Goalrail adapter behavior

The future GitLab adapter must be implemented against an explicit endpoint
allowlist. Endpoints outside the allowlist are blocked by default.

Repository Files API is not part of metadata-only discovery.

## GitLab.com and Self-Managed Instance Identity

Provider instance identity is first-class for GitLab.

Future GitLab implementation must:

- distinguish GitLab.com from each self-managed GitLab instance
- normalize the selected instance base URL
- derive the API base from the normalized instance base URL
- avoid hard-coding GitLab.com as the only provider instance
- treat private self-managed hostnames as sensitive customer infrastructure
  metadata
- support rate-limit, feature, version, tier, SAML/SSO, and admin-policy
  variation as provider metadata or operational state rather than kernel truth
- keep GitLab Group, subgroup, Project, namespace, and OAuth application
  ownership semantics inside the adapter or adapter-owned metadata

Self-managed OAuth application ownership modes, administrator policy, trusted
applications, root namespace parameters, reverse proxy behavior, and instance
version differences must be evaluated by later implementation slices before
live support is claimed.

This ADR does not add config, environment variables, OAuth app setup, secrets,
or deployment files.

## Credentialless `pending_setup` VcsConnection

Future code may create credentialless `pending_setup` `VcsConnection` records.

Rules:

- the record contains non-secret setup state only
- it is Organization-scoped inside an Installation
- it is not `active`
- it cannot list provider metadata
- it cannot refresh provider metadata
- it cannot authorize checkout
- it cannot create or mutate RepoBindings by itself
- it cannot store provider tokens, OAuth codes, PKCE verifiers, or app secrets
- it must expire, be cancelable, or be cleanable
- user-visible copy must not say connected, synced, authorized, ready, scanned,
  cloned, verified, or proof

This decision unblocks a future D2 provider-neutral `VcsConnection` skeleton
only if the first code slice remains credentialless and non-live.

## Relationship to RepoBinding / RepositoryRecord / Checkout

`RepoBinding` remains the current Project-to-repository metadata reference.
It is not permission to clone, inspect code, create branches, create commits,
open merge requests, open pull requests, run checks, gate, or proof.

`RepositoryRecord` remains deferred. Do not introduce it only because provider
credential storage exists. Its trigger conditions remain catalog/search
independent of Project binding, provider sync independent of a Project,
multi-Project repository reuse semantics, repo-level policy, or repository
lifecycle tracking.

`VcsConnection` plus stored provider credentials do not grant checkout
eligibility by themselves.

Checkout credentials remain a later provider/runner adapter concern. Runner
checkout remains governed by ADR-0008. Customer-hosted runner compatibility must
remain possible without Goalrail server-side provider credentials.

## Future Implementation Gates

Unblocked after this ADR:

- D2 provider-neutral `VcsConnection` schema/API skeleton may start only as a
  credentialless, non-live `pending_setup` state slice.
- D2 may define provider-neutral state fields needed to represent setup,
  active/blocked concepts, and redacted provider instance/account metadata.

Still blocked after this ADR:

- provider token tables
- token encryption implementation
- OAuth authorization routes
- OAuth callbacks
- OAuth token exchange
- provider app/client configuration
- GitLab provider client
- live provider metadata calls
- repository metadata listing/search
- RepositoryRecord implementation
- checkout credentials
- runner checkout
- gate
- proof

GitLab OAuth/token implementation remains blocked until:

- D2 provider-neutral `VcsConnection` skeleton exists
- D3 provider-neutral repository metadata contract exists
- D4 GitLab metadata adapter mapping skeleton with fake/test fixtures exists,
  if GitLab remains the first provider
- a later implementation slice chooses concrete encryption/key management,
  token storage schema, redaction tests, refresh concurrency, and audit event
  behavior under this ADR

## Rejected Alternatives

### Store provider tokens directly on VcsConnection

Rejected. `VcsConnection` is a domain relationship and metadata-discovery
boundary. Tokens are secrets and credential artifacts.

### Require Goalrail cloud to hold repository credentials for all repository work

Rejected. Customer-hosted runners are first-class, and some teams must keep
repository contents and credentials inside customer infrastructure.

### Use browser storage for provider tokens

Rejected. Browser storage would widen credential exposure and conflict with the
repository connection UX boundary.

### Treat GitLab `read_api` as safe metadata-only access

Rejected. GitLab docs indicate `read_api` can also access repository file APIs.
Goalrail must rely on its own adapter allowlist and user consent, not provider
scope names alone.

### Implement GitLab OAuth before a provider-neutral VcsConnection skeleton

Rejected. That would make the first provider shape the core implementation and
would skip the accepted sequencing plan.

### Block all credentialless VcsConnection setup state

Rejected. A non-secret `pending_setup` record is useful for honest setup state
and can be implemented without credentials if it cannot perform live metadata
discovery or checkout.

## Non-goals

This ADR does not implement or define final details for:

- server code
- database migrations
- token tables
- provider credential implementation
- encryption implementation
- key-management implementation
- API routes
- OAuth routes
- OAuth callback handling
- GitLab app/client configuration
- environment variables
- secrets
- provider clients
- live provider calls
- repository metadata listing/search
- `RepositoryRecord` implementation
- `VcsConnection` implementation
- checkout credentials
- repository clone
- Repository Files API use
- runner
- gate
- proof
- branch creation
- commit creation
- merge request or pull request creation
- frontend code
- deployment or workflow changes

## Implementation Implications

The next implementation slice should be D2:

```text
provider-neutral VcsConnection skeleton with credentialless pending_setup only
```

That slice may add schema/API state only if it remains non-secret and non-live.
It must not add OAuth, token persistence, provider clients, provider metadata
listing, checkout, runner, gate, or proof.

Any provider credential storage implementation after D2 must include encryption
and key boundary decisions, redaction tests, refresh concurrency behavior,
revocation/deletion behavior, and audit events before storing real provider
tokens.

Any GitLab OAuth implementation must explicitly address `read_api` breadth,
instance identity, strict endpoint allowlist, redacted consent, token refresh
single-writer behavior, and no Repository Files API use.

## Open Questions

- What exact encryption mechanism and key source should self-hosted
  deployments use first?
- What exact KMS or envelope encryption posture should future SaaS use?
- Should provider app/client credentials be configured per Installation,
  per Organization, or deployment-wide for the first GitLab slice?
- What is the exact token storage schema after D2 defines `VcsConnection`?
- What retry and backoff policy should token refresh use after provider
  unavailability?
- How much provider metadata should remain after credential deletion?
- Should protected branch metadata wait for runner/checkout policy design?
- Which audit event names should become stable once event implementation starts?
