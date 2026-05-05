# ADR-0024 — Provider-neutral VCS connection boundary

Status: accepted
Date: 2026-05-05

## Context

ADR-0008 separates VCS discovery, repository binding, and checkout permission.
It also states that `VcsConnection != CheckoutCredential`,
`RepositoryRecord != RepoAccess`, and `RepoBinding != permission to clone`.

ADR-0010 keeps the MVP project context small by using direct `RepoBinding`
before a separate `RepositoryRecord`, and by deferring `VcsConnection` until a
real provider integration needs it. ADR-0022 adds `Installation` above
`Organization`, and ADR-0023 defines auth and CLI login without adding
provider-side repository authorization.

Goalrail now has metadata-only repository-context init and repository context
snapshots, but no provider connection, GitLab OAuth, GitHub App, Bitbucket
OAuth, provider client, token storage, checkout credential, repository clone,
runner, gate, proof, provider sync job, or repository catalog implementation.

Before any GitLab-first implementation, Goalrail needs a provider-neutral
boundary for VCS provider connections. Without this boundary, GitLab-specific
terms could leak into the core domain model, provider account/group/workspace
concepts could be confused with Goalrail `Organization`, provider tokens could
be mistaken for domain objects, or repository metadata could be treated as
checkout permission.

This ADR is documentation-only. It accepts the provider-neutral domain boundary
for future implementation and blocks provider-specific backend implementation
until this boundary is accepted in the architecture canon.

## Decision

Goalrail accepts `VcsConnection` as the future provider-neutral connection and
metadata-discovery boundary for VCS providers.

`VcsConnection` represents a provider connection / account authorization /
metadata-discovery relationship. It is not a checkout credential, not a raw
provider token, not a repository catalog, and not permission to clone.

The current MVP does not implement `VcsConnection` in backend schema, API
routes, provider clients, CLI commands, or web UI. The current metadata-only
`RepoBinding` and repository context snapshot flows remain valid before
provider integration.

Any future GitLab, GitHub, Bitbucket, self-managed Git, or custom Git provider
implementation must conform to this provider-neutral boundary before adding
backend schema/API behavior.

## Core object roles

### VcsConnection

`VcsConnection` is a future Goalrail core-domain object for provider connection
metadata and authorization posture.

Conceptual fields may include:

- `id`
- `installation_id`
- `organization_id`
- `provider_kind`
- `provider_account_ref`
- optional `provider_namespace_ref`
- `connection_state`
- metadata about granted discovery scope
- timestamps
- adapter-owned provider metadata reference

Provider tokens, refresh tokens, installation access tokens, OAuth grants,
deploy keys, SSH keys, and short-lived checkout credentials must not become
`VcsConnection` itself. Secrets belong in a provider/credential store boundary
defined by a later implementation ADR.

### RepoBinding

`RepoBinding` remains the current Project-to-repository reference.

It stores enough metadata for Goalrail to know which repository a Project works
with, such as provider kind, repository external id when known, repository full
name, repository URL, default branch, workflow base branch, and path scope.

`RepoBinding` is separate from checkout permission. A `RepoBinding` may be
manual/dev-seeded, locally declared by `goalrail init`, provider-discovered, or
runner-reported later. None of those modes grants clone/read/write access by
itself.

### RepositoryRecord

`RepositoryRecord` remains deferred.

Goalrail should extract a separate `RepositoryRecord` only when it needs a
repository catalog, provider sync, multi-project repository reuse independent
of direct binding, repository-level policy, or repository lifecycle tracking
outside a single Project binding.

`RepositoryRecord` is not repository access. It would describe repository
metadata and lifecycle, not clone/read/write authority.

### CheckoutCredential

`CheckoutCredential` is not accepted as a core MVP object by this ADR.

Future checkout credentials, if needed, belong behind provider and runner
adapters and should be short-lived where possible. Checkout authority is
evaluated per bounded runner job, not inferred from `VcsConnection`,
`RepositoryRecord`, or `RepoBinding`.

## Relationship to Installation / Organization / Project / RepoBinding

Conceptual relationship:

```text
Installation
  -> Organization
     -> VcsConnection[]
     -> Project
        -> RepoBinding
```

`Installation` is the running Goalrail control plane / instance.

`Organization` is the internal Goalrail tenant/workspace inside an
`Installation`. Goalrail `Organization` is not a GitLab Group, GitHub
Organization, Bitbucket Workspace, provider account, provider namespace,
repository owner, provider installation, or VCS account.

`VcsConnection` belongs to a Goalrail `Organization` inside an `Installation`
because provider authorization is made available to that Goalrail tenant.

`Project` is a delivery container inside a Goalrail `Organization`. Goalrail
`Project` is not a GitLab Project, GitHub repository, Bitbucket repository, or
provider project.

`RepoBinding` links one Goalrail `Project` to repository metadata. A
`RepoBinding` may optionally reference a future `VcsConnection` when the
metadata was discovered or can be refreshed through that provider connection.
The reference still does not grant checkout permission.

## Provider namespace anti-corruption rules

Provider-specific concepts stay behind provider adapters or in adapter-owned
metadata.

Core Goalrail domain terms must remain provider-neutral:

- use `provider_kind`, not provider-specific core field names such as
  `gitlab_group_id`
- use `provider_account_ref` for the connected account / installation / app
  installation / OAuth account shape
- use `provider_namespace_ref` for provider group / owner / workspace /
  namespace shape when needed
- use `repository_external_id`, `repository_full_name`, `repository_url`, and
  `default_branch` for repository metadata
- keep provider-specific ids, token ids, installation ids, project ids, group
  ids, scopes, and permission payloads in adapter metadata

Hard rules:

1. GitLab Group must not become Goalrail `Organization`.
2. GitLab Project must not become Goalrail `Project`.
3. GitHub Organization must not become Goalrail `Organization`.
4. Bitbucket Workspace must not become Goalrail `Organization`.
5. Provider account, namespace, group, workspace, repository owner, or provider
   installation must not become Goalrail `Organization`.
6. Provider tokens must not become `VcsConnection`.
7. Provider repository access must not become `RepoBinding`.
8. Provider-specific permission payloads must not define Goalrail policy
   semantics directly.

GitLab may be the first provider candidate, but GitLab terminology stays in the
GitLab adapter and provider metadata. It must not shape the core Goalrail
object model.

## VcsConnection lifecycle states

The conceptual `VcsConnection` lifecycle states are:

- `pending_setup` - setup has started but provider authorization is not ready
- `active` - provider metadata discovery is available under current policy
- `needs_reconnect` - the connection cannot be used until an owner reconnects
  or refreshes authorization
- `revoked` - provider authorization was revoked or invalidated
- `disabled` - Goalrail policy or an admin disabled the connection
- `failed` - provider setup or refresh failed for a recorded reason

These names are conceptual in this ADR. They are not implemented enums.

## Repository metadata states

Repository metadata can be known without checkout access.

Conceptual metadata states:

- `unknown` - Goalrail has no current repository metadata
- `declared` - metadata was manually/dev/CLI declared, not provider-discovered
- `discovered` - metadata was discovered through a provider connection
- `stale` - previously known metadata may be out of date
- `unavailable` - metadata cannot currently be read
- `removed_at_provider` - provider discovery indicates the repository no longer
  exists or is no longer visible

Metadata freshness must not be treated as clone/read/write permission.

## RepoBinding states

Conceptual `RepoBinding` states:

- `metadata_only` - the binding is known only as repository metadata
- `active` - the binding is valid for Goalrail planning and contract context
- `needs_reconnect` - the binding depends on a provider connection that needs
  reconnect before metadata refresh or checkout eligibility can be evaluated
- `disabled` - the binding is disabled by Goalrail policy/admin action
- `invalid` - the binding no longer points to a valid or allowed repository

Current MVP metadata-only init remains valid. A `metadata_only` binding may be
usable for intent, contracts, local markers, and customer-hosted runner flows
without Goalrail cloud having provider-side repository access.

## Reconnect / revoked / disabled handling

Reconnect affects provider metadata discovery and future provider-mediated
checkout eligibility only.

If a provider connection is revoked, disabled, failed, or needs reconnect:

- existing `RepoBinding` records should remain reviewable as historical or
  declared metadata unless policy explicitly disables them
- metadata refresh should stop or report unavailable/stale state
- future provider-mediated checkout eligibility should move to
  `requires_reconnect` or an equivalent blocked state
- customer-hosted runner flows may remain possible if policy allows them and
  they do not require Goalrail cloud-side provider authorization

Revoking a provider connection must not silently delete Goalrail Projects,
Goals, Contracts, WorkItems, or proofs.

## Metadata-only mode

Goalrail may operate in metadata-only mode.

Metadata-only mode means Goalrail knows repository identity and bounded context,
but has no provider-side clone credential and no provider-side repository
contents.

Valid metadata-only sources include:

- manual/admin declared repository metadata
- dev seed data
- `goalrail init` local Git metadata
- repository context snapshot metadata
- customer-hosted runner reported metadata
- future provider discovery without checkout authorization

Metadata-only mode is compatible with customer-hosted runners because repository
contents and credentials may stay inside customer infrastructure.

## Checkout eligibility is separate

Checkout eligibility is evaluated later from policy, runner mode, provider
connection state, repository metadata, requested operation, and a bounded task
or verification bundle.

Conceptual checkout eligibility states:

- `not_evaluated`
- `metadata_only`
- `eligible_by_policy`
- `blocked_by_policy`
- `requires_runner`
- `requires_reconnect`

Eligibility is not permission by itself. A future runner job still needs
bounded checkout instructions, scoped credentials or customer-owned access, and
a receipt boundary.

This ADR preserves:

- `VcsConnection != CheckoutCredential`
- `RepositoryRecord != RepoAccess`
- `RepoBinding != permission to clone`

## RepositoryRecord deferral rule

Do not introduce `RepositoryRecord` only because a provider integration is
starting.

Keep direct `RepoBinding` until Goalrail needs at least one of:

- repository catalog or repository search independent of Project binding
- provider sync records independent of a specific Project
- multi-project repository reuse semantics that direct RepoBinding cannot
  express cleanly
- repository-level policy or ownership rules
- repository lifecycle history independent of the Project spine

If a later ADR introduces `RepositoryRecord`, it must still preserve the
checkout separation in ADR-0008 and this ADR.

## Runner relationship

Runners own checkout, workspace preparation, code inspection, checks, receipts,
and artifacts behind ADR-0008.

`VcsConnection` may help a future hosted runner obtain short-lived provider
checkout credentials if policy allows that path. It still does not authorize
checkout by itself.

Customer-hosted runners remain first-class. Goalrail may operate with a
customer-hosted runner even when Goalrail cloud has no VCS provider connection,
no clone credential, and no visibility into repository contents beyond bounded
receipts and approved artifacts.

## GitLab-first candidate posture

GitLab can be the first provider candidate for implementation learning.

GitLab-first does not mean GitLab-shaped core.

Any GitLab implementation must translate GitLab Groups, Projects, OAuth/token
details, repository visibility, permissions, and namespace data through a
GitLab adapter into provider-neutral Goalrail concepts. GitLab-specific fields
must stay adapter-owned unless a later ADR accepts a provider-neutral core
field.

## Boundary rules

1. `VcsConnection` is provider-neutral.
2. `VcsConnection` is a connection / account authorization / metadata-discovery
   boundary.
3. `VcsConnection` is not a checkout credential.
4. Raw provider tokens are secrets, not domain objects.
5. `RepoBinding` remains the current Project-to-repository metadata reference.
6. `RepoBinding` does not grant clone/read/write access.
7. `RepositoryRecord` remains deferred until catalog, sync, reuse, lifecycle,
   or repo-level policy pressure exists.
8. Provider-specific concepts stay in adapters and adapter metadata.
9. Goalrail `Organization` remains distinct from all provider group/account/
   workspace/namespace concepts.
10. Goalrail `Project` remains distinct from all provider project/repository
    concepts.
11. GitLab, GitHub, Bitbucket, self-managed Git, and custom Git must map into
    the same provider-neutral core.
12. Provider implementation must not add runner, checkout, gate, proof, queue,
    branch, commit, merge request, or pull request behavior by implication.

## Rejected alternatives

### Start with GitLab-shaped core schema

Rejected. It would let GitLab Group and GitLab Project semantics leak into
Goalrail `Organization` and `Project`, making later GitHub, Bitbucket,
self-managed, and customer-hosted runner paths harder.

### Treat provider tokens as VcsConnection

Rejected. Tokens are secrets and credential artifacts. `VcsConnection` is a
domain relationship and metadata-discovery boundary.

### Treat RepoBinding as repository access

Rejected. `RepoBinding` identifies which repository a Project works with. It
does not authorize checkout, clone, read, write, branches, commits, merge
requests, pull requests, or provider API calls.

### Introduce RepositoryRecord before need

Rejected. Direct `RepoBinding` remains enough for the current MVP and
metadata-only init. `RepositoryRecord` should appear only when catalog, sync,
reuse, lifecycle, or repo-level policy pressure makes it necessary.

### Require Goalrail cloud provider connection for all repository work

Rejected. Customer-hosted runner compatibility is a core architecture posture.
Some teams can use Goalrail with repository credentials and contents kept
inside customer infrastructure.

## Non-goals

This ADR does not implement or define:

- server code
- database migrations
- backend API routes
- GitLab OAuth
- GitHub App integration
- Bitbucket OAuth
- provider clients
- provider token storage
- checkout credentials
- deploy keys
- SSH key management
- repository clone
- repository write access
- branch, commit, merge request, or pull request creation
- repository catalog implementation
- provider sync jobs
- runner implementation
- checkout jobs
- check execution
- gate or proof behavior
- generic queue behavior
- SaaS onboarding
- public registration
- broad integration platform behavior

## Implementation implications

Future provider work should start with a bounded provider-neutral design that
references this ADR.

The first backend implementation slice, if approved later, should be narrow and
should not begin by adding checkout, repository clone, runner jobs, gate, proof,
or queue behavior.

GitLab-specific implementation, if selected first, should be an adapter slice
that maps GitLab concepts to provider-neutral Goalrail concepts. It should not
add GitLab-specific core field names such as `gitlab_group_id` or
`gitlab_project_id`.

Provider credential storage, provider metadata refresh, repository catalog,
checkout eligibility, and runner checkout remain separate future boundaries.

## Open questions

- Whether the first implemented provider slice should start with
  metadata-discovery only or account connection plus repository selection.
- Whether provider credential storage needs a separate ADR before any schema is
  added.
- Whether future `VcsConnection` should be scoped to one Goalrail Organization
  only or allow installation-level administration with organization-specific
  grants.
- Whether repository selection UX belongs first in web settings, CLI, or an
  operator/admin tool.
