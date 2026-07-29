## Why

The local-run contract has no way for a run to end because the work item cannot
be completed as specified. Every terminal status either claims an implementation
attempt or reports a verification gap, so an agent that discovers a conflict in
its own WorkSpec has no machine-accepted way to say so.

Measurement shows this is not a prompting problem. On a conflicting-requirements
kata, 6/6 agents returned a structured question when the environment accepted one
machine-checkably, and 0/6 asked when it did not; every agent in the second group
guessed, and the strongest documented the conflict in prose and shipped anyway.
The channel that produced the good behaviour was built inside the kata, which
made the kata solve itself and proved nothing about the harness.

Two facts confine how the outcome can be represented. Receipt validation
whitelists exactly five terminal statuses and the receipt carries no schema
identifier, so any new status breaks existing readers however it is introduced.
`DecodeWorkSpec` rejects unknown fields and the canonical WorkSpec is
digest-bound, so a new WorkSpec field would fork `goalrail.work-spec/v0` and put
frozen digests at risk.

## What Changes

- Add the terminal run state `blocked`: the provider run ended without an
  implementation because the work item cannot be completed as specified and an
  owner decision is required. Blocked is terminal; answering never resumes the
  run.
- Reserve one constant repository-relative escalation path instead of adding a
  WorkSpec field, leaving `goalrail.work-spec/v0` byte-identical.
- Fail `gr prepare` closed when that path is already populated in the frozen
  baseline observation, so a stale or foreign question cannot attach itself to a
  new run.
- Capture the artifact eagerly at provider observation in `Start`, mirroring the
  existing `failureReceipt` path, and retain its exact bytes append-only in the
  run store. The artifact stays inside the observed worktree delta and is
  excluded only from scope violations.
- Validate the artifact for hygiene only — regular file, non-empty, bounded
  size, valid UTF-8, no unsupported control characters, no secret-shaped content
  — and otherwise treat it as opaque bytes.
- Define the state rules: `blocked` requires a valid artifact and an empty
  in-scope delta; artifact plus in-scope edits yields `failed`; `unlinked`,
  `denied`, and `launch_failed` take precedence; supplied check results are
  recorded verbatim; no input combination yields `passed` while the artifact
  exists.
- Version the terminal receipt explicitly as `goalrail.terminal-receipt/v1` with
  a schema field, an escalation block, and the frozen intent reference, keeping
  existing schema-less receipts readable as v0.
- Let an answering intent version record the escalation it resolves and a
  disposition, so the question-to-decision chain is followable from recorded
  identifiers alone.
- Publish the escalation payload shape as a small provider-neutral capability
  that downstream tooling validates.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Add `blocked` as a terminal state with no resume path. | OUT-1, SIG-2 | NG-1, NG-5 |
| Reserve a constant escalation path and leave the WorkSpec schema untouched. | OUT-2, SIG-4 | NG-2 |
| Fail preparation closed when the reserved path is already populated. | OUT-2, OUT-3, SIG-3 | NG-5 |
| Retain the artifact eagerly and append-only, inside the observed delta. | OUT-2, SIG-1, SIG-3 | NG-5 |
| Validate hygiene only and treat the payload as opaque bytes. | OUT-2, OUT-6 | NG-4 |
| Define the hedge rule, the precedence rules, and verbatim check results. | OUT-3, SIG-2, SIG-3 | NG-5 |
| Version the terminal receipt and keep existing receipts readable. | OUT-4, SIG-4 | NG-1 |
| Record the resolved escalation and disposition on the answering intent version. | OUT-5, SIG-5 | NG-3, NG-6 |
| Publish the escalation payload shape as a downstream-validated capability. | OUT-6, SIG-1 | NG-4 |

## Capabilities

### New Capabilities

- `escalation-question`: the provider-neutral shape of the escalation payload,
  validated outside Goalrail by downstream tooling.

### Modified Capabilities

- `local-run`: preparation rejects a pre-populated reserved escalation path; the
  lifecycle gains the terminal `blocked` outcome with its retention, state, and
  precedence rules; the terminal receipt carries an explicit schema version, an
  escalation block, and the frozen intent reference.
- `intent-snapshot`: an answering intent version may record the escalation it
  resolves together with a disposition, without gaining effect authority.

## Impact

- `internal/localrun`: a preparation gate, eager artifact capture at provider
  observation, the `blocked` state and its precedence rules, receipt schema
  versioning, and the escalation block on the receipt.
- `internal/domain`: a bounded hygiene check for the retained artifact, reusing
  the existing retained-text rules rather than introducing a second family.
- `internal/adapters/openspec`: parsing of the optional `resolves` and
  `disposition` records on an intent version.
- `goalrail.work-spec/v0` is unchanged and is protected by a frozen fixture
  digest regression test. Command surface, JSON result grammar, and the
  `--result` grammar are unchanged.
- Existing receipts without a schema field remain readable as v0. Readers that
  whitelist terminal statuses must accept the new status; this is the one
  intended break and is the reason the schema version is introduced together
  with it.
- No new dependency, service, credential, deployment surface, daemon, or
  external effect.

## Non-Goals

- No mid-run dialogue, control plane, session resume, or background
  continuation. The loop closes only through a new intent version and a new run.
- No change to `goalrail.work-spec/v0`: no new WorkSpec field and no digest
  impact on frozen WorkSpecs.
- No recurrence registry and no claim that an answered conflict cannot recur.
  The delivered property is a machine-followable chain, not prevention.
- No semantic validation of the question inside Goalrail: no source parsing, no
  example execution, no citation grounding in this slice.
- Blocked is never success-adjacent: it cannot be recorded as `passed`, cannot
  suppress supplied check results, and is not a fourth per-check result state.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, spec promotion, provider run, or external effect. Each
  remains a separate owner gate.
