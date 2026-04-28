---
id: pilot_intake_ru_internal_review_notes
title: Pilot Intake RU — Internal Review Notes
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - pilot_intake_ru_review
  - public_demo_readiness
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
# Pilot Intake RU — Internal Review Notes

## 1. Purpose

This note records the internal review of `apps/web/pilot-intake-ru` before
deciding whether to publish it as the public RU landing/demo. It captures
visual, product-flow, responsive, keyboard/a11y, copy/boundary, and local
deterministic checks performed during this review pass, plus follow-ups and
the decision gate.

Review date: 2026-04-28.
Reviewer: ops (browser-driven inspection via local dev server at
`http://localhost:5174/`).

## 2. Review scope

In scope:
- visual review at desktop / mid / narrow / very-narrow widths;
- product-flow review across the 5-step walkthrough;
- responsive review of demo rail, panels, action rows, and right-column
  cards;
- keyboard / a11y review (skip-link, main landmark, step rail
  `aria-current`, textarea description);
- copy / boundary verification against `D-0047`;
- local deterministic boundary verification through DOM inspection.

Out of scope:
- app source changes;
- new scenarios beyond `manual_review_gate` and `bounded_task`;
- analytics or session tracking;
- backend / email submission;
- deployment wiring;
- repo / LLM / runtime integration;
- copy rewrite beyond findings recorded here.

## 3. Current implementation summary

- 5-step walkthrough: Goal Intake → Clarification → Contract Draft → Review
  → Honest Outcome.
- Two scenarios with deterministic detection:
  `manual_review_gate` (signals: review/reviewer/approval/approve/proof/
  gate/manual/провер/ревью/аппрув/соглас/утвержд) and `bounded_task`
  (fallback).
- Three outcome tones derived from structural review fields only
  (`deriveOutcomeTone(review)`): `ready` (olive), `readyWithCaveats`
  (amber), `blocked` (warm rust).
- Pure local helpers: `detectScenario`, `buildContractDraft`,
  `buildReviewReport`, `deriveOutcomeTone`, `buildOutcomeReport`.
- Phase 6A polish: responsive rail, RU risk titles, review digest counters,
  outcome CTA email-input focus + temporary highlight.
- Phase 6B polish: skip-link, scroll-snap rail with right-edge mask,
  <420px breakpoint, subtle CSS-only step fade-in with
  `prefers-reduced-motion` respect.
- Phase 6C polish: `<main id="main-content" tabIndex={-1} aria-label="Демо
  GoalRail">`, skip-link `onClick` focus handler, `data-skip-animation`
  back-navigation suppression.
- 67/67 unit tests pass; typecheck and Vite production build pass at
  review time.
- D-0047 firm boundary: local-only, deterministic, no
  backend/LLM/repo/execution/persistence/analytics, no chat UI, no file
  upload, no model selector.

## 4. Review matrix

Status legend: `PASS` — observed and OK; `WARN` — minor finding worth a
follow-up; `FAIL` — blocking; `PENDING` — not inspected in this pass.

### A. Desktop visual review (1440 × 900)

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| Topbar `GOALRAIL / ПИЛОТ` + 3 status chips | PASS | brand left, chips right; no overflow | — |
| Hero hierarchy (eyebrow → title → subtitle → rail → composer) | PASS | clear vertical rhythm, 720 max-width on subtitle | — |
| Composer clarity (header label, textarea, footer with safety + counter + CTA) | PASS | "GOAL INTAKE · ШАГ 1 ИЗ 5" header, "0 / 1500" counter, disabled CTA | — |
| Right column readability (3 cards) | PASS | "Что вы увидите", "Безопасно для демо", "Итог пилота" stacked cleanly at 340px | — |
| Outcome card readability (verdict header + 4 sections + actions) | PASS | verified across all 3 tone variants in §H | — |
| Skip-link not visible until focus | PASS | `top: -200px` until `:focus`/`:focus-visible` reveals it | — |

### B. Mid-width review (1024 × 820)

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| mainGrid stacks vertically (hero on top, right column below) | PASS | grid-template-areas collapse per `≤1180` rule | — |
| Right-column 2×2 layout when 4 cards (Step 4/5) | PASS | Phase 6A change — symmetric, not 3+1 asymmetry | — |
| Sidebar still readable (220px) | PASS | nav items, status card, footer all fit | — |
| Composer footer single row | PASS | safety + counter + CTA fit in horizontal row | — |
| Demo rail single row of 5 cells | PASS | 01–05 readable at 1024 | — |
| Contract / review / outcome card readability | PASS | structured rows fit within stage | — |

### C. Narrow review (600 × 800)

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| Topbar stacks (brand row, chips row) | PASS | Phase 1 `≤900` rule | — |
| Sidebar hidden | PASS | display:none ≤900 | — |
| Demo rail horizontal scroll discoverability | PASS | scroll-snap + right-edge `mask-image` fade visible | — |
| Composer footer collapses (safety + counter row, CTA full-width below) | PASS | Phase 6A `≤900` change | — |
| Action rows stretch (back / change / primary fill width) | PASS | `.contractActions` flex stretch ≤900 | — |
| Right column stacks under center column | PASS | grid-template-areas single-column ≤900 | — |

### D. Very narrow review (380 × 760)

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| Demo rail readability | PASS | min-width 76px, num 8.5px, label 9.5px (Phase 6B `≤420`); 4 cells visible, 05 reachable via scroll | — |
| Hero title wraps cleanly across 3 lines | PASS | "Покажите задачу — мы / проведём вас по / контракту" | — |
| Verdict badge wrapping | PASS | "СНАЧАЛА НУЖНЫ РЕШЕНИЯ" badge fits on one line within verdict card padding | — |
| CTA visibility (full-width primary + secondary stack) | PASS | actions row collapses cleanly | — |
| Email CTA highlight visibility | PASS | `ctaCard--highlight` ring + email input ring visible after primary outcome CTA click (verified at 1024 in §J; same CSS applies) | — |
| Hero title near right-edge wrap appears tight | WARN | line "проведём вас по" pushes near right padding at 380px; readable but visually tight | minor copy/CSS tweak in a future patch (not blocking) |

### E. Keyboard review

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| Skip-link revealed on first Tab from page top | PASS | `document.activeElement.textContent` was `К основному содержимому`, `href="#main-content"` | — |
| Skip-link Enter moves focus to `<main>` | PASS | after Enter: `location.hash="#main-content"`, `document.activeElement.id="main-content"`, `aria-label="Демо GoalRail"` | — |
| Textarea reachable | PASS | tab order from main → top chips → sidebar nav → composer textarea | — |
| Example chips reachable | PASS | three buttons after textarea CTA in tab order | — |
| Clarification radios usable | PASS | `role="radiogroup"` per question, options as `<button role="radio" aria-checked>` (verified earlier in Phase 2 tests; structure unchanged) | — |
| Final outcome CTA focuses email | PASS | `document.activeElement === pilot-email` after click on outcome primary CTA (Phase 6A behavior, retested in §J) | — |

### F. Screen-reader / semantics review

| Check | Status | Notes | Follow-up |
|------|--------|-------|-----------|
| `<main>` has `aria-label="Демо GoalRail"` | PASS | confirmed via DOM | — |
| Sidebar `<aside aria-label="Разделы лендинга">` | PASS | preserved | — |
| Right column `<aside aria-label="О демо">` | PASS | preserved across all step transitions | — |
| Step rail active uses `aria-current="step"` | PASS | one item carries it; `data-step-state` provides non-color cue | — |
| Textarea `aria-describedby="demo-intake-safety"` references safety span | PASS | both attribute and target span present | — |
| Feedback regions not noisy | PASS | only deliberate `role="status" aria-live="polite"` on local feedback (e.g. `clarificationSaved` block, outcome CTA highlight) | — |

### G. Product-flow review

| Step | Check | Status | Notes |
|------|-------|--------|-------|
| 1. Intake | textarea + safety + chips + CTA disabled <20 chars | PASS | counter increments, CTA enables when chip clicked |
| 2. Clarification | structured `radiogroup`s, not chat | PASS | 3 deterministic questions per scenario, progress `Ответы: N / 3` |
| 3. Contract Draft | object-like structured `<dl>` with named rows | PASS | Phase 3 sections present (Название / Цель / Scope / правило review / failure mode / out of scope / criteria / ambiguity / next step) |
| 4. Review | risks/gaps without fake score | PASS | counters are real `Array.filter().length`, not synthesized %; Demo execution is simulated advisory always present |
| 5. Honest Outcome | does not imply task completion | PASS | `notDone` block has 4 explicit items; verdict copy honest across tones |

### H. Outcome path review

All four flows verified through deterministic browser-driven walkthrough.

| Flow | Input + answers | Expected | Observed | Status |
|------|------------------|----------|----------|--------|
| 1 — ready | "Добавить manual review перед proof approval" / только новые контракты / один назначенный reviewer / proof блокируется | tone=ready, label=ГОТОВ К ПИЛОТУ | tone=`ready`, label=`ГОТОВ К ПИЛОТУ` (DOM `[data-outcome-tone]`) | PASS |
| 2 — readyWithCaveats | same chip / только новые контракты / один назначенный reviewer / нужно решить вручную | tone=readyWithCaveats, label=ПОДХОДИТ С ОГОВОРКАМИ, CTA=Обсудить оговорки | tone=`readyWithCaveats`, label=`ПОДХОДИТ С ОГОВОРКАМИ`, CTA=`Обсудить оговорки` | PASS |
| 3 — blocked | same chip / пока не уверен / пока не определено / нужно решить вручную | tone=blocked, label=СНАЧАЛА НУЖНЫ РЕШЕНИЯ, CTA=Разобрать кейс | tone=`blocked`, label=`СНАЧАЛА НУЖНЫ РЕШЕНИЯ`, CTA=`Разобрать кейс` | PASS |
| 4 — bounded_task fallback | "Разобрать PR с неясными критериями приёмки" | bounded_task questions only, no review-gate-specific fields | clarification prompts: "Какая граница задачи?" / "Что должно быть видно в контракте?" / "Какой честный итог нужен?" — no `mrg-*` ids in DOM | PASS |

### I. Boundary review

DOM scan on Step 1 confirmed all critical boundary copy is visible:

| Check | Status | Observed |
|------|--------|----------|
| Safety microcopy: "Не вставляйте секреты, токены и приватные данные." | PASS | present, referenced by `aria-describedby` |
| "код не выполняется" visible | PASS | in Безопасно для демо card |
| "repo не подключается" visible | PASS | in Безопасно для демо card |
| "данные не сохраняются как production-сущности" visible | PASS | in Безопасно для демо card |
| "результат не является выполненной задачей" visible (Step 5) | PASS | confirmed in `notDone` 4-item list |
| No model selector | PASS | no `<select name~"model">` and no `[role="combobox"][aria-label~="model"]` in DOM |
| No file upload | PASS | no `<input type="file">` anywhere |
| No chat history / assistant turns | PASS | no `chat history` / `история сообщений` / `assistant` text in body |
| No fake numeric readiness score | PASS | no `Готовность N / 100` or `readiness N / 100` regex match |
| No backend submit on outcome CTA | PASS | code path is `target.focus()` + `scrollIntoView()` only; no `fetch`/`XMLHttpRequest` (verified by source inspection — Phase 5 implementation) |

### J. Lead CTA review

| Check | Status | Notes |
|------|--------|-------|
| Primary outcome CTA focuses email input | PASS | `document.activeElement === #pilot-email` after click on outcome primary CTA |
| Email CTA card highlight visible | PASS | `ctaCard.dataset.ctaHighlighted="true"`, `className` contains `ctaCard--highlight`; ring + glow on container and inner input |
| No backend submit implied | PASS | email CTA stays a `mailto:hello@goalrail.dev` form; no fetch initiated by primary CTA click |
| `mailto` / manual handoff remains honest | PASS | `<form action="mailto:hello@goalrail.dev">` + `<a href="mailto:hello@goalrail.dev">hello@goalrail.dev</a>` |

## 5. Findings

### Passes

- All A/B/C visual reviews at 1440 / 1024 / 600 widths.
- All keyboard checks (E): skip-link, main focus, full tab path.
- All semantic checks (F): `aria-label` on `<main>`, complementary
  landmarks, `aria-current`, `aria-describedby`.
- All product-flow checks (G): structured intake, clarification, draft,
  review, outcome.
- All four outcome flows (H): ready / readyWithCaveats / blocked / BT
  fallback resolve to the expected tones, labels, and CTAs.
- All boundary checks (I): D-0047 firm boundaries hold in the rendered
  DOM.
- Lead CTA (J): focus + highlight behavior preserved, no backend submit.

### Warnings

- D / hero title at 380px sits visually tight to right padding when
  Russian first line wraps to "проведём вас по". Content is readable; this
  is a polish-grade nit, not a blocker. Future small CSS patch could
  tighten `.heroCard` padding or shrink the lower `clamp()` bound on
  `.heroTitle` font-size at very-narrow widths.

### Failures

- None.

### Pending human review

- Comparative review with reference designs in
  `docs/reference/design/reference_screens/` if any apply (visual brand
  alignment was not scored in this pass).
- Real-device testing (iOS Safari, Android Chrome) — only
  emulation-via-Playwright was performed.
- `prefers-reduced-motion` empirical confirmation in a real OS; the CSS
  rule exists but was not validated against an OS-level setting in this
  pass.
- Russian copy proofreading by a native speaker (the canonical strings
  were not edited in this pass; only verified for presence and absence
  of forbidden affordances).

## 6. Recommendation

**READY WITH WARNINGS — READY FOR PUBLIC-DOMAIN DECISION.**

The 5-step walkthrough is functionally complete, deterministically correct
across all four major flows, accessible to keyboard and screen-reader
users, responsive across the four width tiers, and conforms to D-0047
firm boundaries. The only finding is a single very-narrow visual nit in
section D that does not block public publication.

The review surface itself is therefore ready for the
public-domain decision (see §8). The visual nit, the pending native-speaker
proofread, and real-device testing are appropriate for separate small
patches and do not require gating publication.

## 7. Proposed follow-up patches

Each of these is a separate small patch; none should be folded into one
change.

1. **Very-narrow hero polish patch.** Adjust `.heroCard` left/right
   padding or lower `clamp()` bound on `.heroTitle` `font-size` to remove
   the slight right-edge tightness at ~380px. Pure CSS, no behavior
   change.
2. **Russian copy proofread patch.** A native speaker reviews
   `GOALRAIL_LANDING_COPY_PILOT_FIRST.md` canonical strings vs `App.tsx`
   constants. Any wording adjustments land in both places in lock-step.
3. **Real-device manual-test note patch.** A short note recording iOS
   Safari and Android Chrome observations, appended to this review doc or
   filed as a sibling note.
4. **Public-domain deployment-prep decision** (D-NNNN). If §8 picks
   "publish", the next decision records the chosen domain (e.g.
   `pilot.goalrail.ru`), the deployment surface, and any wiring required
   without expanding scope beyond static hosting.
5. **Email lead-capture decision** (D-NNNN). Separate explicit decision
   on whether email lead capture stays `mailto:` / focus-only or moves
   to a backend submission. Required by D-0047 if any move beyond
   `mailto:` is contemplated.
6. **Truth-surfaces vocabulary consolidation** (low priority docs
   patch). Audit and formalize the `truth_surfaces:` vocabulary across
   canonical docs; new labels added in Phase 7B
   (`landing_copy_canon`, `public_demo_governance`) and this note
   (`pilot_intake_ru_review`, `public_demo_readiness`) could be lifted
   into a shared enum.

## 8. Decision gate

The next decision should answer:

1. **Publish as public RU landing demo?** If yes, pick a target domain
   and record the choice as a new decision in `docs/ops/DECISIONS.md`.
   Acceptable answer is also "keep internal" — both preserve D-0047.
2. **Email lead capture posture for v1 public.** Default per D-0047 is
   `mailto:` / focus-only / manual handoff. Any move to a backend
   submission requires its own decision before implementation.
3. **Run another review pass before publishing?** Optional. If the very
   narrow polish or Russian copy proofread is felt to be a hard
   pre-publish requirement, run those small patches first and re-run a
   short visual pass.

D-0047 boundaries remain intact regardless of which option is selected.
This note does not by itself authorize publication; it provides the
evidence base for the publication decision.
