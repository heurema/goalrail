# Context Pack

- **Context Pack ID:** CTXP-canon-template-gaps-v0
- **Version:** 1
- **Previous version:** none
- **Started at:** 2026-08-02T19:00:00Z
- **Completed at:** 2026-08-02T19:20:00Z
- **Outcome:** sufficient

> Every claim carries the command that reproduces it. The one time this pack's
> predecessor did not, it carried an invented fact into a confirmed intent.

## Context Items

| ID | Kind | Claim | Verification (run this) | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | `input-space` is the largest finding class by a factor of two — 22 of 78 recorded findings | `jq -r .class ~/personal/heurema/docs/portfolio/REVIEW-FINDINGS.jsonl \| sort \| uniq -c \| sort -rn \| head -3` | 2026-08-02T19:05:00Z | The class the ratchet most owes a working promotion to |
| CTX-2 | repository | Its promotion is recorded as `partially-pending`, and the pending half is exactly this template change | `jq -r 'select(.class=="input-space")' ~/personal/heurema/docs/portfolio/PROMOTIONS.jsonl` | 2026-08-02T19:05:00Z | The prose half landed in the reviewer instructions on 2026-08-01 and the class kept recurring; the ladder says the next rung is a schema requirement |
| CTX-3 | repository | The design template has eight sections and none of them asks what happens when an input is absent, empty, differently shaped, or user-edited | `grep '^## ' internal/harness/canon/templates/design.md` → Context, Goals / Non-Goals, Decisions, Correlation and Evidence, Measurement and Stop Conditions, Risks / Trade-offs, Rollback, Open Questions | 2026-08-02T19:10:00Z | A design can be complete by the schema's own definition and still never have enumerated its input space |
| CTX-4 | repository | The intent template's outcomes table is headed "Confirmed wording" while the snapshot's own status may be `candidate` | `grep -n 'Confirmed wording' internal/harness/canon/templates/intent.md` and `grep -n 'candidate' internal/harness/canon/templates/intent.md` | 2026-08-02T19:10:00Z | The heading asserts confirmation the status denies; a downstream reader cannot tell interpretation from agreement |
| CTX-5 | repository | Changing a canon template changes the canon digest, so every adopting repository's overlay reports as drifted until `gr update` runs; the diagnosis reports it and nothing breaks | `gr doctor` reports `overlay: current (sha256:12cf770f…)` today; `internal/harness/overlay.go` marks a differing file `FileEdited` and `gr update` re-materializes | 2026-08-02T19:12:00Z | The blast radius is every adopter, which is why this is a change rather than an edit — but it is visible and one command repairs it |
| CTX-6 | repository | The Context Pack template gained a Verification column by hand in `pre-pr-review-v0`, and nothing in the schema requires one | `grep -c 'Verification' internal/harness/canon/templates/context.md` → 0, against this pack, which has the column | 2026-08-02T19:15:00Z | The discipline exists in one change by hand and in no template; the `unprovable-claim` class stands at 10 |
| CTX-7 | external | A reviewer raised a third gap: the schema requires a Context Pack once, before intent, and never again — while the owner's stated requirement is a context check before each architectural decision | goalrail#49, section 2 | 2026-08-02T19:15:00Z | One occurrence, and the fix would add a per-decision artifact to every change. Cost known, benefit unmeasured — deferred rather than built, per the lean rule in the workspace `GOALS.yaml` |

## Material Unknowns

None. The one judgement left — whether per-decision context binding earns its
cost — is recorded as deferred with its reason rather than carried as an
unknown, because the change proceeds correctly without answering it.
