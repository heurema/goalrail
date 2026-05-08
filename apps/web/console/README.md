# Goalrail Console Web

Real Goalrail console shell.

Canonical source: `apps/web/console`.

Current scope:
- public `/start` route for English global entry traffic; it uses local quick
  questions and static answers as fallback, and can submit short public
  questions to same-origin `POST /api/start-chat` when the separate start
  assistant Worker is routed; no browser OpenAI key, repo scan, code execution,
  analytics, cookies, sessions, uploads, CRM, or chat history
- multilingual EN/RU source using `react-i18next` + `i18next`
- login-only entry screen with no registration path
- auth client for the existing `apps/server` auth API:
  `POST /v1/auth/login`, optional `POST /v1/auth/change-password`,
  `GET /v1/me`, and `POST /v1/auth/logout`
- first-login password-change flow for bootstrapped users with
  `must_change_password = true`
- authenticated shell entry only after `/v1/me` succeeds
- access and refresh tokens are held in React memory only
- left navigation with Contracts, Delivery Readiness, and Proof product
  surfaces; Delivery Readiness now polls read-only `GET /v1/qualification-feed`
  while authenticated and renders Qualification / Clarification / Contract /
  Blocked lanes
- bottom-left Settings utility with Appearance theme presets and API-backed
  Organization Users add/edit/temporary-password reset UI plus read-only
  Repository context metadata
- Settings / Users blocks self-demotion from owner, self membership
  deactivation, and self temporary-password reset; own password changes use
  the existing password-change flow
- selected theme persists only as a local browser visual preference under `goalrail.console.theme`
- locale is not persisted in browser storage; runtime switching updates i18next,
  `document.documentElement.lang`, and the URL `lng` query param
- `VITE_GOALRAIL_API_BASE_URL` configures the API base URL at build time; empty
  means same-origin `/v1/...`
- no cookies, token `localStorage` / `sessionStorage`, public registration,
  signup, SSO, invite/reset email, self-service password reset, password reset
  email delivery, analytics,
  repo integration, runner, gate, proof, or product data loop
- live `console.goalrail.ru` may still point to an older static release until a
  separate deployment migration / API routing slice
- live `https://goalrail.dev/start` is served from this app through the main
  console deployment, while same-origin `POST /api/start-chat` is owned by the
  separate Cloudflare Worker, not by `apps/server`; external static serving
  must preserve SPA fallback for `/start`
- live `https://goalrail.dev/` is a temporary public root entry that redirects
  client-side to `/start` until a separate public landing page exists
- local Vite dev proxies `/api/start-chat` to `https://goalrail.dev` by default
  so the public assistant works without a local Worker; set
  `START_ASSISTANT_PROXY_TARGET=http://127.0.0.1:8787` when intentionally
  testing a local start-assistant Worker

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- Settings / Users uses `/v1/me` to determine the current `organization_id`
  and then consumes the ADR-0027 Organization user-management routes:
  `GET /v1/organizations/{organization_id}/users`,
  `POST /v1/organizations/{organization_id}/users`,
  `PATCH /v1/organizations/{organization_id}/users/{user_id}`, and
  `POST /v1/organizations/{organization_id}/users/{user_id}/temporary-password-resets`
- Settings / Repository uses `/v1/me` to determine the current
  `organization_id` and then consumes
  `GET /v1/organizations/{organization_id}/repository-context`
- Delivery Readiness polls `GET /v1/qualification-feed?limit=50` only while the
  authenticated surface is open; it renders stored feed snapshots and does not
  call Goal continuation, readiness recomputation, clarification creation, or
  contract creation automatically
- Delivery Readiness has explicit user-triggered actions for
  `continue_goal`, `answer_clarification`, and `draft_contract`; they call
  `POST /v1/goals/{id}/continuation`,
  `POST /v1/clarifications/{id}/answers/continuation`, and
  `POST /v1/contracts`, then refresh the feed once after the action returns
- clarification answers are submitted only for text-safe mappings
  (`goal.summary`, `goal.scope_hint`, `goal.acceptance_hint`); actor-mapped
  questions such as `goal.intent_owner` are shown as unsupported instead of
  being posted as invalid plain text
- Repository context data is server-owned metadata only: Organization summary,
  active Project metadata, and active RepoBinding metadata
- Repository context does not imply provider authorization, checkout
  permission, readiness/proof status, execution status, or runner state
- Users data is loaded from the server API; local state is only the fetched
  view, filters, form draft, and one-time create/reset response panel
- Users persistence uses backend-aligned roles only:
  `owner`, `admin`, `member`, and `viewer`; `observer` is not a documented
  target role
- generated temporary passwords are shown only from the immediate successful
  create/reset response and must not be stored in browser storage
- user management remains Console/admin API-backed; there are no CLI
  user-management commands
- product surfaces, auth state, locale, users, and settings screen are not persisted
