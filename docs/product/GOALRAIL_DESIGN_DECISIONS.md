# GOALRAIL — Design Decisions

## Current decision

Use a **two-scene public entry flow**.

- **Scene 1:** incoming task -> draft working contract
- **Scene 2:** contract breakdown -> pilot request

## Why

This is the closest visual expression of the current product thesis:

> Goalrail is a shared source of truth between business intent and AI-assisted delivery.

The public experience should not present Goalrail as:

- a prompt tool
- an AI IDE
- a dashboard / OS
- a replacement for Jira / Linear

Instead it should present Goalrail as a layer that helps teams turn a vague incoming task into a shared working contract before execution starts.

## Visual baseline

- Dark, restrained, premium, editorial tone
- No top navigation / company-site rhythm
- No provider buttons on the first scenes
- No prompt export framing
- Contract is the central object

## Tooling decision

- **Stitch:** primary tool for visual exploration of the 2-scene narrative
- **Figma:** follow-up tool for fixing the chosen direction, copy lock, grid, spacing, and handoff

## CTA decision

The sales motion should be pilot-first, not self-serve:

- Primary CTA in scene 1: `Открыть разбор`
- Primary CTA in scene 2: `Получить пилотный разбор`

## Relationship to the larger product

The public flow shows only the first honest slice of the broader product loop:

`Goal -> Clarify -> Contract -> Tasks -> Change -> Verify -> Proof -> Feedback`

## Console shell v0

The Goalrail console uses a restrained terminal-adjacent visual direction.

Main product surfaces remain exactly:

- Contracts
- Delivery Readiness
- Proof

Login, Settings, Appearance, and Users are support UI. They do not become
product surfaces.

Console v0 includes:

- login-only entry screen
- no registration path
- bottom-left Settings utility
- Appearance theme presets inspired by common terminal palettes
- Users add/edit UI under Settings

Current boundary:

- UI shell only
- no production auth
- no user CRUD backend
- no sessions/cookies
- no analytics
- no repo/runtime execution claims
- no gate/proof implementation claims

The UI must avoid fake scores, fake scans, fake live proof, and fake backend
status.
