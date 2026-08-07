## Why

A pre-PR review of the current admission branch was attempted twice and produced
nothing both times. Each run consumed its entire deadline — 25 minutes, then 50 —
and reported only that the reviewer had not finished in time. No finding, no
receipt, nothing to disposition. The branch it was meant to check is still
unreviewed (CTX-1).

The reviewer was not slow. Its session recorded zero events for the remaining 23
and 49 minutes after it issued one call to an MCP server configured on the
machine (CTX-2). That call reproduces on demand and never returns (CTX-3), while
the same tool, project, and arguments answer immediately for a different client
(CTX-4). Approval policy (CTX-5) and the reviewer's read-only sandbox (CTX-6) do
not change it.

Two Goalrail defects made a foreign defect this expensive. The reviewer inherits
the machine's MCP servers, although the same invocation deliberately strips the
environment so the reviewer does not inherit the author's session; there is no
way to run without them (CTX-7, CTX-10). And a reviewer that stops entirely is
indistinguishable, at the caller, from one that is merely slow: both are reported
as a deadline overrun, after the full deadline is paid (CTX-11).

The failure is not exotic. Any provider component that blocks — a hung tool, an
unreachable endpoint, a lock never released — produces exactly this shape, and
today it always costs the whole budget and yields silence.

## What Changes

- Bound reviewer **progress**, not only total elapsed time. A reviewer that emits
  nothing for a bounded period is stopped, and the stop is reported as a stalled
  reviewer rather than as an exhausted deadline.
- Report what the stalled reviewer last did, taken from the output already
  captured, so the caller can act without reading provider session files.
- Let a caller run the reviewer isolated from the machine's provider integrations,
  expressed as caller-supplied invocation arguments so that Goalrail names no
  provider component.
- Keep a stalled review outside the receipt record: it completed nothing, and a
  receipt claiming otherwise is worse than none.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Bound reviewer progress and stop a stalled reviewer early | OUT-1, SIG-1 | NG-1, NG-3, NG-6 |
| Report stalling separately from a deadline overrun | OUT-1, SIG-2 | NG-5 |
| Name the stalled reviewer's last observed activity | OUT-2, SIG-2 | NG-1 |
| Allow a reviewer invocation isolated from machine integrations, without naming any | OUT-3, SIG-3, SIG-4 | NG-2, NG-4, NG-6 |
| Write no receipt for a stalled review | OUT-1, SIG-5 | NG-5 |
| Complete the pre-PR review of the admission branch | OUT-4, SIG-3 | NG-3 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `independent-review`: bound reviewer progress in addition to total time,
  classify a stalled reviewer as its own outcome carrying its last observed
  activity, keep it out of the receipt record, and extend the existing capability
  boundary so a caller can isolate the reviewer from the machine's provider
  integrations without Goalrail naming one.

## Impact

- Affected code: `internal/review` (reviewer invocation, deadline handling,
  outcome reporting) and the `gr review` command surface.
- Affected behavior: a blocked reviewer ends in minutes with a distinct outcome
  instead of at the deadline with a generic one; an isolating option becomes
  available and is off by default.
- Public contract impact: one new outcome and one new option on a public command;
  existing outcomes, receipts, and defaults are unchanged.
- Dependencies and infrastructure: none. No new runtime dependency, service, or
  provider integration.
- Operations: nothing is activated, published, or installed; the owner's machine
  configuration and credentials are untouched.

## Non-Goals

- Do not repair, work around, or take a position on the provider defect that
  triggered this; it stays broken and outside this repository.
- Do not name any specific MCP server, tool, or provider component in Goalrail
  code or configuration.
- Do not weaken the independent-review contract: the reviewer keeps a fresh
  context, remains the provider that did not author where one exists, and gains
  no write capability.
- Do not modify the owner's machine configuration, credentials, or provider
  installation.
- Do not treat an early stall stop as a finding-free, passing, or completed
  review.
- Do not add a provider-specific reviewer runtime, a supervision service, or a
  second review implementation.
- This proposal does not authorize implementation, commit, push, pull-request
  creation, merge, release, or activation.
