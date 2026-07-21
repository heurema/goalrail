## Context

Goalrail currently has no runtime implementation. The repository contains its service hypothesis, the new OpenSpec scaffolding, and an existing repository-local Langfuse tag configuration. The first useful slice must therefore establish a workflow contract that the team can use while building Goalrail and make its value falsifiable before larger architecture is introduced.

The critical failure mode is upstream: an agent can perform valid implementation and checks against the wrong interpretation. The owner remains the decision maker for intent and final outcome assessment. OpenSpec can order development artifacts, and Langfuse can supply execution observations, but neither owns Goalrail's canonical product semantics.

Current verified integration facts are deliberately narrow:

- Codex lifecycle hooks receive the current `session_id`; subagent hooks use the parent session ID.
- The installed Langfuse Codex plugin already groups observations by Codex session and records the Codex thread identity.
- CLI launches can receive per-run Langfuse metadata through supported environment configuration.
- The live probe verified CLI lifecycle-hook binding. The tested App background-task path did not execute the temporary project hook, while the App launch API returned an immutable `threadId` in its creation receipt.
- OpenSpec 1.6.0 validates the project-local schema but its current `new change` path still resolves the repository default as `spec-driven`; Goalrail must pass `--schema goalrail-intent` explicitly until that behavior changes.

## Goals / Non-Goals

**Goals:**

- Make a versioned, owner-confirmed Intent Snapshot the required predecessor of a proposal.
- Define a minimal provider-neutral record for intent lineage, execution lineage, owner assessment, and canary metrics.
- Prove automatic `change_id → run_id → session_id` binding in a bounded adapter probe before relying on it.
- Prepare a 15-change real-work canary whose assignment, metrics, pass signals, and stop signals are frozen before it starts.
- Keep the whole slice removable without changing future runtime architecture.

**Non-Goals:**

- Implement the Goalrail control plane or a durable workflow engine.
- Select Temporal/Restate, OpenFGA/SpiceDB, or a final sandbox runtime.
- Fork the Langfuse plugin, build a dashboard, add retention automation, or split Langfuse projects.
- Capture raw prompts, transcripts, credentials, or code in the repository evidence record.
- Introduce shadow mutations or claim causal proof from the small canary.

## Decisions

### 1. Use an intent-first OpenSpec schema as a replaceable development adapter

The project-local `goalrail-intent` schema inserts `intent.md` before proposal, specs, design, and tasks. The proposal compiler may consume only a confirmed intent version and must preserve intent IDs. OpenSpec files express this development workflow; canonical intent fields and lifecycle states remain provider-neutral so a future compiler can replace OpenSpec.

Alternatives considered:

- Keep intent only in Explorer conversation — rejected because it leaves no versioned owner-confirmed artifact and cannot support reliable traceability or measurement.
- Build a Goalrail orchestration service first — rejected because the canary can test the semantic contract without a durable engine or deployed service.

### 2. Use three semantic intent groups and explicit lifecycle metadata

Desired outcomes, non-goals, and observable success signals are the only semantic groups. Evidence, ambiguity, confirmation, amendments, and run references are lifecycle/provenance fields. The lifecycle is `candidate → confirmed`; a material amendment creates a new candidate version linked to its predecessor. Proposal generation is blocked until active owner confirmation is recorded.

This keeps the first artifact small enough to review while preserving the distinction between what the owner said, what the agent inferred, and what the owner accepted.

### 3. Require automatic lineage from provider-authoritative identity

The adapter combines immutable change/run context with provider-authoritative root session identity. CLI runs use the `session_id` supplied by Codex lifecycle hooks. Goalrail-launched App tasks use the immutable `threadId` returned in the `create_thread` response. Shared mutable files such as a repository-wide `current-change` pointer are forbidden because concurrent tasks could produce wrong joins.

The adapter has three acceptable outcomes:

1. deterministic automatic binding works through a lifecycle hook or launch receipt;
2. the record is explicitly `UNLINKED` and one bounded resolution attempt succeeds without guessing; or
3. the slice stops at an owner gate because the surface cannot supply an unambiguous binding.

Human copying of session IDs and heuristic transcript parsing are not fallback paths. Existing or manually created App tasks without a Goalrail launch receipt remain `UNLINKED` and are ineligible for the v0 canary. The existing Langfuse plugin remains unchanged; Goalrail joins to its traces using the verified session identity when available.

The live probe produced a verified CLI join and no App project-hook receipt. At the required owner gate, the owner selected the provider-authoritative App launch receipt on 2026-07-21.

Alternatives considered:

- Limit v0 to CLI-only execution — rejected because Goalrail can preserve the same automatic identity guarantee for tasks it launches through the App API.
- Continue probing App hook trust or parse task content — rejected for v0 because the creation response already supplies authoritative identity, while parsing content would weaken the contract.

Rollback is to make App tasks ineligible and retain the verified CLI hook path. Revisit this decision if the App provider stops returning immutable task identity, a returned `threadId` differs from the root identity observed by Langfuse, or any wrong join occurs.

### 4. Keep one minimal repository evidence record and use Langfuse as an observation backend

The change directory will own a small, structured, append-only canary manifest/event record containing only stable IDs, timestamps, categorical outcomes, numeric measurements, source/check references, trace references, and correction links. It is the durable source for canary decisions. Langfuse supplies detailed execution observations referenced by session or trace identity; it is not the sole source of truth and no second custom database or service is added.

The existing shared Langfuse project and `project:goalrail` tag remain. Project splitting, retention automation, and sensitive/external workloads trigger a later privacy decision rather than being solved in this slice.

### 5. Freeze a small directional comparison before the first assignment

The canary manifest freezes eligibility, the `flow → flow → baseline` rotation, check recording, assessment timing, overhead calculation, and rate formulas before ordinal 1. Baseline changes use the team's ordinary workflow; flow changes add the Intent Snapshot and proposal gate. Both variants receive the same stable identity, lineage capture, verification recording, and terminal owner assessment so they can be compared.

The canary reports directional rates only. Fifteen changes are sufficient to expose workflow failures and gross cost, not to establish general causality.

### 6. Keep owner judgment explicit and checks orthogonal

The owner records `match`, `partial`, or `miss` after delivery. Recorded checks independently determine `green`; therefore `wrong-but-green` is a first-class observable outcome. Material correction is recorded independently of final outcome and counts only when it changes the meaning of an outcome, non-goal, or success signal and changes work before delivery.

No model may infer the owner's outcome rating or silently convert missing assessment into `match`.

### Retained target architecture constraints

This slice neither implements nor removes the following target layers:

- Go kernel/control plane.
- Temporal as reference durable runtime and Restate as bounded challenger; running workflows need not migrate across engines.
- OpenFGA as reference ReBAC adapter and SpiceDB as challenger, plus server-side task grants and child-permission subset rules.
- Effect enforcement through a Tool/Effect Gateway, sandbox launcher, and credential broker; user credentials are not passed into sandboxes.
- Rootless OCI followed by gVisor locally, with Agent Sandbox plus Kata considered later for stronger multi-host or multi-tenant isolation; an ordinary container is not a sufficient untrusted-code boundary.
- Coarse-grained versioned out-of-process Protobuf/gRPC module contracts.
- Vendor-free canonical domain, run-pinned WorkSpec snapshots, replaceable OpenSpec compiler/provider, and future evidence, repair, collaboration, permission, and PR traceability layers.

## Correlation and Evidence

The minimum v0 lineage is:

```text
Intent Snapshot version
        ↓
OpenSpec change_id → run_id → root Codex session_id → Langfuse trace references
        ↓
recorded source/check ref → terminal owner assessment → canary report
```

Identity rules:

- `change_id` is stable from assignment through assessment.
- `run_id` identifies the one v0 execution run for that change.
- The root Codex `session_id` is captured from a lifecycle-hook input or a provider-authoritative launch receipt, never typed by the owner.
- Resume and compaction preserve the root session association; child agents inherit it.
- A join is valid only when all three identities agree with immutable run context.
- Missing or conflicting context emits an append-only `UNLINKED` event. At most one bounded resolution attempt is allowed; a wrong join is a hard stop.

The implementation probe must test startup, resume, compaction where exposed, child-agent inheritance, absent context, conflicting context, and two concurrent changes. It must not depend on the transcript format because Codex documents that format as unstable.

## Measurement and Stop Conditions

Before ordinal 1, a versioned canary manifest will define:

- what qualifies as a real eligible change;
- the fixed 15-position variant assignment;
- the check set selection and source-reference rule;
- how flow-only agent time and owner-review time form `flow_overhead_minutes` using available trace/event timestamps;
- when the owner records outcome and `repeat_optin`;
- formulas and missing-data treatment.

For each variant, `non_match_rate = (partial + miss) / delivered assessed changes`. Abandonments are reported separately and never dropped. The report compares rates rather than the unequal raw counts from 10 flow and 5 baseline assignments.

Pass requires all confirmed thresholds: at least 13 verified lineage joins with zero wrong joins, lower directional flow non-match rate, at least one contemporaneously prevented material misunderstanding, median flow overhead at most 30 minutes, and `repeat_optin=yes` for at least two-thirds of flow changes.

Hard stop before completion occurs on any wrong join, retroactive evidence rewrite, median flow overhead above 30 minutes, at least 3 process-caused flow abandonments, or at least 2 unresolved links after bounded resolution. At completion, no prevention plus no lower non-match rate plus higher cost yields `RESHAPE`. These verdicts apply to this flow version, not to the broader importance of intent.

## Risks / Trade-offs

- **Small, uneven sample can be noisy** → Report directional rates and raw evidence; make no statistical or causal claim.
- **Owner knows the variant and may assess differently** → Freeze definitions and assessment timing before the first change; preserve evidence for review.
- **Instrumentation may become the product** → Limit v0 to one adapter, one structured record, deterministic reporting, and existing Langfuse traces; no dashboard or service.
- **Codex App tasks created outside Goalrail lack immutable run context** → Treat them as `UNLINKED` and ineligible for v0 instead of parsing task text or accepting manual joins.
- **OpenSpec custom schemas are currently experimental upstream** → Pin OpenSpec 1.6.0 for this change, keep the schema in-repository, and keep canonical semantics independent of its file format.
- **OpenSpec 1.6.0 does not honor the configured custom schema as the `new change` default** → Put the explicit schema command in project instructions and verify the schema recorded in every new `.openspec.yaml` file.
- **Shared Langfuse retention may later be inappropriate** → Keep the canary internal and non-sensitive; revisit before client, external, production, or sensitive work.
- **Process overhead could hide any quality benefit** → Enforce the 30-minute median threshold and collect repeat opt-in rather than expanding the workflow.

## Activation Sequence

1. Validate the provider-neutral record and deterministic correlation path with synthetic local fixtures.
2. Run the correlation adapter against one disposable local Codex session; do not start the canary if the join is ambiguous.
3. Freeze the canary manifest and measurement formula.
4. Start the 15-change sequence and stop automatically on any hard stop condition.
5. Produce the deterministic report for owner review; do not roll the result into broader architecture automatically.

## Rollback

Before the canary starts, rollback is deletion or disabling of the project-local adapter/hook and canary files. During the canary, stop new assignments, preserve existing append-only evidence, and mark the version `STOP`; do not rewrite or delete observations. OpenSpec artifacts remain readable planning records and do not constrain the future runtime.

## Resolved Questions

- The timestamp-derived overhead formula is frozen in `canary/manifest-v1.md`: sum the envelope duration of distinct flow-only root `Codex Turn` traces, add the explicit owner-review interval, and divide by 60 without rounding. Nested observations are not summed, and missing required evidence is never imputed. The manifest remains `frozen_not_activated`; starting the 15-change canary still requires a separate owner instruction.
