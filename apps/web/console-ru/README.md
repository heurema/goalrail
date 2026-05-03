# Goalrail Console RU Web

Russian Goalrail console shell.

Target subdomain: `console.goalrail.ru`.
Deployment status: live static HTTPS surface; wiring and smoke evidence live in
`docs/ops/CONSOLE_RU_DEPLOYMENT_WIRING.md`.

Current scope:
- login-only entry screen with no registration path
- left navigation with three structured empty product surfaces: Контракты, Оценка готовности, Проверка результата
- bottom-left Settings utility with Appearance theme presets and Users add/edit UI
- selected theme persists only as a local browser visual preference under `goalrail.console.theme`
- no data, API calls, backend integration, production auth, sessions/cookies, routing, user-settings persistence, analytics, or CLI/server status
- visual shell only, based on the existing change-packet demo design language

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- users UI currently stores changes in component state; no API persistence yet
- product surfaces, login state, users, and settings screen are not persisted
