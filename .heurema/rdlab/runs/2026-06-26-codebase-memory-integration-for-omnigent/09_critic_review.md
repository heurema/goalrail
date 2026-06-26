# 09 Critic Review

## Provider availability

`provider_router.py doctor --probe` result:

- `claude-sonnet`: ok
- `claude-haiku`: ok
- `vibe-default`: ok
- `agy-default`: failed; exit 0 with empty stdout/stderr

The run used `claude-sonnet` as proposer and `vibe-default` as skeptic. The
agy route was excluded from debate because the probe did not produce the
required token.

## Proposer summary

The proposer supported the companion-tool path:

- Do not vendor codebase-memory-mcp into Omnigent.
- Do not make it a hidden Omnigent wheel dependency.
- Install as a separate `uv tool` using the existing installer pattern.
- Add `--skip-codebase-memory`.
- Make failures warning-not-fatal.
- Add setup-side repair/check for `auto_index=true`.
- Keep remote sandbox host-image work separate.

Accepted where backed by local code evidence.

## Skeptic summary

The skeptic highlighted:

- platform binary/download failures;
- air-gapped or enterprise network failures;
- remote sandbox parity gap;
- hidden config mutation;
- default automatic behavior being too broad.

Useful objections:

- Warning-not-fatal is required, but warnings must be explicit.
- Remote sandbox support must not be implied by local installer work.
- Full agent config repair should not be silent.

Over-strict objections:

- Making the entire CBM integration opt-in by default is stricter than the
  product goal; a default binary install is acceptable with `--skip`.
- Hard-failing platform/download failures conflicts with the constraint that
  Omnigent must still install.

## Additional local-code critique

Local code inspection found a stronger concrete issue than the provider passes:

- `codebase-memory-mcp install -y` can auto-confirm deletion of existing index
  DBs when indexes are present.

This changes the recommendation:

- default install can include `uv tool install`, `--version`, and
  `config set auto_index true`;
- full `install -y` should be interactive, guarded, or replaced by a safer CBM
  repair command.

## Residual risk

The recommendation relies on current CBM behavior. If CBM changes `install`
semantics, Omnigent should pin a minimum version and keep tests around the
commands it invokes.

## Critical issues


## Medium issues


## Minor issues


## Missing primary sources


## Unsupported claims


## Provider / critic debate

Record independent critique here. Prefer distinct providers or agents when available. Use `scripts/provider_router.py doctor --probe` and `scripts/provider_router.py route --task critic --exclude-provider <primary-provider>` when operating in a project lab.

If only one agent/provider was used, or if a provider was blocked by auth, location, or token refresh, say so explicitly and list the missing independent perspectives.

| Reviewer | Provider / role | Main objection | Change made or reason rejected |
|---|---|---|---|


## Overconfident recommendations


## Final confidence rating
