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
  - docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md
  - docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
---
# Pilot Intake RU — Deployment Wiring

## 1. Purpose

This note records the deployment-wiring status for `apps/web/pilot-intake-ru`
under D-0053's chosen target (active target `pilot.goalrail.ru` / public
path `/` / public status `candidate-public`; D-0053 supersedes D-0049
for target domain and canonical public URL — D-0049's original
`pilot.goalrail.dev` is now reserved for a later global-market
rollout) and D-0051's hosting path (operator-managed SSH static
server). It does **not** record an actual live deployment. The surface
is not deployed, not live, and no remote state has been changed by
this note.

Wiring date: 2026-04-28 (Phase 8H); same-day re-attempt (Phase 8I);
Phase 8J stopped at env gate; Phase 8K target-domain realignment to
`.ru`.
Owner: ops.

## 2. Decision basis

- `docs/ops/DECISIONS.md` D-0047 — public landing demo remains
  local-only and deterministic (firm boundary).
- `docs/ops/DECISIONS.md` D-0048 — apps/web/pilot-intake-ru approved
  as candidate public RU pilot-first landing demo.
- `docs/ops/DECISIONS.md` D-0049 — original target domain
  `pilot.goalrail.dev`, public path `/`, hosting target `static CDN
  target TBD`, public status `candidate-public`. Superseded by
  D-0053 for target domain and canonical public URL; body preserved
  as historical record.
- `docs/ops/DECISIONS.md` D-0053 — active target domain
  `pilot.goalrail.ru` (supersedes D-0049 for target domain and
  canonical public URL); public path `/` and public status
  `candidate-public` are preserved; the `.dev` domain is reserved
  for a later global-market rollout. Any further change of domain
  requires a separate decision.
- `docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md` — Phase 7C
  internal review evidence (recommendation `READY WITH WARNINGS`).
- `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md` — Phase 8B
  deployment-prep evidence (recommendation `READY WITH WARNINGS`).
- `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md` — canonical copy
  and governance reference.

## 3. Target surface

| Field | Value |
|-------|-------|
| Domain | `pilot.goalrail.ru` (active per D-0053; supersedes D-0049's `pilot.goalrail.dev`, which is reserved for a later global-market rollout) |
| Public path | `/` |
| Public status | `candidate-public` |
| Hosting target | `static CDN target TBD` (concrete provider not picked) |
| Live status | **NOT DEPLOYED** — surface is not live, no remote state has been provisioned in this slice. |

## 4. Build and output

| Aspect | Value |
|--------|-------|
| App path | `apps/web/pilot-intake-ru` |
| Workspace package | `@goalrail/pilot-intake-ru-web` |
| Build command | `npm run pilot-intake-ru:build` (root: `npm run build --workspace @goalrail/pilot-intake-ru-web`); script body: `tsc -b && vite build` |
| Test command | `npm run pilot-intake-ru:test` |
| Typecheck | `npm run pilot-intake-ru:typecheck` |
| Preview command | `npm run pilot-intake-ru:preview` (workspace exposes `vite preview`); local default `http://localhost:4173/` |
| Output directory | `apps/web/pilot-intake-ru/dist/` |
| `dist/` contents | `index.html`, `favicon.svg`, `assets/index-*.{js,css}`, `assets/ibm-plex-{sans,mono}-*.{woff,woff2}` |
| `dist/` size | ~908 KB (fonts dominate); index-*.css 245.36 KB / gzip 36.95 KB; index-*.js 267.36 KB / gzip 79.05 KB |
| Vite `base` setting | not set in `vite.config.ts` → defaults to `/`, which matches D-0049 `PUBLIC_PATH=/`; **no config change required** |
| Root-path asset references | `<script src="/assets/index-*.js">`, `<link href="/assets/index-*.css">`, `<link href="/favicon.svg">` — all root-absolute, root-safe |
| Env vars at build | NONE (`grep -nE "import\.meta\.env\|process\.env\|VITE_"` against `src/*` returns no matches) |
| Env vars at runtime | NONE |
| Secrets required | NONE |

## 5. Hosting wiring status

**READY FOR SSH DEPLOY — RUNTIME VALUES REQUIRED.**

Phase 8H attempted SSH static deployment wiring per D-0051. Phase 8I
re-attempted the same slice. In both attempts the local preflight,
boundary audit, and local preview smoke passed (see §7 below). In both
attempts no remote SSH operation was performed because the required
runtime environment variables were not provided in the operator's
shell:

| Required env var | Present? |
|------------------|----------|
| `GR_PILOT_REMOTE_DEPLOY=yes` | NO |
| `GR_PILOT_SSH_TARGET` | NO |
| `GR_PILOT_RELEASE_ROOT` | NO |
| `GR_PILOT_CURRENT_LINK` | NO |
| `GR_PILOT_DOMAIN` | NO |

Per Phase 8H Scope C and Phase 8I Scope A, no SSH connection was
opened, no `rsync` or `scp` was run, and no remote state was touched.
The repo did not acquire any server identifiers, credentials,
hostnames, IP addresses, SSH ports, usernames, keys, or tokens. The
previously-pinned hosting target table (below) is unchanged.

Active provider decision: `docs/ops/DECISIONS.md` D-0051 —
**operator-managed SSH static server** for `pilot.goalrail.ru`
(active per D-0053; supersedes D-0049's `pilot.goalrail.dev`).
D-0051 explicitly supersedes D-0050 for hosting provider and
deployment mode; D-0049 is preserved.

| Field | Value |
|-------|-------|
| Hosting provider | operator-managed SSH static server |
| Hosting target detail | operator-managed Linux server reachable over SSH; exact host, IP address, SSH port, SSH user, and credentials are kept out of repo; static web root and release directory will be confirmed during deployment wiring |
| DNS strategy | DNS handled externally by the operator; `pilot.goalrail.ru` (per D-0053) will point to the SSH server or upstream reverse proxy using A / AAAA / CNAME as appropriate; if the DNS zone is currently managed through Cloudflare, the record must be DNS-only / non-proxied or otherwise configured so public traffic does **not** depend on Cloudflare Pages, Cloudflare proxy, Cloudflare Workers, or Cloudflare CDN services |
| TLS strategy | server-managed HTTPS via existing reverse proxy or Let's Encrypt; HTTPS for `https://pilot.goalrail.ru/` must be verified before any public use |
| Deployment mode | manual static upload over SSH after a local production build; preferred mechanism is `rsync` / `scp` to a timestamped release directory with an atomic `current` symlink switch; **no automatic redeploys**; no CI deploy workflow |
| Preview mode | local `vite preview` smoke check plus a server smoke check after manual upload; an optional staging vhost / path is allowed only if the operator explicitly provides one |

Supersession trail (kept for history):
- A repo-wide search in Phase 8D returned **no evidence** of an
  existing static hosting convention (none of `netlify.toml`,
  `vercel.json`, `wrangler.toml`, `fly.toml`, `firebase.json`,
  `.firebaserc`, `amplify.yml`, `_redirects`, `_headers`,
  `Dockerfile*`, `render.yaml`, `Caddyfile`, `nginx.conf`, root
  `Makefile`). Only CI workflow was
  `.github/workflows/docs-check.yml`. Phase 8D therefore left this
  slice `BLOCKED ON HOSTING PROVIDER SELECTION`. Phase 8E resolved
  the canonical-link metadata WARN.
- D-0050 then selected Cloudflare Pages Direct Upload as the
  provider, moving status to `PROVIDER SELECTED — WIRING PENDING`.
- D-0051 supersedes D-0050 for hosting provider and deployment mode:
  Cloudflare Pages Direct Upload is no longer the selected RU launch
  path. Cloudflare Pages, Workers, Functions, KV / R2 / D1 / Durable
  Objects / Queues, proxy / CDN, and Web Analytics remain disallowed
  for this surface. The active path is now an operator-managed SSH
  static server with the values pinned above.

What is unblocked by D-0051:
- the next slice may identify the operator-managed SSH server out of
  repo, confirm its static web root and release directory layout
  (without committing host / IP / credentials), define the manual
  upload procedure (rsync / scp), exercise it once, and switch the
  `current` symlink to expose the new release;
- adding `pilot.goalrail.ru` (per D-0053) to point at the server
  (or its upstream reverse proxy) via externally-managed DNS, with
  the record kept DNS-only / non-proxied if the zone is in
  Cloudflare;
- verifying HTTPS for `https://pilot.goalrail.ru/` (server-managed
  via existing reverse proxy or Let's Encrypt) before public use;
- running a server-side smoke check against the manually-uploaded
  release.

What remains explicitly out of scope under D-0047 + D-0048 + D-0049 +
D-0051 in the wiring slice:
- backend, persistence, LLM/API, repo provider integration, code
  execution;
- analytics or session tracking;
- CI deploy workflow / automatic Git-based deploys / any automatic
  redeploys;
- Cloudflare Pages, Workers, Functions, KV / R2 / D1 / Durable
  Objects / Queues, proxy / CDN, Web Analytics;
- email lead capture beyond `mailto:` / focus / manual handoff;
- any change of target domain or public path (D-0049 invariant);
- committing SSH keys, server credentials, hostnames, IP addresses,
  ports, usernames, deploy scripts, or reverse-proxy config to the
  repository.

## 6. Boundary verification (D-0047)

Source-level + preview-runtime verification.

| Boundary | Status | Evidence |
|----------|--------|----------|
| No backend (`fetch(`, `XMLHttpRequest`, `sendBeacon`) | PASS | source grep returns no matches in `src/{App.tsx,App.css,main.tsx,theme.ts}`; preview-runtime test recorded zero non-static network requests during a full walkthrough including primary outcome CTA click |
| No LLM/API endpoints | PASS | source grep returns no matches for `openai`, `anthropic`, `claude.ai`, `api.openai`, `api.anthropic` |
| No repo provider integration | PASS | source grep returns no matches for `api.github`, `api.gitlab`, `bitbucket.org/api`, `github.com/api` |
| No code execution / runtime | PASS | no `eval`, no `Function()` constructor, no Web Worker calls; React component memory only |
| No persistence | PASS | no `localStorage`, no `sessionStorage`, no `indexedDB`, no cookies set by app |
| No analytics / session tracking | PASS | no `gtag`, no `googletagmanager`, no `mixpanel`, no `sentry`, no `datadog`, no `analytics` references |
| No chat UI | PASS | tests assert absence of `chat history`, `assistant`, model selector, file upload across all 5 steps |
| No file upload | PASS | no `<input type="file">` |
| No model selector | PASS | no model `<select>` or model-named `combobox` |
| No backend email POST | PASS | `<form action="mailto:hello@goalrail.dev">` only; outcome primary CTA implementation calls only `target.focus()` + `target.scrollIntoView()` + temporary local CSS class |
| Email lead capture stays mailto/focus/manual | PASS | preview test confirmed `document.activeElement === pilot-email` after CTA click; mailto `<a href>` and `<form action>` preserved |

## 7. Smoke check

Performed on the production build via `npm run pilot-intake-ru:preview`
(default `http://localhost:4173/`) with browser automation. Re-run in
Phase 8H and again in Phase 8I to confirm nothing regressed since
Phase 8D / 8E. **No remote / domain smoke was run** — neither Phase 8H
nor Phase 8I opened an SSH connection (see §5).

| Smoke item | Result |
|------------|--------|
| `index.html` loads | PASS |
| Hero title (`Покажите задачу — мы проведём вас по контракту`) visible | PASS (`document.querySelector('h1#hero-title')` found) |
| 5-step rail visible (`[aria-label="Шаги демо"] li`) | PASS (5 items) |
| Full walkthrough reaches outcome (RG flow with clean answers) | PASS — `data-outcome-tone="ready"` |
| Primary outcome CTA focuses email input | PASS — `document.activeElement.id === "pilot-email"` |
| Console errors during walkthrough | 0 errors |
| Console warnings during walkthrough | 0 warnings |
| Non-static network requests during walkthrough | 0 (no `fetch`, no XHR, no `sendBeacon`) |
| `index.html` `<link rel="canonical" href>` value | `https://pilot.goalrail.ru/` (active per D-0053) — first realigned from `https://goalrail.ru/` to D-0049's `https://pilot.goalrail.dev/` in Phase 8E pre-publish hygiene patch, then realigned from `https://pilot.goalrail.dev/` to D-0053's `https://pilot.goalrail.ru/` in Phase 8K target-domain supersession (HTML-metadata only — no copy / behavior change in either pass); built `dist/index.html` line 12 contains the active `.ru` canonical after Phase 8K rebuild |
| `dist/` asset paths root-safe under `PUBLIC_PATH=/` | PASS — `/favicon.svg`, `/assets/index-*.{js,css}` |
| Phase 8H local preflight (typecheck/test/build) | PASS — typecheck 0 errors; tests 67/67; build ~208ms |
| Phase 8H source boundary scan (`fetch(`, `XMLHttpRequest`, `sendBeacon`, `localStorage`, `sessionStorage`, `indexedDB`, `gtag`, `analytics`, `mixpanel`, `sentry`, `datadog`, `openai`, `anthropic`, `claude.ai`, `api.github`, `api.gitlab`) | PASS — no matches in `index.html` / `App.tsx` / `App.css` / `main.tsx` / `theme.ts` |
| Phase 8H local preview smoke at `http://localhost:4173/` | PASS — full RG walkthrough → outcome `ready`; primary outcome CTA focuses email; canonical at the time = `https://pilot.goalrail.dev/` (then-active D-0049 value; later realigned to `https://pilot.goalrail.ru/` in Phase 8K per D-0053); 0 console errors, 0 console warnings, 0 non-static network requests |
| Phase 8I env gate check (`GR_PILOT_REMOTE_DEPLOY=yes` plus four required runtime values) | NOT SATISFIED — `printenv` showed none of `GR_PILOT_REMOTE_DEPLOY`, `GR_PILOT_SSH_TARGET`, `GR_PILOT_RELEASE_ROOT`, `GR_PILOT_CURRENT_LINK`, `GR_PILOT_DOMAIN` set; per Phase 8I Scope A this disallowed any remote SSH/rsync/scp activity |
| Phase 8I local preflight (typecheck/test/build) | PASS — typecheck 0 errors; tests 67/67 (~18.91s); build OK (~198ms) |
| Phase 8I dist canonical line check | PASS at the time — `dist/index.html` line 12 = `<link rel="canonical" href="https://pilot.goalrail.dev/" />` (the then-active D-0049 value; Phase 8K realigned to `https://pilot.goalrail.ru/` per D-0053 and rebuilt `dist/`) |
| Phase 8I source boundary scan | PASS — no forbidden patterns in `index.html` / `App.tsx` / `App.css` / `main.tsx` / `theme.ts` |
| Phase 8I local preview smoke at `http://localhost:4173/` | PASS — full RG walkthrough → outcome `ready`; primary outcome CTA focuses email; canonical at the time = `https://pilot.goalrail.dev/` (then-active D-0049 value; later realigned to `https://pilot.goalrail.ru/` in Phase 8K per D-0053); 0 console errors, 0 console warnings, 0 non-static network requests |

## 8. Remaining deployment tasks

| Task | Status | Notes |
|------|--------|-------|
| Choose concrete static hosting path | DONE per `docs/ops/DECISIONS.md` D-0051 (which supersedes D-0050 for hosting provider and deployment mode) | operator-managed SSH static server with manual rsync/scp upload, atomic release directory + `current` symlink, server-managed HTTPS, externally-managed DNS, no automatic redeploys, no CI deploy workflow. |
| Identify SSH server and release layout (out of repo) | PENDING | The wiring slice identifies the operator-managed SSH server and confirms the static web root and release directory layout. Server hostnames, IP addresses, SSH ports, usernames, keys, and credentials must remain out of the repository. The actual confirmed release-root path may be summarised in this doc but server identifiers must not be. |
| Add repo-side server config | NONE PLANNED | D-0051 explicitly does not authorise committing reverse-proxy config (nginx / Caddy / Apache / etc.) or SSH-related scripts to this repo. Any such config lives on the operator-managed server unless a separate explicit decision authorises a repo-side artefact. |
| DNS for `pilot.goalrail.ru` | PENDING | Externally-managed by the operator per D-0051: A / AAAA / CNAME as appropriate to the SSH server or upstream reverse proxy. The active target domain is `pilot.goalrail.ru` per D-0053 (which supersedes D-0049's `pilot.goalrail.dev`, now reserved for a later global-market rollout). If the DNS zone is in Cloudflare, the record must be DNS-only / non-proxied so public traffic does not depend on Cloudflare Pages, Cloudflare proxy, Cloudflare Workers, or Cloudflare CDN services. |
| TLS for `https://pilot.goalrail.ru/` | PENDING | Server-managed HTTPS via existing reverse proxy or Let's Encrypt per D-0051; HTTPS must be verified active on the active `.ru` target before any public use. |
| Publish-time canonical link fix in `index.html` | RESOLVED in Phase 8E (against D-0049's `.dev`), realigned in Phase 8K (to D-0053's `.ru`) | Phase 8E updated `apps/web/pilot-intake-ru/index.html` line 12 from `<link rel="canonical" href="https://goalrail.ru/" />` to `<link rel="canonical" href="https://pilot.goalrail.dev/" />` to match the then-active D-0049. Phase 8K then updated the same line from `https://pilot.goalrail.dev/` to `https://pilot.goalrail.ru/` to match D-0053 (which supersedes D-0049 for target domain and canonical public URL). HTML-metadata only — no copy rewrite, no behavior change in either pass. The built `dist/index.html` contains the active `.ru` canonical after the Phase 8K rebuild. Source-grep boundary scan re-run: PASS. Production preview smoke (Phase 8K): PASS — canonical link is `https://pilot.goalrail.ru/`, hero visible, 5-step rail visible, 0 console errors/warnings, 0 non-static network requests. Inspection of `index.html` again found no other absolute public URL metadata (no `og:url`, no `twitter:url`, no other canonical-like links); only this one entry needed alignment. |
| Production preview / smoke check at chosen provider | PENDING | Local preview pass recorded in §7. Provider-context preview is part of the wiring slice once provider is picked. |
| Real-device pass (iOS Safari + Android Chrome) | PENDING | Recorded in `PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md` and `PILOT_INTAKE_RU_DEPLOYMENT_PREP.md`. |
| Native-speaker proofread of canonical Russian copy | PENDING | Recorded in same prior notes. |
| Behavioural screen-reader audit (VoiceOver / NVDA / TalkBack) | PENDING | Recorded in same prior notes. |
| Verify `mailto:` handoff in deployed context | PENDING | Local preview confirmed; needs re-confirmation at the chosen provider. |
| Re-confirm D-0047 boundaries in deployed context | PENDING | Source + local preview confirmed; needs re-confirmation post-deploy. |

## 9. Recommendation

**READY FOR SSH DEPLOY — RUNTIME VALUES REQUIRED.**

Phase 8H executed the local half of the SSH static deployment wiring
slice (build, boundary scan, local preview smoke). Phase 8I
re-attempted the same slice and re-ran the local half end-to-end with
identical PASS results. In both phases the remote half (SSH
connection, `rsync` upload, atomic `current` symlink switch, DNS / TLS
verification, server-side smoke) was **not** performed because the
operator did not provide the required runtime environment variables
(`GR_PILOT_REMOTE_DEPLOY=yes`, `GR_PILOT_SSH_TARGET`,
`GR_PILOT_RELEASE_ROOT`, `GR_PILOT_CURRENT_LINK`, `GR_PILOT_DOMAIN`).
Per Phase 8H Scope C and Phase 8I Scope A, this is the correct
outcome: no remote state was changed, no server identifiers were
committed.

The surface itself remains **READY WITH WARNINGS** for static
deployment:
- build/test/typecheck pass (Phase 8H re-ran them; Phase 8I re-ran
  them again with the same PASS outcomes);
- output is root-path safe with no env or secret requirements;
- boundary verification PASS at source and preview runtime in both
  Phase 8H and Phase 8I;
- local preview smoke check confirms the full 5-step walkthrough works
  and the primary outcome CTA correctly focuses the email input;
- no provider-specific wiring is required by the app code itself
  (Vite default `base="/"` matches D-0049);
- the canonical-link metadata mismatch identified in Phase 8D §8 was
  resolved in Phase 8E (HTML-metadata only, no copy/behavior change).

To unblock the next attempt, the operator provides the five runtime
env vars in their shell and re-runs the SSH static deployment wiring
slice (see §10 / `docs/ops/NEXT.md`). Optional `GR_PILOT_SSH_OPTS`,
`GR_PILOT_RSYNC_OPTS`, `GR_PILOT_RELEASE_ID`, `GR_PILOT_KEEP_RELEASES`,
and `GR_PILOT_PREVIOUS_RELEASE` may also be exported.

D-0047 + D-0048 + D-0049 + D-0051 boundaries are intact. D-0050 is
preserved in the file as historical record but is `superseded by
D-0051 for hosting provider and deployment mode`. The surface remains
not deployed and not live until the wiring slice completes with HTTPS
verified active on `https://pilot.goalrail.ru/` (active per D-0053).
