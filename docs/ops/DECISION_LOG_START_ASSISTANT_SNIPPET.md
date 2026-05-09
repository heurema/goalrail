# Decision Log Snippet - Start Assistant

Suggested decision entry:

```text
Date: 2026-05-07
Project: Goalrail
Decision: Make `/start` a guided AI entry page, implemented first as static guided answers and then as a source-grounded public assistant over approved Goalrail knowledge.
Reason: English-first global traffic needs a low-friction explanation surface connected to current GitHub-backed project knowledge.
What this prevents: sending global readers to RU-only pages, overbuilding a long landing before validation, exposing OpenAI keys in browser code, and allowing public answers to hallucinate beyond approved docs.
Review date: 2026-06-07
```

If accepted, add this to `docs/ops/DECISIONS.md` following the existing decision-log format.
