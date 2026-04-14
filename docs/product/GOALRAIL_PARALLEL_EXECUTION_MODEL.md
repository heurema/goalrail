# Goalrail Parallel Execution Model

> Canonical model for safe parallel execution.

## 1. Purpose

Goalrail must support parallel work in two distinct ways:
1. different tasks may execute in parallel
2. the same task may be reviewed or explored by multiple runtimes in parallel

These are different systems with different trust boundaries.
Parallel work must not turn into hidden concurrent mutation of one workspace.

## 2. Two parallel systems

### ExecutionGroup
A bounded set of different tasks that may start together under one scheduling decision.
Each task still produces its own Run, Receipt, Decision, and Proof.

### AdvisoryPanel
A bounded multi-runtime process over one task, one question, or one frozen bundle.
It may run one of several advisory protocols such as `panel`, `quorum`, `verify`, or `diverge`.
An AdvisoryPanel never becomes the authoritative writer of the final decision.

## 3. Core rules

1. Parallelism is explicit and plan-driven, not ad hoc.
2. Each writable run gets its own isolated workspace.
3. One writable run uses one primary writer runtime.
4. Disjoint tasks may run in parallel.
5. Overlapping or uncertain writable scope must not run concurrently in the same mutable workspace.
6. Every multi-run Execution Group must end with a fan-in barrier before downstream verification or release.
7. AdvisoryPanel outputs are advisory only; gate remains authoritative.
8. Risk and policy select whether a task gets zero, one, or many advisory lanes.
9. Sensitive policy may prohibit multi-vendor fan-out even for high-risk work.

## 4. Canonical objects

### ExecutionGroup
```json
{
  "id": "grp_123",
  "contract_id": "ctr_10",
  "task_ids": ["tsk_1", "tsk_2"],
  "strategy": "parallel",
  "status": "planned"
}
```

### IsolationDecision
```json
{
  "group_id": "grp_123",
  "task_id": "tsk_1",
  "mode": "worktree|docker|serialized",
  "reason": "disjoint_scope"
}
```

### BarrierRecord
```json
{
  "group_id": "grp_123",
  "status": "ready_for_gate",
  "run_ids": ["run_1", "run_2"],
  "summary_ref": "art_group_summary_9"
}
```

### AdvisoryPanel
```json
{
  "id": "panel_41",
  "task_id": "tsk_7",
  "mode": "panel|quorum|verify|diverge",
  "runtime_ids": ["codex", "claude_code", "gemini"],
  "status": "running",
  "exposure_policy": "diff_only"
}
```

### ConsensusRecord
```json
{
  "panel_id": "panel_41",
  "status": "split",
  "result": "escalate",
  "summary_ref": "art_consensus_12"
}
```

## 5. Task scheduling taxonomy

### TaskClass
- `blocking`
- `non_blocking`
- `review_only`
- `serialized`

### DependencyType
- `hard_block`
- `soft_block`
- `fan_in_required`

Scheduler inputs must consider:
- dependency edges
- predicted writable scope
- overlap confidence
- touched roots or files
- shared resources
- risk level
- policy restrictions
- runtime availability

## 6. Isolation modes

### worktree
Use when:
- writable scope is disjoint
- merge semantics are simple
- the fast local path is safe

### docker
Use when:
- scope overlap exists
- scope confidence is low
- sandboxing must be stronger
- tools or runtime isolation is required

### serialized
Use when:
- concurrent mutation would be unsafe
- one task must follow another

## 7. Scheduling decision

An Execution Group may run in parallel only when all conditions hold:
1. tasks are independently gateable
2. writable scopes are disjoint or isolated strongly enough
3. merge and fan-in semantics are explicit
4. blocking dependencies are satisfied
5. policy does not require serialization

## 8. Advisory protocols

### `panel`
Use when:
- the task needs parallel opinions
- architecture tradeoffs or failure modes should be compared

### `quorum`
Use when:
- the task needs an explicit `approve|block|escalate` style advisory outcome
- risk is high enough to justify structured voting

### `verify`
Use when:
- important claims must be challenged across runtimes
- migration safety, compatibility, or security statements need adversarial checking

### `diverge`
Use when:
- the solution space is broad enough to justify multiple isolated implementations or design options
- the task benefits from comparative evaluation before committing to one path

Rules:
- advisory protocols operate on bounded questions, diffs, or frozen bundles
- comparative implementations must remain isolated from the primary writable run until selected intentionally
- advisory protocols may feed Gate but may not bypass it

## 9. Fan-in and synthesis

### ExecutionGroup barrier
Barrier responsibilities:
- collect run statuses
- ensure all run artifacts are present
- ensure no unsafe overlap escaped the isolation model
- expose one inspectable group summary
- allow downstream verification to proceed in a controlled way

### AdvisoryPanel synthesis
Panel responsibilities:
- collect all advisory outputs
- normalize consensus and split points
- persist one inspectable panel summary
- expose whether the result supports accept, block, or escalate as advisory input only

Gate consumes both when present:
- group summary from ExecutionGroup barriers
- panel summary or ConsensusRecord from AdvisoryPanels

## 10. Non-goals

This model does not imply:
- concurrent writes by two runtimes into one mutable workspace
- uncontrolled “swarm” execution
- replacing per-run gate and proof with one opaque group verdict
- allowing advisory panel consensus to overwrite policy or scope failures

## 11. MVP rollout

Recommended order:
1. single-run local runtime
2. worktree-based parallel execution for clearly disjoint tasks
3. one advisory review lane on a frozen bundle
4. risk-based multi-runtime advisory panels
5. `diverge`-style comparative exploration later
