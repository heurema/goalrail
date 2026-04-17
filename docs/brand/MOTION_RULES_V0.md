# Goalrail Motion Rules v0

> Первая рабочая motion-system спецификация для Goalrail / Goal on Rails / Punk.
> Не финальный animation bible.
> Не попытка заранее описать весь motion design.
> Это bounded v0, которая нужна, чтобы видео и анимированные assets не расползались по характеру, темпу и смыслу.

## 1. Purpose

Этот документ фиксирует:
- motion thesis v0
- как должен двигаться Punk
- как должны двигаться rails / signals / accent motifs
- какой темп и ритм считаются default
- чего нельзя делать, чтобы motion не ломал business-readable подачу
- как motion должен работать в двух tonal modes

## 2. Boundary

Этот документ не заменяет:
- `PUNK_CHARACTER_SYSTEM.md`
- `SHORT_FORM_CONTENT_SYSTEM.md`
- `VISUAL_IDENTITY_V0.md`
- `TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`
- `MASCOT_ASSET_RULES.md`

Он не описывает production pipeline конкретного Adobe toolchain.
Он описывает **motion behavior rules**.

## 3. Motion thesis

Current motion thesis:

**controlled movement with sharp intent**

Это значит:
- motion должен передавать направление и управляемость
- motion может быть резким, но не хаотичным
- motion может быть энергичным, но не перегруженным
- motion должен поддерживать мысль, а не устраивать шоу вместо мысли

## 4. Core motion principles

### Principle 1 — Direction over spectacle
Движение должно показывать направление.
Не спецэффект ради спецэффекта.

### Principle 2 — Control over chaos
Даже в более edgy режиме motion не должен выглядеть “потерявшим рельсы”.

### Principle 3 — Short beats win
Короткие, понятные акценты лучше длинной бессмысленной анимации.

### Principle 4 — Message first
Если motion мешает читать текст или понимать тезис, motion проиграл.

### Principle 5 — Repeatable motifs beat unique tricks
Лучше несколько повторяемых motion-приёмов, чем постоянно выдумывать новые.

## 5. Tonal modes

## 5.1 Mode A — Serious Business
Use when:
- CTO / pilot / product explainers
- trust-heavy content
- process walkthroughs
- proof-oriented messaging

Motion characteristics:
- cleaner transitions
- restrained timing
- lower amplitude
- fewer decorative pulses
- more stable compositions
- less jitter / less intensity

Effect:
- controlled
- credible
- calm but not dead

## 5.2 Mode B — Serious Cyberpunk Fun
Use when:
- build in public
- sharper hooks
- Punk-led clips
- more social-native content

Motion characteristics:
- stronger accent pulses
- sharper in/out behavior
- more visible directional energy
- slightly higher contrast in movement
- more edge, but still readable

Effect:
- alive
- memorable
- sharp
- non-corporate

## 5.3 Switching rule

Switching between the two modes should happen via:
- speed/intensity
- accent pulse density
- transition sharpness
- motif energy

Switching modes should **not** require changing:
- the whole motion vocabulary
- core symbolic anchors
- brand stack
- product readability rules

## 6. Punk motion rules

### 6.1 Default Punk motion character
Punk should move like:
- sharp
- intentional
- controlled
- slightly aggressive in emphasis
- never sloppy

### 6.2 What Punk should not feel like
Avoid making Punk feel:
- floppy
- overly cute
- hyper-bouncy cartoon character
- chaotic meme puppet
- random idle-noise machine

### 6.3 Default motion behaviors
Allowed default behaviors:
- quick head turn or lean
- short point / direct gesture
- controlled entrance
- slight pause before key line
- tight emphasis on critical words

### 6.4 Use sparingly
Use sparingly:
- exaggerated squash-and-stretch
- long idle loops
- full-body theatrical acting
- constant bobbing
- endless micro-motion

Rule:
Punk should look like he means something,
not like he is trying to entertain between ideas.

## 7. Rail motion rules

Rails are a core metaphor.
Their motion should imply:
- direction
- track
- guided path
- progress under constraint

Recommended rail motions:
- linear reveal
- guided sweep
- forward line extension
- subtle directional travel

Avoid:
- wavy wandering path motion
- overly organic motion
- uncontrolled drift-like animation

## 8. Signal / gate motion rules

Signals and gates represent:
- verify
- checkpoint
- go/no-go
- control point

Recommended motions:
- blink / pulse with purpose
- quick state change
- minimal confirmation flash
- restrained lock / latch motion

Avoid:
- disco flashing
- repeated noisy blinking
- alarm aesthetics unless truly needed

## 9. Accent pulse rules

Accent pulses are allowed,
but they must support hierarchy.

Good use:
- emphasize a hook word
- punctuate a key transition
- mark a proof/control moment

Bad use:
- pulsing everything
- constant glow breathing
- over-animated UI chrome

Rule:
Accent pulse is a signal, not a background wallpaper behavior.

## 10. Text motion rules

### 10.1 Entry behavior
Default text behavior:
- quick reveal
- simple slide or fade with direction
- no decorative gymnastics

### 10.2 Emphasis behavior
Allowed:
- one-word emphasis pulse
- short scale/opacity accent
- directional underline/rail cue

Avoid:
- text flipping
- chaotic letter-by-letter effects
- unreadable glitch text
- aggressive wobble

### 10.3 Readability rule
If the viewer has to wait for a long animation to understand the text,
the text animation is too slow or too decorative.

## 11. Subtitle motion rules

Subtitles should feel:
- stable
- fast to scan
- minimally animated
- supportive, not performative

Recommended subtitle motion:
- simple appearance / refresh
- clean emphasis on selected words
- restrained transition between subtitle blocks

Avoid:
- karaoke gimmicks everywhere
- bouncing subtitle lines
- too many color changes
- animated subtitle decoration that competes with meaning

Rule:
Subtitles are a comprehension layer first.

## 12. Transition rules

### 12.1 Default transitions
Recommended default transitions:
- quick directional cut
- restrained slide
- fade with directional intent
- hard cut when energy benefits clarity

### 12.2 Avoid
Avoid by default:
- random zooms
- spinning transitions
- liquid morphs
- over-stylized scene changes
- transitions that call attention to themselves first

## 13. Timing rules

### 13.1 Default pace
Default motion pace should feel:
- decisive
- efficient
- not rushed beyond comprehension
- not slow beyond tension

### 13.2 Hold rule
Important frames may need a short hold.

Hold just long enough for:
- headline reading
- concept landing
- CTA recognition

Not long enough to feel dead.

### 13.3 Over-animation rule
If every element enters, pulses, slides, glows, and moves,
there is no hierarchy left.

## 14. Layout-motion interaction

Motion should follow layout hierarchy.

This means:
- primary element moves first
- supporting element moves second or less strongly
- background motifs move least

Rule:
background motion should almost never outshout foreground meaning.

## 15. Recommended motion vocabulary v0

This is the small repeatable motion vocabulary to start with:
- directional slide
- fast fade-in/out
- short accent pulse
- linear rail reveal
- signal blink
- sharp cut
- restrained hold

This is enough for v0.

## 16. Motion by asset type

### Talking mascot clip
- Punk motion: moderate
- text motion: restrained
- subtitle motion: minimal
- background motion: low

### Hook clip
- Punk motion: moderate to high
- text motion: moderate
- background motion: low to moderate
- transition sharpness: higher

### Pilot / process explainer
- Punk motion: low
- text motion: low
- background motion: low
- signal/rail motion: restrained and functional

### Build in public update
- Punk motion: low to moderate
- text motion: low to moderate
- artifact reveal: functional
- accent pulse: moderate

## 17. Anti-patterns

Avoid these by default:
- glitch everywhere
- shaking frames
- always-on scan noise
- constant breathing lights
- animation that makes product look unstable
- mascot motion that makes Punk look unserious or goofy
- transition spam

## 18. Production rule v0

For the first production cycle:
- choose 4–6 repeatable motion moves only
- reuse them often
- vary intensity, not vocabulary size

This keeps the system controllable.

## 19. Flexibility rule

Allowed future change:
- increase polish
- add richer mascot acting
- expand motion vocabulary slightly
- define separate motion packs by channel

Not allowed to drift:
- controlled movement thesis
- message-first rule
- business-readable floor
- anti-chaos principle

## 20. Immediate recommendation summary

For v0 production:
- keep motion sharp and restrained
- let direction and control lead
- animate text minimally
- animate Punk intentionally
- use rails/signals as movement logic,
  not as decorative noise

## 21. Current one-line summary

**Goalrail motion should feel like controlled movement under tension: sharp enough to be memorable, restrained enough to remain trustworthy.**
