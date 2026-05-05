---
id: goalrail_gitlab_vcs_connection_research
title: Goalrail GitLab VCS Connection Research
kind: research_note
authority: advisory
status: current
owner: architecture-research
truth_surfaces:
  - gitlab_provider_research
  - vcs_connection_boundary_input
  - repository_metadata_mapping_input
lifecycle: incubating
review_after: 2026-06-05
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md
  - docs/product/GOALRAIL_REPOSITORY_CONNECTION_UX.md
  - docs/PROJECT_SPINE_SCHEMA.md
  - docs/adr/ADR-0008-runner-checkout-boundary.md
  - docs/adr/ADR-0010-organization-project-repo-binding-persistence-boundary.md
  - docs/adr/ADR-0024-provider-neutral-vcs-connection-boundary.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/DECISIONS.md
  - docs/ops/COMPONENTS.yaml
---
# Goalrail GitLab VCS Connection Research

> Official-docs-based research note. This document is advisory background for
> ADR-0024 and future GitLab metadata adapter planning. It does not override
> ADR-0024 or authorize GitLab implementation, OAuth routes, token storage,
> checkout, cloning, runners, gate, proof, or GitLab client code.

Researched on: 2026-05-05.

## Purpose

This note maps current official GitLab documentation into Goalrail boundary
inputs for the accepted provider-neutral VCS connection boundary and future
GitLab metadata adapter planning.

This phase is intentionally limited to research and mapping. It should be read
as:

1. background for ADR-0024 as the accepted provider-neutral VCS connection
   boundary
2. a later GitLab metadata adapter plan
3. only then, a bounded implementation slice if a later authorized slice adopts
   ADR-0024's boundary

This note does not define a final schema, API contract, credential store,
checkout protocol, runner behavior, or provider abstraction. Where it differs
from ADR-0024, ADR-0024 wins.

## Goalrail boundary recap

Goalrail's current product and architecture truth must remain intact:

- ADR-0024 is the accepted architecture truth for the provider-neutral VCS
  connection boundary.
- Goalrail is a productized operating layer for AI-assisted delivery, not a
  Git provider, AI IDE, CI system, or DevOps suite.
- Goalrail `Organization` is an internal tenant/workspace. It is not a GitLab
  Group.
- Goalrail `Project` is a delivery container inside a Goalrail Organization. It
  is not a GitLab Project.
- `RepoBinding` identifies which repository a Goalrail Project works with. It
  does not grant checkout permission.
- `VcsConnection` is a future provider connection layer. It is not a checkout
  credential.
- Repository metadata discovery, repository binding, and checkout authority are
  separate concerns.
- The Goalrail API server must not clone repositories or run checks in-process.
- Repository checkout, workspace preparation, code inspection, and check
  execution belong behind the runner boundary.
- Clone URLs, branch metadata, protected branch metadata, and GitLab project
  IDs are provider metadata only unless a later checkout boundary separately
  authorizes use for a bounded runner job.

GitLab-specific terms must therefore stay inside a provider adapter or provider
metadata surface. They must not become Goalrail core domain concepts.

## Official GitLab findings

GitLab exposes OAuth and REST API surfaces that are sufficient for user-approved
repository metadata discovery, but the official scope model does not appear to
offer a narrow "repository metadata only, no source content" OAuth scope.

The most important finding for Goalrail is the separation problem:

- GitLab's `read_api` scope is the apparent read-only API scope for user,
  group, and project metadata discovery.
- GitLab also documents repository file access with `read_api`.
- GitLab's `read_repository` scope explicitly covers Git-over-HTTP and the
  Repository Files API.
- Therefore, a metadata discovery connection cannot rely on GitLab OAuth scopes
  alone to prove that source-code/file access is impossible.

Goalrail should treat GitLab metadata discovery as a provider-specific input to
a provider-neutral connection boundary, not as direct checkout authority.

## OAuth and PKCE

GitLab documents support for multiple OAuth 2.0 flows through its OAuth 2.0
identity provider API, including authorization code with PKCE, authorization
code without PKCE, resource owner password credentials, and device
authorization grant. GitLab describes authorization code with PKCE as the most
secure option and recommends it for both client and server apps. Source:
[OAuth 2.0 identity provider API](https://docs.gitlab.com/api/oauth2/).

For authorization code with PKCE, GitLab's documented flow uses:

- `GET /oauth/authorize`
- `POST /oauth/token`
- `state` as an unpredictable value and CSRF token
- `code_verifier`
- S256 `code_challenge`
- optional `root_namespace_id` when SAML SSO is configured for the associated
  group

GitLab's OAuth documentation shows the authorization URL shape as
`https://gitlab.example.com/oauth/authorize` and the token endpoint as
`https://gitlab.example.com/oauth/token`. Source:
[OAuth 2.0 identity provider API](https://docs.gitlab.com/api/oauth2/).

Boundary implication for Goalrail:

- A future GitLab connection should use authorization code with PKCE unless a
  later ADR documents a narrower reason not to.
- OAuth callback and token exchange routes are out of scope for this research
  phase.
- `state`, redirect URI validation, PKCE verifier storage, token exchange, and
  token refresh handling must be defined in the later provider-neutral ADR
  before implementation.
- The optional GitLab `root_namespace_id` should be treated as provider-specific
  OAuth behavior for SAML SSO support, not as a Goalrail Organization mapping.

## OAuth application setup considerations

GitLab documents three OAuth application ownership modes:

- user-owned applications
- group-owned applications
- instance-wide applications

User-owned and group-owned applications can be created through settings pages
and provide an OAuth Client ID and Client Secret. GitLab says the Client Secret
is available at creation time and that renewing the secret prevents the existing
application from functioning until credentials are updated. Source:
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/).

Instance-wide applications are documented for GitLab Self-Managed and require
administrator access. GitLab also documents a "trusted" instance-wide
application option that skips the user authorization step. Source:
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/).

Boundary implication for Goalrail:

- GitLab.com and each self-managed GitLab instance require instance-aware
  OAuth configuration.
- A future ADR should decide whether Goalrail supports user-owned,
  group-owned, instance-wide, or customer-provided application registration in
  the first GitLab slice.
- Trusted instance-wide applications should be treated cautiously. Skipping user
  authorization is an instance-admin policy decision, not a Goalrail default.
- OAuth application credentials are deployment secrets. This research phase
  must not introduce secrets, env vars, or deploy config.

## Scope candidates and least-privilege notes

GitLab's OAuth provider documentation lists the relevant scopes:

- `read_user`: read-only access to the authenticated user's profile through
  `/user`, including username, public email, and full name; also read-only
  endpoints under `/users`.
- `read_api`: read access to the API, including all groups and projects,
  container registry, and package registry.
- `read_repository`: read-only access to repositories on private projects using
  Git-over-HTTP or the Repository Files API.
- `write_repository`: read-write access to repositories on private projects
  using Git-over-HTTP, not using the API.
- `api`: complete read/write API access.

Source:
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/).

Preliminary candidate for metadata discovery:

- `read_user` for authenticated user identity and display context.
- `read_api` for read-only API access to groups and projects.

Scopes to avoid for the metadata-only phase:

- `read_repository`, because GitLab documents it as repository access through
  Git-over-HTTP or Repository Files API.
- `write_repository`, because this phase has no repository writes.
- `api`, because this phase has no read/write API need.
- runner, registry, observability, AI, sudo, and admin scopes.

Important limitation:

GitLab's Repository Files API documents that `read_api` and `read_repository`
allow read access to repository files, and its file retrieval response includes
Base64-encoded file contents. Source:
[Repository files API](https://docs.gitlab.com/api/repository_files/).

This means official docs do not establish a clean OAuth scope that allows
private project/group/repository metadata discovery while technically excluding
all source file reads. A later ADR should account for this by requiring:

- an internal adapter endpoint allowlist
- no Repository Files API calls in metadata discovery
- no Git-over-HTTP use in metadata discovery
- no repository checkout or clone credential derivation from `VcsConnection`
- clear user-facing consent text that the provider token may be broader than
  Goalrail's metadata adapter behavior

## GitLab.com vs self-managed

The GitLab REST API root is the GitLab host name plus `/api/v4`. Official
examples use `https://gitlab.example.com/api/v4/projects`; for GitLab.com, the
same rule yields `https://gitlab.com/api/v4`. Source:
[REST API](https://docs.gitlab.com/api/rest/).

Self-managed GitLab should be first-class in Goalrail's future design:

- OAuth authorization and token endpoints live on the selected GitLab instance.
- The API base URL must be derived from a normalized instance base URL, not
  hard-coded to GitLab.com.
- Instance-wide OAuth applications are only documented for GitLab Self-Managed
  and require administrator access.
- Self-managed administrators can configure several project and group API rate
  limits.
- Feature availability can differ by GitLab version, tier, and instance
  settings.
- Self-managed instances may use SAML SSO and the OAuth `root_namespace_id`
  parameter may be relevant for the associated group.
- Reverse proxies can affect URL-encoded path parameters; GitLab's REST API
  docs warn that namespaced paths, branch names, tags, and file paths with `/`
  must be URL-encoded.

Sources:
[REST API](https://docs.gitlab.com/api/rest/),
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/),
[Groups API](https://docs.gitlab.com/api/groups/).

Boundary implication for Goalrail:

- A future provider-neutral connection model should include provider instance
  identity for GitLab. It should not assume one global GitLab cloud.
- The provider-specific adapter can know "GitLab.com" versus "self-managed
  instance"; the Goalrail core should only depend on neutral provider and
  connection state.
- Self-managed policy variability should be reflected as connection metadata
  and operational checks, not hidden assumptions in the kernel.

## Repository/project metadata mapping inputs

GitLab's provider object is a GitLab Project. Goalrail must not conflate that
with a Goalrail Project.

GitLab's Projects API documents response fields useful for future repository
metadata mapping:

| GitLab field | Future Goalrail mapping input | Boundary note |
| --- | --- | --- |
| `id` | Provider-scoped repository external ID | Not globally unique outside one GitLab instance. |
| `path_with_namespace` | Provider repository full name / canonical path candidate | Do not map namespace to Goalrail Organization. |
| `web_url` | Browser URL metadata | Safe as metadata, still potentially sensitive for private repositories. |
| `ssh_url_to_repo` | SSH clone URL metadata | Metadata only; does not authorize checkout. |
| `http_url_to_repo` | HTTP clone URL metadata | Metadata only; do not add credentials or use for checkout in this phase. |
| `default_branch` | Provider default branch metadata | Not necessarily the Goalrail workflow base branch. |
| `visibility` | Provider visibility metadata | Values are documented as `private`, `internal`, or `public`. |
| `archived` | Repository state signal | Use only if returned by the API response for the endpoint/version. |
| `namespace` | Provider namespace metadata | Useful for display/disambiguation; not Goalrail tenant truth. |

The Projects API also states that returned fields vary based on the
authenticated user's permissions. Source:
[Projects API](https://docs.gitlab.com/api/projects/).

Path handling notes:

- GitLab project endpoints can use an integer ID or URL-encoded namespaced
  project path.
- GitLab REST docs require URL encoding for namespaced paths and branch names
  that contain `/`.
- GitLab REST docs say moved project paths can return a redirect with a
  `Location` header.

Source:
[REST API](https://docs.gitlab.com/api/rest/).

Boundary implication for Goalrail:

- A future GitLab metadata adapter should store or compare provider identity
  using instance base URL plus GitLab project `id` where available.
- `path_with_namespace` remains useful for display and idempotency, but path
  changes must be expected.
- Clone URLs must remain metadata-only until a separate runner checkout
  boundary issues bounded checkout instructions.
- Repository metadata should be treated as sensitive customer information even
  when it does not contain source code.

## Groups, subgroups, and project listing

GitLab's Groups API documents:

- `GET /groups` lists visible groups for the authenticated user; without
  authentication, only public groups are returned.
- Group listing defaults to 20 results per page because API results are
  paginated.
- Filters include `owned`, `min_access_level`, `top_level_only`, `visibility`,
  `active`, and `archived`.
- `GET /groups/:id/projects` lists projects in a group; without authentication,
  only public projects are returned.
- `GET /groups/:id/projects` supports `include_subgroups` to include projects in
  subgroups, defaulting to `false`.
- `GET /groups/:id/subgroups` lists visible immediate subgroups.
- `GET /groups/:id/descendant_groups` lists visible descendant groups.

Source:
[Groups API](https://docs.gitlab.com/api/groups/).

Boundary implication for Goalrail:

- GitLab Group discovery should be provider metadata for selection and
  filtering only.
- A GitLab Group must not become a Goalrail Organization.
- A GitLab subgroup tree can help a user find repositories, but it should not
  define Goalrail Project structure automatically.
- Shared projects need careful display. GitLab's Groups API notes that the
  `namespace` attribute can distinguish a project in the group from a project
  shared to the group.

## Branches and protected branches

GitLab's Branches API documents:

- `GET /projects/:id/repository/branches`
- `GET /projects/:id/repository/branches/:branch`
- branch fields such as `name`, `merged`, `protected`, `default`,
  `developers_can_push`, `developers_can_merge`, `can_push`, `web_url`, and
  `commit`
- commit subfields such as full SHA `id`, `short_id`, dates, title, message,
  author/committer metadata, and commit `web_url`

Source:
[Branches API](https://docs.gitlab.com/api/branches/).

GitLab's Protected branches API documents:

- `GET /projects/:id/protected_branches`
- `GET /projects/:id/protected_branches/:name`
- response fields including `name`, `allow_force_push`,
  `code_owner_approval_required`, `push_access_levels`,
  `merge_access_levels`, `unprotect_access_levels`, and optional inherited
  protection data on some tiers
- access level constants such as `0` no access, `30` developer, `40`
  maintainer, and `60` administrator for self-managed instances

Source:
[Protected branches API](https://docs.gitlab.com/api/protected_branches/).

Boundary implication for Goalrail:

- Branch and protected-branch metadata can inform future repository readiness,
  branch selection UI, and checkout policy checks.
- `can_push` and protected branch permissions must not be treated as Goalrail
  write authorization in this research phase.
- Future checkout or write behavior must be separately authorized by a runner
  and credential boundary.
- Protected branch metadata may vary by tier, inherited group settings, and
  self-managed capabilities; adapters should treat missing fields as unknown,
  not as permissive defaults.

## Pagination, rate limits, and errors

GitLab REST API pagination:

- Default offset pagination uses `page` and `per_page`.
- `per_page` defaults to `20` and has a max of `100`.
- GitLab returns `Link` headers with `rel` values such as `prev`, `next`,
  `first`, and `last`; docs say clients should use these links instead of
  generating URLs.
- Some pagination headers may not be returned for GitLab.com users.
- Keyset pagination exists for selected resources and is preferred for large
  collections when available.
- For queries over 10,000 records, GitLab may omit `x-total`,
  `x-total-pages`, and `rel="last"`.

Source:
[REST API](https://docs.gitlab.com/api/rest/).

GitLab.com rate limits relevant to metadata discovery include:

- authenticated API traffic for a user: 2,000 requests per minute
- projects list requests: 2,000 requests every 10 minutes
- group projects requests: 600 requests per minute
- single project requests: 400 requests per minute
- groups list requests: 200 requests per minute
- single group requests: 400 requests per minute
- repository files: 500 requests per minute

When GitLab.com rate-limits a request, GitLab responds with `429`; GitLab says
Projects, Groups, and Users API rate-limit responses do not include
informational headers. Source:
[GitLab.com settings](https://docs.gitlab.com/user/gitlab_com/).

Self-managed and Dedicated rate-limit considerations:

- GitLab documents configurable Projects API rate limits for Self-Managed and
  Dedicated, including defaults for `GET /projects`, `GET /projects/:id`, and
  user project-list endpoints.
- GitLab documents configurable Groups API rate limits for Self-Managed and
  Dedicated, including defaults for `GET /groups`, `GET /groups/:id`, and
  `GET /groups/:id/projects`.
- Administrators can adjust these limits or set some limits to `0` to disable
  them.

Sources:
[Rate limits on Projects API](https://docs.gitlab.com/administration/settings/rate_limit_on_projects_api/),
[Rate limits on Groups API](https://docs.gitlab.com/administration/settings/rate_limit_on_groups_api/).

REST error behavior relevant to provider adapters:

- `401 Unauthorized`: user is not authenticated and a valid user token is
  necessary.
- `403 Forbidden`: the request is not allowed.
- `404 Not Found`: the resource could not be accessed, including when the user
  is not authorized to access it.
- `429 Too Many Requests`: application rate limit exceeded.
- `503 Service Unavailable`: server is temporarily overloaded.

Source:
[Troubleshooting the REST API](https://docs.gitlab.com/api/rest/troubleshooting/).

Boundary implication for Goalrail:

- A future GitLab metadata adapter must be paginated by design.
- It should treat 401 as reconnect/token-refresh territory, 403 as permission
  or insufficient-scope territory, 404 as either missing resource or masked
  authorization failure, and 429 as backoff territory.
- It should preserve enough provider response context for operator diagnosis
  without storing tokens or sensitive response bodies unnecessarily.
- It should not use Repository Files API calls during metadata discovery, even
  if rate limits are documented.

## Token lifecycle and reconnect

GitLab documents that OAuth access tokens expire after two hours and that
integrations must use the `refresh_token` attribute to generate new ones.
GitLab also says the access token expiration is not configurable. Source:
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/).

The OAuth 2.0 identity provider API documents refresh-token use with
`POST /oauth/token` and `grant_type=refresh_token`; the refresh response sends
new tokens and invalidates the existing access token and refresh token. Source:
[OAuth 2.0 identity provider API](https://docs.gitlab.com/api/oauth2/).

GitLab documents several revocation/deletion paths:

- Users can revoke authorized application access from their applications page.
- Deleting an application deletes associated grants and tokens.
- The OAuth API includes token information and token revocation surfaces.

Sources:
[Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/),
[OAuth 2.0 identity provider API](https://docs.gitlab.com/api/oauth2/).

Boundary implication for Goalrail:

- A future `VcsConnection` state model should include at least connected,
  refresh-needed, needs-reconnect, revoked/invalid, and unavailable/error
  concepts. Exact names are for the ADR, not this research note.
- Reconnect should refresh provider authorization without mutating historical
  Goalrail proof or past repository metadata snapshots.
- Token refresh must be single-writer and race-aware because GitLab refresh
  invalidates the previous refresh token.
- Token storage is not authorized by this research phase. A later ADR must
  define encryption, access boundaries, redaction, audit events, retention,
  deletion, and operator recovery behavior before implementation.

## Security risks

GitLab metadata discovery carries risks even without checkout:

- OAuth access and refresh tokens are bearer credentials for the user's GitLab
  permissions and must be treated as secrets.
- `read_api` appears necessary for private group/project metadata discovery, but
  official docs also allow repository file reads with `read_api`.
- Repository names, namespace paths, visibility, archived state, default branch,
  branch names, protected branch rules, clone URLs, and commit metadata can
  reveal sensitive product, customer, or internal project information.
- `ssh_url_to_repo` and `http_url_to_repo` are clone URL metadata. Logging them
  casually can reveal private namespace paths even without credentials.
- Self-managed GitLab instances may expose private hostnames. Instance base URLs
  can be sensitive customer infrastructure metadata.
- Group and subgroup names may reveal organization structure but must not be
  mapped into Goalrail tenancy automatically.
- Rate-limit and permission errors can leak enough state to infer whether a
  private resource exists; adapters should normalize user-facing errors
  carefully.

Minimum security posture for the future ADR:

- request only the scopes needed for metadata discovery
- avoid `read_repository`, `write_repository`, and `api` for the first
  metadata-only connection unless a later ADR explicitly changes scope
- use a strict provider API allowlist
- never call repository file/content/blob APIs in metadata discovery
- never place tokens in repo files, logs, docs, browser storage, or local
  project markers
- mark clone URLs as metadata-only wherever displayed or stored
- separate `VcsConnection` state from `RepoBinding` identity and checkout
  credentials
- make token revocation and reconnect visible without claiming checkout access

## Provider-neutral implications for Goalrail

The GitLab findings point to a provider-neutral VCS boundary with these
candidate concepts. These are advisory research inputs; ADR-0024 is the
accepted boundary where it has already made a decision:

- `VcsProvider`: provider kind plus provider instance, for example GitLab.com or
  a self-managed GitLab base URL.
- `VcsConnection`: user/admin-approved provider API connection for metadata
  discovery and repository selection.
- `RepositoryCandidate`: provider-returned repository metadata before binding.
- `RepoBinding`: Goalrail Project to repository identity mapping, still not
  checkout permission.
- `CheckoutCredential`: separate future runner/checkout artifact, not derived
  automatically from `VcsConnection`.
- `ConnectionState`: provider auth health and reconnect state.
- `ProviderMetadataSnapshot`: bounded metadata evidence with source instance,
  fetched time, and adapter version.

ADR-0024 now defines the provider-neutral boundary before any GitLab-specific
code because GitLab exposes a broad `read_api` surface and because self-managed
GitLab makes instance identity first-class.

## Open questions for future provider slices

1. Is `read_user read_api` acceptable for metadata discovery given that
   `read_api` also enables repository file reads through the Repository Files
   API?
2. Should Goalrail support only GitLab.com first, or require self-managed GitLab
   support in the first provider slice even if implementation starts
   with GitLab.com smoke tests?
3. Which OAuth application ownership modes are allowed first: user-owned,
   group-owned, customer-provided instance-wide, or all of them?
4. How should the future product explain GitLab Group selection without implying
   that a GitLab Group is a Goalrail Organization?
5. Should repository candidates be persisted before `RepoBinding`, or only shown
   transiently until the user binds a repository?
6. What is the minimum provider metadata snapshot needed for auditability
   without creating a repository catalog too early?
7. How should refresh-token rotation be serialized if two Goalrail processes
   attempt refresh at the same time?
8. Which provider errors become `needs_reconnect`, `permission_denied`,
   `rate_limited`, or `not_found` in a provider-neutral state model?
9. How should self-managed GitLab base URLs be validated, normalized, and
   displayed without leaking private infrastructure details?
10. Should protected branch metadata be captured during initial repository
    selection, or deferred until runner/checkout policy design?
11. What retention and deletion policy applies to repository metadata after a
    GitLab app is revoked or a Goalrail Project unbinds a repository?
12. Should the first metadata adapter use only REST, or should GraphQL be
    evaluated later for pagination and field minimization?

## Preliminary recommendation

Do not implement GitLab integration next.

ADR-0024 is now the accepted provider-neutral VCS connection boundary. Any next
phase should conform to ADR-0024 and stay limited to a bounded GitLab metadata
adapter plan only if that slice is explicitly authorized. That later plan must:

- keep Goalrail `Organization`, Goalrail `Project`, `RepoBinding`,
  `VcsConnection`, and checkout authority separate
- explicitly model provider instance identity for GitLab.com and self-managed
  GitLab
- define a metadata-only connection scope and API allowlist
- document the residual risk that GitLab OAuth scopes do not appear to provide
  repository-metadata-only access isolated from repository file reads
- define connection state, reconnect, revocation, refresh, pagination,
  rate-limit, and error semantics
- define security requirements before any token persistence
- keep checkout credentials and runner instructions out of `VcsConnection`
- leave GitLab client implementation for a later bounded metadata adapter plan

A GitLab metadata adapter plan can define the exact endpoints, fields, tests,
mocks, and fixture strategy for repository candidate discovery only after it
adopts ADR-0024's boundary and non-goals.

## Sources

Official GitLab documentation reviewed:

- [Configure GitLab as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/)
- [OAuth 2.0 identity provider API](https://docs.gitlab.com/api/oauth2/)
- [REST API](https://docs.gitlab.com/api/rest/)
- [REST API authentication](https://docs.gitlab.com/api/rest/authentication/)
- [Troubleshooting the REST API](https://docs.gitlab.com/api/rest/troubleshooting/)
- [Projects API](https://docs.gitlab.com/api/projects/)
- [Groups API](https://docs.gitlab.com/api/groups/)
- [Branches API](https://docs.gitlab.com/api/branches/)
- [Protected branches API](https://docs.gitlab.com/api/protected_branches/)
- [Repository files API](https://docs.gitlab.com/api/repository_files/)
- [GitLab.com settings](https://docs.gitlab.com/user/gitlab_com/)
- [Rate limits on Projects API](https://docs.gitlab.com/administration/settings/rate_limit_on_projects_api/)
- [Rate limits on Groups API](https://docs.gitlab.com/administration/settings/rate_limit_on_groups_api/)
