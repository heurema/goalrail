# 01 Question Graph

## Root question

How should Omnigent integrate codebase-memory-mcp so local agents get code
intelligence by default without vendoring a binary or hiding external config
mutation inside a Python wheel dependency?

## Decision branches

1. Package dependency
   - Add `codebase-memory-mcp` to `pyproject.toml`.
   - Risk: dependency install cannot safely run post-install side effects.
   - Status: reject for first integration.

2. Vendored binary/source
   - Bundle `../codebase-memory-mcp` or built artifacts into Omnigent.
   - Risk: wheel size, platform matrix, release drift, ownership confusion.
   - Status: reject for first integration.

3. Companion tool from installer/setup
   - `install_oss.sh` installs codebase-memory-mcp explicitly.
   - `omnigent setup` checks/repairs configuration explicitly.
   - Risk: must handle download failures and config/index side effects.
   - Status: accept with guardrails.

4. Remote sandbox host image layer
   - Bake codebase-memory-mcp into Omnigent host image.
   - Risk: separate from local installer; wheel overlays use `--no-deps`.
   - Status: separate follow-up.

## Subquestions

- Where can Omnigent install the companion binary without breaking base install?
- Which codebase-memory-mcp commands are safe as automatic warning-not-fatal
  steps?
- Which commands mutate agent configs or local indexes and need prompt/guardrails?
- What tests prove the skip flag and warning behavior?
- What should be deferred to host image work?

## Decision gates

- Gate A: Does the command mutate external agent configs?
- Gate B: Can the command delete or rebuild existing user state?
- Gate C: Can failure leave Omnigent unusable?
- Gate D: Does the command affect remote sandboxes?
- Gate E: Is the behavior testable without network and without real config writes?

## Root question


## Sub-questions


## Alternative framings


## Stakeholders


## Schools of thought


## Likely controversies


## Evidence gaps


## Follow-up branches


## Kill criteria / what would change the conclusion
