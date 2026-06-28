# code-intel-memory CLI contract (Goalrail integration)

Goalrail talks to the engine through one subprocess shape only. This
note pins the contract Goalrail depends on; the engine should converge
to the **target** behavior below.

## Invocation

```
<binary> cli <tool> '<json-args>'
<binary> --version
```

Binary names probed, in order: `code-intel-memory`, then
`codebase-memory-mcp` (rename in progress). Override with
`GOALRAIL_CODE_INTEL_BIN`.

The agent never supplies a path. Goalrail resolves the repo root
server-side from session context (`ToolContext.workspace`, falling back
to the runner cwd), canonicalizes it, discovers the git root without
leaving the session boundary, and only then calls the engine. An
explicit root (future `repo_id` / UI path) must stay inside the
boundary or the call is rejected.

## Target contract

- **success** → JSON object on **stdout**, exit code **0**
- **error** → JSON object on **stderr**, **non-zero** exit code
- **logs** stay on stderr (e.g. `level=info ...`) and must not break
  JSON parsing — the envelope is a single compact JSON line

## Tolerated legacy behavior (current engine)

A logical error (e.g. "project not indexed") is emitted as a JSON
envelope on **stderr** while the process still exits **0** with empty
stdout. `CodeIntelClient._parse` normalizes this into a
`CodeIntelToolError`, but it is not the desired end state.

## Project resolution

`index_status` and friends key off the engine **project name** (a slug
of the absolute path). Goalrail does not reimplement the slug rule: it
calls `list_projects` and matches by canonicalized `root_path`, so the
two stay in sync by construction. A repo with no matching project is
reported as `status: "not_indexed"` without relying on an engine error.

## Error envelope

```json
{ "error": "<code>", "message": "<human text>", "...": "hints" }
```

`CodeIntelClient` maps this to `CodeIntelToolError(message, code, payload)`.
Future hardening: add `schema_version` to every envelope and stabilize
exit codes (Phase 0).
