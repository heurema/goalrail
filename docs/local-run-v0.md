# Local Run v0

`gr` is the provider-neutral operator surface for one prepared trusted local
run. Its lifecycle is `prepare → inspect → start → finish`: the owner prepares
and reviews the frozen WorkSpec, explicitly starts a separately authorized run,
then reviews the bounded terminal receipt. The v0 implementation is
intentionally inert by default: without a separately implemented exact
activation boundary, every production `start` returns `ACTIVATION_REQUIRED`
before a run ID, launch claim, or provider invocation.

## WorkSpec

The input is a strict `goalrail.work-spec/v0` JSON document. It binds one
confirmed Intent Snapshot, exact repository `HEAD`, bounded path scope,
verification checks, stop conditions, and the fixed
`trusted-local-provider-enforced-v0` posture.

Provider names, provider arguments, provider session identity, approval state,
prompts, transcripts, credentials, and source bodies are not WorkSpec fields.
Check `argv` values are frozen instructions; `gr` does not execute them.

## Commands

View the lifecycle and command list:

```sh
go run ./cmd/gr help
```

Every command also has built-in flag guidance:

```sh
go run ./cmd/gr help prepare
go run ./cmd/gr start --help
```

Help does not prepare, launch, or mutate a run.

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

## Blocked runs

A run can end because the work item cannot be completed as specified. The run
writes its question to one reserved repository-relative path,
`.goalrail/blocked.md`, and leaves the declared scope untouched. The terminal
status is then `blocked`: no implementation exists and an owner decision is
required.

`blocked` is terminal. Answering never resumes the run — the answer creates a
new intent version and a new one-shot run. The intent version that answers may
record which run and question it resolves, and whether the escalation was
`answered`, `spurious`, or `withdrawn`.

Goalrail treats the question as opaque bytes. It checks retention hygiene only —
regular file, non-empty, bounded, valid UTF-8, no unsupported control
characters, no secret-shaped content — and retains the exact bytes append-only
at provider observation. Deleting the worktree file afterwards does not change
the recorded outcome. The payload's shape is
[`goalrail.escalation/v0`](schemas/goalrail-escalation-v0.md), validated by
downstream tooling rather than by Goalrail.

The status rules are fail-closed:

- `prepare` refuses to run when the reserved path is already populated, so a
  stale question cannot attach itself to a new run;
- a question together with edits inside the declared scope is `failed`, not
  `blocked`, and the question is still retained;
- `unlinked`, `denied`, and `launch_failed` take precedence — an unattributable
  session records the question as evidence without minting a blocked status;
- supplied check results are recorded verbatim, and an explicit check failure
  stays `failed`;
- no combination of inputs yields `passed` while a question is retained.

Receipts carry an explicit `goalrail.terminal-receipt/v1` schema identifier and
the frozen intent reference. A receipt stored before the identifier existed
remains readable and is never rewritten.

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
