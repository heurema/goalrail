## Why

The consumed dogfood run remained correctly fail-closed because Goalrail's
invocation-local Codex override registered an empty `SessionStart` matcher
group. The smallest repair is to make the exact supported Codex hook contract
explicit and locally testable while preserving the failed run and stopping
before another provider attempt.

## What Changes

- Add one bounded Codex-adapter helper that renders the exact invocation-local
  `SessionStart` matcher group and its single synchronous command handler for
  the pinned supported contract.
- Safely encode the absolute local capsule command as the provider's required
  command string; reject unsafe or unsupported input instead of emitting a
  partial hook definition.
- Add deterministic regression coverage for the spent v1 override, the
  repaired one-handler override, safe command quoting, and the existing
  provider-authoritative correlation path.
- Keep the failed v1 claim, receipt, task diff, and local activation capsule
  unchanged. A future activation must use a new exact WorkSpec and a separate
  owner gate.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Render exactly one pinned-contract `SessionStart` command handler at the Codex adapter boundary. | OUT-1, SIG-1 | NG-3, NG-4 |
| Reject unsafe or structurally incomplete hook definitions before launch material can consume them. | OUT-2, SIG-1, SIG-2 | NG-1, NG-3 |
| Exercise the repaired definition and existing correlation path with deterministic local tests only. | OUT-2, OUT-3, SIG-2, SIG-3 | NG-2, NG-3 |
| Preserve the consumed v1 lineage, receipt, capsule, and three-file task diff as historical evidence. | OUT-4, SIG-4, SIG-5 | NG-1, NG-2 |
| Stop after bounded local implementation evidence and diff review, with no provider or Git action. | OUT-3, SIG-3, SIG-4, SIG-5 | NG-2, NG-5 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-run`: Require an invocation-local Codex lifecycle hook definition to
  represent exactly one valid synchronous `SessionStart` command handler
  before it can be used for provider-authoritative lineage.

The existing `work-spec` capability remains provider-neutral and is consumed
unchanged.

## Impact

- Affected implementation is bounded to the existing
  `internal/adapters/codex` package and focused tests.
- The repair adds no external dependency, service, canonical domain field,
  WorkSpec field, credential path, persistent provider configuration, or
  execution authority.
- Existing fail-closed correlation reasons and receipt semantics remain
  compatible.
- The spent `.goalrail/activate-dogfood-run-v1` capsule and
  `/Users/vi/.local/state/goalrail/activate-dogfood-run-v1` evidence remain
  read-only.

## Non-Goals

- Do not run `gr finish`, alter or reclassify the failed v1 receipt, retry or
  resume its consumed claim, or treat provider terminal text as linked
  evidence.
- Do not launch Codex, create a new WorkSpec, run ID, claim, or provider
  attempt, or authorize a future repair run.
- Do not weaken Codex sandbox, approval, or hook-trust enforcement; add
  fallback identity; retain raw hook payloads; or leak Codex fields into the
  provider-neutral WorkSpec.
- Do not create a reusable activation service, retry loop, background process,
  additional agent, automatic Git workflow, Temporal integration, OpenFGA
  integration, Effect Gateway, credential broker, or sandbox platform.
- Do not commit, push, open or merge a PR, publish, or deploy.
