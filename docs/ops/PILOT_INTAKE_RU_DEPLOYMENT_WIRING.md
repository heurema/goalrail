---
id: pilot_intake_ru_deployment_wiring
title: Pilot Intake RU — Deployment Wiring
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - pilot_intake_ru_deployment_wiring
  - public_demo_candidate
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md
  - docs/ops/DECISIONS.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
---
# Pilot Intake RU — Deployment Wiring

## Current status

**LIVE VIA SSH STATIC SERVER — SMOKE PASSED.**

The business-first RU pilot landing from `apps/web/pilot-intake-ru` has been
built locally, uploaded to the operator-managed SSH static server, and exposed
through the server-side `current` symlink. Public DNS for
`pilot.goalrail.ru` now reaches the operator-managed server, public HTTPS smoke
passes, and same-origin `POST /api/pilot-lead` public smoke passes.

D-0056 allows one narrow lead-capture exception for this surface:
`POST /api/pilot-lead` validates an email, dedupes already-submitted addresses
by the local JSONL lead log, sends a notification to the configured
notification recipient, and records local JSONL notification status. D-0057
allows a server-local direct recipient override; D-0073 sets
`pilot@goalrail.dev` as the public/manual pilot contact address and requires
the visible Telegram channel `@goalrail`. D-0058 allows a server-local daily digest
from the same JSONL lead log at 07:00 GMT+3 for the previous local calendar day;
empty days send no digest. D-0059 allows Resend HTTPS transport through
`skill7.dev` for the same notification/digest emails; the API key is configured
server-locally at `/srv/goalrail/pilot/backend/resend-api-key.local`. D-0061
keeps failed notification attempts retryable without allowing concurrent
duplicate notification attempts. D-0062 migrates the active repo source for the
endpoint/digest from transitional PHP-FPM scripts to a narrow Go sidecar under
`apps/web/pilot-intake-ru/server`. D-0065 minimizes new lead records by
omitting user-agent, adds a local dry-run-first JSONL purge command with a
90-day default retention window, and keeps reverse-proxy rate limiting as
operator-managed deployment posture rather than repo-side server config.

The previous operator-managed server install used PHP-FPM. On 2026-04-30, that
live endpoint wiring was migrated to the repo-side Go sidecar through
operator-managed process supervision and reverse proxy configuration outside
this repository. Reverse-proxy rate limiting was applied as operator-managed
posture; no concrete Nginx/Caddy config is committed here. No analytics,
tracking, Google Sheets, CRM, cookies, sessions, user accounts, LLM/API calls,
repo integration, runtime execution, broad backend platform, CI/CD deployment
workflow, deploy script, concrete reverse-proxy config, or repo-side server
config was added.

## Decision basis

- D-0047 remains in force except for the narrow D-0056 lead-capture exception:
  no analytics, tracking, broad backend platform, persistence beyond the local
  JSONL lead log, LLM/API, repo integration, runtime execution, chat UI, file
  upload, model selector, or real scan claim.
- D-0048 remains in force: `apps/web/pilot-intake-ru` is approved as the RU
  pilot-first public candidate surface.
- D-0049 remains superseded by D-0053 for target domain and canonical public
  URL.
- D-0050 remains superseded by D-0051 for hosting provider and deployment
  mode; Cloudflare Pages / Workers / proxy / CDN / Web Analytics remain out of
  scope for this surface.
- D-0051 remains in force: operator-managed SSH static server, manual static
  upload, timestamped release directory, atomic `current` symlink, externally
  managed DNS, server-managed HTTPS, no automatic redeploys.
- D-0053 remains in force: active target domain and canonical URL are
  `https://pilot.goalrail.ru/`.
- D-0055 remains in force: the business-first Founding Pilot landing supersedes
  the old technical interactive walkthrough as the primary public RU landing.
- D-0056 remains in force: the RU landing may use only the narrow
  `POST /api/pilot-lead` endpoint for lead capture. It may validate email,
  dedupe by the local JSONL lead log, send notification email, write/update
  local JSONL notification status, omit user-agent from new lead records,
  provide a local retention purge command, and use local sendmail/Postfix
  fallback where available; it does not approve analytics, tracking, IP
  logging, fingerprinting, CRM, Google Sheets, cookies, sessions, user
  accounts, LLM/API calls, repo integration, runtime execution, or a broad
  backend platform.
- D-0057 remains in force: form notifications may use a server-local direct
  recipient override at `/srv/goalrail/pilot/backend/lead-recipient.local`. The
  override is operator-managed server state and the actual recipient address is
  not committed to repo docs/code/tests. If absent, the endpoint falls back to
  `pilot@goalrail.dev` per D-0073.
- D-0058 remains in force: a server-local daily digest may read the existing
  JSONL lead log, send one previous-day digest at `07:00 GMT+3` only when
  leads exist, and send nothing on empty days.
- D-0059 remains in force: Resend may be used only as a narrow HTTPS
  transactional transport for the existing lead notification/digest emails,
  with sender `GoalRail Pilot <noreply@skill7.dev>` and a server-local API key.
- D-0061 remains in force: `notification_failed` remains retryable, while
  `received` / `pending` are in-flight states and missing / unknown statuses are
  conservative duplicates.
- D-0062 remains in force: active repo source for the endpoint/digest is a
  landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, not PHP-FPM
  and not the core `apps/server` product API.
- D-0065 remains in force: new lead rows omit user-agent, local JSONL retention
  purge is dry-run by default, and abuse rate limiting is an operator-managed
  reverse-proxy guardrail without committed Nginx/Caddy config.
- D-0073 remains in force: the public/manual pilot contact email is
  `pilot@goalrail.dev`, the direct `mailto:` fallback uses that address, and
  the RU pilot landing exposes Telegram channel `@goalrail` at
  `https://t.me/goalrail`.

## Target surface

| Field | Value |
|-------|-------|
| Domain | `pilot.goalrail.ru` |
| Canonical URL | `https://pilot.goalrail.ru/` |
| Public path | `/` |
| App | `apps/web/pilot-intake-ru` |
| Hosting path | operator-managed SSH static server per D-0051 |
| Release root | `/srv/goalrail/pilot/releases` |
| Current symlink | `/srv/goalrail/pilot/current` |
| Lead endpoint | `POST /api/pilot-lead` |
| Endpoint source | `apps/web/pilot-intake-ru/server/cmd/goalrail-pilot-intake-ru` + `apps/web/pilot-intake-ru/server/internal/pilotlead` |
| Server endpoint mode | Go sidecar `serve` mode on an operator-managed local listen address |
| Local lead log | `/srv/goalrail/pilot/leads/leads.jsonl` |
| Direct recipient override | `/srv/goalrail/pilot/backend/lead-recipient.local` (server-local, not committed) |
| Daily digest source | Go sidecar `digest` mode |
| Shared mail transport | Go sidecar mail transport; Resend HTTPS when configured, local sendmail/Postfix fallback where available |
| Resend API key | `/srv/goalrail/pilot/backend/resend-api-key.local` (server-local, not committed) |
| Resend sender | `GoalRail Pilot <noreply@skill7.dev>` |
| Daily digest cron | `/etc/cron.d/goalrail-pilot-leads-digest`; `04:00 UTC` / `07:00 GMT+3`, previous GMT+3 day, only if leads exist |
| Repo migration status | Active repo source migrated from PHP to Go sidecar per D-0062 |
| Operator server migration status | Migrated from previous PHP-FPM wiring to the Go sidecar outside repo source control |
| Current deployment status | **LIVE VIA SSH STATIC SERVER — SMOKE PASSED** |
| Public live status | **LIVE** after public DNS, HTTPS, and `/api/pilot-lead` smoke passed |

Server hostnames, IP addresses, SSH ports, usernames, key paths, tokens,
credentials, and provider-specific identifiers are intentionally not recorded
in this repository.

## Run result

### SSH aliases / access

- Operator-provided root SSH target: reachable.
- Operator-provided deploy SSH target: reachable.
- No actual host, IP, username, port, or key path was recorded in repo docs.

### Remote bootstrap

- Remote OS path used `apt`; unsupported-distro blocker did not occur.
- Minimal static-hosting packages installed or confirmed present: `nginx`,
  `certbot`, `python3-certbot-nginx`, `rsync`, `ufw`.
- Previous D-0056 lead-capture packages installed or confirmed present:
  `php-fpm`, `postfix`, and local `mail` support. D-0062 changes repo source
  to Go; the active endpoint wiring has now migrated to the Go sidecar. Package
  removal, if any, remains an operator-managed server hygiene task and is not
  represented in repo source.
- Deploy user exists.
- Deploy SSH directory and authorized keys were ensured idempotently.
- SSH hardening drop-in was installed with:
  - `PasswordAuthentication no`
  - `PubkeyAuthentication yes`
  - `PermitRootLogin prohibit-password`
- SSH config test passed before reload.
- UFW was enabled with SSH, HTTP, and HTTPS allowed.

### Release layout and placeholder

- `/srv/goalrail/pilot/releases` exists.
- Placeholder release exists at `/srv/goalrail/pilot/releases/placeholder`.
- Placeholder `index.html` was written with correct heredoc syntax.
- `/srv/goalrail/pilot/current` was ensured and was later switched to the
  uploaded timestamped release.
- `current/index.html` is readable.

### Nginx

- `/etc/nginx/sites-available/pilot.goalrail.ru` was created/updated on the
  server as a static config rooted at `/srv/goalrail/pilot/current`.
- After D-0056, the same site config included a narrow
  `location = /api/pilot-lead` routed to PHP-FPM. On 2026-04-30, the active
  operator-managed wiring was switched to reverse proxy that same path to the
  local Go sidecar; the concrete server config is not committed here.
- The site is enabled through `sites-enabled`.
- `nginx -t` passed.
- Nginx reload succeeded.
- Server-local static smoke via Host header passed for index and asset bundle.
- `nginx -t` passed after adding the lead endpoint location.
- Nginx reload succeeded after adding the lead endpoint location.

### DNS / TLS

- Phase 8M DNS check result: historical blocker; DNS did not appear to point to
  the operator-managed server at that time.
- 2026-04-30 DNS verification: PASS. Public resolver comparison now shows
  `pilot.goalrail.ru` reaching the operator-managed server without recording
  the server IP in repo docs.
- Server static root verification: PASS. `/srv/goalrail/pilot/current` exists,
  `current/index.html` is readable, `current/assets/` exists, and the deployed
  canonical is `https://pilot.goalrail.ru/`.
- `nginx -t`: PASS.
- Certbot result: PASS on the operator-managed server.
- `certbot renew --dry-run`: PASS.
- Server-local HTTPS smoke with the `pilot.goalrail.ru` host: PASS for HTTP
  200 and canonical metadata.
- Public HTTPS smoke: PASS. A public `curl` request to
  `https://pilot.goalrail.ru/` returns HTTP 200, the canonical
  `https://pilot.goalrail.ru/`, and the hero `ИИ-кодинг без хаоса`.
- Public `/api/pilot-lead` smoke: PASS. A fresh operator smoke lead returned
  HTTP 200 / `{"duplicate":false,"ok":true}`, duplicate retry returned HTTP 200
  / `{"duplicate":true,"ok":true}`, invalid email returned HTTP 400, and
  honeypot returned HTTP 400.

### Lead capture endpoint

D-0056 allows the only server-side exception for this surface. Implemented
behavior:

- Frontend contact form submits JSON to same-origin `POST /api/pilot-lead`.
- Payload includes `email`, `source: ru-pilot`, `page: pilot.goalrail.ru`, and
  a hidden `website` honeypot.
- Invalid email is rejected client-side before calling `fetch`.
- Duplicate email submissions return success with a distinct duplicate message,
  without appending a new JSONL line or sending another notification.
- Direct fallback remains `mailto:pilot@goalrail.dev`.
- Active repo endpoint/digest source lives in the Go module at
  `apps/web/pilot-intake-ru/server`.
- The command entrypoint is
  `apps/web/pilot-intake-ru/server/cmd/goalrail-pilot-intake-ru`.
- Server deployment of the Go sidecar is complete as an operator-managed wiring
  step; this repo doc does not record live server hostnames, listen addresses,
  process managers, or reverse-proxy config.
- Local lead log path is `/srv/goalrail/pilot/leads/leads.jsonl`.
- New JSONL rows omit user-agent and do not replace it with IP logging, hashed
  IP, cookies, sessions, fingerprinting, or browser identifiers.
- Local retention command examples:
  - `goalrail-pilot-intake-ru purge` — dry-run by default.
  - `GOALRAIL_PURGE_CONFIRM=yes goalrail-pilot-intake-ru purge` — confirmed
    local purge.
- Retention defaults to 90 days and may be overridden with
  `GOALRAIL_LEAD_RETENTION_DAYS` in the bounded 7–365 day range.
- Reverse-proxy rate limiting is applied by the operator-managed web server /
  reverse proxy; no concrete Nginx/Caddy config is committed here.
- Public/manual contact remains `pilot@goalrail.dev`.
- Public Telegram channel remains `@goalrail` at `https://t.me/goalrail`.
- Notification recipient may be a server-local direct override from
  `/srv/goalrail/pilot/backend/lead-recipient.local`; the configured value is
  not stored in repo docs/code/tests.
- If the override file is absent, notification recipient falls back to
  `pilot@goalrail.dev`.
- Notification subject starts with `Пилот`.
- The Go sidecar prefers Resend HTTPS transport when
  `/srv/goalrail/pilot/backend/resend-api-key.local` exists and is valid.
- Resend sender: `GoalRail Pilot <noreply@skill7.dev>`.
- Resend API key status: configured on the operator-managed server; the key
  value is not recorded in repo docs/code/tests/logs.
- Resend transport smoke: PASS. A one-off current-day digest reported
  `transport=resend` and included the expected non-empty lead count.
- If the Resend API key is absent, the sidecar falls back to local
  sendmail/Postfix where available, with `noreply@pilot.goalrail.ru` as the
  envelope sender.
- Previous server-local Postfix wiring is operator-managed server state and is
  not committed here.
- UFW remains limited to SSH, HTTP, and HTTPS; inbound SMTP was not opened.
- Server-local Go sidecar process smoke: PASS.
- Server-local endpoint smoke through the reverse proxy: valid JSON lead
  returned HTTP 200 / `{ "ok": true, "duplicate": false }`.
- Server-local duplicate smoke: HTTP 200 / `{ ok: true, duplicate: true }`,
  with unchanged JSONL line count and no new notification.
- Server-local invalid email smoke: HTTP 400.
- Server-local honeypot smoke: HTTP 400.
- Lead log append smoke: PASS.
- New JSONL row smoke: PASS. The smoke row includes `notification_status`,
  `submitted_at`, `submitted_at_local`, and `submitted_date_local`; it omits
  `user_agent` and does not include IP, cookie, session, fingerprint, or browser
  identifiers.
- Previous email send smoke: PHP `mail()` accepted the notification and the
  local mail queue was empty after the smoke. Cloudflare Email Routing
  classified form-generated `noreply@pilot.goalrail.ru` mail to
  `hello@goalrail.dev` as `unauthenticatedForward`; this used the then-current
  manual address before D-0073 moved the public/manual pilot contact to
  `pilot@goalrail.dev`. D-0057 direct recipient override remains available for
  form notifications. This is historical server evidence, not a claim that PHP
  remains active repo source after D-0062.
- Direct recipient override status: configured on the operator-managed server
  with a validated operator-provided address; the address is not committed to
  repo docs/code/tests.
- Direct recipient override smoke: HTTP 200 / `{ ok: true, duplicate: false }`,
  lead log appended, and local mail queue remained empty after relay handoff.

Generic reverse-proxy shape for this endpoint after the Go sidecar migration:

```nginx
location = /api/pilot-lead {
    limit_except POST { deny all; }
    proxy_pass http://<local-go-sidecar-upstream>;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    # Visitor IP does not need to be forwarded to the Go sidecar.
}
```

The concrete process manager, listen address, reverse-proxy config, and restart
policy are operator-managed server wiring and are intentionally not committed.
The committed repository does not contain server hostnames, IPs, SSH users,
ports, key paths, credentials, live Nginx config, or direct recipient email
addresses.

Daily digest behavior:

- Source: Go sidecar `digest` mode from `apps/web/pilot-intake-ru/server`.
- Binary command: `goalrail-pilot-intake-ru digest`.
- Cron path: `/etc/cron.d/goalrail-pilot-leads-digest` on the operator-managed
  server.
- Schedule: `04:00 UTC`, equivalent to `07:00 GMT+3`.
- Window: previous GMT+3 calendar day.
- Empty previous day: no email is sent.
- One or more leads: one digest email is sent to the same D-0057 recipient
  selection.
- New JSONL rows include `submitted_at` (UTC), `submitted_at_local`, and
  `submitted_date_local` for digest/audit readability, and omit `user_agent`.
  Existing rows without local fields are converted from `submitted_at` when the
  digest is generated; legacy rows with `user_agent` remain readable but the
  digest does not include user-agent.
- Previous server install status: superseded. `pilot-lead.php` and
  `pilot-leads-digest.php` were installed under `/srv/goalrail/pilot/backend/`
  and passed `php -l` on the operator-managed server before D-0062. On
  2026-04-30, the active operator-managed endpoint wiring was migrated to the Go
  sidecar; PHP-FPM is no longer the active endpoint path for `/api/pilot-lead`.
- Cron install status: PASS. `/etc/cron.d/goalrail-pilot-leads-digest` runs the
  digest as `www-data` at `04:00 UTC`.
- Digest dry-run smoke: PASS. A non-empty local-day dry run reported that a
  digest would be sent; a known-empty day reported `no_leads` and exited
  without sending. No real digest email was sent during this smoke to avoid
  duplicate notification noise.
- Go sidecar digest dry-run smoke after live migration: PASS, exit 0.
- Go sidecar purge dry-run smoke after live migration: PASS, exit 0; dry-run
  output reported `would_purge` with 90-day retention and did not delete rows.
- One-off digest send smoke after envelope-sender alignment: server-side
  Postfix accepted and relayed the message, and the local queue was empty;
  mailbox delivery still depended on recipient/provider filtering and sender
  authentication posture.
- One-off digest send smoke after D-0059 Resend key installation: PASS; the
  digest reported `sent ... transport=resend` with the expected non-empty lead
  count.


### Local build / preflight

Run from `apps/web`:

- `npm run pilot-intake-ru:typecheck` — PASS.
- `npm run pilot-intake-ru:test` — PASS, 20 tests. Existing Vitest warning:
  `--localstorage-file was provided without a valid path`.
- `npm run pilot-intake-ru:build` — PASS.
- `apps/web/pilot-intake-ru/dist/index.html` exists.
- `apps/web/pilot-intake-ru/server/go.mod` exists for the landing-owned Go
  sidecar.
- `apps/web/pilot-intake-ru/server/cmd/goalrail-pilot-intake-ru` exists as the
  Go command source for `serve`, `digest`, and `purge` modes.
- `apps/web/pilot-intake-ru/server/internal/pilotlead` contains the JSONL store,
  HTTP handler, mail transport, digest behavior, purge behavior, and tests for
  the sidecar.
- Go sidecar pre-deploy validation on 2026-04-30: `go test ./...`, `go vet
  ./...`, and `go build ./cmd/goalrail-pilot-intake-ru` all passed from
  `apps/web/pilot-intake-ru/server`.
- `apps/web/pilot-intake-ru/dist/assets/` exists.
- Built canonical is `https://pilot.goalrail.ru/`.
- Built `dist/` contains no `pilot.goalrail.dev` references.
- Root-absolute asset paths are valid for `/`.

### Local preview smoke

Run from `apps/web`:

- `npm run preview --workspace @goalrail/pilot-intake-ru-web -- --host 127.0.0.1` — PASS.
- Page loads.
- Hero `ИИ-кодинг без хаоса` visible.
- Primary CTA `Обсудить пилот` visible.
- Canonical in DOM is `https://pilot.goalrail.ru/`.
- Contact email `pilot@goalrail.dev` visible.
- Telegram channel `@goalrail` visible.
- Console errors: 0.
- Failed requests: 0.
- Non-static requests on load: 0.
- Contact form visible. Valid-submit behavior is covered by Vitest with a
  mocked `/api/pilot-lead` response; local Vite preview does not run the Go
  sidecar.

### Boundary audit

Source grep against production files passed with the D-0056 exception:

- `fetch('/api/pilot-lead')` is the only allowed browser network call;
- no external `fetch`, `XMLHttpRequest`, `sendBeacon`;
- no `localStorage`, `sessionStorage`, `indexedDB`;
- no `gtag`, `googletagmanager`, `analytics`, `mixpanel`, `sentry`, `datadog`;
- no `openai`, `anthropic`, `claude.ai`;
- no `api.github`, `api.gitlab`;
- no `input type="file"`;
- no `model selector`;
- no `chat history`.

### Upload and symlink

- A new timestamped release directory was created under
  `/srv/goalrail/pilot/releases`.
- After D-0056, a newer timestamped static release containing the lead-capture
  frontend was uploaded and switched through the same atomic symlink pattern.
- `apps/web/pilot-intake-ru/dist/` was uploaded with `rsync --delete` through
  the operator-provided deploy SSH target.
- Remote release verification passed:
  - release `index.html` exists;
  - release `assets/` exists;
  - release canonical contains `https://pilot.goalrail.ru/`;
  - no `pilot.goalrail.dev` reference was found in the uploaded release.
- `/srv/goalrail/pilot/current` was atomically switched to the new timestamped
  release using `ln -sfn` + `mv -Tf`.
- `current/index.html` is readable.
- `current/assets/` exists.

### Public smoke

- Server-local static smoke via Nginx and Host header: PASS.
- Server-local HTTPS smoke with forced local resolution: PASS for HTTP 200 and
  canonical metadata.
- Remote static-root smoke over SSH: PASS.
- Public DNS resolver comparison: PASS; the public domain reaches the
  operator-managed server. The server IP is not recorded in repo docs.
- Public HTTPS smoke at `https://pilot.goalrail.ru/`: PASS for HTTP 200,
  canonical metadata, and hero copy.
- Public `/api/pilot-lead` smoke: PASS for fresh lead, duplicate retry, invalid
  email, and honeypot cases with the expected generic JSON responses.
- Public page-load boundary scan: PASS. Public HTML loads only same-origin
  static resources, sets no cookies, contains no analytics / tracking /
  LLM-provider / repo-provider calls, and submit remains same-origin
  `/api/pilot-lead`.
- Public deployed asset scan confirms `pilot@goalrail.dev` and `@goalrail`
  remain present and the primary CTA / lead-form focus behavior remains covered
  by the existing frontend test suite.
- Public live status: **LIVE VIA SSH STATIC SERVER — SMOKE PASSED**.

## Rollback note

No previous non-placeholder release was found during this run. The placeholder
release remains present. Once there is at least one previous real release, the
generic rollback procedure is:

1. Switch `/srv/goalrail/pilot/current` back to the previous release with the
   same atomic `ln -sfn` + `mv -Tf` pattern.
2. Re-run server-local smoke.
3. Re-run public HTTPS and `/api/pilot-lead` smoke after rollback.

No releases were deleted in this run.

## Next action

1. Perform a real-device mobile smoke pass on iOS Safari and Android Chrome.
2. Perform a native-speaker proofread pass of the live Russian copy.
3. Keep monitoring the operator-managed Go sidecar, local JSONL retention
   lifecycle, and reverse-proxy rate limiting posture without committing server
   config or secrets.
