# Demo Change Packet Web RU

Separate Russian-language copy of the Goalrail change-packet demo.

This is not in-app i18n for `apps/web/demo-change-packet`; it is a separate runnable workspace intended for an independent RU demo domain.

## Source

Copied from `apps/web/demo-change-packet` and localized in place.

## Stack

- React
- Vite
- Mantine runtime shell
- Plain CSS for pixel-oriented demo styling
- Vitest + React Testing Library

## Commands

From `apps/web`:

```bash
npm run demo-change-packet-ru:dev
npm run demo-change-packet-ru:build
npm run demo-change-packet-ru:test
npm run demo-change-packet-ru:typecheck
```

## Notes

This is a prototype RU demo surface for the change packet walkthrough.
It is intentionally a demo shell, not a claim that the full Goalrail product loop is implemented.
