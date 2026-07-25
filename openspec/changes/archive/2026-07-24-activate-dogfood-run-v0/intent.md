# Intent Snapshot

- **Intent ID:** `intent-activate-dogfood-run-v0`
- **Version:** 1
- **Status:** confirmed
- **Owner:** Goalrail owner
- **Context Pack:** `context-activate-dogfood-run-v0` version 1
- **Run references:** pending; no WorkSpec has been prepared and no provider run is authorized

## Source Evidence

- **SE-1 — Owner statement:** Define the minimal fail-closed activation boundary for exactly one owner-assisted trusted local Codex run through the implemented provider adapter while keeping Codex sandbox and approval behavior as the enforcement boundary.
- **SE-2 — Owner task:** The first WorkSpec should make built-in `gr` help useful: show a concise lifecycle in `gr help`, provide help for individual commands, and preserve all existing JSON outputs and execution semantics.
- **SE-3 — Owner scope:** Bound the task to `cmd/gr/main.go`, `cmd/gr/main_test.go`, and `docs/local-run-v0.md`; require `go test ./cmd/gr` and `go test ./...`; stop after a local terminal receipt and presentation of the worktree diff.
- **SE-4 — Owner boundaries:** Do not add Temporal, OpenFGA, an Effect Gateway, a new sandbox platform, credentials, external effects, automatic Git operations, retries, background execution, multi-agent execution, or any commit, push, pull request, merge, deployment, or publication. Preserve these excluded layers as retained target architecture where applicable.
- **SE-5 — Owner milestone boundary:** Create only the bounded Context Pack and candidate Intent Snapshot now. Do not create proposal, specs, design, tasks, implementation, preparation, admission, provider execution, or Git actions before a later explicit authorization.
- **SE-6 — Repository fact:** The confirmed `dogfood-run-v0` contract already defines a provider-neutral immutable WorkSpec, an explicit operator start, at most one launch, provider-authoritative identity, bounded evidence, and a separate real-activation gate.
- **SE-7 — Repository fact:** Production local-run `start` currently returns `ACTIVATION_REQUIRED` before a run ID, launch claim, or provider invocation, while duplicate claims are rejected after an allowed launch begins.
- **SE-8 — Repository fact:** The Codex local-run adapter is implemented but not activated by the production `gr` service; it passes bounded run context to Codex and reports provider outcome and session identity without replacing provider enforcement.
- **SE-9 — Repository fact:** `gr help` currently emits only a one-line usage string, command flag sets already define individual flags, command results are JSON, and the local-run guide documents the inert lifecycle and excluded effects.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Goalrail permits exactly one owner-selected, prepared WorkSpec to cross the current production activation denial and invoke the implemented Codex adapter at most once. Missing, mismatched, stale, invalid, or already-claimed input remains denied before another provider launch; the slice creates no general, reusable, or durable activation surface. | Attempt the exact authorized WorkSpec once, then attempt absent, different, stale, malformed, and duplicate inputs; verify that only the first exact attempt can reach the adapter and every other case fails closed. | SE-1, SE-4, SE-6, SE-7, SE-8, CTX-3, CTX-4, CTX-6, CTX-7, CTX-10 |
| OUT-2 | The authorized WorkSpec remains provider-neutral and pins one small task: make `gr help` explain the `prepare → inspect → start → finish` lifecycle concisely, make built-in help available for each command, and update the local-run guide without changing existing JSON outputs or command execution semantics. Its allowed paths are exactly `cmd/gr/main.go`, `cmd/gr/main_test.go`, and `docs/local-run-v0.md`. | Inspect the frozen WorkSpec and resulting diff; verify that no provider-specific field was added, all changed paths are in the allowlist, help covers the lifecycle and every command, and existing machine-readable command results remain unchanged. | SE-2, SE-3, SE-6, SE-9, CTX-5, CTX-8, CTX-9 |
| OUT-3 | The run is explicitly owner-assisted: the owner initiates the one start, observes and answers normal Codex approval prompts when applicable, and retains review authority. Goalrail supplies bounded context and records the outcome but does not grant permissions, inject credentials, bypass a denial, or reinterpret Codex sandbox and approval decisions. | Exercise a provider-gated or denied action during bounded verification and confirm that normal Codex enforcement remains authoritative and that Goalrail cannot convert the result into permission. | SE-1, SE-4, SE-8, CTX-2, CTX-3, CTX-7 |
| OUT-4 | The run executes only the frozen checks `go test ./cmd/gr` and `go test ./...`, produces one bounded terminal receipt correlated to the WorkSpec, run, pinned base revision, provider session, check results, and worktree delta, presents the local diff, and then stops for owner review without a retry, second run, or follow-on effect. | Inspect the terminal receipt and worktree after the single run; verify both frozen checks are represented, the receipt and diff are reviewable, and no second launch or subsequent action occurs. | SE-3, SE-4, SE-6, SE-7, CTX-3, CTX-5, CTX-6, CTX-9, CTX-10 |
| OUT-5 | The activation remains a narrow dogfood canary. Planning artifacts, local code, or a confirmed Intent Snapshot do not by themselves authorize preparation or provider execution; each later phase retains its explicit owner gate. | Before each later phase, verify that no preparation state, launch claim, provider process, or Git effect exists without the corresponding owner instruction. | SE-5, SE-6, CTX-2, CTX-3, CTX-4 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | This slice does not create a general activation flag, environment override, reusable authorization, multi-WorkSpec admission path, or long-lived mechanism broader than the one exact owner-selected WorkSpec and run. | SE-1, SE-4, SE-7, CTX-3, CTX-4, CTX-6 |
| NG-2 | The help task does not change existing JSON outputs, prepare, inspect, start, or finish execution semantics, WorkSpec semantics, or terminal receipt semantics beyond what is strictly required to admit the one run. | SE-2, SE-3, SE-6, SE-9, CTX-5, CTX-8 |
| NG-3 | The slice does not add Temporal or another workflow engine, a queue, scheduler, daemon, retry or repair loop, background continuation, or multi-agent or subagent execution. | SE-4, CTX-2, CTX-3, CTX-9 |
| NG-4 | The slice does not add OpenFGA, a custom task-grant or authorization system, an Effect Gateway, a credential broker, credentials, external effects, or a new sandbox platform. | SE-1, SE-4, CTX-2, CTX-3, CTX-7, CTX-9 |
| NG-5 | The run does not create or switch branches, stage, commit, push, open a pull request, merge, rebase, deploy, publish, release, or perform any automatic Git lifecycle action. | SE-3, SE-4, CTX-3, CTX-9 |
| NG-6 | The slice does not make Codex or OpenSpec types canonical, add provider-specific fields to WorkSpec, build a general provider control plane, or finalize the target architecture. | SE-4, SE-6, SE-8, CTX-2, CTX-5, CTX-7 |
| NG-7 | Excluding durable execution, authorization, effect enforcement, stronger sandboxing, credentials, collaboration, retries, and repair from this slice does not reject or remove those retained architecture layers. | SE-4, CTX-2, CTX-3 |
| NG-8 | This candidate Intent Snapshot does not authorize proposal, specs, design, tasks, implementation, WorkSpec preparation, admission, provider execution, or Git actions. | SE-5, CTX-2, CTX-4 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Activation is exact and fail-closed. | Only the owner-selected WorkSpec digest at its pinned repository revision can reach the adapter once; every absent, mismatched, invalid, stale, or duplicate request creates no additional provider invocation. | SE-1, SE-6, SE-7, CTX-4, CTX-5, CTX-6, CTX-10 |
| SIG-2 | The single-run ceiling is preserved. | The bounded activation produces at most one immutable launch claim, one generated run ID, one provider-authoritative root session reference or explicit unlinked reason, and one terminal receipt; no automatic retry or second launch occurs. | SE-1, SE-4, SE-6, SE-7, SE-8, CTX-3, CTX-6, CTX-7 |
| SIG-3 | Built-in help is useful without breaking automation. | `gr help` shows the concise lifecycle and points to help for `prepare`, `inspect`, `start`, and `finish`; each command has built-in guidance; existing JSON-producing command tests remain byte- and shape-compatible except where tests only add help coverage. | SE-2, SE-9, CTX-8, CTX-9 |
| SIG-4 | The exact frozen checks pass. | `go test ./cmd/gr` and `go test ./...` both complete successfully in the authorized run and their results are represented in the terminal receipt. | SE-3, CTX-5 |
| SIG-5 | Provider enforcement remains authoritative. | Codex sandbox or approval gating behaves normally, no Goalrail input can widen it, no credential is introduced, and a denial remains denied in the recorded provider outcome. | SE-1, SE-4, SE-8, CTX-3, CTX-7 |
| SIG-6 | Scope drift fails closed. | The diff contains changes only under the three allowed paths, the repository base revision remains the pinned revision, and any out-of-scope path or `HEAD` change prevents a successful terminal result. | SE-3, SE-6, CTX-5, CTX-9, CTX-10 |
| SIG-7 | The run stops at the local owner-review boundary. | After the terminal receipt and diff are shown, there are zero retries, second runs, background continuations, Git lifecycle actions, external effects, publications, or deployments. | SE-3, SE-4, CTX-3, CTX-6, CTX-9 |
| SIG-8 | This planning milestone remains inert. | The change contains only OpenSpec change metadata, `context.md`, and candidate `intent.md`; no downstream planning artifact, implementation edit, preparation state, launch claim, provider process, or Git action exists. | SE-5, CTX-2, CTX-4 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** Goalrail owner
- **Confirmed at:** 2026-07-24T12:31:42Z
- **Verification action:** The owner actively confirmed candidate Intent Snapshot version 1 during the Goalrail live development session.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
