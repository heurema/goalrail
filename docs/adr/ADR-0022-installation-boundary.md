# ADR-0022 — Installation boundary

Status: accepted
Date: 2026-05-03

## Context

ADR-0010 defines the current MVP `Organization -> Project -> RepoBinding`
persistence boundary. That model remains valid, but the next auth, CLI login,
and SaaS onboarding discussions need one explicit boundary above
`Organization`.

Without that boundary, self-hosted bootstrap, future SaaS tenancy, canonical
public URLs, first-user language, and organization-aware server code can blur
together too early. Goalrail needs the term `Installation` before implementing
auth, JWT, CLI login, SaaS onboarding, organization creation APIs, or web setup
flows.

This ADR is documentation-only. It defines the product and data-model boundary
that a later schema slice should implement.

## Decision

Goalrail accepts `Installation` as the concrete running Goalrail control plane
or instance.

An `Organization` remains the tenant/workspace boundary inside an
`Installation`.

The existing ADR-0010 model remains valid:

```text
Installation
  -> Organization
  -> Project
  -> RepoBinding
```

`Self-hosted` is a deployment mode, not a separate product model. Goalrail has
only two deployment modes:

- `self_hosted`
- `saas`

No `managed_dedicated`, `dedicated_enterprise`, or reserved dedicated enterprise
deployment mode is documented or reserved by this boundary. If a later customer
needs special operation, that must be expressed through deployment practice,
commercial packaging, infrastructure topology, or a new explicit decision
without adding a third core deployment mode by implication.

MVP starts with `self_hosted` mode.

In `self_hosted` mode:

- one primary `Organization` is bootstrapped for the `Installation`
- organization creation is disabled
- the backend must still remain organization-aware
- no self-hosted shortcut may bypass `organization_id`

Future `saas` mode may support multiple `Organization` records in one Goalrail
service.

Managed, guided, or founder-led rollout language describes deployment practice
and service motion. It does not create additional Goalrail deployment modes.

## Installation fields

The `Installation` should own `public_base_url` as the canonical externally
visible URL for the running Goalrail control plane.

`public_base_url` is required for real bootstrap. It must be normalized without
a trailing slash.

`public_base_url` must use HTTPS except for localhost/dev URLs. Examples:

- `https://goalrail.example.com`
- `https://console.example.com/goalrail`
- `http://localhost:8080`

The later schema slice should define the exact validation rules, but the
boundary is:

- production/self-hosted bootstrap requires `public_base_url`
- SaaS bootstrap requires `public_base_url`
- localhost/dev can use HTTP
- non-localhost real deployments require HTTPS
- stored value has no trailing slash

## First user and roles

Product and admin language may call the first user a `super admin`.

The data model should map that user to an `OrganizationMembership` role
`owner` for the bootstrapped primary `Organization`. This ADR does not create a
separate super-admin role in the MVP data model.

Future SaaS operator/admin roles may be defined by a later ADR if the control
plane needs service-wide administration. That is not part of this boundary.

## CLI naming

ADR-0003 remains valid. The canonical CLI binary is `goalrail`.

This boundary does not rename the CLI to `glr`, and no login or bootstrap
command should introduce `glr` as the canonical name.

## Consequences

Backend code must remain organization-aware even when running in
`self_hosted` mode with a single bootstrapped primary `Organization`.

Self-hosted code paths must not special-case away `organization_id` or collapse
`Organization` into `Installation`.

Future SaaS work can add multiple organizations inside one service without
changing the ADR-0010 `Organization -> Project -> RepoBinding` chain.

Installation mode stays a small enum/check boundary with only `self_hosted` and
`saas` accepted values.

`public_base_url` becomes installation-owned configuration rather than
organization-owned, project-owned, CLI-owned, or ad hoc environment-only truth.

## Rejected alternatives

### Treat Organization as the running instance

Rejected. It makes self-hosted bootstrap convenient but blocks clean SaaS
multi-organization semantics and blurs tenant/workspace identity with the
running service instance.

### Treat self-hosted as a separate product model

Rejected. Self-hosted and SaaS share the same Goalrail product model and core
object chain. They differ in deployment mode and bootstrap rules.

### Reserve managed dedicated / dedicated enterprise mode now

Rejected. The current product boundary only needs `self_hosted` and `saas`.
Reserving a third mode would create product and schema surface before there is
evidence that the core model needs it.

### Allow self-hosted code to omit organization_id

Rejected. It would make the MVP simpler locally but would create a different
data path from SaaS and undermine organization-aware authorization,
configuration, and audit semantics.

### Make super admin a separate MVP data role

Rejected. First-user product language can say `super admin`, but the current
data model maps it to `OrganizationMembership(owner)`.

## Non-goals

This ADR does not implement or define:

- server code
- migrations
- installation schema
- auth
- JWT
- refresh tokens
- CLI login
- CLI changes
- web UI
- SaaS onboarding
- billing
- SSO/OIDC
- organization creation API
- runner, gate, or proof behavior
- generic queue behavior
