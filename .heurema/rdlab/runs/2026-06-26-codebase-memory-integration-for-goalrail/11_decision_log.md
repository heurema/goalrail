# 11 Decision Log

```text
Date: 2026-06-26
Project: Goalrail
Decision: Integrate codebase-memory-mcp as an explicit companion tool through installer/setup, not as a vendored binary or hidden Python dependency.
Reason: codebase-memory-mcp install mutates external agent configs and its PyPI wrapper downloads a release binary on first run; those side effects belong in visible setup flows. Current install -y can also delete existing indexes, so full repair needs guardrails.
What this prevents: hidden wheel side effects, vendored binary drift, broken remote sandbox assumptions, and silent deletion/rebuild of existing CBM indexes.
Review date: Before first Goalrail implementation PR or after CBM provides a non-destructive repair command.
```
