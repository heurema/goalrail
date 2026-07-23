# Local Run v0

`gr` is the provider-neutral operator surface for one prepared trusted local
run. The v0 implementation is intentionally inert: it can prepare and inspect
a WorkSpec, but every production `start` returns `ACTIVATION_REQUIRED` before a
run ID, launch claim, or provider invocation.

## WorkSpec

The input is a strict `goalrail.work-spec/v0` JSON document. It binds one
confirmed Intent Snapshot, exact repository `HEAD`, bounded path scope,
verification checks, stop conditions, and the fixed
`trusted-local-provider-enforced-v0` posture.

Provider names, provider arguments, provider session identity, approval state,
prompts, transcripts, credentials, and source bodies are not WorkSpec fields.
Check `argv` values are frozen instructions; `gr` does not execute them.

## Commands

Prepare and display the canonical snapshot and digest:

```sh
go run ./cmd/gr prepare \
  --file /absolute/path/to/work-spec.json \
  --state-dir /absolute/path/outside/the/repository
```

Inspect prepared state:

```sh
go run ./cmd/gr inspect \
  --digest sha256:<digest> \
  --state-dir /absolute/path/outside/the/repository
```

Inspect a later run receipt:

```sh
go run ./cmd/gr inspect \
  --run run-<id> \
  --state-dir /absolute/path/outside/the/repository
```

The syntax reserved for a separately activated future Codex start is:

```sh
go run ./cmd/gr start \
  --digest sha256:<digest> \
  --adapter codex \
  --state-dir /absolute/path/outside/the/repository \
  -- <adapter arguments>
```

In v0 this always stops with `ACTIVATION_REQUIRED`. There is no activation
flag, fixture flag, fallback adapter, environment override, retry, or
background continuation.

A future activated run would record check results explicitly:

```sh
go run ./cmd/gr finish \
  --run run-<id> \
  --result 'check-id=pass,local:evidence-ref,sha256:<digest>' \
  --state-dir /absolute/path/outside/the/repository
```

`finish` records results and read-only worktree fingerprints. It does not run
checks, stage files, commit, push, create a pull request, deploy, publish, or
perform another effect.

## State and evidence

Local state is stored under `--state-dir`, `GOALRAIL_STATE_HOME`, or the
platform user-state default, never inside the target worktree. Stored evidence
contains canonical metadata, hashes, bounded references, and changed paths.
It contains no patch, copied source, command output, raw provider payload,
prompt, transcript, or credential.

Worktree observation includes tracked, untracked, and ignored files under
explicit path-count and output-size bounds. Any repository `HEAD` change makes
the terminal receipt fail with `HEAD_CHANGED`; a commit cannot hide mutations
from v0 verification.

Real provider activation, automated checks, Git lifecycle actions, external
effects, credentials, retries, and stronger execution infrastructure require
separate confirmed intent and implementation changes.
