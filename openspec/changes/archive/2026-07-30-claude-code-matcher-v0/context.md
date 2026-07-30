# Context Pack

- **Context Pack ID:** context-claude-code-matcher-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-30T06:45:00Z
- **Completed at:** 2026-07-30T07:10:24Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | external | The second scaffold's session-start matcher accepts one exact value per entry — `startup`, `resume`, `clear`, `compact`, `fork` — and an omitted, empty, or `*` matcher makes the hook fire on every occurrence of the event. | claude-code-docs:code.claude.com/docs/en/hooks | 2026-07-30T06:48:00Z | Registration that omits the matcher therefore announces on resumption, clearing, compaction, and forking, which the promoted contract forbids. |
| CTX-2 | repository | The connection routine for that scaffold writes one group per event with no matcher field at all, verified by connecting on an isolated stand and reading the produced settings file. | repo:internal/ambient/connect.go | 2026-07-30T06:53:00Z | The defect is in shipped code, not hypothetical: the produced registration is exactly the "fires on every occurrence" shape. |
| CTX-3 | repository | The owner's own working configuration for that scaffold registers four separate session-start groups, one per matcher value, for a single unrelated hook command. | local:user-scaffold-settings-2026-07-30 | 2026-07-30T06:47:00Z | Confirms the one-value-per-entry shape in practice, independently of the documentation. |
| CTX-4 | repository | The promoted requirement states that the announcement accompanies only the event that opens a run, and that recurring session events carry none. The first scaffold's transport enforces this by inspecting the hook source; the second scaffold's registration has no equivalent guard. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-30T06:50:00Z | One scaffold satisfies a promoted rule and the other silently does not. |
| CTX-5 | external | Project-level hooks for the second scaffold are an ordinary settings layer that merges with user settings rather than being gated behind per-hook review; the only precondition recorded is the folder's workspace trust. | claude-code-docs:code.claude.com/docs/en/hooks | 2026-07-30T06:49:00Z | The promoted limitation about user-scope registration was generalised from one scaffold and does not hold for this one. |
| CTX-6 | repository | The promoted requirement records that registering inside the repository is externally blocked, stated without naming which scaffold that evidence came from. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-30T06:51:00Z | An accurate observation about one provider was written as a property of attachment in general. |
| CTX-7 | repository | Live verification of the second scaffold could not be performed: an isolated home does not satisfy its login state, and the remaining routes required either writing into the owner's working configuration or copying an account file containing unrelated project history. | local:isolated-verification-attempt-2026-07-30 | 2026-07-30T07:05:00Z | The matcher defect is fixed from documentation and the owner's own configuration; end-to-end behaviour for this scaffold remains unobserved. |

## Material Unknowns

None blocking. One limit is recorded rather than resolved: no live session has
exercised the second scaffold, so its announcement delivery, question retention,
and whether it prompts for any approval remain unobserved. The fix rests on the
provider's published matcher contract and on the shape of the owner's own
working configuration, which agree with each other.
