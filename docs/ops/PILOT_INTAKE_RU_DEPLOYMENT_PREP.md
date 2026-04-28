---
id: pilot_intake_ru_deployment_prep
title: Pilot Intake RU — Deployment Prep
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - pilot_intake_ru_deployment_readiness
  - public_demo_candidate
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md
  - docs/ops/DECISIONS.md
  - docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
---
# Pilot Intake RU — Deployment Prep

## 1. Purpose

This note records deployment-prep readiness for the candidate public RU
pilot-first landing demo (`apps/web/pilot-intake-ru`). It does **not**
record actual deployment, hosting wiring, domain provisioning, or any
backend/email-submit work. It is a pre-publish checklist whose result
feeds the next operational slice.

Prep date: 2026-04-28.
Owner: ops.

## 2. Decision basis

- `docs/ops/DECISIONS.md` D-0047 — public landing demo remains
  local-only and deterministic (firm boundary).
- `docs/ops/DECISIONS.md` D-0048 — public RU pilot-first landing demo
  candidate approved (pending deployment-prep).
- `docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md` — internal review
  evidence base, recommendation `READY WITH WARNINGS — READY FOR
  PUBLIC-DOMAIN DECISION`.
- `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md` — canonical copy
  and governance reference.

## 3. Current candidate surface

- App path: `apps/web/pilot-intake-ru`.
- Package: `@goalrail/pilot-intake-ru-web` (workspace under
  `apps/web/package.json`).
- Stack: React 19.2.5 + Vite 8.0.10 + Mantine 9.1.0 + IBM Plex Sans/Mono
  fonts (per `goalrail-web-stack` skill).
- Current behavior: complete deterministic local 5-step walkthrough
  (Goal Intake → Clarification → Contract Draft → Review → Honest
  Outcome), driven by pure helpers
  (`detectScenario`, `buildContractDraft`, `buildReviewReport`,
  `deriveOutcomeTone`, `buildOutcomeReport`).
- Current public status: candidate approved per D-0048. **Not deployed.
  Not live.** No backend exists.
- Lead capture posture: `mailto:hello@goalrail.dev` form + focus +
  manual handoff. The outcome primary CTA only focuses the email input
  and applies a temporary local CSS highlight.

## 4. Build readiness

| Check | Command / method | Result | Notes |
|------|------------------|--------|-------|
| Typecheck | `npm run pilot-intake-ru:typecheck` | PASS | `tsc --noEmit`, 0 errors |
| Tests | `npm run pilot-intake-ru:test` | PASS | 67/67 (1 file, ~21s) |
| Production build | `npm run pilot-intake-ru:build` | PASS | `tsc -b && vite build`, ~283ms; emits `apps/web/pilot-intake-ru/dist/` |
| Build output size | `du -sh apps/web/pilot-intake-ru/dist/` | PASS | ~908 KB total (fonts dominate); `index-*.css` 245.27 KB / gzip 36.93 KB; `index-*.js` 267.36 KB / gzip 79.05 KB |
| Preview command | `npm run pilot-intake-ru:preview` (workspace exposes `vite preview`) | PASS (available) | `apps/web/pilot-intake-ru/package.json` has `"preview": "vite preview"`; not invoked here, no smoke check yet beyond dev-server visual review |
| Smoke check on production preview | `vite preview` | PENDING | Recommended as a small step in the next deployment-wiring slice; not required to issue the public-domain decision |

## 5. Static hosting readiness

| Aspect | Status | Notes |
|--------|--------|-------|
| Static build output | PASS | `dist/` contains `index.html`, `assets/*.{js,css,woff,woff2}`, `favicon.svg`. No SSR, no server runtime. |
| Required env vars at build | NONE | No `VITE_*` env reads in source; `vite.config.ts` has no env-driven fields. |
| Required env vars at runtime | NONE | App is pure static: it does not read environment variables at runtime. |
| Required secrets | NONE | No tokens, keys, or credentials embedded or referenced in source or build. |
| Backend dependency | NONE | App makes zero network calls. Email lead is a `mailto:` form (no POST). |
| Public path / base path assumption | DEFAULT | Vite default base `/`; if hosted at a non-root path, `vite.config.ts` `base` would need to be set, but no such requirement exists yet for a root-domain target. |
| Target domain | DECIDED — active target is `pilot.goalrail.ru` with public path `/` per `docs/ops/DECISIONS.md` D-0053 (which supersedes D-0049 for target domain and canonical public URL; D-0049 originally selected `pilot.goalrail.dev`, which is now reserved for a later global-market rollout) | Public status: `candidate-public`. The hosting path is now operator-managed SSH static server per D-0051 (D-0050 superseded for hosting provider and deployment mode). Because public path is `/`, no `vite.config.ts` `base` adjustment is required; root-path behavior is verified during wiring. Any further change of domain or path must be recorded as a separate explicit decision per D-0053. |

## 6. Boundary audit

Forbidden patterns scanned across `apps/web/pilot-intake-ru/src/` source:

| Boundary (per D-0047) | Status | Evidence / notes |
|-----------------------|--------|------------------|
| No backend (no `fetch(`, no `XMLHttpRequest`, no `sendBeacon`) | PASS | `grep -nE "fetch\\(\|XMLHttpRequest\|sendBeacon"` against `App.tsx` / `App.css` / `main.tsx` returns no matches. |
| No LLM/API endpoints | PASS | Source scan finds no `openai.com`, `anthropic`, `claude.ai`, `api.openai`, `api.anthropic`, etc. |
| No repo provider integration | PASS | Source scan finds no `github.com/api`, `api.github`, `api.gitlab`, `bitbucket.org/api`. |
| No code execution / runtime | PASS | App is purely declarative React; no `eval`, no `Function()`, no shell-out, no Web Workers calling external code. |
| No persistence | PASS | No `localStorage`, no `sessionStorage`, no `indexedDB`, no cookies set by the app. State lives in React component memory only. |
| No analytics / session tracking | PASS | No `gtag`, no `googletagmanager`, no `mixpanel`, no `sentry`, no `datadog`, no `analytics` references. No script tags injected. |
| No chat UI | PASS | `App.test.tsx` actively asserts the absence of `chat history`, `assistant`, model selector, file upload across the walkthrough; only test files reference these strings (as negative assertions). |
| No file upload | PASS | No `<input type="file">` element; tests assert this. |
| No model selector | PASS | No `<select>` for model; no `[role="combobox"][aria-label*="model"]`. |
| Email CTA stays mailto / focus / manual | PASS | `<form action="mailto:hello@goalrail.dev">` + `<a href="mailto:hello@goalrail.dev">`; outcome primary CTA implementation calls only `target.focus()` + `target.scrollIntoView()` + sets a local CSS class via `setTimeout` cleanup. No fetch initiated. |

Built `dist/assets/*.js` was size-checked but not byte-grepped beyond
`du`. The source-level scan above is the authoritative evidence; if a
future automated verification is desired, a small `grep` rule on the
emitted bundle could be added in a separate decision/patch.

## 7. Copy parity

Canonical strings in
`docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md` were diff-checked
against `apps/web/pilot-intake-ru/src/App.tsx`.

| Copy area | Canonical source line(s) | Implementation status | Notes |
|-----------|--------------------------|------------------------|-------|
| Hero title | `Покажите задачу — мы проведём вас по контракту` (LANDING_COPY §B/E) | MATCH | `App.tsx:1045` |
| Intake placeholder | `Опишите задачу, PR, изменение или кейс…` (LANDING_COPY §E) | MATCH | `App.tsx:1081` placeholder; `1076` srOnly label |
| Safety microcopy | `Не вставляйте секреты, токены и приватные данные.` (LANDING_COPY §E) | MATCH | `App.tsx:1091` |
| 5-step rail labels | `Запрос`, `Уточнения`, `Контракт`, `Проверка`, `Итог` (LANDING_COPY §E) | MATCH | `App.tsx` `const demoSteps = ['Запрос', 'Уточнения', 'Контракт', 'Проверка', 'Итог']` |
| Outcome verdict labels | `ГОТОВ К ПИЛОТУ` / `ПОДХОДИТ С ОГОВОРКАМИ` / `СНАЧАЛА НУЖНЫ РЕШЕНИЯ` (LANDING_COPY §H) | MATCH | `App.tsx:504`, `:512`, `:520` |
| Outcome `notDone` (4 items) | `код не выполнялся`, `repo не подключался`, `production-сущности не создавались`, `результат не является выполненной задачей` (LANDING_COPY §E) | MATCH | `App.tsx:85–88` |

Mismatches: none. No copy edits required by this prep pass.

## 8. Known pre-publish risks

| Risk | Status after this pass | Notes |
|------|------------------------|-------|
| Very narrow hero title tightness around 380px | RESOLVED | CSS-only fix applied: at `@media (max-width: 420px)`, `.heroCard` padding reduced from `26px 22px` (≤900) to `22px 18px`, and `.heroTitle` `font-size` lowered to `clamp(28px, 8vw, 34px)` with `letter-spacing: -0.04em`. Verified visually at 380px (3-line wrap with breathing room). Desktop 1440 unchanged. Test/typecheck/build still pass. |
| Real-device iOS Safari / Android Chrome testing | PENDING | Only Playwright Chromium emulation performed in §4 of `PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md`. Real-device pass should happen in the deployment-wiring slice. Non-blocking. |
| Real screen-reader audit (VoiceOver / NVDA / TalkBack) | PENDING | DOM-level a11y attributes verified (`aria-label="Демо GoalRail"`, `aria-current="step"`, skip-link, `aria-describedby`). Behavioural SR audit not performed. Non-blocking but recommended in deployment slice. |
| `prefers-reduced-motion` empirical confirmation | PENDING | CSS rule exists; not validated against an OS-level setting. Non-blocking. |
| Russian copy native-speaker proofread | PENDING | Strings verified for presence/absence and against canonical doc. Native-speaker review not performed. Non-blocking. |
| Target domain not chosen | RESOLVED | Originally recorded in `docs/ops/DECISIONS.md` D-0049 as `pilot.goalrail.dev`; superseded by D-0053 to active target `pilot.goalrail.ru` with public path `/`, public status `candidate-public`. Hosting path was originally pinned to Cloudflare Pages Direct Upload by D-0050, then changed to operator-managed SSH static server by D-0051 (D-0050 superseded for hosting provider and deployment mode). The `.dev` domain is reserved for a later global-market rollout. The deployment-wiring slice continues under D-0051 + D-0053. |
| Email lead capture posture | PRESERVED | Stays `mailto:` / focus / manual handoff per D-0047 + D-0048. Any move to backend submission requires its own decision. |
| Analytics posture | PRESERVED | Disallowed per D-0047 + D-0048. Enabling analytics requires its own decision. |

## 9. Deployment-prep recommendation

**READY WITH WARNINGS** — pre-publish prep is complete; the surface is
ready to enter a deployment-wiring slice once a target domain is chosen.

Basis:
- All build/test/typecheck checks pass.
- All boundary checks pass.
- All copy parity checks match.
- The single non-blocking visual warning from the internal review is now
  resolved by a CSS-only patch.
- Remaining items (real-device test, native-speaker proofread, screen
  reader audit, target domain) are appropriate for a separate
  deployment-wiring slice and do not block readiness.

D-0047 boundary remains intact. D-0048 candidate-approval remains
intact. No new product behavior was introduced.

## 10. Next operational slice

The next slice is `Slice — Pilot intake RU deployment wiring`. It
should:

1. Target domain is recorded in `docs/ops/DECISIONS.md` D-0053 as
   active target `pilot.goalrail.ru` with public path `/` and public
   status `candidate-public` (D-0053 supersedes D-0049 for target
   domain and canonical public URL; D-0049 originally selected
   `pilot.goalrail.dev`, which is now reserved for a later
   global-market rollout). The hosting path is operator-managed SSH
   static server per D-0051 (D-0050 superseded for hosting provider
   and deployment mode); the deployment-wiring slice executes
   manual `rsync` / `scp` upload to a timestamped release directory
   with atomic `current` symlink switch. Any later change of domain
   or public path must be recorded as a separate explicit decision
   before it is implemented.
2. Configure static hosting only (no server runtime, no backend, no
   serverless functions). Vite `base` may need adjustment if hosted
   at a non-root path.
3. Run `vite preview` and a brief smoke walkthrough through all 5
   steps + 4 outcome flows on the production build.
4. Real-device pass on iOS Safari and Android Chrome.
5. Native-speaker proofread of canonical Russian copy; reconcile any
   findings in both `App.tsx` and
   `GOALRAIL_LANDING_COPY_PILOT_FIRST.md` in lock-step.
6. Optional behavioural screen-reader audit.
7. Keep D-0047 boundaries intact: no backend, no analytics, no email
   submission, no repo/LLM/runtime integration, no persistence, no new
   scenarios, no new outcome tones.

If any of points 4–6 surface a blocker, file it as a separate small
patch and re-run prep; do not fold them into the wiring slice.
