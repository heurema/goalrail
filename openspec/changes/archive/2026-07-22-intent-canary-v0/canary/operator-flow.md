# Intent Canary v0 Operator Flow

This is the complete local operator surface for one canary change. It is a
command plus an append-only JSONL record, not a daemon, dashboard, or workflow
engine.

The real 15-change canary is **not activated**. `start` currently rejects every
real assignment and accepts only an explicit `--synthetic` assignment. Removing
that guard, installing a persistent hook, or starting ordinal 1 with real work
requires a separate owner instruction.

## Build

Run from the repository root:

```sh
go build -o /tmp/goalrail-canary ./cmd/goalrail-canary
```

The default evidence path is:

```text
openspec/changes/intent-canary-v0/canary/events.jsonl
```

Global `--repo` and `--store` flags, when needed, go before the command. Values
stored as actors, reasons, sources, and checks are bounded metadata tokens or
`scheme:identifier` references: no whitespace, prompt text, source content, or
credentials.

## Automatic identity boundary

The operator supplies `change_id`; Goalrail generates `run_id` and every event
ID. There is no command flag for a Codex session ID.

For a CLI run, launch the process through `start`:

```sh
/tmp/goalrail-canary start \
  --change change-synthetic-001 \
  --intent-version 1 \
  --actor operator \
  --source request:synthetic-001 \
  --synthetic \
  -- codex
```

`start` gives the child process an immutable, process-scoped
`GOALRAIL_RUN_CONTEXT`, plus the exact repository and evidence paths. The Codex
lifecycle-hook integration invokes this target with the provider hook JSON on
standard input:

```sh
/tmp/goalrail-canary bind-hook
```

The target reads `session_id` only from that provider hook input, writes a
sanitized lineage event, and emits nothing to model-visible standard output.
The environment is per launch, so concurrent changes do not share a mutable
`current-change` file. A missing or conflicting receipt is appended as
`UNLINKED`; it is never guessed from task text or a transcript.

For a Goalrail-launched Codex App task, the launcher inherits the same run
context, calls the provider, and pipes the exact `create_thread` response to:

```sh
/tmp/goalrail-canary bind-launch
```

`bind-launch` accepts the root identity only from the response `threadId`.
There is no manual session fallback. A manually created App task therefore
remains ineligible for v0.

When no launcher follows `--`, `start` returns a machine-readable receipt for a
Goalrail launcher integration. That receipt is not a request for an operator to
copy IDs by hand.

## Inspect

```sh
/tmp/goalrail-canary inspect --change change-synthetic-001
```

The JSON projection includes immutable assignment, latest lineage, terminal
state, latest owner assessment, correction count, and event count. The JSONL
record remains authoritative; `inspect` does not mutate it.

The aggregate report is derived from that same verified event chain:

```sh
/tmp/goalrail-canary report
```

Before 15 terminal assignments its normal verdict is `PENDING`. Missing owner
assessment, lineage, or flow overhead remains missing; the command does not
impute it. A digest-chain integrity failure stops report generation.

## Disable and rollback

Stop new assignments for this manifest version with an explicit owner action:

```sh
/tmp/goalrail-canary disable \
  --actor owner \
  --source review:canary-stop-v1 \
  --reason owner-stop
```

`disable` appends one canary-level `canary_stopped` event to the same verified
digest chain. It never deletes or rewrites prior events. The report then exposes
`assignments_stopped=true` and verdict `STOP`; every later `start` fails closed.
A second `disable` also fails instead of hiding the original stop receipt.

Already assigned changes remain inspectable and may still append their lineage,
terminal, assessment, or correction evidence. The stop marker is scoped to the
selected `--store`: it does not install or remove Codex hooks, mutate process
environment, change `.codex/langfuse.json`, touch credentials, or disable
unrelated Codex and Langfuse use.

## Record a material correction before delivery

Only an owner-confirmed meaning change that changes the work is material:

```sh
/tmp/goalrail-canary correct \
  --kind material \
  --change change-synthetic-001 \
  --owner owner \
  --source review:intent-correction-001 \
  --reason non-goal-misunderstood
```

A typo or wording-only edit emits no material-correction event. Material
corrections are rejected after delivery or abandonment.

## Deliver

Record every selected check reference. Add `--green` only when the entire
selected set passed and none is missing:

```sh
/tmp/goalrail-canary deliver \
  --change change-synthetic-001 \
  --actor operator \
  --source review:handoff-001 \
  --check-ref check:go-test \
  --green \
  --overhead-minutes 12.5
```

`--overhead-minutes` is valid only for a `flow` assignment and must be the
manifest-v1 timestamp-derived measurement. Baseline delivery omits it. Delivery
means a reviewable handoff; it does not imply merge or deploy.

## Abandon

An undelivered change terminates explicitly and keeps its ordinal:

```sh
/tmp/goalrail-canary abandon \
  --change change-synthetic-001 \
  --actor operator \
  --source review:abandonment-001 \
  --reason flow-too-costly \
  --process-caused \
  --overhead-minutes 8.0
```

Flow-only flags are rejected for baseline. An abandoned change receives no
intent-match assessment.

## Owner assessment

Only after delivery, the owner records intent outcome explicitly:

```sh
/tmp/goalrail-canary assess \
  --change change-synthetic-001 \
  --owner owner \
  --source review:owner-assessment-001 \
  --outcome partial \
  --repeat-opt-in yes
```

`flow` requires `--repeat-opt-in yes|no`; baseline rejects that field. The
command derives `green` from the immutable terminal check record and derives
the material-prevention flag from prior material-correction events. It never
infers `match`, `partial`, `miss`, or repeat opt-in for the owner.

## Correct an owner assessment

No event ID is copied. The command automatically supersedes the latest
assessment and preserves every earlier event:

```sh
/tmp/goalrail-canary correct \
  --kind assessment \
  --change change-synthetic-001 \
  --owner owner \
  --source review:assessment-correction-001 \
  --reason owner-outcome-correction \
  --outcome match \
  --repeat-opt-in no
```

Every command validates the complete digest chain before appending. Invalid
transitions, duplicate ordinals, changed run identity, changed verified root
identity, assessment/check disagreement, and correction of a non-latest
assessment fail closed.
