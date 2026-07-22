# Intent Snapshot

- **Intent ID:** STABILIZE-CANARY-RUNTIME-BOUNDARY
- **Version:** 1
- **Status:** confirmed
- **Owner:** t3chn
- **Run references:** Codex thread `019f836a-81ce-7bf0-a3ae-94835a80b10e`; GitHub PR `heurema/goalrail#5`

## Source Evidence

- **SE-1 — Owner statement:** The owner actively confirmed the proposed runtime-boundary outcomes, non-goals, and success signals on 2026-07-22.
- **SE-2 — Repository fact:** PR #5 moves the command's default evidence path from the active OpenSpec change directory to `canary/intent-canary-v0/events.jsonl` while archiving that change.
- **SE-3 — Repository fact:** PR review thread `PRRT_kwDOTewKTc6S0LfF` requires a versioned, owner-confirmed Intent Snapshot and proposal for the runtime boundary change.
- **SE-4 — Repository fact:** The PR review notes that the retained operator flow still names the removed active-change evidence path.
- **SE-5 — Repository fact:** OpenSpec is a replaceable development-time compiler/provider and canonical runtime behavior must not depend on its active or dated archive paths.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Archiving completed OpenSpec changes leaves the canary harness with one stable, provider-neutral manifest and default evidence location. | Run the command-path regression test and inspect the resulting path. | SE-1, SE-2, SE-5 |
| OUT-2 | Current operator guidance names the same stable evidence path used by the command and remains available outside archived planning history. | Compare the documented default with the command default and exercise the documented synthetic flow. | SE-1, SE-4 |
| OUT-3 | The runtime-boundary change is traceable to owner-confirmed intent before PR #5 is merged. | Validate the versioned intent, proposal coverage, and PR change metadata. | SE-1, SE-3 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not activate the real 15-change canary or create ordinal 1. | SE-1 |
| NG-2 | Do not change the frozen manifest rules, assignment rotation, evidence semantics, or pass/stop thresholds. | SE-1, SE-5 |
| NG-3 | Do not add an artifact registry, database, service, provider abstraction, or other infrastructure. | SE-1 |
| NG-4 | Do not implement eligible/excluded admission evidence or check-set freezing in this repair; that remains the next separately reviewed change. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Code and current operator guidance agree on the default path. | Both resolve exactly to `canary/intent-canary-v0/events.jsonl`; no current guidance directs operators to the removed active-change path. | SE-2, SE-4 |
| SIG-2 | Runtime behavior is independent of the OpenSpec lifecycle location. | The default-store regression passes in a repository root with no `openspec/` directory. | SE-2, SE-5 |
| SIG-3 | Archived planning history and canonical specs remain intact. | Mechanical spec-body comparisons and strict OpenSpec validation pass. | SE-2, SE-5 |
| SIG-4 | The repair introduces no canary activation or evidence. | The manifest remains `frozen_not_activated`, no repository evidence file exists, and real assignment remains rejected. | SE-1 |
| SIG-5 | The bounded repair is regression-safe. | `go vet ./...`, `go test -race ./...`, and `git diff --check` pass. | SE-1, SE-2 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** t3chn
- **Confirmed at:** 2026-07-22T08:04:36Z
- **Verification action:** The owner reviewed the explicit outcomes, non-goals, and success signals in Codex thread `019f836a-81ce-7bf0-a3ae-94835a80b10e` and replied “да”.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
