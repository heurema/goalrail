# Context Pack

- **Context Pack ID:** context-hook-trust-disclosure-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-29T18:00:00Z
- **Completed at:** 2026-07-29T18:56:32Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The first live verification of background attachment established the whole chain: the announcement reached the agent, which quoted the reserved path verbatim; given a contradictory task the same model wrote a question instead of guessing and left the source file untouched; and the stop hook retained the question with its own identifier, digest, session reference, and an explicit unbound reason. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-29T18:34:00Z | The built capability works end to end once its hooks actually run; only the path to running them was missing. |
| CTX-2 | external | A registered hook does not run until the exact hook definition is reviewed and trusted, and trust is recorded against the hook's current hash, so a changed command is skipped again until re-reviewed. Users manage this through the `/hooks` command, and Codex prints a warning at startup when hooks need review. | codex-docs:learn.chatgpt.com/docs/hooks | 2026-07-29T18:50:00Z | Registration is only half of connection; without disclosure the user sees silence and concludes the product is broken. |
| CTX-3 | repository | Observed on the isolated stand: after connection the first interactive session recorded trust entries in the user configuration but ran no hook; only the following session executed them. Three earlier non-interactive runs executed nothing and printed nothing. | local:isolated-verification-stand-2026-07-29 | 2026-07-29T18:33:00Z | The silent first session is the concrete defect this change addresses, reproduced rather than inferred. |
| CTX-4 | external | Project-local hooks load only from a trusted project layer. On the stand the project carried `trust_level = "trusted"` and a correctly parsed `.codex/hooks.json`, yet `/hooks` listed zero installed hooks for every event. | codex-docs:learn.chatgpt.com/docs/hooks | 2026-07-29T18:52:00Z | Moving the attachment into the repository, which would have made the privacy boundary structural, does not currently work. |
| CTX-5 | external | An open defect filed 2026-07-25 against version 26.721.31836 reports that project-level hooks receive no trust prompt and are silently skipped, that the session-start event has already passed by the time the user finds the setting, and that a new session is required afterwards. No maintainer response and no workaround are recorded. | codex-issue:openai/codex#35306 | 2026-07-29T18:54:00Z | The project-level route is externally blocked, so the global route is not a preference but the only working one. |
| CTX-6 | external | An open request asks for a supported way for local wrapper installers to obtain consented hook trust, and records that integrators currently reproduce the trust hash and write it directly, which depends on private implementation details. | codex-issue:openai/codex#21615 | 2026-07-29T18:55:00Z | Forging trust is technically possible and already practised; it must be refused explicitly rather than left undecided. |
| CTX-7 | repository | The promoted capability requires connection to be consented and reversible and requires background failure to stay invisible to the session, but states nothing about what the user must do after connecting. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-29T18:20:00Z | The gap is in the contract, not only in the command output. |

## Material Unknowns

None blocking. One limit is recorded rather than resolved: whether the second
supported scaffold applies the same trust gate has not been observed, because
the live verification ran against one scaffold only.
