# Context Pack

- **Context Pack ID:** context-repair-stale-executable-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-30T08:00:00Z
- **Completed at:** 2026-07-30T08:20:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | `gr health`'s `checkRegisteredExecutable` stats the executable path recorded inside the managed hook command; when the path is missing it refuses to report `working` and its next action tells the user to re-run `gr connect --scaffold <name> --yes`. | repo:internal/ambient/health.go | 2026-07-30T08:05:00Z | The diagnosis is correct and the advice is the only remedy the tool offers. |
| CTX-2 | repository | `isConnected` decides presence for both scaffolds by checking for the managed marker inside a registered handler command; it never compares the registered executable path against the one connection was invoked with. `PlanConnection` calls it without the executable, though `PlanConnection` itself already receives and resolves that path. | repo:internal/ambient/connect.go | 2026-07-30T08:07:00Z | A stale registration is indistinguishable from a current one to `isConnected`, so `PlanConnection` reports `AlreadyPresent=true` and `Connect` changes nothing. |
| CTX-3 | repository | For Codex, `connectCodex` unconditionally appends a fresh managed block; it never removes an existing one first. For Claude Code, `connectClaudeCode` only replaces a session-start handler when it is unscoped (`claudeCodeSessionStartIsScoped` returns false) or adds a Stop handler when none is present (`claudeCodeHasGoalrail` returns false) — neither check inspects the handler's command for a stale path. | repo:internal/ambient/connect.go | 2026-07-30T08:09:00Z | Even if presence detection is corrected, naively re-running today's write path would either duplicate the Codex block or leave a scoped-but-stale Claude Code handler untouched. |
| CTX-4 | repository | The regression test written for the Claude Code matcher fix (`TestClaudeCodeRepairsAnUnscopedRegistration`) seeds a registration with an old executable path and explicitly comments that the stale path is "a separate defect, recorded rather than fixed here — connect currently reports 'already present' for it, which makes health's 're-run connect' advice useless." | repo:internal/ambient/connect_test.go | 2026-07-30T08:10:00Z | The scope boundary of the prior change and the existence of this defect are both pinned by a committed test comment, not just by memory. |
| CTX-5 | repository | Issue heurema/goalrail#25 records the defect, its cause in both files, the constraints a fix must hold (consented and idempotent connection; residue-free disconnection that never touches foreign handlers; Claude Code session-start stays scoped to `startup`; Goalrail never writes or reproduces scaffold trust records), and four suggested regression tests. | github:heurema/goalrail#25 | 2026-07-30T08:11:00Z | This change's scope is the issue's scope: repair-on-reconnect for a stale executable path, nothing wider. |
| CTX-6 | repository | The promoted `ambient-connect` requirement states connection and disconnection SHALL be safe to repeat, disconnection SHALL leave no residue and MUST NOT touch entries it did not add, Claude Code session-start registration SHALL stay scoped to the occurrence that opens a session, and Goalrail never writes, computes, or reproduces scaffold trust records. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-30T08:13:00Z | These are the exact constraints a repair-on-reconnect scenario must not violate; the delta for this change extends this requirement rather than replacing it. |
| CTX-7 | repository | Replacing a handler's command changes its literal text, and the promoted requirement records that the scaffold gates a hook's execution on the user reviewing and trusting its exact current definition. A repair that swaps in a new command therefore returns the affected hook to an untrusted state on the scaffold that enforces this, the same way any other command edit would. | repo:openspec/specs/ambient-connect/spec.md, internal/ambient/health.go | 2026-07-30T08:14:00Z | A silent repair would trade one silent failure (stale path) for another (repaired but untrusted, still not running, with no notice). The connection disclosure already carries the trust-step notice for a fresh connection; repair must not bypass it. |

## Material Unknowns

None blocking. The fix is scoped to what `PlanConnection`/`Connect` do with a
registration whose command differs from the currently invoked executable; it
does not touch trust detection, health's existing missing-binary check, the
announcement, retention, intent binding, or the wrapper lifecycle.
