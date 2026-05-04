# Goalrail Console RU Web

Russian Goalrail console shell.

Target subdomain: `console.goalrail.ru`.
Deployment status: live static HTTPS surface; wiring and smoke evidence live in
`docs/ops/CONSOLE_RU_DEPLOYMENT_WIRING.md`.

Current scope:
- login-only entry screen with no registration path
- repo-side auth client for the existing `apps/server` auth API:
  `POST /v1/auth/login`, optional `POST /v1/auth/change-password`,
  `GET /v1/me`, and `POST /v1/auth/logout`
- first-login password-change flow for bootstrapped users with
  `must_change_password = true`
- authenticated shell entry only after `/v1/me` succeeds
- access and refresh tokens are held in React memory only
- left navigation with three structured empty product surfaces: Контракты, Оценка готовности, Проверка результата
- bottom-left Settings utility with Appearance theme presets and Users add/edit UI
- selected theme persists only as a local browser visual preference under `goalrail.console.theme`
- no cookies, token `localStorage` / `sessionStorage`, public registration,
  signup, SSO, invite/reset email, password reset, admin user API, analytics,
  repo integration, runner, gate, proof, or product data loop
- live `console.goalrail.ru` is still a static deployment and needs a separate
  API routing / proxy / CORS deployment slice before deployed auth can work
  against a backend

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- users UI currently stores changes in component state; no API persistence yet
- product surfaces, auth state, users, and settings screen are not persisted
