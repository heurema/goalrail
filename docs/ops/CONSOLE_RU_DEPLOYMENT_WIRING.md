---
id: console_ru_deployment_wiring
title: Console RU — Deployment Wiring
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - console_ru_deployment_wiring
  - public_console_shell
lifecycle: active-core
review_after: 2026-08-01
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/DECISIONS.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
  - apps/web/console-ru/README.md
---
# Console RU — Deployment Wiring

## Current status

**LIVE VIA SSH STATIC SERVER — SMOKE PASSED.**

The RU console shell from `apps/web/console-ru` has been built locally,
uploaded to the operator-managed SSH static server, exposed through the
server-side `current` symlink, and served at `https://console.goalrail.ru/`.

This is a static visual shell only. It does not add production auth, sessions,
cookies, persistence, backend routes, analytics, CRM, LLM/API calls, repo
integration, runtime execution, gate, proof, or any product data loop.

Server hostnames, IP addresses, SSH ports, usernames, key paths, credentials,
DNS provider details, concrete Nginx config, and private keys are intentionally
not recorded in this repository.

## Decision basis

- D-0022 remains in force: the RU console shell lives in
  `apps/web/console-ru` and targets `console.goalrail.ru`.
- The console remains a prototype/public packaging surface only, not a mature
  Goalrail web product loop.
- Future cards and detail views must wait for underlying CLI/server
  functionality instead of implying unimplemented backend behavior.

## Target surface

| Field | Value |
|-------|-------|
| Domain | `console.goalrail.ru` |
| Canonical URL | `https://console.goalrail.ru/` |
| Public path | `/` |
| App | `apps/web/console-ru` |
| Hosting path | operator-managed SSH static server, colocated with the RU pilot static server |
| Release root | `/srv/goalrail/console-ru/releases` |
| Current symlink | `/srv/goalrail/console-ru/current` |
| Active release | `/srv/goalrail/console-ru/releases/20260503T153849Z-console-ru-ux-polish` |
| Runtime/backend | none |
| Current deployment status | **LIVE VIA SSH STATIC SERVER — SMOKE PASSED** |

## Run result

### Local build / preflight

Run from `apps/web`:

- `npm run console-ru:typecheck` — PASS.
- `npm run console-ru:test` — PASS, 9 tests.
- `npm run console-ru:build` — PASS.
- `apps/web/console-ru/dist/index.html` exists.
- Built canonical is `https://console.goalrail.ru/`.
- Source boundary scan passed: no app `fetch`, `XMLHttpRequest`, `sendBeacon`,
  `document.cookie`, `sessionStorage`, analytics/tracking strings, provider API
  references, file input, model selector, or chat-history surface in
  `apps/web/console-ru/src` or `apps/web/console-ru/index.html`. The only
  accepted browser storage usage is the local visual theme preference under
  `goalrail.console.theme`.

### Upload and symlink

- A new timestamped release directory was created under
  `/srv/goalrail/console-ru/releases`.
- `apps/web/console-ru/dist/` was uploaded with `rsync --delete` through the
  operator-provided deploy SSH target.
- Remote release verification passed:
  - release `index.html` exists;
  - release `assets/` exists;
  - release canonical contains `https://console.goalrail.ru/`;
  - no `console.goalrail.dev` or `pilot.goalrail.ru` reference was found in the
    uploaded release.
- `/srv/goalrail/console-ru/current` was atomically switched to the new
  timestamped release using the same `ln -sfn` + `mv -Tf` pattern as the RU
  pilot static release.
- On 2026-05-03, a follow-up static release
  `20260503T091453Z-console-ru-rail-switch` refreshed the console brand mark
  from the hamburger-like three-line mark to Rail Switch Mark v0 and switched
  `/srv/goalrail/console-ru/current` to that release.
- On 2026-05-03, a second follow-up static release
  `20260503T092658Z-console-ru-wordmark-only` removed the custom mark after
  visual review and switched the console shell to wordmark-only branding.
- On 2026-05-03, release `20260503T093017Z-console-ru-minimal-login` removed
  the login-card heading and helper copy, leaving only the wordmark and login
  fields.
- On 2026-05-03, release `20260503T153849Z-console-ru-ux-polish` refreshed the
  live console with Appearance theme switching, structured empty states for the
  three product surfaces, and the local-only `goalrail.console.theme` visual
  preference. Users, login state, product surfaces, sessions, cookies, API
  calls, analytics, and backend behavior were not persisted or introduced.

### Nginx / TLS

- A server-local Nginx vhost for `console.goalrail.ru` was installed outside
  repo source control.
- The vhost is static-only and rooted at `/srv/goalrail/console-ru/current`.
- It does not define API locations or proxy to a backend.
- `nginx -t` passed.
- Nginx reload succeeded.
- Certbot issued and installed a certificate for `console.goalrail.ru`.
- Console-specific `certbot renew --dry-run --cert-name console.goalrail.ru`
  passed.
- A whole-host `certbot renew --dry-run` also attempted the unrelated
  `pilot.goalrail.ru` certificate and reported a pilot renewal dry-run failure.
  The console-specific renewal dry-run passed; the pilot renewal result should
  be handled as separate operator follow-up.

### Public smoke

- DNS for `console.goalrail.ru` resolves to the operator-managed server.
- HTTP redirects to `https://console.goalrail.ru/`.
- Public HTTPS smoke at `https://console.goalrail.ru/`: PASS for HTTP 200,
  canonical metadata, static asset paths, `X-Content-Type-Options`,
  `Referrer-Policy`, and no `Set-Cookie` response header.
- Remote static-root smoke over SSH: PASS.
- Deployed source-target check: `data-deployment-target="console.goalrail.ru"`
  is present in the built app bundle.
- Public source boundary is unchanged from local source: no app API surface,
  backend route, analytics/tracking, provider API, file upload, model selector,
  or chat-history feature is introduced by deployment. Browser storage remains
  limited to the local visual theme preference under `goalrail.console.theme`.
- Public UX-polish smoke passed: the deployed JavaScript bundle contains the
  structured empty-state copy for `Оценка готовности`, `Проверка результата`,
  `Goal → Contract → Task → Proof`, and the `goalrail.console.theme` key.
- Public brand smoke passed: the deployed JavaScript bundle contains the
  `GOALRAIL` wordmark and no longer contains `svg.brandMark` or the rejected
  Rail Switch Mark v0 paths.
- Public minimal-login smoke passed: the deployed JavaScript and CSS bundles no
  longer contain `GoalRail Console`, `Вход в рабочее пространство`, `Доступ
  выдает`, `loginSubtitle`, or `loginHelper`.

## Rollback note

The current release can be rolled back by switching
`/srv/goalrail/console-ru/current` back to a previously-known-good release with
the same atomic `ln -sfn` + `mv -Tf` pattern, then rerunning server-local and
public HTTPS smoke.

No release deletion was performed in this run.

## Next action

1. Keep console content constrained to the visual shell until real CLI/server
   functionality exists.
2. If a future console deploy adds data, auth, API calls, analytics, or
   backend routes, record a separate decision before implementation.
3. Investigate the unrelated `pilot.goalrail.ru` renewal dry-run failure as a
   separate operator task.
