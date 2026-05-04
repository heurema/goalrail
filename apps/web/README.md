# Goalrail Web Rules

This directory is a shared boundary for Goalrail web stack conventions.

## Layout rule

- runnable frontend apps live in `apps/web/<resource>`
- no runnable app should live directly in `apps/web`
- this directory should stay lightweight and contain shared rules only

## Standard stack

- React
- Vite
- Mantine
- PostCSS with `postcss-preset-mantine`
- Vitest + React Testing Library

## Official AI tooling

- React docs: `https://react.dev/llms.txt`
- Vite docs: `https://vite.dev/llms.txt`
- Mantine docs: `https://mantine.dev/llms.txt`
- Mantine MCP: `@mantine/mcp-server`
- Mantine skills live in `.codex/skills/mantine-*`

Project-local config:

- Codex: `.codex/config.toml`
- Generic MCP clients: `.mcp.json`

Rule:

- React uses official docs only
- Mantine uses official MCP plus official skills

## Current resources

- `apps/web/console` is the canonical multilingual EN/RU console source with
  bounded auth API client flow and structured empty product surfaces
- `apps/web/demo-change-packet` is the current EN change-packet demo prototype
- `apps/web/demo-change-packet-ru` is a separate RU copy intended for an independent demo domain, not an in-app translation layer
- `apps/web/pilot-intake-ru` is the Russian pilot-intake landing prototype
- `apps/web/reference-designs` stores imported design handoff bundles for later implementation; it is not a runnable workspace

## Console source and targets

- Canonical product console source: `apps/web/console`
- Product frontend target direction for later deployment slices: `goalrail.dev`
- API target direction for later deployment slices: `api.goalrail.dev`
- Demo sandbox remains separate at `demo.goalrail.dev`
- Legacy/live `console.goalrail.ru` may continue serving an older static release until a separate deployment migration slice

The shared console app does not hard-code deployment-specific canonical
metadata. Deployment-specific metadata, routing, CORS, DNS, and server-host
configuration are later deployment concerns, not part of the source
consolidation slice.
