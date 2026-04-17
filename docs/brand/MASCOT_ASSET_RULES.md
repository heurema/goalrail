# Punk Mascot Asset Rules v0

> Production-facing asset rules for Punk as mascot / on-screen carrier.
> This document turns the character from a narrative role into a repeatable content asset system.
> It is still v0: practical, bounded, and evolvable.

## 1. Purpose

This document fixes:
- where Punk can appear
- where Punk should not appear
- how strongly Punk may dominate an asset
- reusable asset roles for video, thumbnails, title cards, and carousels
- pose / framing / crop defaults
- consistency rules for future mascot generation and editing

## 2. Boundary

This document does **not** redefine:
- product truth
- public narrative
- short-form content grammar
- full visual identity
- final illustration bible
- final motion system

It depends on:
- `docs/brand/PUNK_CHARACTER_SYSTEM.md`
- `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
- `docs/brand/VISUAL_IDENTITY_V0.md`
- `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`

## 3. Core rule

Punk is a **brand carrier**, not the product itself.

Rule:
- Punk may amplify attention
- Punk may sharpen memory
- Punk may carry tone
- Punk may not replace product explanation

## 4. Asset roles

Punk may appear in assets in one of five roles.

### 4.1 Signal role
A quick visual interrupt that punctuates a point.

Use when:
- a clip needs a strong hook
- a post needs an anti-drift signal
- a title card needs attitude, not explanation

### 4.2 Guide role
Punk points toward the core idea, system, or problem.

Use when:
- explaining a delivery issue
- introducing a build step
- walking through a simplified process idea

### 4.3 Critic role
Punk visually or verbally challenges a bad state.

Use when:
- calling out drift
- contrasting patch vs proof
- highlighting weak process or vague goals

### 4.4 Builder role
Punk appears as the one constructing / shaping / shipping.

Use when:
- build in public updates
- repo/docs/system-progress clips
- product construction storytelling

### 4.5 Accent role
Punk appears lightly, almost like a signature.

Use when:
- business-facing cards need a small dose of identity
- system visuals are the main focus
- product-first assets need minimal mascot presence

## 5. Asset intensity levels

### Level 0 — No Punk
Use when the product/process/system should be fully primary.

### Level 1 — Accent presence
Small cameo, silhouette, or partial crop.

### Level 2 — Supporting presence
Visible, but not dominant.
Punk supports the headline or concept.

### Level 3 — Lead presence
Punk is the primary focal anchor, but the message still stays readable.

### Level 4 — Full mascot focus
Avoid by default.
Only use if the asset is explicitly mascot-first and still tied to a product idea.

Rule:
Default range for Goalrail assets is **Level 1–3**.
Level 4 should be rare.

## 6. Recommended intensity by asset type

### Title cards
- default: Level 0–2
- rare: Level 3

### Thumbnails
- default: Level 1–3
- especially for `TH2`, Level 2–3

### Carousels
- opening card: Level 1–2
- middle explanation cards: Level 0–1
- signal/quote card: Level 2–3

### Product / pilot explainers
- default: Level 0–1

### Build in public clips
- default: Level 1–3

## 7. Tonal mode interaction

### Serious Business
Default Punk use:
- Level 0–1, sometimes 2
- cleaner crops
- less expressive exaggeration
- less visual noise
- more guide/accent than critic

### Serious Cyberpunk Fun
Default Punk use:
- Level 1–3
- more visible edge
- stronger silhouette power
- more signal/critic/builder energy

Rule:
The same mascot may appear in both modes.
What changes is emphasis, crop, density, and energy.

## 8. Framing defaults

Recommended reusable framings:

### F1 — Head/shoulders crop
Best for:
- talking mascot shots
- thumbnails with short hooks
- focused attention

### F2 — 3/4 torso crop
Best for:
- guide/builder role
- stronger gesture use
- title cards with space for text

### F3 — silhouette / partial profile
Best for:
- accent presence
- dark-mode title cards
- subtle brand signature

### F4 — side-facing directional crop
Best for:
- movement implication
- guiding attention toward product/process area
- rail metaphor compositions

Rule:
Do not overuse full-body shots unless they add actual storytelling value.

## 9. Pose defaults

Recommended reusable pose families:

### P1 — Point / direct
Guides attention to the message or process.

### P2 — Lean / inspect
Suggests review, scrutiny, verification.

### P3 — Forward / build
Suggests motion, construction, progress.

### P4 — Sharp stillness
A static anti-bullshit stare/silhouette for signal assets.

Rule:
For v0, a small pose vocabulary is better than an overgrown library.

## 10. Expression defaults

Allowed expression families:
- focused
- sharp
- slightly ironic
- skeptical
- determined

Avoid by default:
- exaggerated cartoon joy
- slapstick expressions
- goofy overreaction faces
- hyper-aggressive rage

## 11. Text coexistence rule

Whenever Punk appears with text:
- text must remain primary or co-primary
- Punk must not cover key words
- silhouette edges should support text hierarchy, not fight it

Rule:
If the mascot makes the hook harder to read, the asset is wrong.

## 12. Product coexistence rule

Whenever Punk appears with product or process visuals:
- product/process area must stay interpretable
- Punk may point, frame, or punctuate
- Punk may not obscure the actual object of explanation

Especially important for:
- system visuals
- operating flow cards
- pilot explainer assets

## 13. Background interaction rule

Punk should sit cleanly on:
- dark graphite backgrounds
- controlled technical textures
- rail/signal motifs
- limited contrast fields

Avoid:
- overly busy collage backgrounds
- unrelated decorative environments
- bright playful scenes that break the tone

## 14. Color interaction rule

Punk should carry or harmonize with:
- electric pink accent logic
- dark neutral base
- restrained secondary signals

Rule:
Do not recolor Punk arbitrarily per asset just for variation.
Variation should come from composition and role, not random palette shifts.

## 15. Asset families to produce first

Recommended first production set:

### A. Core silhouette pack
- 3 clean silhouette variants
- for accent/signature use

### B. Talking crop pack
- 3–5 head/shoulders variants
- for video + thumbnails

### C. Builder pack
- 2–3 torso crops with forward/build energy
- for build in public assets

### D. Critic/signal pack
- 2–3 sharper poses
- for drift / proof / control claims

This is enough for v0.

## 16. Asset file logic (conceptual)

For future production organization, assets should eventually be grouped by:
- role
- framing
- pose
- tonal mode suitability
- background compatibility

Example grouping logic:
- `signal/head_shoulders/`
- `builder/torso/`
- `accent/silhouette/`

No need to over-engineer this now, but the logic should remain stable.

## 17. Thumbnail recommendation

For thumbnails, prefer:
- F1 or F2 framing
- P1, P3, or P4 pose family
- Level 2 mascot intensity
- very short text hook

This gives the strongest recall without letting Punk eat the whole frame.

## 18. Title card recommendation

For title cards, prefer:
- Level 0–2
- F3 or F4 when Punk is present
- silhouette or partial profile over full focal portrait unless the card is intentionally mascot-led

## 19. Carousel recommendation

For carousels, default flow:
- card 1: Level 1–2
- cards 2–4: Level 0–1
- card 5 or quote/signal card: Level 2–3 if needed
- final CTA card: Level 0–1

This avoids mascot overload.

## 20. Anti-overuse rule

If Punk appears in every frame, every card, every thumbnail, and every CTA,
brand recall may go up briefly, but product clarity goes down.

Rule:
Punk should feel intentional, not compulsive.

## 21. Evolution rule

Allowed to evolve:
- exact silhouette refinement
- render polish
- expression sharpness
- crop library
- pose library
- text-safe composition rules

Not allowed to drift:
- anti-drift builder role
- serious cyberpunk fun tonal identity
- business-readable use standard
- product-first coexistence rule

## 22. Immediate recommendation summary

For v0 production:
- keep Punk mostly in Level 1–3 usage
- use head/shoulders and torso crops first
- build a small reusable pose library
- prefer signal, guide, builder roles over pure mascot spectacle
- protect text and product clarity at all times

## 23. Current one-line summary

**Punk should behave like a reusable production asset that sharpens memory and attention without ever overwhelming the product, the process, or the message.**
