# ADR-0023 — User bootstrap, auth, and CLI login boundary

Status: accepted
Date: 2026-05-03

## Context

ADR-0022 defines `Installation` as the running Goalrail control plane above
`Organization`, with `self_hosted` and `saas` as the only deployment modes.
The smallest Installation schema foundation now exists, but Goalrail still has
no auth, JWT, refresh-token store, CLI login, user management flow, SaaS
onboarding, organization creation API, or web UI.

Before adding auth schema or login endpoints, Goalrail needs one explicit
boundary for the self-hosted first-user path, user creation, token direction,
and CLI login shape. Without this boundary, auth work could blur product/admin
language with data roles, make the JWT carry stale authorization state, turn
CLI login into an implicit SaaS-only flow, or accidentally introduce public
registration and organization creation into the MVP.

This ADR is documentation-only. It defines the product and architecture
direction for later bounded schema and API slices.

## Decision

Goalrail self-hosted MVP uses bootstrap-first user creation.

In `self_hosted` mode, bootstrap creates the first super admin for the
bootstrapped primary `Organization`.

`Super admin` is product/admin language. In the MVP data model, it maps to an
`OrganizationMembership` with role `owner` for the bootstrapped primary
`Organization`. This ADR does not create a separate super-admin data role.

The MVP has no public registration.

After bootstrap, a super admin / admin creates users inside the Organization.
Created users receive a backend-generated temporary password. Created users
must change that temporary password on first login before normal product use.

Email invite delivery and password reset email delivery are deferred. The MVP
boundary may use generated temporary credentials and explicit admin/user
handoff, but it must not claim invite email, reset email, transactional email,
or mail transport for the core Goalrail product unless a later bounded slice
implements it.

Future auth schema should keep password credentials separate from the `users`
table. The direction is a dedicated credentials table such as
`user_password_credentials`, rather than storing password hash material directly
on `users`.

## Token direction

Goalrail should use a simple short-lived JWT access token.

Goalrail should use an opaque DB-backed refresh token. Refresh token state
belongs in a server-owned store such as `user_sessions` or a dedicated refresh
token table.

JWTs must not carry broad or stale permission state. They may carry narrow
identity/session claims needed for request authentication, but role and access
checks happen server-side through current `OrganizationMembership` state.

The server remains organization-aware. Auth must not create a self-hosted
shortcut that bypasses `organization_id`.

## CLI login

ADR-0003 remains valid. The canonical CLI binary remains `goalrail`.

The future CLI login command is `goalrail login`.

For self-hosted use, `goalrail login` requires an explicit `server_url`.
The CLI must not assume a Goalrail SaaS default server for self-hosted login.

CLI login uses a browser-based localhost loopback callback. The CLI opens or
prints an authorization URL, listens on localhost for the callback, and stores
the resulting authenticated profile after the server completes the login flow.

The CLI stores:

- `server_url`
- selected `organization_id` or organization profile
- selected `project_id` or project profile
- selected `repo_binding_id` or repo binding profile
- token/session material required for subsequent authenticated CLI calls

The stored CLI profile represents one selected Organization / Project /
RepoBinding working context for a server. It does not replace server-side
authorization, membership checks, or repo binding truth.

## SaaS onboarding

SaaS onboarding is deferred.

SaaS organization creation is deferred.

This ADR does not define public SaaS signup, organization creation API,
billing-gated onboarding, invitations, SSO/OIDC, or SaaS operator/admin roles.

## Consequences

The first real auth schema slice should add password credentials and refresh
token/session storage before login endpoints.

Bootstrap owner creation remains separate from public registration.

Admin-created users and first-login password change become explicit product
requirements for the self-hosted MVP.

Authorization checks must load current membership state from the server side
instead of trusting long-lived role claims embedded in JWTs.

The CLI login profile is tied to an explicit server URL and selected project
context, which keeps self-hosted and future SaaS flows compatible without
renaming the CLI or hard-coding a SaaS default.

## Rejected alternatives

### Public registration in the MVP

Rejected. Public registration would imply broader SaaS onboarding, abuse
handling, invite/reset email, and organization creation flows before the
self-hosted MVP needs them.

### Store password hashes directly on users

Rejected. User identity and password credentials should stay separate so later
auth methods, password rotation, credential disabling, and SSO/OIDC can evolve
without overloading the `users` table.

### Long-lived JWTs with embedded roles

Rejected. Role and permission state can change. Broad or stale JWT role claims
would weaken organization-aware authorization and make membership changes slow
to take effect.

### Refresh tokens as JWTs

Rejected. Opaque DB-backed refresh tokens give the server a revocation and
session-management boundary without trusting long-lived client-carried claims.

### CLI login without explicit server_url for self-hosted

Rejected. Self-hosted deployments have deployment-owned URLs. The CLI must bind
to the intended server explicitly instead of assuming a hosted default.

### Rename the CLI for auth work

Rejected. ADR-0003 keeps `goalrail` as the canonical binary. Auth and login do
not introduce a new canonical CLI name.

## Non-goals

This ADR does not implement or define:

- server code
- migrations
- JWT implementation
- password hashing implementation
- login endpoints
- bootstrap endpoint shape
- CLI changes
- web UI
- public registration
- email invite delivery
- password reset delivery
- SaaS onboarding
- organization creation API
- billing
- SSO/OIDC
- runner, gate, or proof behavior
- generic queue behavior
