# Context Pack

- **Context Pack ID:** CTXP-canon-template-gaps-v0
- **Version:** 3
- **Previous version:** 2
- **Started at:** 2026-08-02T18:55:27Z
- **Completed at:** 2026-08-02T18:59:58Z
- **Outcome:** sufficient

> Every claim carries the command that reproduces it. The one time this pack's
> predecessor did not, it carried an invented fact into a confirmed intent.
>
> Version 2 corrected CTX-5 after the adversarial exchange refuted v1's upgrade
> claim. Version 3 re-runs every bounded recipe, records the PR review findings
> and the remaining compare/write window, and replaces the earlier artifact's
> mislabeled local-wall-clock `Z` metadata with live UTC observations. The exact
> predecessor is retained in `context-v2.md`.

## Context Items

| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
|---|---|---|---|---|---|---|
| CTX-1 | repository | `input-space` remains the largest finding class: 22 recorded findings, ahead of `contract-seam` at 12 and `unprovable-claim` at 10 | repo:heurema-docs/portfolio/REVIEW-FINDINGS.jsonl | `jq -r .class ~/personal/heurema/docs/portfolio/REVIEW-FINDINGS.jsonl \| sort \| uniq -c \| sort -rn \| head -3` returns `input-space` first with 22 | 2026-08-02T18:59:58Z | The class the ratchet most owed a working promotion to |
| CTX-2 | repository | The `input-space` promotion is now recorded as `active`, with this change's Consumed Inputs template as a landing place | repo:heurema-docs/portfolio/PROMOTIONS.jsonl | `jq -r 'select(.class=="input-space")' ~/personal/heurema/docs/portfolio/PROMOTIONS.jsonl` returns status `active` and names the canon landing place | 2026-08-02T18:59:58Z | The external half of task 4.2 is complete; merge readback still decides whether the default branch carries the landing place |
| CTX-3 | repository | The base design template has eight sections and none asks what happens when an input is absent, empty, differently shaped, or user-edited | git:1d3e75665ae8:internal/harness/canon/templates/design.md | `git show 1d3e75665ae8:internal/harness/canon/templates/design.md` shows no Consumed Inputs section | 2026-08-02T18:59:58Z | A design can be complete by the schema's own definition and still never have enumerated its input space |
| CTX-4 | repository | The base intent template heads candidate-capable tables with "Confirmed wording" and "Confirmed boundary" | git:1d3e75665ae8:internal/harness/canon/templates/intent.md | `git show 1d3e75665ae8:internal/harness/canon/templates/intent.md` contains both confirmed headings | 2026-08-02T18:59:58Z | The headings assert confirmation the status may deny |
| CTX-5 | repository | The original upgrade claim was false: without a previous-canon record, untouched adopter overlays become `edited`; the current migration test now proves the recorded shipped digest crosses cleanly | repo:openspec/changes/canon-template-gaps-v0/evidence/adversarial-exchange-2026-08-02.md | Inspect the retained exchange and run `go test ./internal/harness -run TestThePreviousCanonUpgradesCleanly -count=1`; expect a pass | 2026-08-02T18:59:58Z | The blast radius is every adopter, and OUT-1 exists because the original claim was false |
| CTX-6 | repository | The base Context Pack template contains no Verification recipe column | git:1d3e75665ae8:internal/harness/canon/templates/context.md | `git show 1d3e75665ae8:internal/harness/canon/templates/context.md` contains no Verification heading | 2026-08-02T18:59:58Z | The discipline existed only by hand; the reader must retain and validate the promoted field rather than discard it |
| CTX-7 | external | Issue 49 asks for context rebinding around architectural decisions and is already closed with a resolution comment for this change | issue:heurema/goalrail/49 | `gh issue view 49 --repo heurema/goalrail --json state,body,comments` shows the request, closed state, and resolution comment | 2026-08-02T18:59:58Z | The issue needs no second closure action; PR merge and archive are the remaining repository tail |
| CTX-8 | repository | The opponent's amendments define the Consumed Inputs shape, falsifiable empty form, recipe wording, and decision-citation rule | repo:openspec/changes/canon-template-gaps-v0/evidence/adversarial-exchange-2026-08-02.md | Not independently reproducible — a live exchange with a metered provider; retained evidence: `evidence/adversarial-exchange-2026-08-02.md` | 2026-08-02T18:59:58Z | The design's text decisions rest on retained evidence rather than memory |
| CTX-9 | external | PR 56 has eight unresolved review threads: seven valid defects to fix and one test-order claim refuted by existing restore logic and a shuffled run | pr:heurema/goalrail/56 | Read the eight PR 56 review threads; expect seven fixed dispositions and one refuted disposition before merge | 2026-08-02T18:59:58Z | A green old-head CI result does not close unresolved review findings |
| CTX-10 | repository | The first remediation attempt bulk-compares the overlay and then writes in a separate loop, leaving a wider TOCTOU window before later file replacements | repo:internal/harness/overlay.go | Inspect `materializeExpected`; expect comparison to finish before the write loop, then run the deterministic late-write regression after the fix | 2026-08-02T18:59:58Z | P1 is not closed until each replacement performs a just-in-time optimistic digest check |
| CTX-11 | repository | Context Pack v2 labeled local wall-clock values with `Z`; its `20:35Z` completion is later than the `16:36Z` commit that carried it | git:e7ae4c9:openspec/changes/canon-template-gaps-v0/context.md | Compare `git show e7ae4c9:openspec/changes/canon-template-gaps-v0/context.md` with `git show -s --format=%cI e7ae4c9`; expect the impossible ordering | 2026-08-02T18:59:58Z | v3 uses a live UTC window instead of repairing the old evidence in place |

> The judgement whether a per-decision Context Pack artifact earns its cost is
> deliberately deferred, not unresolved: the change proceeds correctly with
> the minimal citation binding in OUT-5.

## Material Unknowns

None.
