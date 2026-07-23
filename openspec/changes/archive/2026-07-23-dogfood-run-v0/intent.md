# Intent Snapshot

- **Intent ID:** `intent-dogfood-run-v0`
- **Version:** 3
- **Previous version:** 2
- **Status:** confirmed
- **Owner:** Goalrail owner
- **Context Pack:** `context-dogfood-run-v0` version 1
- **Run references:** pending; no implementation run or real activation exists

## Source Evidence

- **SE-1 — Owner statement:** Determine the minimal provider-neutral WorkSpec and operator-assisted `gr` flow for one trusted local Codex run in the Goalrail repository.
- **SE-2 — Owner boundary:** In v0, the existing Codex sandbox and approval mechanisms remain the enforcement boundary.
- **SE-3 — Owner boundary:** Do not add Temporal, OpenFGA, a custom Effect Gateway, a separate sandbox platform, automatic Git operations, credentials, external effects, or real activation in this slice.
- **SE-4 — Owner boundary:** Preserve excluded layers as retained target architecture, stop at candidate intent confirmation, and do not create proposal, specs, design, tasks, implementation, or Git and deployment actions.
- **SE-5 — Repository fact:** Goalrail currently has only the canary-specific `goalrail-canary` executable; its tested local flow already provides immutable run context, provider-authoritative lineage, frozen checks, append-only evidence, and operator inspection.
- **SE-6 — Repository fact:** The accepted project lifecycle keeps OpenSpec replaceable, separates intent from effect authority, and states that an already-started run remains governed by its frozen WorkSpec or manifest snapshot.
- **SE-7 — Owner instruction:** After the intent-confirmation milestone completed, the owner explicitly authorized one proposal-only milestone while leaving specs, design, tasks, implementation, Git actions, and real activation out of scope.
- **SE-8 — Owner decision:** Phase-specific planning authority belongs to the current task rather than semantic intent. The owner confirmed that planning may advance through separately requested milestones while real activation and external actions retain their independent gates.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Goalrail defines the smallest canonical WorkSpec snapshot needed for exactly one trusted repository-local run: stable WorkSpec identity and version, repository and pinned base revision, confirmed intent or change reference, bounded task and path scope, required verification checks, explicit stop conditions, and the declared local execution and effect posture. Provider launch arguments, provider session identity, and approval UI state remain outside canonical WorkSpec semantics. | Review one complete WorkSpec example and remove any field that is not required to prepare, constrain, verify, stop, or correlate this single run; verify that no field names Codex or OpenSpec. | SE-1, SE-2, SE-6, CTX-2, CTX-3, CTX-5, CTX-7 |
| OUT-2 | An operator-assisted `gr` flow resolves and validates the WorkSpec, shows the exact frozen snapshot and digest, waits for an explicit operator start action, launches at most one Codex run through a replaceable adapter, and returns a terminal receipt without an automatic retry or second run. | Exercise the flow with a local fixture and verify that preparation is inspectable, no launch occurs before the start gate, and one start can create no more than one run. | SE-1, SE-2, SE-5, CTX-1, CTX-4, CTX-6 |
| OUT-3 | For this trusted local v0, Codex continues to enforce filesystem sandbox and interactive approval decisions; `gr` passes the bounded run context but does not duplicate, weaken, bypass, or reinterpret provider enforcement. | Attempt an action that Codex denies or gates and verify that `gr` cannot convert it into an allowed effect and records the provider outcome without claiming its own authorization decision. | SE-2, SE-3, SE-6, CTX-2, CTX-3, CTX-8 |
| OUT-4 | The run remains traceable without provider lock-in: Goalrail records the frozen WorkSpec identity and digest, generated run identity, provider-authoritative root session reference, pinned repository revision, selected checks, terminal outcome, and bounded evidence references while keeping prompts, transcripts, credentials, and raw source bodies out of canonical state. | Inspect the terminal receipt and retained local evidence for a fixture run and verify deterministic correlation, required fields, bounded references, and absence of raw or secret-shaped payloads. | SE-1, SE-5, SE-6, CTX-3, CTX-5, CTX-6, CTX-7 |
| OUT-5 | After separate implementation and activation approvals, the v0 flow can dogfood one small trusted local Goalrail change from an owner-confirmed WorkSpec through reviewable local verification, then stop for owner review. | With a separately authorized dogfood task, complete exactly one local run, present its worktree delta, checks, and terminal receipt, and perform no follow-on run or Git lifecycle action. | SE-1, SE-3, SE-4, CTX-1, CTX-3, CTX-8 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Completing planning artifacts or local implementation does not itself authorize or activate a real dogfood run. Real activation and every external, Git, deployment, or publication action retain their separate owner gates. | SE-3, SE-4, SE-7, SE-8, CTX-2, CTX-3, CTX-8 |
| NG-2 | V0 does not add Temporal or another durable workflow engine, queue, scheduler, daemon, background loop, multi-run orchestration, automatic retry, repair loop, or subagent orchestration. | SE-3, SE-4, CTX-2 |
| NG-3 | V0 does not add OpenFGA, a custom authorization or task-grant system, an Effect Gateway, a credential broker, credentials, or external effects. | SE-2, SE-3, SE-6, CTX-2, CTX-3, CTX-8 |
| NG-4 | V0 does not build or select a separate sandbox platform. It assumes one trusted local repository and leaves enforcement to the existing Codex sandbox and approval mechanisms. | SE-2, SE-3, CTX-2, CTX-8 |
| NG-5 | V0 does not perform automatic branch creation, staging, commit, push, pull-request, merge, rebase, deploy, publish, or release actions. Local code edits and checks in a later owner-authorized run remain reviewable worktree changes. | SE-3, SE-4, CTX-3, CTX-8 |
| NG-6 | V0 does not make Codex or OpenSpec types canonical, build a general multi-provider control plane, or finalize the retained target architecture. | SE-1, SE-4, SE-6, CTX-2, CTX-5, CTX-7 |
| NG-7 | Excluding durable execution, authorization, effect enforcement, stronger sandboxing, credentials, and collaboration from this slice does not reject or remove those future architecture layers. | SE-3, SE-4, CTX-2 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The WorkSpec is minimal, deterministic, and provider-neutral. | One schema and fixture cover every OUT-1 field, serialize to a stable digest, contain no Codex or OpenSpec field, and contain no unused field. | SE-1, SE-6, CTX-2, CTX-3, CTX-7 |
| SIG-2 | Intent and operator gates are fail-closed. | Candidate, missing, changed, or invalid intent and WorkSpec inputs produce no launch; the operator can inspect the resolved frozen snapshot before one explicit start action. | SE-1, SE-4, SE-6, CTX-3, CTX-5 |
| SIG-3 | One start means at most one run. | A successful start produces one generated run ID, one immutable WorkSpec digest, and one provider-authoritative root session reference; duplicate start, implicit retry, or a second launch is rejected. | SE-1, SE-5, CTX-5, CTX-6 |
| SIG-4 | Existing Codex enforcement remains authoritative. | Sandbox or approval denial remains denied, requires the normal Codex interaction when applicable, and cannot be overridden by a WorkSpec or `gr` flag. | SE-2, SE-3, CTX-3, CTX-8 |
| SIG-5 | Terminal evidence is sufficient for operator review. | The receipt identifies the pinned base revision, effective scope, exact selected checks and results, terminal status, WorkSpec and run identities, provider session reference, and worktree delta reference without raw prompts, transcripts, or credentials. | SE-5, SE-6, CTX-3, CTX-5, CTX-6, CTX-7 |
| SIG-6 | The bounded v0 produces no hidden follow-on effects. | Tests and the separately authorized dogfood receipt show zero automatic Git lifecycle actions, external effects, credential use, background continuation, retries, or second runs. | SE-3, SE-4, CTX-3, CTX-8 |
| SIG-7 | Planning progress remains inert until separate activation. | Context, intent, proposal, specs, design, tasks, and local implementation can exist with zero provider launches; a real provider session begins only after a distinct owner activation instruction. | SE-4, SE-7, SE-8, CTX-2, CTX-3, CTX-8 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** Goalrail owner
- **Confirmed at:** 2026-07-23T07:30:15Z
- **Verification action:** The owner reviewed the exact correction that separates semantic intent from phase-specific task authority and explicitly replied `да`, confirming version 3.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
