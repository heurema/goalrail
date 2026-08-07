# Intent Snapshot

- **Intent ID:** review-stall-isolation-v0
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** confirmed
- **Owner:** role:repository-owner
- **Context Pack:** review-stall-isolation-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 — Owner statement:** The owner asked to fix `gr review` itself rather than
  disable the offending MCP server in the machine configuration, having been shown
  that a configuration override is the immediate alternative.
- **SE-2 — Owner statement:** The owner asked for both defects in one change: the
  stall detector and the ability to run a reviewer without the machine's MCP
  servers, rather than either alone.
- **SE-3 — Owner statement:** The owner reported that reviews succeed when Codex is
  the author, which is the case in which the reviewer is the other provider.
- **SE-4 — Repository fact:** A blocked reviewer costs the whole deadline and yields
  neither a finding nor a receipt (CTX-1, CTX-11).
- **SE-5 — Repository fact:** The reviewer inherits the machine's MCP servers, and
  one of them blocks it indefinitely; Goalrail offers no way to run without them
  (CTX-3, CTX-7, CTX-10).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | A reviewer that stops making progress is stopped early and reported as a stalled reviewer, distinctly from one that ran out of time | Start a review against a reviewer that blocks, and read the reported outcome and the elapsed time | SE-4, CTX-1, CTX-2, CTX-11 |
| OUT-2 | The report of a stalled reviewer names what it last did, so the caller can act without reading provider session files | Read the output of a stalled run and check that it identifies the last observed reviewer activity | SE-4, CTX-2, CTX-13 |
| OUT-3 | A review can be run without the machine's MCP servers, without Goalrail naming any of them | Run a review with the isolating option against the branch that previously blocked, and observe it complete | SE-1, SE-2, SE-5, CTX-7, CTX-8, CTX-9, CTX-10 |
| OUT-4 | The pre-PR review of the current admission branch completes and leaves a receipt | Run the full pre-PR review on `codex/admission-hardening-after-pr62` and read the written receipt | SE-2, CTX-1, CTX-12 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not repair, work around, or take a position on the Codex/codebase-memory-mcp defect itself; it stays broken and outside this repository | CTX-3, CTX-4 |
| NG-2 | Do not name any specific MCP server, tool, or provider component in Goalrail code or configuration | SE-1, CTX-8, CTX-9 |
| NG-3 | Do not weaken the independent-review contract: the reviewer keeps a fresh context, remains the provider that did not author, and gains no write capability | SE-3, CTX-6 |
| NG-4 | Do not modify the owner's machine configuration, credentials, or provider installation | SE-1, CTX-10 |
| NG-5 | Do not treat an early stall stop as a finding-free review, a passing review, or a substitute for a completed one | SE-4, CTX-1 |
| NG-6 | Do not add a provider-specific reviewer runtime, a supervision service, or a second review implementation | CTX-10 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A blocked reviewer is stopped in minutes rather than at the deadline | A run against a reviewer that blocks ends well inside its deadline, and its elapsed time no longer equals the deadline as it did at 25 and 50 minutes | CTX-1, CTX-2 |
| SIG-2 | Stalling and running out of time are separately reported | The two conditions produce different outcomes in the command's own output | CTX-11 |
| SIG-3 | An isolated review completes where the same review previously blocked | The same branch and base that blocked twice produce a completed review and a receipt | CTX-3, CTX-7 |
| SIG-4 | Isolation carries no provider name | A search of the repository for the blocking server's name returns nothing outside this change's evidence | NG-2, CTX-8 |
| SIG-5 | A stalled review is never recorded as a completed one | No receipt claims completion for a run that was stopped for lack of progress | NG-5, CTX-11 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** role:repository-owner
- **Confirmed at:** 2026-08-06
- **Verification action:** The owner was shown a plain-language view of the outcomes, boundaries, and success signals, together with the evidence that the diff size, approval policy, sandbox, index, and MCP server were each excluded by experiment, and selected the scope covering both defects.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
