# Goalrail — Figma Workflow

## 1. Recommendation

**Separate FigJam is not required right now.**

Go directly into Figma, but do not start with “beautiful design”.
Use one Figma file as the working source of truth for:
- message lock
- competitor map
- wireframes
- copy lock
- scene transitions
- handoff notes

Use FigJam only if later we need a collaborative workshop or a fast alignment board with multiple people.

## 2. Why Figma now

The current problem is not “lack of ideas”.
The current problem is lack of fixation:
- the message keeps changing
- the object being sold keeps changing
- the scenes drift away from the product brief

Figma is better for locking structure and states.

## 3. Recommended file structure

### Page 0 — Source of Truth
Keep this page text-only.
Include:
- primary pain statement
- one-line product thesis
- what Goalrail is not
- central object = shared working contract
- public flow = scene 1 -> scene 2 -> pilot request

### Page 1 — Competitor Map
Use a simple matrix or table.
Rows:
- Factory
- Atlassian Rovo Dev
- GitHub Copilot coding agent
- GitLab Duo Agent Platform
- Sourcegraph
- Devin
- Harness

Columns:
- task framing
- shared contract
- execution
- verification
- business visibility
- provider-native vs supplement layer
- what not to compete on
- our wedge

### Page 2 — Scene 1 Wireframe
Black and white only.
No palette exploration.
No decorative details.
Only:
- incoming task
- shared contract
- CTA: Открыть разбор

### Page 3 — Scene 2 Wireframe
Black and white only.
Only:
- detailed breakdown artifact
- quiet continuation strip
- pilot request panel
- CTA: Получить пилотный разбор

### Page 4 — Copy Lock
All text in one place:
- headlines
- sublines
- labels
- artifact block titles
- form labels
- CTA text
- tiny notes

No layout exploration on this page.
Only wording.

### Page 5 — Visual Pass
Only after pages 0–4 are fixed.
This is where palette, spacing, typography and final polish happen.

### Page 6 — Handoff / Interaction Notes
Document:
- what happens on “Открыть разбор”
- what happens on “Получить пилотный разбор”
- field behavior
- what is static vs interactive
- what is only visual framing

## 4. What not to do in Figma

Do not start with:
- styles
- gradients
- hero explorations
- logo-heavy pages
- long website sections
- provider buttons
- dashboard-like chrome

Do not make a full marketing site first.

## 5. Working sequence

1. Lock pain statement on Page 0
2. Build competitor map on Page 1
3. Build wireframes for Scene 1 and Scene 2
4. Lock copy
5. Only then do visual design
6. Only then build prototype links between scenes

## 6. Prototype rule

The first clickable prototype should only answer:
- what the first screen is
- what opens after “Открыть разбор”
- how the pilot request is reached

It should not try to simulate the whole product.

## 7. When FigJam becomes useful

Use FigJam later only if:
- we need an alignment workshop with several people
- we need a fast whiteboard for research clustering
- we need to compare many competitor directions visually

Until then, normal Figma is enough.

## 8. Done criteria

### Scene 1 is done when:
- it clearly communicates “incoming task -> working contract” in 5 seconds
- nobody mistakes it for a prompt tool or dashboard

### Scene 2 is done when:
- it clearly communicates “breakdown -> next delivery step -> pilot request”
- the CTA feels like a pilot motion, not SaaS signup

### File is ready for implementation when:
- copy is locked
- interaction notes are written
- visual language is consistent across both scenes

## 9. Suggested next tool after Figma

After Figma lock:
- use Framer for a live public landing test
- use a coded prototype later only if interaction depth is needed

## 10. References

1. FigJam overview: https://www.figma.com/figjam/
2. Figma Dev Mode overview: https://www.figma.com/dev-mode/
3. Dev Mode guide: https://help.figma.com/hc/en-us/articles/15023124644247-Guide-to-Dev-Mode
