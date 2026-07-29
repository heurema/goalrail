## Context

Intent Snapshot `intent-escalation-discoverability-v0` version 1 is confirmed.

The blocked terminal outcome landed on `main` as `3f32dc3` and was promoted as
`68dbd5e`. It admits, retains, and scores an escalation — and nothing tells a run
the channel exists. The reserved path is a constant in Goalrail; the Codex
adapter passes the provider a working directory, the operator's verbatim
arguments, and the run-context environment value; the frozen task statement never
reaches the provider at all.

Two repository facts shape the answer:

- Goalrail already renders an invocation-local `SessionStart` hook that invokes
  the local capsule, and that hook is already trusted for lineage correlation
  (`internal/adapters/codex/hook_config.go`). A launch-time delivery point exists.
- The provider's published hook output contract defines, for `SessionStart`, an
  optional `additionalContext` string alongside `continue`, `stopReason`,
  `suppressOutput`, and `systemMessage`. Text can travel the path Goalrail
  already establishes.

And one measurement fact shapes the wording: the A/B that motivates the channel
recorded its treatment branch as loud — the channel occupied roughly a third of
the task statement, and a hint leaked through a version name — so its authors
flagged 6/6 as possibly inflated. Announcement volume is a variable.

## Goals / Non-Goals

**Goals:**

- Make the channel reachable by a run that was never told about it in its task.
- Reuse the provider surface already established rather than adding a second.
- Keep announced content provider-neutral and transport provider-specific.
- Make an undeliverable announcement fail the launch rather than degrade it.
- Keep the announcement minimal enough that a later measurement measures the
  environment rather than the hint.

**Non-Goals:**

- No task delivery: composing the work prompt stays with the operator.
- No canonical schema change, in either direction.
- No dialogue, acknowledgement, handshake, or retry.
- No change to `goalrail.escalation/v0` and no attempt to resolve its known
  incompatibility with the kata oracle's mechanical example check.
- No kata edits and no measurement.

## Decisions

### 1. Announce through the hook already rendered, not through a new surface

The alternatives were an environment variable, adapter arguments, or a file
written into the worktree.

An environment variable is already carried (`GOALRAIL_LOCAL_RUN_CONTEXT`) and is
read by the capsule, not by the agent; nothing puts it in the agent's context.
Adapter arguments are composed by the operator, not by Goalrail, so relying on
them would make the capability depend on operator prose — exactly the
unreliability this change exists to remove. Writing a file into the worktree
would violate the preparation boundary, would appear in the delta, and would put
Goalrail's own instructions inside the repository under evaluation.

The hook is already rendered, already invocation-local, and already carries a
provider-authoritative exchange. Its published output contract has a field for
session context. Reusing it adds no new trust surface.

### 2. Fixed provider-neutral text, provider-specific transport

The announced content names no provider, so a second adapter delivers the same
statement in its own contract's terms. This keeps the boundary where the promoted
requirements already put it: canonical state is provider-neutral, the adapter is
replaceable.

The text is fixed rather than composed per run. A per-run announcement would be a
place for task-specific hints to accumulate, and the hint is precisely the
confound the measurement warned about.

### 3. An adapter that cannot announce fails the launch

Silent degradation is the dangerous option: the run proceeds, the agent has no
channel, it guesses, and the receipt looks ordinary. That is the pre-change world
wearing a post-change receipt — worse than the pre-change world, because the
evidence now implies a capability that was not present.

Failing the launch is loud, recoverable, and cannot be mistaken for a result.

### 4. Announce once, as a statement

Anything else becomes a protocol. An acknowledgement implies a reply path; a
retry implies observing whether the first delivery worked, which implies reading
provider state mid-run. Both cross into the control plane the workspace contract
defers. One statement at launch is the whole mechanism.

### 5. Deliberately boring wording

The announcement says what the channel is and when it applies. It does not say
that this repository might contain a conflict, does not suggest checking whether
requirements agree, and does not describe the work.

This is a direct consequence of the recorded caveat. A loud announcement would
reproduce the loud treatment branch, and a later measurement would again be
unable to separate "the environment accepts a question" from "the task told the
agent to ask one". Keeping the announcement minimal is what makes the next
measurement mean something.

### 6. What the announcement is not allowed to become

It is not a system prompt, a policy statement, a style guide, or a place to put
future capabilities. Each addition would be another variable in the same
measurement. If a second capability needs announcing later, that is a separate
decision with its own evidence, not an extension of this string.

### 7. Corrections from automated review of the implementation

Four findings, all confirmed against the code and all fixed. One repeats a
pattern worth naming: a fail-closed check is only as honest as the thing it asks.

- **The adapter reported delivery it does not perform.** `VerifyAnnouncementDelivery`
  returned success unconditionally, while the value only reaches the session if a
  launcher installs a capsule that reads it and answers the hook. No production
  code in this repository does that — the renderer was called only by tests. The
  check passed and the run could launch with no announcement, which defeats the
  entire mechanism. The report now comes from the launcher, which is the
  component that knows whether the path exists.
- **The announcement would have repeated inside a run.** The provider re-signals
  session start on resumption, clearing, and compaction — all four sources are
  recognised in the correlation path — and the rendered matcher does not
  distinguish them. The renderer now emits the announcement only for the source
  that opens a run, so one statement cannot become a recurring instruction.
- **An undeliverable announcement burned the one-time authorization.** The check
  ran after admission was consumed, so a failed launch discarded the owner's
  grant with nothing to show for it and no retry could reuse it. Verification now
  precedes every irreversible step of the launch.
- **The Context Pack cited evidence that does not resolve.** Two items pointed at
  paths in a separate repository, and a third attributed the one-shot and
  control-plane constraints to `AGENTS.md`, where they do not appear — they live
  in the promoted `local-run` capability. The measurement claims are now retained
  verbatim beside the pack so it can be audited without leaving the repository,
  and the misattributed item points at its real source.

The confirmed intent was left unmodified. These corrections make the shipped
behaviour honest against outcomes version 1 already states; none of them changes
what the owner confirmed, so none of them justifies editing a confirmed snapshot
in place.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-escalation-discoverability-v0`
  version 1 and Context Pack `context-escalation-discoverability-v0` version 1.
- **Specification evidence:** one `ADDED` and one `MODIFIED` requirement for
  `local-run`.
- **Implementation evidence:** each scenario maps to a deterministic test. The
  announcement text is pinned by a test so it cannot drift into a hint. The
  undeliverable-announcement path is proven with a fixture adapter that has no
  delivery mechanism.
- **Ownership:** `local-run` owns the announcement and the adapter obligation.
  `work-spec` is untouched. `escalation-question` continues to own the payload
  shape.
- **Failure handling:** an adapter that cannot deliver fails before the provider
  is launched, leaving no run ID, launch claim, or receipt.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation reports this change and every promoted spec
  as valid.
- The frozen `goalrail.work-spec/v0` fixture digest is unchanged and the terminal
  receipt schema is unchanged.
- A test pins the announcement text and asserts it contains no provider name and
  nothing about conflicts, documents, or the work item.
- A fixture adapter without a delivery path fails the launch, proven by test.
- Stop and reshape if delivering the announcement would require a WorkSpec field,
  a receipt field, a write into the target worktree, or any reply path.

## Risks / Trade-offs

- **[The provider may not surface `additionalContext` as expected]** → Taken from
  the provider's published hook schema, not observed end to end. This is the
  same class of open question as sandbox writability of the reserved path, and
  both are closed by the same eventual real run. Recorded rather than hidden.
- **[A fixed string becomes a dumping ground]** → Explicitly ruled out in the
  design and pinned by a test; a second capability needs its own decision.
- **[Failing the launch on an undeliverable announcement is strict]** → It is,
  and deliberately: the alternative produces receipts that overstate what the
  run could do.
- **[The announcement is still a hint, just a quieter one]** → True, and
  unavoidable: a channel must be knowable to be usable. What the design can
  control is that the hint is about the channel and never about the work. A
  future measurement should treat announcement wording as a declared variable
  rather than a constant.
- **[Reusing the lineage hook couples two purposes]** → Accepted. They share a
  lifetime and a trust boundary, and separating them would mean a second
  provider surface for no gain.

## Rollback

Delete the change directory. Once implemented, rollback is reverting the
implementation commits: no schema, stored evidence, or promoted spec changes
until a separate archival step promotes the delta. Runs already launched with an
announcement remain valid evidence and are not rewritten.

## Open Questions

None.
