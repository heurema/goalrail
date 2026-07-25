# Intent Snapshot

- **Intent ID:** intent-repair-codex-hook-linkage-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-repair-codex-hook-linkage-v0 version 1
- **Run references:** failed source run `run-cad86018a6a3e1fe9241249ad3d48cef`; repair run pending and out of scope

## Source Evidence

- **SE-1 (owner):** Continue with a separate bounded milestone that diagnoses and locally fixes `INVALID_HOOK_INPUT`, does not mask the failure with manual `gr finish`, and stops before a second provider run.
- **SE-2 (local receipt):** The source run completed at unchanged HEAD but produced a fail-closed `unlinked` receipt with `INVALID_HOOK_INPUT` and unavailable check results.
- **SE-3 (repository and pinned provider source):** The v1 activation rendered an empty Codex matcher group because its invocation-local hook override used a top-level command array instead of the pinned Codex 0.144.4 nested command-handler shape.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Provide the smallest local repair that makes an invocation-local hook definition for the exact supported Codex contract register one synchronous `SessionStart` command handler and deliver the bounded provider-authoritative lifecycle payload to Goalrail's existing correlation path. | Review the emitted configuration and a deterministic regression test against the pinned handler and payload contracts. | SE-2, SE-3, CTX-2, CTX-3, CTX-4 |
| OUT-2 | Preserve fail-closed evidence semantics: missing, malformed, conflicting, or unsupported hook evidence remains unlinked and cannot be replaced by manual success, retry, fallback identity, or provider terminal text. | Review negative-path tests and the unchanged terminal receipt of the source run. | SE-1, SE-2, CTX-1, CTX-5, CTX-6 |
| OUT-3 | Produce local proof of the repair without launching Codex or consuming another provider attempt, while keeping the WorkSpec provider-neutral and leaving Codex sandbox and approval policy as the execution enforcement boundary. | Run the bounded repair tests and repository test suite, then inspect local state and diff before stopping. | SE-1, CTX-6, CTX-7 |
| OUT-4 | Leave the failed v1 run, its launch claim, its terminal receipt, and its three-file task diff intact as historical evidence. | Compare the source run state artifacts and task paths before and after the repair slice. | SE-2, CTX-1, CTX-6 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not run `gr finish`, rewrite or delete the source receipt, retry or resume the consumed v1 launch, or claim that its unavailable checks are linked Goalrail evidence. | SE-1, SE-2, CTX-1, CTX-6 |
| NG-2 | Do not launch a second provider process, activate another WorkSpec, create a new run ID or launch claim, or use credentials or external effects in this milestone. | SE-1, CTX-7 |
| NG-3 | Do not weaken Codex sandbox or approval settings, bypass hook trust, add an alternate identity fallback, retain raw hook payloads, or move provider-specific fields into the provider-neutral WorkSpec. | CTX-4, CTX-5, CTX-7 |
| NG-4 | Do not add Temporal, OpenFGA, an Effect Gateway, a sandbox platform, workflow retries, background or multi-agent execution, automatic Git operations, or a general long-lived activation service. These remain outside this slice, not removed from the target architecture. | CTX-7 |
| NG-5 | Do not commit, push, open a PR, merge, publish, or deploy. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The original v1 hook override is represented by a regression fixture that proves it registers no handler under the pinned contract, while the repaired override represents exactly one synchronous `SessionStart` command handler with a safely quoted capsule command. | Deterministic local test passes without starting Codex or a model session. | CTX-2, CTX-3 |
| SIG-2 | A pinned-schema-compatible `SessionStart` payload reaches the existing one-shot IPC and correlation path and produces verified lineage; missing and malformed payload cases remain unlinked. | Focused Go tests cover one positive payload and the existing negative matrix. | CTX-4, CTX-5 |
| SIG-3 | All selected repair-package tests and `go test ./...` pass. | Both commands exit 0 in the local implementation cycle. | SE-1 |
| SIG-4 | No new provider attempt exists and the failed source run's claim, observation, receipt, HEAD, and task diff remain unchanged. | Read-only state and Git comparison shows no new run and no mutation of the source evidence. | CTX-1, CTX-6 |
| SIG-5 | The final local diff is limited to the confirmed repair implementation, its tests, and this change's planning artifacts; no Git or external action occurred. | Review `git status` and the bounded diff at the terminal stop. | SE-1, CTX-7 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-23T17:27:33Z
- **Verification action:** Owner actively confirmed candidate version 1 in the current Goalrail task.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
