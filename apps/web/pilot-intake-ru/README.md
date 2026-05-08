# Goalrail Pilot Intake RU

Russian pilot-intake landing prototype for Goalrail.

## Scope

- Static React + Vite + Mantine resource under `apps/web/pilot-intake-ru`
- Owns the current Russian pilot landing at `/` for `pilot.goalrail.ru`
- Also owns the adjacent Russian public start route at `/start`, targeted at `https://goalrail.ru/start`
- Keeps `/` as a lead-capture/public landing prototype, not a product runtime
- Keeps `/start` as a public assistant entry surface; it uses only same-origin `POST /api/start-chat` when that operator-managed route is wired

## Commands

Run from `apps/web`:

```bash
npm run pilot-intake-ru:dev
npm run pilot-intake-ru:test
npm run pilot-intake-ru:typecheck
npm run pilot-intake-ru:build
```

Local dev proxies `/api/start-chat` to `https://goalrail.dev` by default so
`/start` can exercise the existing public start assistant. Use
`START_ASSISTANT_PROXY_TARGET` to point at another operator-approved target.
