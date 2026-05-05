---
id: goalrail_repository_connection_ux
title: Goalrail Repository Connection UX
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - repository_connection_ux
  - console_settings_boundary
  - repo_binding_ux
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md
  - docs/product/GOALRAIL_DESIGN_DECISIONS.md
  - docs/PROJECT_SPINE_SCHEMA.md
  - docs/adr/ADR-0008-runner-checkout-boundary.md
  - docs/adr/ADR-0010-organization-project-repo-binding-persistence-boundary.md
  - docs/adr/ADR-0023-user-bootstrap-auth-and-cli-login-boundary.md
  - docs/ops/STATUS.md
  - docs/ops/COMPONENTS.yaml
---
# Goalrail Repository Connection UX

## Purpose

This document defines the honest console UX boundary for connecting repositories
through VCS providers, with GitLab as the first provider candidate.

It is a Phase 1 product-shape spec. It does not implement provider APIs, OAuth,
GitLab API calls, token storage, repository checkout, repository scan, runner,
gate, proof, or frontend provider clients.

Repository connection supports the contract-to-proof operating layer by helping
a Goalrail Project identify the repository it works with. It does not claim
checkout, code inspection, execution, gate, proof, or provider replacement
behavior.

## Current repo truth

Current canon and implementation status set these boundaries:

- Goalrail is a productized operating layer over existing tools, not an AI IDE,
  DevOps suite, provider replacement, or hosted execution platform.
- Goalrail Organization is the internal workspace / tenant boundary. It is not
  a GitHub Organization, GitLab Group, or Bitbucket Workspace.
- Goalrail Project is the delivery container inside a Goalrail Organization. It
  is not a repository.
- RepoBinding identifies which repository a Goalrail Project works with. It is
  not permission to clone.
- MVP uses direct RepoBinding before RepositoryRecord. RepositoryRecord and
  VcsConnection remain future layers.
- Existing server and CLI slices can create metadata-only repository context
  through authenticated init paths. Those paths are not provider connection,
  OAuth, provider account sync, repository picker, repository scan, checkout,
  runner, gate, or proof.
- Current `apps/web/console` has login, first-login password change, `/v1/me`,
  logout, three empty product surfaces, Settings -> Appearance, and Settings ->
  Users. Settings is support UI.
- Current console browser storage is limited to the visual theme key
  `goalrail.console.theme`. Tokens and profile state stay in memory.
- No console repository settings UI, provider connection status, provider
  account identity, GitLab group/project listing, repository metadata search,
  live binding picker, reconnect behavior, revoked state, runner checkout,
  gate, or proof exists.

## Console entry point

The intended console entry point is:

```text
Settings -> Integrations -> Repositories
```

Settings remains support UI. It must not become a fourth product surface next
to Contracts, Delivery Readiness, and Proof.

Console navigation rules:

- Product surfaces remain Contracts, Delivery Readiness, and Proof.
- Settings can expose utility sections such as Appearance, Users, and
  Integrations.
- Repositories belongs under Integrations because it configures external
  product context; it is not a delivery board, contract workspace, or proof
  surface.
- A first static placeholder may use this label structure without adding live
  provider actions.
- Any future visible Organization / Project / RepoBinding context must be
  displayed as selected working context, not as a new product surface.

## State model

These states are user-visible labels over separate concerns: provider
connection, provider metadata, repository selection, and Goalrail RepoBinding.
They are not a promise of one linear wizard and they must not be faked.

| State | What the user sees | What it means | What it does not mean | Backend dependency |
| --- | --- | --- | --- | --- |
| No provider connected | Empty Repositories settings state; GitLab may appear as a disabled candidate; warning copy is visible. | The console has no server-backed provider connection state to show. Before provider-neutral backend APIs exist, this can be a static docs-backed placeholder. | It does not mean the server has checked GitLab, checked account access, listed repositories, or found no repositories. | Static placeholder needs no provider API. Real status needs provider-neutral connection status API. |
| Provider connected | Provider card shows the provider name, connection status, and provider account identity returned by the server. | Goalrail server has a recorded provider connection state for the authenticated Goalrail context. | It does not mean any repository is selected, bound, cloned, scanned, or available for checkout. | Future VcsConnection or equivalent provider-neutral connection API. |
| Repository metadata available | Repository rows or search results show provider metadata such as provider, full name, URL, default branch, visibility where allowed, and external ID where available. | Provider-neutral backend APIs have returned repository metadata. For GitLab, groups/projects/repositories are provider metadata only. | It does not mean the repository is a Goalrail Project, bound to a Goalrail Project, scanned, checked out, inspected, verified, ready, or safe. | Future provider-neutral repository metadata list/search API. |
| Repo selected | A repository row is selected inside the picker and pending confirmation. | The user has chosen one provider metadata item in the UI flow. | It does not mean RepoBinding exists, server truth changed, checkout permission exists, or code can be inspected. | Real selection requires repository metadata from backend APIs. Transient UI selection must be clearly pending until server confirmation. |
| Repo bound to Goalrail Project | The Repositories settings view shows one active primary RepoBinding for the selected Goalrail Project, with provider, repository full name, URL, default/workflow branch where available, access mode, and state from the server. | Server-owned RepoBinding says this Goalrail Project works with that repository. For MVP, one Project should have at most one active primary RepoBinding unless later canon changes. | It does not grant checkout permission, run scans, create tasks, start runners, write gate decisions, or generate proof. | Existing RepoBinding persistence is the canonical backend foundation. Console display and mutation still need authorized server-backed read/write/context APIs. |
| Reconnect needed | Provider card shows a non-destructive reconnect-needed state with no repository actions that imply live access. | Server reports that provider connection health requires user/admin action before live metadata can be refreshed. | It does not prove token revocation cause, repository deletion, RepoBinding invalidity, checkout failure, or proof failure. | Future provider connection health API. Must be server-reported, not inferred from frontend timers. |
| Revoked | Provider card shows revoked or disconnected by provider/user/admin, with binding context preserved only if the server still reports it. | Server reports that the provider connection can no longer be used for provider metadata refresh. | It does not delete RepoBinding, delete Goalrail Project state, remove local markers, or prove checkout is impossible for customer-hosted or metadata-only modes. | Future provider connection lifecycle API and server-side revocation state. |
| Metadata-only | The repository context or binding is marked metadata-only and warning copy is visible. | Goalrail has repository identity or metadata for context/binding only. This matches the current low-level server direction for RepoBinding init and repository context snapshots. | It does not mean Goalrail can clone, inspect code, run checks, read private files, access branches, execute runtime work, gate, or generate proof. | Existing metadata-only RepoBinding/server init can create this kind of state. Console display still needs a server-backed read/context API; provider metadata-only listing needs future provider-neutral APIs. |

## GitLab-first picker model

GitLab is the first provider candidate for the picker model, but the UX must
stay provider-neutral.

GitLab vocabulary rules:

- GitLab groups, subgroups, projects, and repositories are provider metadata.
- A GitLab Group is not a Goalrail Organization.
- A GitLab project is not a Goalrail Project.
- A GitLab repository is not a RepoBinding until the Goalrail server creates or
  returns a RepoBinding for a Goalrail Project.

Picker behavior before provider-neutral backend APIs:

- The GitLab candidate may be shown only as disabled, unavailable, or planned.
- The UI may explain that provider connection is not live yet.
- The UI must not show fake provider accounts, fake avatars, fake groups, fake
  projects, fake repositories, fake branches, or fake connection timestamps.
- The UI must not include an enabled Connect, Reconnect, Select, Bind, Scan, or
  Clone action.

Picker behavior after provider-neutral backend APIs exist:

- The picker may show provider metadata returned by the server.
- Provider account identity, connection health, repository listing, and search
  must come from backend APIs, not browser-only mocks.
- Selecting a provider metadata item is only a pending UI choice until the
  server creates or returns the relevant RepoBinding.
- The picker may show enough metadata for a human to choose a repository, but
  must not claim repository scan, checkout, branch analysis, code inspection,
  execution, gate, proof, or readiness unless those behaviors are implemented
  and documented separately.

## Project-to-RepoBinding binding model

Goalrail Project-to-repository binding uses Goalrail-native objects:

- Goalrail Organization is the internal workspace / tenant.
- Goalrail Project is the delivery container inside that Organization.
- RepoBinding is the project-to-repository reference.
- RepoBinding belongs to exactly one Goalrail Project.
- One MVP Goalrail Project should have at most one active primary RepoBinding
  unless later canon changes.
- RepoBinding may store repository reference directly in the MVP.
- RepositoryRecord remains deferred until repository catalog, multi-project
  repository reuse, repo-level policy, or independent provider sync requires it.
- VcsConnection remains a future provider connection layer and is not required
  for direct metadata-only RepoBinding.

The console binding flow, when later implemented, should make the context
explicit:

```text
Goalrail Organization -> Goalrail Project -> active primary RepoBinding
```

Binding confirmation should show the selected Goalrail Organization, selected
Goalrail Project, selected provider metadata, and the RepoBinding warning copy
before the server write.

## Selected working context in the console

Selected Organization / Project / RepoBinding context should eventually appear
in the console after server-backed context selection exists.

Rules:

- The selected context should be visible enough that users know which Goalrail
  Organization, Goalrail Project, and RepoBinding they are working in.
- The context must come from server-owned state or a server-validated context
  endpoint.
- The context must not be stored in browser storage before a documented storage
  boundary exists.
- The context must not replace server-side authorization, membership checks, or
  RepoBinding truth.
- The context must not imply checkout permission, provider token validity,
  repository scan, code inspection, runner readiness, gate status, or proof.
- The CLI login profile may store one selected Organization / Project /
  RepoBinding working context for CLI calls, but that does not make the browser
  console profile a source of truth.

## Required warning copy

The console repository settings surface must include this language, verbatim or
with only grammar-preserving surrounding text:

```text
RepoBinding identifies a repository.
RepoBinding is not checkout permission.
Metadata-only connection does not mean Goalrail can clone or inspect code.
```

Recommended contextual warning:

```text
Goalrail uses repository connection to identify the repository a Goalrail Project works with.
It does not grant checkout, scan, execution, gate, or proof behavior by itself.
```

## Frontend allowed before backend APIs

Before provider-neutral backend APIs exist, frontend work is limited to:

- static docs-backed Settings -> Integrations -> Repositories entry
- disabled or unavailable placeholder, if clearly non-live
- explanatory empty state
- required warning copy
- no-provider-connected state
- unavailable GitLab candidate card, only if disabled and clearly not connected
- links to relevant docs if useful
- no browser storage beyond the existing `goalrail.console.theme` key

Allowed copy must use future or unavailable language. It must not say Connected,
Synced, Selected, Bound, Scanned, Cloned, Verified, Ready, Passing, Failing, or
Proof unless backed by implemented server state for that exact behavior.

## Must wait for provider-neutral backend APIs

The following must wait for provider-neutral backend APIs and documented server
state:

- live provider connection status
- OAuth / connect / reconnect / revoked behavior
- provider account identity
- GitLab groups/projects/repositories list
- repository metadata listing and search
- selecting a repository through UI from provider metadata
- binding a repository through UI
- provider token handling
- VcsConnection implementation
- RepositoryRecord implementation
- any real reconnect or revoked state
- any browser or console API client for provider APIs

Provider token handling must be server-side or runtime-side according to a later
documented provider boundary. It must not be introduced as browser storage.

## Anti-fake-state rules

The console repository settings surface must avoid product-looking mock state:

- no seeded fake repositories
- no fake provider avatars or accounts
- no Connected UI without real backend state
- no metadata available rows without real provider-neutral API data
- no selected repository claim without user action and server-backed metadata
- no bound repository claim without server-owned RepoBinding truth
- no reconnect needed or revoked state without server-reported provider
  lifecycle state
- no Scan, Clone, Proof, Gate, Runner, Verified, Pass, Fail, Ready, Branch
  analysis, Code inspection, or Checkout wording in repository settings unless
  backed by implemented behavior and documented boundaries
- no localStorage, sessionStorage, cookies, IndexedDB, or browser cache for
  provider state, repository metadata, selected repository, RepoBinding context,
  provider account identity, or tokens before a documented storage boundary
  exists

## Later implementation phases

Suggested sequence after this docs-only boundary:

1. Add a static console placeholder under Settings -> Integrations ->
   Repositories with the no-provider-connected state and warning copy.
2. Define provider-neutral backend APIs for provider connection status,
   repository metadata listing/search, and authorized RepoBinding reads before
   enabling live UI.
3. Add GitLab as the first provider adapter behind the provider-neutral API,
   keeping GitLab groups/projects as provider metadata.
4. Add console repository picker and binding only after backend state exists.
5. Define checkout, runner, scan, gate, and proof UI only in their own later
   bounded slices.

This sequence is advisory. It does not mark any later phase as implemented.

## Non-goals

This document does not define or implement:

- backend implementation
- OAuth implementation
- GitLab API calls
- provider token storage
- database schema changes
- VcsConnection implementation
- RepositoryRecord implementation
- full RepoBinding CRUD
- checkout
- repository scan
- runner
- proof
- gate
- public signup
- admin user management
- SSO
- billing
- analytics
- live frontend provider UI
- frontend API client code for provider APIs
- mock live provider data
- changes to product surface navigation beyond documenting future Settings UX
