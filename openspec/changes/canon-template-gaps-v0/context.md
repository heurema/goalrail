# Context Pack

- **Context Pack ID:** CTXP-canon-template-gaps-v0
- **Version:** 2
- **Previous version:** 1 (CTX-5 corrected: v1 claimed the upgrade path was
  safe, the adversarial exchange refuted it, and this change's own rule forbids
  settling a decision on a refuted fact without a new pack version)
- **Started at:** 2026-08-02T19:00:00Z
- **Completed at:** 2026-08-02T20:35:00Z (v2; v1 completed 19:20Z, before CTX-5's correction existed)
- **Outcome:** sufficient

> Every claim carries the command that reproduces it. The one time this pack's
> predecessor did not, it carried an invented fact into a confirmed intent.

## Context Items

| ID | Kind | Claim | Verification (run this) | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | `input-space` is the largest finding class by a factor of two — 22 of 78 recorded findings | `jq -r .class ~/personal/heurema/docs/portfolio/REVIEW-FINDINGS.jsonl \| sort \| uniq -c \| sort -rn \| head -3` | 2026-08-02T19:05:00Z | The class the ratchet most owes a working promotion to |
| CTX-2 | repository | Its promotion is recorded as `partially-pending`, and the pending half is exactly this template change | `jq -r 'select(.class=="input-space")' ~/personal/heurema/docs/portfolio/PROMOTIONS.jsonl` | 2026-08-02T19:05:00Z | The prose half landed in the reviewer instructions on 2026-08-01 and the class kept recurring; the ladder says the next rung is a schema requirement |
| CTX-3 | repository | The design template has eight sections and none of them asks what happens when an input is absent, empty, differently shaped, or user-edited | `git show 1d3e75665ae8:internal/harness/canon/templates/design.md | grep '^## '` → Context, Goals / Non-Goals, Decisions, Correlation and Evidence, Measurement and Stop Conditions, Risks / Trade-offs, Rollback, Open Questions | 2026-08-02T19:10:00Z | A design can be complete by the schema's own definition and still never have enumerated its input space |
| CTX-4 | repository | The intent template's outcomes table is headed "Confirmed wording" while the snapshot's own status may be `candidate` | `git show 1d3e75665ae8:internal/harness/canon/templates/intent.md | grep -c 'Confirmed wording'` → 1 (at the pre-change revision; the working tree now differs by design) | 2026-08-02T19:10:00Z | The heading asserts confirmation the status denies; a downstream reader cannot tell interpretation from agreement |
| CTX-5 | repository | **v2, corrected.** v1 claimed nothing breaks and one command repairs — refuted: `previousCanons` was empty, so a canon change would classify every adopter's overlay as `edited` and `gr update` would stop with `ErrLocalEdits`, demanding `--discard-local-edits` for untouched files. Clean crossing exists only once the shipped canon is recorded in the history | v1's own citation showed it: `overlay.go` marked a differing file `FileEdited`; the exchange record and `TestThePreviousCanonUpgradesCleanly` prove the corrected claim | 2026-08-02T20:30:00Z | The blast radius is every adopter, and OUT-1 exists because the original claim was false |
| CTX-6 | repository | The Context Pack template gained a Verification column by hand in `pre-pr-review-v0`, and nothing in the schema requires one | `git show 1d3e75665ae8:internal/harness/canon/templates/context.md | grep -c 'Verification'` → 0 at the pre-change revision, against this pack, which has the column | 2026-08-02T19:15:00Z | The discipline exists in one change by hand and in no template; the `unprovable-claim` class stands at 10 |
| CTX-7 | external | A reviewer raised a third gap: the schema requires a Context Pack once, before intent, and never again — while the owner's stated requirement is a context check before each architectural decision | goalrail#49, section 2 | 2026-08-02T19:15:00Z | One occurrence, and the fix would add a per-decision artifact to every change. Cost known, benefit unmeasured — deferred rather than built, per the lean rule in the workspace `GOALS.yaml` |

| CTX-8 | repository | The opponent's exact amendments — the Consumed Inputs table shape, the falsifiable empty form, the recipe column wording, the decision-citation rule — are the starting text for every template edit | Not independently reproducible — a live exchange with a metered provider; retained evidence: `evidence/adversarial-exchange-2026-08-02.md`, verbatim verdict | 2026-08-02T20:30:00Z | The design's text decisions settle on these amendments; without this item they would rest outside the pack |

## Material Unknowns

None. The one judgement left — whether per-decision context binding earns its
cost — is recorded as deferred with its reason rather than carried as an
unknown, because the change proceeds correctly without answering it.
