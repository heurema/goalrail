## Why

WorkSpec preparation already verifies the exact confirmed intent bytes and digest, then parses that intent against the sibling Context Pack the intent declares, and fails closed before any prepared state exists. The behaviour is implemented and covered by regression tests, but no promoted capability states it.

The only existing wording sits inside an unmerged change, embedded in a `MODIFIED Requirements` block for `Real activation remains separately gated` — a requirement that `main` rewrote when the exact admission boundary landed. That wording cannot be transplanted, and its change declares an intent ID that an archived change on `main` already claims.

Without a promoted requirement, an implementation that fails closed on invalid context is indistinguishable from an accident, and a later change could remove it without violating any stated contract.

## What Changes

- Modify the promoted `local-run` requirement `Run preparation is inspectable and does not launch` so it states that preparation validates the confirmed intent together with the sibling Context Pack the intent declares.
- Require both artifacts to resolve inside the canonical repository root, and require preparation to fail before any prepared state, run ID, launch claim, or provider invocation when the Context Pack is missing, unreadable, mismatched, malformed, oversized, or escaping that root.
- Keep the requirement provider-neutral: no Context Pack fields enter the canonical WorkSpec, and the check depends on no proposal, specification, design, task, provider, or activation artifact.
- Add no implementation. The behaviour ships unchanged from the existing commit; this change only gives it a contract.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| State preparation-time validation of the intent together with its declared sibling Context Pack, both confined to the canonical repository root. | OUT-1, SIG-1 | NG-1, NG-3 |
| State that every invalid, unreadable, mismatched, oversized, or escaping Context Pack fails preparation before prepared state, run ID, claim, or provider invocation. | OUT-2, SIG-1 | NG-1, NG-5 |
| Keep the requirement provider-neutral and free of canonical WorkSpec schema changes. | OUT-3, SIG-4 | NG-1, NG-3, NG-6 |
| Give the behaviour a spec home under a new intent identity without editing merged evidence. | OUT-4, SIG-3 | NG-2, NG-4 |
| Map every stated scenario onto an existing regression test rather than new code. | OUT-1, OUT-2, SIG-2 | NG-1 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-run`: Preparation validates the confirmed intent together with the sibling Context Pack it declares, confines both to the canonical repository root, and fails closed before any prepared state, run ID, launch claim, or provider invocation when that binding is absent or invalid.

## Impact

- No production code changes. The behaviour already exists in `internal/adapters/openspec/work_spec_intent.go` with regression coverage in its sibling test file.
- Canonical WorkSpec content, receipt format, `gr` command behaviour, JSON results, and error contracts remain unchanged.
- No merged file under `openspec/changes/archive/` is edited, and no existing intent ID or version is reused.
- No new dependency, service, credential, deployment surface, or breaking change.
- The `dogfood-activation`, `operator-cli-help`, `intent-snapshot`, `context-pack`, and `work-spec` capabilities are untouched.

## Non-Goals

- Do not change the existing implementation, the canonical WorkSpec or receipt schemas, or any command behaviour. This change is specification-only.
- Do not reuse, re-version, amend, or edit `intent-activate-dogfood-run-v0`, the archived `2026-07-24-activate-dogfood-run-v0` change, or any other merged evidence.
- Do not restate the intent-level Context Pack semantics already promoted in `intent-snapshot` and `context-pack`; state only preparation-time enforcement.
- Do not touch the hook-linkage requirement, the open pull request that carries it, or the `dogfood-activation` and `operator-cli-help` capabilities.
- Planning completion does not authorize implementation, commit, push, pull request, merge, archival, provider launch, cleanup, or any external effect. Each remains a separate owner gate.
- Do not add a workflow engine, authorization system, effect gateway, credential broker, sandbox platform, queue, daemon, database, dashboard, or general activation service.
