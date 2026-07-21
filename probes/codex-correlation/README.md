# Codex correlation probe

This is a dependency-free feasibility probe, not the production Goalrail adapter.
The corresponding project-local Go adapter is implemented in
`internal/adapters/codex`; this probe remains its provider-contract fixture and
does not become a second runtime path.

It tests one narrow contract:

```text
process-scoped run context + Codex hook input -> deterministic lineage result
```

The Codex hook JSON is read from standard input. The immutable launch context is supplied as one JSON value in `GOALRAIL_RUN_CONTEXT`:

```json
{
  "schema_version": 1,
  "change_id": "intent-canary-v0",
  "run_id": "run-probe-001",
  "repo_root": "/workspace/goalrail"
}
```

The probe accepts the documented common Codex hook fields it needs: `session_id`, `cwd`, and `hook_event_name`. It also validates `SessionStart.source` or the required `SubagentStart` fields. Current Codex hooks already supply the parent session ID to subagent hooks, so the probe does not perform a heuristic child-to-parent lookup. It never reads `transcript_path`.

The output is deterministic JSON:

- `VERIFIED` contains `change_id`, `run_id`, and `root_session_id` plus a fingerprint of the launch context.
- `UNLINKED` contains a stable reason code and no guessed lineage.

`GOALRAIL_RUN_CONTEXT` is process-scoped so concurrent launches do not share a repository-wide mutable `current-change` file. The context fingerprint detects which exact value produced a result; it is not a signature or authorization boundary.

Fixture shapes follow the current [Codex hooks contract](https://learn.chatgpt.com/docs/hooks).

## Run the deterministic matrix

```sh
python3 probes/codex-correlation/test_probe.py
python3 probes/codex-correlation/test_live_hook.py
```

The tests use only the Python standard library and synthetic identifiers. They cover:

- `SessionStart` from startup, resume, and compaction;
- `SubagentStart` inheritance of the parent session ID;
- missing launch context;
- conflicting repository context;
- two concurrent changes with isolated process environments.

`live_hook.py` is the one-shot wrapper used by OpenSpec task 1.3. It runs the
same probe, emits no model-visible output, and writes one append-only sanitized
receipt keyed by the observed hook identity. It never writes hook input,
transcript paths, prompts, credentials, or source content. Its deterministic
wrapper checks are in `test_live_hook.py`.

The live CLI and App results are recorded in the change evidence directory.

## Non-goals

- Installing a Codex hook.
- Writing a lineage or evidence record.
- Parsing transcripts.
- Calling Langfuse or any external service.
- Authenticating the launch context.
