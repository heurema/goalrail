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

## Current resource

- `apps/web/demo-change-packet` is the current change-packet demo prototype
