# Goalrail Brand Layer Index

> Отдельный слой документации внутри того же проекта.
> Нужен, чтобы brand / mascot / character docs жили рядом с продуктом,
> но не смешивались с product canon, architecture canon и ops docs.

## Purpose

Этот слой нужен для:
- mascot / character system
- brand carrier rules
- short-form content grammar
- visual identity drafts
- title-card and thumbnail production defaults
- voice / tone rules for characterized public content
- future asset-system docs

## Read in this order

1. `docs/brand/PUNK_CHARACTER_SYSTEM.md`
2. `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
3. `docs/brand/VISUAL_IDENTITY_V0.md`
4. `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`

## Ownership boundary

### Brand layer owns
- mascot role
- character traits
- on-screen behavior
- voice / dialogue rules for Punk
- brand-carrier constraints
- title-card / thumbnail template defaults
- future mascot/asset-system rules

### Brand layer does not own
- product truth
- operating model
- pilot shape
- landing copy as product copy source of truth
- architecture language
- GTM canon

## Precedence rule

Если возникает конфликт:

1. concept canon wins
2. product summary / message hierarchy wins
3. public narrative / public language wins
4. brand layer adapts

То есть:
- mascot усиливает public narrative
- mascot не переписывает product definition
- mascot не подменяет собой Goalrail как продукт

## Working rule

Если меняется:
- роль Punk
- character tone
- mascot relationship to Goalrail / Goal on Rails / Specpunk
- short-form production defaults
- title-card or thumbnail template rules

то обновляется:
- `docs/brand/PUNK_CHARACTER_SYSTEM.md`
- `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
- `docs/brand/VISUAL_IDENTITY_V0.md`
- `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`
- при необходимости `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
- при необходимости `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`

## Current principle

Внутри этого проекта:
- **Goalrail** — продукт
- **Goal on Rails** — public series / campaign line
- **Specpunk** — sibling builder/runtime project
- **Punk** — mascot / character layer
