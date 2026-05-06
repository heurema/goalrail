# Goalrail Console Web

Real Goalrail console shell.

Canonical source: `apps/web/console`.

Current scope:
- multilingual EN/RU source using `react-i18next` + `i18next`
- login-only entry screen with no registration path
- auth client for the existing `apps/server` auth API:
  `POST /v1/auth/login`, optional `POST /v1/auth/change-password`,
  `GET /v1/me`, and `POST /v1/auth/logout`
- first-login password-change flow for bootstrapped users with
  `must_change_password = true`
- authenticated shell entry only after `/v1/me` succeeds
- access and refresh tokens are held in React memory only
- left navigation with three structured empty product surfaces: Contracts, Delivery Readiness, Proof
- bottom-left Settings utility with Appearance theme presets and Users add/edit UI
- selected theme persists only as a local browser visual preference under `goalrail.console.theme`
- locale is not persisted in browser storage; runtime switching updates i18next,
  `document.documentElement.lang`, and the URL `lng` query param
- `VITE_GOALRAIL_API_BASE_URL` configures the API base URL at build time; empty
  means same-origin `/v1/...`
- no cookies, token `localStorage` / `sessionStorage`, public registration,
  signup, SSO, invite/reset email, password reset, admin user API, analytics,
  repo integration, runner, gate, proof, or product data loop
- live `console.goalrail.ru` may still point to an older static release until a
  separate deployment migration / API routing slice

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- Settings / Users is intended to become the API-backed Organization
  user-management surface for the future
  `/v1/organizations/{organization_id}/users` routes documented in ADR-0027
- users UI currently stores changes in component state; no API persistence yet
- future Users persistence must use backend-aligned roles only:
  `owner`, `admin`, `member`, and `viewer`; `observer` is not a documented
  target role
- generated temporary passwords must be treated as one-time secrets and must
  not be stored in browser storage
- product surfaces, auth state, locale, users, and settings screen are not persisted
