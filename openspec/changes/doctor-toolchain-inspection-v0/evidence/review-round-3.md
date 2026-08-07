# Independent review, round 3 (incremental)

Round 3 was attempted twice. The first attempt stalled: the reviewer produced
nothing for its five-minute progress bound and was stopped, writing no receipt.
That is the promoted progress bound doing its job rather than a review result,
and it reproduces the machine-local integration hang this repository already
records. The attempt was repeated once with the two configured provider
integrations removed from the reviewer session, which is what that flag exists
for. One retry, not a loop.

The bounded execution record for that first attempt — its exit status, the
reviewer output it had produced when the bound stopped it, and the receipt
check for it having written none — is in `review-round-3-stall.md`.

- reviewer: codex (mode cross)
- range: 47d02d13b1a8..ac462c001e82
- effort: medium, duration: 180s
- integrations removed: codebase-memory-mcp, node_repl

## Report

The shared lookup helper changes runtime identity resolution from an exact component-ID match to an ambiguous name-or-ID match. Canonical manifests with colliding names can therefore produce an incorrect readiness result.

Review comment:

- [P2] Prefer the exact runtime ID over component names — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:227-227
  When an accepted manifest contains an earlier component whose `Name` equals the declared runtime ID, `componentNamed` selects that component before reaching the exact ID match. For example, changing the earlier `goalrail` component's name to `node` remains valid because the decoder does not require unique names; diagnosis then inspects Goalrail's entrypoint and reports the runtime incompatible even when the actual `node` component is ready. The previous runtime path matched `BinaryIdentity.ComponentID` exactly, so retain exact-ID lookup for runtimes or prioritize IDs before names.


## Disposition

Accepted, and it is a regression this change introduced: sharing one lookup
between the runtime and the compiler replaced the runtime exact identifier match
with a name-or-identifier match. Component identifiers are unique in an accepted
manifest and names are not, so a colliding name displaced the real component.

An exact identifier now wins outright and a name is consulted only when no
identifier matches; a name two components answer to resolves to neither.

Regressions: TestExactComponentIdentifierWinsOverAName and
TestAmbiguousComponentNameResolvesToNothing, both failing before the fix — the
first verifying Goalrail entrypoint while reporting on the runtime.
