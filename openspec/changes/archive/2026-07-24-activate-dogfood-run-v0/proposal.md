## Why

The local-run lifecycle and Codex adapter already exist, but production start is
intentionally inert. Goalrail needs one tightly scoped dogfood activation
boundary to verify that an exact, owner-selected WorkSpec can reach the existing
adapter once without converting planning completion into general execution
authority. The selected task also needs usable built-in `gr` help so the
operator can understand the bounded lifecycle.

## What Changes

- Add a fail-closed admission boundary that permits exactly one
  owner-selected, prepared WorkSpec at its pinned repository revision to invoke
  the existing Codex adapter once.
- Preserve denial before provider invocation for missing, mismatched, stale,
  malformed, or duplicate input; do not add a reusable activation flag.
- Make `gr help` describe `prepare → inspect → start → finish` and make each
  command expose built-in help without changing machine-readable command
  outputs.
- Update the local-run guide with the owner-assisted activation and review
  boundary.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Exact one-shot admission and provider invocation | OUT-1, OUT-3, OUT-5, SIG-1, SIG-2, SIG-5 | NG-1, NG-3, NG-4, NG-5, NG-6 |
| Useful CLI help and local-run documentation | OUT-2, SIG-3, SIG-6 | NG-2, NG-5 |
| Bounded terminal evidence and owner-review stop | OUT-4, SIG-4, SIG-7 | NG-3, NG-4, NG-5 |

## Capabilities

### New Capabilities

- `dogfood-activation`: exact, owner-assisted admission of one frozen WorkSpec
  through the existing local Codex adapter.
- `operator-cli-help`: human-readable built-in lifecycle and command help for
  the `gr` operator surface.

### Modified Capabilities

- `local-run`: replace the pre-activation blanket production denial only for
  the exact authorized dogfood WorkSpec while preserving one-shot, provider
  enforcement, terminal receipt, and operator-review requirements.

## Impact

- `cmd/gr/main.go` and `cmd/gr/main_test.go`
- `docs/local-run-v0.md`
- Local-run admission wiring and its focused tests, only if required to enforce
  the exact selected WorkSpec
- No new dependency, provider, credential, external service, or Git action

## Non-Goals

- No general activation mechanism, multi-WorkSpec admission path, retry,
  daemon, scheduler, workflow engine, or background continuation.
- No Temporal, OpenFGA, task grants, Effect Gateway, credential broker, new
  sandbox platform, credentials, external effects, or automatic Git lifecycle.
- No provider-specific fields in canonical WorkSpec and no claim that Codex
  sandbox or approval enforcement belongs to Goalrail.
- No real provider launch, commit, push, pull request, merge, deployment, or
  publication as part of this implementation change.
