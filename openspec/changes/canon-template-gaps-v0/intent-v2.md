# Intent Snapshot

- **Intent ID:** INT-canon-template-gaps-v0
- **Version:** 2
- **Previous version:** 1
- **Status:** candidate
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-canon-template-gaps-v0 version 2
- **Run references:** local session, 2026-08-02; adversarial exchange record in `evidence/`

## Source Evidence

- **SE-1 — owner:** On 2026-08-02, picked this work — "49" — and directed the adversarial method: "изучи детальнее предложение на дыры и улучшения и еще я бы с codex как в ревью бы это сделал один доказывает другому".
- **SE-2 — repository measurement:** `openspec validate` enforces no template sections — a gutted `design.md` and `context.md` passed with only delta-related complaints. Nothing here is schema-enforced; a template edit forces an author-side claim a reviewer can contradict concretely.
- **SE-3 — repository review record:** On 2026-08-02, all three original edits were AMENDED, one supporting claim REFUTED, and one blocking discovery was recorded verbatim in `evidence/adversarial-exchange-2026-08-02.md`; the owner accepted that amended composition in full.
- **SE-4 — repository review record:** At base commit `1d3e75665ae8`, `previousCanons` was empty, so the first canon change would classify every adopter's untouched overlay as `edited` rather than `behind` and make `gr update` demand `--discard-local-edits`; the original claim "nothing breaks, one command repairs" was false.
- **SE-5 — repository review record:** The schema-version bump was refuted: `.openspec.yaml` pins only the schema name, so a bump pins nothing, and ADR-0002 reserves the version for structural transitions; `version: 2` stays.
- **SE-6 — repository ratchet:** `input-space` stood at 22 of 78 findings with its template promotion pending; `unprovable-claim` stood at 10, including an invented fact that reached a confirmed intent.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Before any canon byte changes, the overlay upgrade path recognizes a previous canon: an overlay byte-identical to a canon this binary once shipped reads as **behind**, never as edited, and `gr update` brings it forward without `--discard-local-edits`. A migration test pins this against the actual previous canon's digest. | Materialize the old canon in a scratch repository, build with the new one, and confirm doctor says behind and update proceeds clean. | SE-4 |
| OUT-2 | `design.md` gains a **Consumed Inputs** section: for every behavior-affecting external input this slice introduces or changes — input, source and trust, accepted states, refused states, mutation policy, verification. The legitimate empty form is exact and falsifiable: "None — verified within `<scope>` by `<bounded search or trace>`"; a bare N/A counts as unfilled. | Generate a design from the new template and confirm the section and its empty-form rule are present. | SE-3, SE-6 |
| OUT-3 | `intent.md` stops asserting confirmation outside the status: "Confirmed wording" → "Outcome", "Confirmed boundary" → "Boundary", and the schema description says candidate-or-confirmed. Confirmation is established only by Status and the Confirmation block. | Grep the canon for the old headings and the owner-confirmed description; all gone. | SE-3 |
| OUT-4 | `context.md` replaces the ad-hoc discipline with a **Verification recipe** column: bounded, read-only, stating prerequisites and the expected observation; for historical or non-repeatable evidence the honest form is "Not independently reproducible — `<reason>`; retained evidence: `<stable ref>`". A refresh command is never described as reproducing an earlier observation. | Generate a context pack from the new template and confirm the column and both forms. | SE-3, SE-6 |
| OUT-5 | The design instruction binds decisions to context without a new artifact: every material decision cites the Context Pack item IDs it relies on, and a decision resting on a fact absent or stale in the current pack requires a new pack version before it settles. | Read the design instruction in the canon and confirm the binding rule. | SE-3 |
| OUT-6 | Every edit lands in all four places at once — canon templates, canon `schema.yaml` instructions, and the mirrored `openspec/schemas/goalrail-intent/` copies — with the byte-identity test still green, and the canon test stops promising a validation it never runs. | `go test ./internal/harness/` green; the misleading comment gone or the validation actually run. | SE-2, SE-3 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No schema version bump. It pins nothing and would only look like pinning. | SE-5 |
| NG-2 | No per-decision Context Pack artifact. The minimal citation binding in OUT-5 is the whole of it. | SE-3 |
| NG-3 | No claim of schema enforcement anywhere. These are author-side forcing functions and reviewer anchors; validation does not check sections, and saying otherwise is the unprovable-claim class. | SE-2 |
| NG-4 | Existing changes are untouched: they pin their schema by name and their artifacts are already written. | SE-2, SE-5 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The first real canon change upgrades cleanly. | On baseline and goalrail: doctor reports behind (not edited), `gr update` succeeds without flags, doctor then reports current. | SE-4 |
| SIG-2 | The ratchet's pending promotion closes. | `input-space` promotion status moves from partially-pending to active with the template as its landing place. | SE-6 |
| SIG-3 | The class trend is measurable. | Post-promotion `input-space` occurrences are countable by `ratchet.py`; recurrence escalates per the existing rule. | SE-6 |

> The exact wording of template prose is design-level. The opponent's exact
> amendments are the starting text and may be edited for tone, not meaning.

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
