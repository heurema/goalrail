# Intent Canary v0 Operator Flow — Manifest v2

Manifest v2 is frozen but **not activated**. Every command below includes
`--manifest-version 2`, writes only to the separate v2 evidence chain, and still
requires `--synthetic`. These commands are for deterministic rehearsal; they do
not authorize a real assignment or an external Langfuse read.

## Build and evidence path

```sh
go build -o /tmp/goalrail-canary ./cmd/goalrail-canary
```

The default v2 path is:

```text
canary/intent-canary-v0/events-v2.jsonl
```

Use `--repo` or `--store` before the command only when the defaults are not the
intended repository and evidence chain.

## Start and bind verified lineage

```sh
/tmp/goalrail-canary --manifest-version 2 start \
  --change change-synthetic-v2-001 \
  --intent-version 1 \
  --actor operator \
  --source request:synthetic-v2-001 \
  --reason eligibility-confirmed \
  --synthetic \
  -- codex
```

The launched process receives `GOALRAIL_RUN_CONTEXT`. A lifecycle integration
pipes the provider hook JSON to `bind-hook`; no command accepts a manually typed
session ID:

```sh
/tmp/goalrail-canary --manifest-version 2 bind-hook
```

## Flow-only context, basis, and phase

After the Context Pack is sufficient and the owner has confirmed intent, bind
their stable identities without copying wording into runtime evidence:

```sh
/tmp/goalrail-canary --manifest-version 2 bind-context \
  --change change-synthetic-v2-001 \
  --actor operator \
  --source openspec:change-synthetic-v2-001 \
  --context-pack context-synthetic-v2-001 \
  --context-version 1

/tmp/goalrail-canary --manifest-version 2 record-basis \
  --change change-synthetic-v2-001 \
  --actor owner \
  --source owner-review:basis-synthetic-v2-001 \
  --intent-ref openspec:change-synthetic-v2-001 \
  --intent-id intent-synthetic-v2-001 \
  --intent-version 1 \
  --timing pre_execution \
  --desired-outcome-id OUT-1 \
  --non-goal-id NG-1 \
  --success-signal-id SIG-1
```

Record the explicit flow window after it completes and before implementation or
check selection. Times are UTC RFC3339 values captured at the actual boundary:

```sh
/tmp/goalrail-canary --manifest-version 2 record-phase \
  --change change-synthetic-v2-001 \
  --actor operator \
  --source review:flow-phase-synthetic-v2-001 \
  --started-at 2026-07-22T12:00:00Z \
  --completed-at 2026-07-22T12:05:00Z
```

Baseline omits Context Pack and phase. After baseline delivery, record the basis
from the original request with `--timing post_delivery`; this visible timing is
intentional and prevents a false claim of symmetric pre-execution handling.

## One bounded Langfuse reconciliation

Credentials are read only from process environment and are never accepted as
flags or written to JSONL:

```sh
export LANGFUSE_BASE_URL=https://cloud.langfuse.com
export LANGFUSE_PUBLIC_KEY=...
export LANGFUSE_SECRET_KEY=...
```

Do not put actual values in repository files, shell examples, reviews, or chat.
Use the existing trusted local secret setup. `LANGFUSE_HOST` is accepted when
`LANGFUSE_BASE_URL` is unset.

For flow, provide the separately observed owner-review interval:

```sh
/tmp/goalrail-canary --manifest-version 2 reconcile \
  --change change-synthetic-v2-001 \
  --actor operator \
  --source langfuse-api:observations-v2 \
  --owner-review-ref review:owner-review-synthetic-v2-001 \
  --owner-review-start 2026-07-22T12:05:00Z \
  --owner-review-end 2026-07-22T12:06:00Z
```

Baseline uses the same command without owner-review flags. The adapter performs
one read using exact verified `session_id`, explicit time bounds, at most 100
rows per page and 10 pages, a 15-second HTTP timeout, strict decoding, and no
retry. An explicit later run may use `--correction-reason REASON`; it appends a
correction and never rewrites the earlier observation.

No rows, read failure, mixed flow/implementation trace, malformed response, or
session conflict remains `unavailable` or `conflict`; it never becomes zero.
Delivery and owner assessment can still be recorded, but report completion
readiness remains false when required flow overhead is unavailable.

## Checks, delivery, and structured owner assessment

Freeze checks as in v1, then deliver without a manual overhead value:

```sh
/tmp/goalrail-canary --manifest-version 2 freeze-checks \
  --change change-synthetic-v2-001 \
  --actor operator \
  --source review:checks-synthetic-v2-001 \
  --check-ref test:go-test

/tmp/goalrail-canary --manifest-version 2 deliver \
  --change change-synthetic-v2-001 \
  --actor operator \
  --source review:delivery-synthetic-v2-001 \
  --check-ref test:go-test \
  --green
```

Manifest v2 rejects `--overhead-minutes`; a flow total is derived from recorded
trace envelopes plus owner review, while baseline never receives a flow total.

The owner judges every frozen basis item and supplies the consistent aggregate:

```sh
/tmp/goalrail-canary --manifest-version 2 assess \
  --change change-synthetic-v2-001 \
  --owner owner \
  --source owner-review:assessment-synthetic-v2-001 \
  --outcome match \
  --repeat-opt-in yes \
  --desired-outcome OUT-1=achieved \
  --non-goal NG-1=preserved \
  --success-signal SIG-1=observed
```

Valid values are `achieved|partial|missed`, `preserved|violated`, and
`observed|missing` for the three respective groups. Missing, duplicate, foreign,
wrong-category, or aggregate-inconsistent judgments fail before append.
Assessment correction uses the same judgment flags with `correct --kind
assessment`; it supersedes only the latest assessment and retains history.

## Inspect, report, and rollback

```sh
/tmp/goalrail-canary --manifest-version 2 inspect --change change-synthetic-v2-001
/tmp/goalrail-canary --manifest-version 2 report
/tmp/goalrail-canary --manifest-version 2 disable \
  --actor owner \
  --source review:synthetic-rollback-v2 \
  --reason rollback-exercise
```

`inspect` projects effective context, basis, phase, telemetry components, and
item judgments. `report` derives its result from the verified v2 JSONL chain.
`disable` appends a stop marker for v2 only; it does not delete evidence, change
v1, remove hooks, or modify Langfuse.
