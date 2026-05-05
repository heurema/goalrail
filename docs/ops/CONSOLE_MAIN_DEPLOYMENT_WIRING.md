---
id: console_main_deployment_wiring
title: Console Main — Deployment Wiring
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - console_main_deployment_wiring
  - main_console_api_routing
lifecycle: active-core
review_after: 2026-08-05
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
  - apps/web/README.md
  - apps/web/console/README.md
  - apps/server/README.md
---
# Console Main — Deployment Wiring

## Current status

**LIVE VIA `11me/infra` FLUX GITOPS — SMOKE PASSED.**

The main Goalrail console and API are now deployed through the external
`11me/infra` Flux GitOps path:

- frontend: `https://goalrail.dev`
- API: `https://api.goalrail.dev`
- console source: `apps/web/console`
- API source: `apps/server`
- deployment source of truth: `11me/infra`

The demo sandbox remains separate at `https://demo.goalrail.dev`. The legacy
`https://console.goalrail.ru/` deployment remains separate and is not migrated
by this slice.

This document records live routing and smoke evidence only. It does not claim
that Goalrail has a complete product web loop, durable console product data,
real Delivery Readiness / Proof backend data, runner execution, gate decisions,
repo checkout, tracker sync, analytics, CRM, or SaaS onboarding.

Private server hostnames, IP addresses, SSH ports, usernames, key paths,
kubeconfig contents, provider tokens, database credentials, private keys,
concrete reverse-proxy snippets, and private infrastructure details are
intentionally not recorded in this repository.

## Source and deployment truth

| Field | Value |
|-------|-------|
| Frontend public URL | `https://goalrail.dev` |
| API public URL | `https://api.goalrail.dev` |
| Console source | `apps/web/console` |
| API source | `apps/server` |
| Deployment source of truth | `11me/infra` |
| Infra merge revision | `main@sha1:f4cb3db22853d0d92291f37acb055cd28e8abec7` |
| Flux Kustomization | `flux-system/apps-personal` |
| Console source ref used by infra build | `060bbe8e6f306861f81732ab83171059af786505` |
| Console API build env | `VITE_GOALRAIL_API_BASE_URL=https://api.goalrail.dev` |
| Console default locale build env | `VITE_GOALRAIL_DEFAULT_LOCALE=en` |
| Demo sandbox | `https://demo.goalrail.dev`, separate |
| Legacy RU console | `https://console.goalrail.ru/`, separate / not migrated |

## Live verification

Flux and rollout evidence from 2026-05-05:

- Flux source revision was
  `main@sha1:f4cb3db22853d0d92291f37acb055cd28e8abec7`.
- Flux Kustomization `flux-system/apps-personal` reported `Ready=True`.
- `goalrail-console` rollout completed successfully.
- `goalrail-server` rollout completed successfully.
- `goalrail.dev` resolved publicly.
- `api.goalrail.dev` resolved publicly.
- `goalrail-dev-tls` reported `Ready=True`.
- `api-goalrail-dev-tls` reported `Ready=True`.

Public smoke:

- `https://goalrail.dev/` returned HTTP 200.
- `https://api.goalrail.dev/livez` returned `{"status":"ok"}`.
- `https://api.goalrail.dev/readyz` returned `{"status":"ok"}`.
- `https://api.goalrail.dev/version` returned
  `{"service":"goalrail-server","version":"0.0.0-dev"}`.
- The frontend response had no `Set-Cookie` header.
- The frontend HTML and JavaScript bundle did not contain
  `console.goalrail.dev`.
- The frontend JavaScript bundle contained `https://api.goalrail.dev`.

Browser CORS preflight smoke:

- `OPTIONS https://api.goalrail.dev/v1/auth/login` with
  `Origin: https://goalrail.dev` returned HTTP 204.
- The response allowed origin `https://goalrail.dev`.
- The response allowed methods `GET, POST, PATCH, OPTIONS`.
- The response allowed headers `Authorization, Content-Type`.

## Temporary CORS bridge

The live `goalrail-server` image at verification time still predates the
Goalrail server source change from PR #120. That image does not implement
application-level `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS`.

For the current deployment, CORS is therefore intentionally provided by nginx
ingress annotations in `11me/infra`, allowing only `https://goalrail.dev` for
the API browser preflight path.

The source-level application CORS implementation exists in `apps/server` after
Goalrail PR #120, but `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS` is intentionally not
enabled in infra yet. A later infra cleanup must pin a post-PR-#120
`goalrail-server` image, enable app-level
`GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS=https://goalrail.dev`, and remove the nginx
ingress CORS annotations in the same change to avoid duplicate
`Access-Control-Allow-Origin` headers.

## Non-goals

- No product source code was changed by this ops record.
- No infra source, deployment automation, kubeconfig, secrets, credentials,
  provider tokens, private hosts/IPs, SSH details, or reverse-proxy snippets
  are recorded here.
- No demo sandbox migration is recorded; `https://demo.goalrail.dev` remains
  separate.
- No legacy RU console migration is recorded; `https://console.goalrail.ru/`
  remains separate.
- No full Goalrail product web loop is claimed.
- Contracts, Delivery Readiness, and Proof console surfaces still must not be
  represented as data-backed product features until their backend boundaries
  exist.
