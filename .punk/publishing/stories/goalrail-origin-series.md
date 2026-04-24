# Goalrail Origin Series

- Status: active-manual
- Audience: RU-speaking developers building with AI coding agents; small product teams evaluating AI delivery workflows
- Canon refs:
  - `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
  - `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`
  - `docs/product/GOALRAIL_OFFER.md`
  - `docs/product/GOALRAIL_PRICING_MODEL.md`
  - `docs/product/GOALRAIL_ICP.md`
  - `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
- Source refs:
  - `https://punks.run/journal`
  - `/Users/vi/personal/heurema/punk/site/src/data/journal.ts`

## Through-line

The series explains in Russian-first public language how the work moved from AI assistance inside the IDE toward
an inspectable operating layer for agentic software delivery.

The first public episode starts from the smallest honest baseline:

> Models helped find solutions, but the human still implemented and verified
> everything inside the editor.

From there, the story can expand into trust, tests, planning, specs, contracts,
gates, proof artifacts, and finally Goalrail as a productized operating layer.

## Episode map

1. **The workflow still lived in the IDE** - AI as assistance, not execution.
2. **Can we stop reading code?** - trust becomes the real problem.
3. **TDD became the first answer** - confidence moves outside code reading.
4. **Planning became mandatory** - vague work is not safe agent input.
5. **Specs and contracts** - scope gets bounded before execution.

## Key claims

- The interesting problem is not faster code generation.
- The hard problem is trust without reading every generated line.
- Trust has to come from the process: tests, explicit scope, gates, and proof.
- Public narrative must not imply that Goalrail is already a broad automated platform.
- Goalrail is currently pilot-first: free qualification, paid managed pilot, and expansion only after proof.

## Early-team easter egg

Do not pitch directly in the body.
Use a quiet line that signals the current design-partner opportunity:

> Сейчас мы начинаем с русскоязычных небольших команд. Там проще честно
> проверить один реальный repo, быстро увидеть боль и повлиять на будущие
> defaults продукта.

This preserves the win-win logic without turning the post into an ad:

- early team win: native-language collaboration, lower entry point, and influence over defaults
- Goalrail win: product shaped under real repo pressure, not in a vacuum
