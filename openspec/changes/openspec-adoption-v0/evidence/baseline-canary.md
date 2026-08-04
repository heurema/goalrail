# Baseline adoption canary

Date: 2026-08-04

The canary was rerun after independent review fixes for line-ending-only
instruction differences and ambiguous rules shapes; the observations below are
from the corrected implementation.

## Source and isolation

- Baseline source commit: `83711b2f00abb1ee03207aa0388a6ec4864f8f99`
  (`Restore the harness overlay to the canon the installed binary carries`).
- Candidate Goalrail worktree base: `03fd7da20371bb8bd6d4483dab1c0976ea8d747b`.
- The Baseline working tree contained unrelated owner work. Its tracked OpenSpec
  configuration, `intent-driven` schema, and existing changes matched the
  source commit. Two untracked changes named `goalrail-intent`, so neither could
  increase the old-schema pin count.
- The canary used `git archive` at the source commit, initialized a new temporary
  Git worktree, and left `/Users/vi/personal/heurema/baseline` untouched.
- To replay the pre-adoption state, only the temporary copy's configuration
  schema line was changed from `goalrail-intent` to `intent-driven` before
  initialization.
- Both commands ran over ordinary pipes with no terminal attached (`tty=false`).
- Instruction span comparison normalizes CRLF to LF before digesting; the
  reported Baseline instruction changes therefore cannot be checkout-only line
  ending differences.

## Initialization result

`gr init --confirm-schema-switch` exited `0` and emitted valid JSON. Its
adoption section reported:

```json
{
  "replaced_schema": "intent-driven",
  "schema_difference": {
    "comparable": true,
    "only_in_adopted": ["context"],
    "dependencies_changed": ["intent", "design"],
    "instructions_changed": ["intent", "proposal", "specs", "design", "tasks"]
  },
  "rules": {
    "present": true,
    "has_rules": true,
    "count": 8,
    "counted": true,
    "sha256": "0238b3e392deb9615f29d012e2cb54b1a1311bb03989d778c19179faf5e22728"
  },
  "rules_disclosure": "These rules were present when the schema was replaced; Goalrail neither interprets nor edits them.",
  "pins": {
    "active": 1,
    "archived": 2
  },
  "schema_directory": "3 change(s) still name \"intent-driven\"; its schema directory must remain"
}
```

The exact rules span disclosed by the report was:

```yaml
rules:
  context:
    - Gather external evidence only for a concrete unknown that could change the intended outcome; store concise claims with sources and observation times, never raw pages or transcripts.
  intent:
    - Use the project-local $grilling skill explicitly and ask one decision question at a time.
    - Treat material human input as primary evidence and model-generated synthesis as derivative.
    - Keep source evidence distinct from interpretation; label every source and quote minimally; do not store full transcripts.
    - Do not mark an interpretation confirmed without explicit owner confirmation; preserve refined, superseded, and rejected meaning by versioning the snapshot rather than overwriting it.
    - Treat status confirmed without recorded confirmation evidence as candidate.
    - Do not turn wider ecosystem hypotheses into scope for the current change.
  proposal:
    - Do not create a proposal until intent is confirmed and the owner explicitly asks to proceed.
```

After initialization, a recursive comparison against a second archive of the
same Baseline commit found the complete `openspec/` tree byte-identical. The
temporary configuration therefore returned to the source bytes, and no spec,
change, or schema directory was changed or removed.

## Diagnosis result and control

The diagnosis carried the intended one-line fact:

> rules present when schema "intent-driven" was replaced at
> 2026-08-04T14:24:30Z have not changed since that adoption

The committed Baseline overlay predates the candidate binary's canon, so both
control runs correctly reported the same four existing `behind` problems and
exited `1`. Removing only the optional adoption object from the temporary marker
and rerunning diagnosis proved:

- exit status remained `1` with and without the advisory;
- `working`, `problems`, and `next_actions` were identical;
- all pre-existing top-level JSON keys were identical;
- `adoption` was the only added top-level key.

The non-zero diagnosis was therefore caused by pre-existing overlay currency,
not by the adoption record or advisory.
