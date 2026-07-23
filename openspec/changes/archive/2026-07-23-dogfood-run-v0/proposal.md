## Why

Goalrail has a canary-specific local operator command but no general, provider-neutral contract for one controlled coding-agent run. This change introduces the smallest dogfoodable WorkSpec and `gr` flow, using Codex as the first replaceable adapter while keeping real activation separately gated.

## What Changes

- Add a canonical, versioned WorkSpec snapshot for one trusted repository-local run. It binds a confirmed intent or change, pinned repository revision, bounded task and path scope, required checks, stop conditions, and local execution/effect posture, then produces a deterministic digest.
- Add a minimal `gr` operator surface that resolves and validates the WorkSpec, displays the frozen snapshot and digest, requires an explicit start action, permits at most one launch, and rejects duplicate starts or automatic retries.
- Add a provider adapter boundary whose launch arguments, root session identity, sandbox behavior, and approval interaction stay outside canonical WorkSpec semantics. Implement Codex only as the first v0 adapter without introducing a general multi-provider control plane.
- Record a bounded terminal receipt containing WorkSpec/run identity, provider-authoritative root session reference, pinned revision, effective scope, selected checks and results, terminal status, and worktree-delta reference without retaining prompts, transcripts, credentials, or raw source content.
- Prove the complete prepare-to-receipt path with deterministic local fixtures and denial/duplicate-start cases. Keep real Goalrail dogfood activation blocked until a separate owner instruction.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Define and deterministically digest the minimal canonical WorkSpec. | OUT-1, SIG-1, SIG-2 | NG-3, NG-4, NG-6, NG-7 |
| Add the explicit operator prepare/start flow with a one-launch ceiling. | OUT-2, SIG-2, SIG-3 | NG-1, NG-2, NG-5 |
| Isolate Codex launch and enforcement details behind a replaceable provider boundary. | OUT-2, OUT-3, SIG-4 | NG-3, NG-4, NG-6, NG-7 |
| Retain a bounded, reviewable terminal receipt and worktree-delta reference. | OUT-4, SIG-5, SIG-6 | NG-3, NG-5 |
| Verify the flow locally while leaving real activation and follow-on actions blocked. | OUT-5, SIG-6, SIG-7 | NG-1, NG-2, NG-3, NG-5 |

## Capabilities

### New Capabilities

- `work-spec`: Provider-neutral definition, validation, freezing, and deterministic identity of the canonical single-run input.
- `local-run`: Operator-gated preparation, one-shot provider launch, provider-authoritative lineage, terminal verification receipt, and activation boundary for a trusted local run.

### Modified Capabilities

None.

## Impact

- Add provider-neutral WorkSpec, run, and receipt types plus validation to the canonical Go domain.
- Add a general local-run operator boundary and a `gr` command without coupling them to canary manifests or OpenSpec runtime types.
- Add the first Codex launch adapter by reusing proven immutable-context and provider-authoritative identity patterns without changing Codex sandbox or approval enforcement.
- Add repository-local deterministic fixtures and focused tests for validation, digest stability, explicit start, duplicate rejection, denial propagation, receipt safety, and absence of hidden follow-on effects.
- Leave existing canary commands, manifests, evidence streams, canonical capability specs, and external interfaces unchanged. Add no third-party dependency, service, database, credential path, or deployment surface.

## Non-Goals

- Do not create specs, design, tasks, implementation, or a real dogfood run in this proposal milestone.
- Do not activate a real run, automatically retry, launch subagents, or continue in the background.
- Do not add Temporal or another workflow engine, OpenFGA or another authorization system, an Effect Gateway, credential broker, separate sandbox platform, dashboard, database, queue, scheduler, or daemon.
- Do not perform automatic branch creation, staging, commit, push, pull-request, merge, rebase, deployment, publication, or release actions.
- Do not add credentials or external effects, bypass provider sandbox/approval decisions, make Codex or OpenSpec types canonical, implement additional provider adapters, or remove retained target architecture layers.
