## Why

WorkSpec preparation verified the confirmed intent's bytes and digest, but never validated the Context Pack that intent declares. A stale, foreign, or truncated pack could therefore back a frozen WorkSpec while every digest check still passed.

The promoted `intent-snapshot` and `context-pack` capabilities already require a flow Intent Snapshot to reference a valid Context Pack and require every derived item to carry resolvable context IDs. Preparation trusted that contract instead of enforcing it, and no promoted requirement stated the enforcement.

Review of the first attempt confirmed four further defects: preparation applied no schema-aware rule, so omitting both the declaration and the artifact bypassed verification entirely; the requirement described verification as occurring before freezing, which the implementation does not do; a bounded read could block indefinitely on a FIFO; and the change's own Context Pack packed two evidence references into single cells, which the accepted reference pattern rejects.

## What Changes

- Verify the confirmed intent against the Context Pack of its change during preparation, confining both artifacts to the canonical repository root under the same bounded size limit.
- Require the pack using the same schema-aware distinction the change loader applies, so a current change under the project intent schema cannot bypass verification by omitting both the declaration and the artifact.
- Reject a non-regular intent artifact or Context Pack before opening it, so a FIFO fails with a bounded reason instead of blocking preparation.
- Modify the promoted `local-run` requirement `Run preparation is inspectable and does not launch` to state this enforcement at the boundary that actually holds: before any prepared state is persisted, not before freezing.
- Keep the change provider-neutral: no Context Pack fields enter the canonical WorkSpec, and the check depends on no proposal, specification, design, task, provider, or activation artifact.
- Carry one valid evidence reference per Context Pack row so this change's own intent passes the verification it introduces.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Verify the confirmed intent together with the Context Pack of its change, both confined to the canonical repository root. | OUT-1, SIG-1 | NG-1, NG-3 |
| Fail before any prepared state is persisted on every invalid, unreadable, oversized, non-regular, or escaping artifact. | OUT-2, SIG-1, SIG-3 | NG-1, NG-5 |
| Apply the change loader's schema-aware requirement rule so the no-pack path cannot become a bypass. | OUT-3, SIG-2 | NG-1 |
| State the boundary that holds in the implementation rather than reordering preparation to fit the wording. | OUT-4, SIG-1 | NG-4 |
| Keep the change provider-neutral and free of canonical schema changes. | OUT-5, SIG-5 | NG-1, NG-3, NG-6 |
| Give the behaviour a spec home under a new intent identity and keep this change's own pack valid. | OUT-6, SIG-4 | NG-2 |
| Map every stated scenario onto a regression test. | OUT-1, OUT-2, OUT-3, SIG-6 | NG-1 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-run`: Preparation verifies the confirmed intent together with the Context Pack of its change, requires that pack exactly when the change loader would, rejects non-regular artifacts before opening them, and fails closed before any prepared state is persisted.

## Impact

- `internal/adapters/openspec/work_spec_intent.go` gains context resolution, the schema-aware requirement rule, and a regular-file guard applied to both bounded artifact reads.
- Canonical WorkSpec content, receipt format, `gr` command behaviour, JSON results, and error contracts remain unchanged.
- No merged file under `openspec/changes/archive/` is edited, and no existing intent ID or version is reused. Version 1 of this change's snapshot is retained unmodified.
- No new dependency, service, credential, deployment surface, or breaking change.
- The `dogfood-activation`, `operator-cli-help`, `intent-snapshot`, `context-pack`, and `work-spec` capabilities are untouched.

## Non-Goals

- Do not change the canonical WorkSpec or receipt schemas, `gr` command behaviour, JSON results, or error contracts.
- Do not reuse, re-version, amend, or edit `intent-activate-dogfood-run-v0`, any archived change, or any other merged evidence.
- Do not restate the intent-level Context Pack semantics already promoted in `intent-snapshot` and `context-pack`; state only preparation-time enforcement.
- Do not reorder preparation to make an aspirational boundary true; state the boundary that holds.
- Planning completion does not authorize merge, archival, spec promotion, deployment, provider launch, or any external effect. Each remains a separate owner gate.
- Do not add a workflow engine, authorization system, effect gateway, credential broker, sandbox platform, queue, daemon, database, dashboard, or general activation service.
