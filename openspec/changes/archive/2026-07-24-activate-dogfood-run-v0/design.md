## Context

The existing local-run service already freezes WorkSpecs, persists one-shot
launch claims, invokes an injected adapter, and records bounded provider
observations and receipts. Production construction deliberately injects no
adapter, so `start` returns `ACTIVATION_REQUIRED` before any state or provider
effect. The current change must preserve that default while making one
owner-selected dogfood activation possible and making the operator CLI
understandable.

## Goals / Non-Goals

**Goals:**

- Preserve the existing provider-neutral WorkSpec, one-shot claim, correlation,
  evidence, and provider-enforcement boundaries.
- Add concise top-level and per-command `gr` help without changing successful
  JSON command results.
- Define an exact, auditable admission for one WorkSpec and one pinned revision.

**Non-Goals:**

- No general authorization, task-grant, credential, sandbox, durable-execution,
  retry, background, multi-agent, or Git lifecycle system.
- No actual provider invocation as part of implementation verification.

## Decisions

### Keep WorkSpec and admission distinct

WorkSpec remains a provider-neutral description of requested work. An explicit
admission record, if selected, binds one frozen digest and pinned revision but
does not become a WorkSpec field or claim authority over other effects. This
preserves the distinction between requested work and permission to begin one
provider launch.

Alternative rejected: adding Codex details or an activation flag to WorkSpec.
That would leak provider and authority semantics into the canonical domain.

### Preserve fail-closed construction by default

The production service continues to have no adapter unless the exact dogfood
admission is present and validates. Every other start follows the existing
`ACTIVATION_REQUIRED` path before state mutation or provider invocation.

Alternative rejected: an environment switch or unrestricted CLI activation
flag. Either would create a reusable broad activation surface.

### Reuse the existing local-run lifecycle after admission

Once the exact admission gate passes, the existing service owns preparation,
atomic launch claim creation, adapter invocation, correlation, provider outcome,
and receipt behavior. The change does not duplicate those paths or add retries.

### Use one fixed out-of-repository admission record

The owner selected option A on 2026-07-24: one record at the fixed
`activate-dogfood-run-v0.admission.json` path under the Goalrail state root.
The record is dedicated to this change and binds exactly one prepared WorkSpec
digest and pinned base revision. It has no provider fields, credentials, actor
roles, delegation, or effect permissions.

```json
{
  "schema": "goalrail.dogfood-admission/v0",
  "change": "activate-dogfood-run-v0",
  "work_spec_digest": "sha256:<exact prepared digest>",
  "base_revision": "<exact pinned revision>"
}
```

Admission validates the immutable record before atomically publishing its fixed
`activate-dogfood-run-v0.admission.consumed.json` hard-link marker. The marker
is published before run ID generation, launch-claim persistence, or adapter
invocation. Missing, malformed, mismatched, stale, replaced, or
already-consumed records return `ACTIVATION_REQUIRED` without creating run
artifacts. Goalrail provides no operation to re-arm a successfully consumed
record at the same state root.

The default `NewService` construction remains adapter-free. A later explicitly
authorized local host must deliberately construct `NewDogfoodService` with the
existing Codex adapter; no command flag or environment variable selects this
path, and this implementation milestone does not launch a provider process.

Alternative rejected: a caller-selected record path, record-creation command,
environment override, or reusable admission directory. Each would make the
one-run authority broader or easier to repeat than this dogfood slice requires.

### Keep help separate from command results

Top-level help is a static lifecycle summary. Per-command help is provided by
the existing Go flag sets and exits before service construction. Successful
command JSON remains unchanged.

## Correlation and Evidence

The exact WorkSpec digest, pinned revision, launch claim, generated run ID,
adapter name/version, provider observation, selected check results, and bounded
worktree delta remain the evidence chain. An admission denial records no run ID
or launch claim. A successful implementation verification uses deterministic
tests only; a future real run produces the normal terminal receipt.

## Measurement and Stop Conditions

- Focused command and local-run tests prove help behavior, exact admission, and
  denial before state mutation.
- `go test ./cmd/gr` and `go test ./...` are the frozen checks for a later
  dogfood run; they are also run locally before requesting review.
- Stop immediately if the admission design requires a general activation flag,
  credentials, provider-enforcement bypass, or a second provider launch.

## Risks / Trade-offs

- [An exact admission mechanism can accidentally become a reusable authority
  surface] → Require an owner decision on its storage and creation before
  implementation; deny everything else by default.
- [Help text can drift from runtime behavior] → Test top-level and each command
  help alongside existing JSON-output compatibility tests.
- [A real provider run is mistaken for implementation verification] → Keep live
  activation as a separate owner gate after code review.

## Rollback

Removing the explicit admission wiring restores the current production behavior:
`start` has no adapter and returns `ACTIVATION_REQUIRED` before state mutation.
The WorkSpec, provider adapter, fixture lifecycle, and existing receipts remain
unchanged.

## Decision Evidence

- Independent critique: Claude Fable 5 selected the one-time record and
  identified atomic check-and-consume before run ID, claim, or provider call as
  the critical guard.
- Owner decision: option A was explicitly selected, later deferred during the
  help-only stream, and explicitly re-authorized for tasks 2.2 and 2.3 on
  2026-07-24.
