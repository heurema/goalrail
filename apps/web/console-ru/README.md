# Goalrail Console RU Web

Russian Goalrail console shell.

Target subdomain: `console.goalrail.ru`.

Current scope:
- login-only entry screen with no registration path
- left navigation with three empty product surfaces: Контракты, Оценка готовности, Проверка результата
- bottom-left Settings utility with Users add/edit UI only
- no data, API calls, backend integration, production auth, sessions/cookies, routing, persistence, analytics, or CLI/server status
- visual shell only, based on the existing change-packet demo design language

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- users UI currently stores changes in component state; no API persistence yet
