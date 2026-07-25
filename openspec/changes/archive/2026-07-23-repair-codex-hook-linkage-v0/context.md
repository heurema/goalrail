# Context Pack

- **Context Pack ID:** context-repair-codex-hook-linkage-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-23T17:16:08Z
- **Completed at:** 2026-07-23T17:22:21Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The single `activate-dogfood-run-v1` provider process completed at unchanged HEAD and changed only its three allowed WorkSpec paths, but its terminal receipt is fail-closed `unlinked` with reason `INVALID_HOOK_INPUT`; both declared check results are therefore `unavailable` in the receipt. | `/Users/vi/.local/state/goalrail/activate-dogfood-run-v1/runs/run-cad86018a6a3e1fe9241249ad3d48cef/terminal-receipt.json` | 2026-07-23T17:13:08Z | Establishes the exact observed failure without treating provider terminal text as linked Goalrail evidence. |
| CTX-2 | repository | The spent v1 capsule renders `hooks.SessionStart=[{command=[<capsule>,"hook"]}]` as an invocation-local Codex override. | `.goalrail/activate-dogfood-run-v1/contract.go:496` | 2026-07-23T17:22:21Z | Identifies the configuration emitted by Goalrail for the failed run. |
| CTX-3 | external | Codex `rust-v0.144.4` defines each `SessionStart` entry as a `MatcherGroup` with an optional `matcher` and a nested `hooks` list. A command handler requires `type="command"` and a string `command`. `MatcherGroup` does not deny unknown fields, so the v1 entry's top-level `command` is ignored while `hooks` defaults to empty. | `https://github.com/openai/codex/blob/rust-v0.144.4/codex-rs/config/src/hook_config.rs` | 2026-07-23T17:20:00Z | Confirms the root cause against the exact pinned provider source: no Goalrail `SessionStart` handler was registered. |
| CTX-4 | external | The pinned Codex `SessionStart` command input schema supplies `session_id`, absolute `cwd`, `hook_event_name="SessionStart"`, and `source` in the same value set already accepted by Goalrail; additional provider fields are present but need not be retained. | `https://github.com/openai/codex/blob/rust-v0.144.4/codex-rs/hooks/schema/generated/session-start.command.input.schema.json` | 2026-07-23T17:20:10Z | Rules out a mismatch in the fields Goalrail needs to bind lineage and keeps the repair at hook registration/transport. |
| CTX-5 | repository | The launcher captures at most one bounded payload over private one-shot Unix IPC and returns an empty payload when no hook connects; the adapter then maps an undecodable or empty lifecycle payload to `INVALID_HOOK_INPUT`. | `.goalrail/activate-dogfood-run-v1/launcher.go:80`; `.goalrail/activate-dogfood-run-v1/launcher.go:204`; `internal/adapters/codex/correlation.go:192` | 2026-07-23T17:22:21Z | Confirms the observed receipt is the intended fail-closed consequence of the missing handler, not evidence of a successful identity join. |
| CTX-6 | repository | The v1 launch claim and terminal receipt already exist, and the activation contract prohibits retry or fallback for the consumed exact run. | `openspec/changes/activate-dogfood-run-v1/tasks.md`; `.goalrail/activate-dogfood-run-v1/one_shot_test.go` | 2026-07-23T17:16:40Z | Requires the repair to preserve the spent v1 evidence and stop before any second provider run. |
| CTX-7 | owner | The owner selected a separate bounded repair milestone: diagnose and locally fix `INVALID_HOOK_INPUT`, do not mask it with manual `gr finish`, and stop before a second provider run. | Owner confirmation in the current Goalrail task | 2026-07-23T17:15:00Z | Bounds the next candidate intent and preserves a separate owner gate for any future live activation. |

## Material Unknowns

None.
