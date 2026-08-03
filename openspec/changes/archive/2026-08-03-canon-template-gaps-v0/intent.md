# Intent Snapshot

- **Intent ID:** INT-canon-template-gaps-v0
- **Version:** 3
- **Previous version:** 2
- **Status:** confirmed
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-canon-template-gaps-v0 version 3
- **Run references:** local session, 2026-08-02; predecessor `intent-v2.md`; adversarial exchange record in `evidence/`; PR 56 review threads

## Source Evidence

- **SE-1 — owner:** On 2026-08-02, picked this work — "49" — and directed the adversarial method: "изучи детальнее предложение на дыры и улучшения и еще я бы с codex как в ревью бы это сделал один доказывает другому".
- **SE-2 — repository measurement:** `openspec validate` enforces no template sections — a gutted `design.md` and `context.md` passed with only delta-related complaints. Nothing here is schema-enforced; a template edit forces an author-side claim a reviewer can contradict concretely.
- **SE-3 — repository review record:** On 2026-08-02, all three original edits were AMENDED, one supporting claim REFUTED, and one blocking discovery was recorded verbatim in `evidence/adversarial-exchange-2026-08-02.md`; the owner accepted that amended composition in full.
- **SE-4 — repository review record:** At base commit `1d3e75665ae8`, `previousCanons` was empty, so the first canon change would classify every adopter's untouched overlay as `edited` rather than `behind` and make `gr update` demand `--discard-local-edits`; the original claim "nothing breaks, one command repairs" was false.
- **SE-5 — repository review record:** The schema-version bump was refuted: `.openspec.yaml` pins only the schema name, so a bump pins nothing, and ADR-0002 reserves the version for structural transitions; `version: 2` stays.
- **SE-6 — repository ratchet:** `input-space` stood at 22 of 78 findings with its template promotion pending; `unprovable-claim` stood at 10, including an invented fact that reached a confirmed intent.
- **SE-7 — repository review record:** PR 56 has eight unresolved review threads: seven valid artifact, reader, lifecycle, or update defects, plus one test-order claim refuted by restore logic and a twenty-run shuffled test.
- **SE-8 — repository audit:** The first review-remediation patch bulk-compared all paths before a separate write loop, so later paths retained a wider compare/write window; the smallest portable repair is a per-file just-in-time optimistic digest check with an explicitly non-atomic contract.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Before any canon byte changes, the overlay upgrade path recognizes a previous canon: an overlay byte-identical to a canon this binary once shipped reads as **behind**, never as edited, and `gr update` brings it forward without `--discard-local-edits`. Before each replacement, update performs a just-in-time optimistic digest check against its inspected state: a visible late change refuses without discard; explicit discard retries at most three times and backs up the latest observed existing bytes. This is not a claim of atomic protection against a non-cooperating writer changing the file after the final check. | Materialize the old canon and cross cleanly; deterministically change behind, missing, and initially-current paths at the final-check seam and confirm refusal/preservation, then confirm explicit discard backs up the latest observed bytes. | SE-4, SE-7, SE-8 |
| OUT-2 | `design.md` gains a **Consumed Inputs** section: for every behavior-affecting external input this slice introduces or changes — input, source and trust, accepted states, refused states, mutation policy, verification. The legitimate empty form is exact and falsifiable: "None — verified within `<scope>` by `<bounded search or trace>`"; a bare N/A counts as unfilled. | Generate a design from the new template and confirm the section and its empty-form rule are present. | SE-3, SE-6 |
| OUT-3 | `intent.md` stops asserting confirmation outside the status: "Confirmed wording" → "Outcome", "Confirmed boundary" → "Boundary", and the schema description says candidate-or-confirmed. Confirmation is established only by Status and the Confirmation block. | Grep the canon for the old headings and the owner-confirmed description; all gone. | SE-3 |
| OUT-4 | `context.md` replaces the ad-hoc discipline with a **Verification recipe** column: bounded, read-only, stating prerequisites and the expected observation; for historical or non-repeatable evidence the honest form is "Not independently reproducible — `<reason>`; retained evidence: `<stable ref>`". A refresh command is never described as reproducing an earlier observation. The seven-column reader retains and validates the recipe with bounded and sensitive-text checks; the legacy six-column form remains readable without an invented recipe. | Generate and read a seven-column pack, reject unbounded or secret-shaped recipes, and load a legacy six-column pack with an empty recipe. | SE-3, SE-6, SE-7 |
| OUT-5 | The design instruction binds decisions to context without a new artifact: every material decision cites the Context Pack item IDs it relies on. A decision resting on an absent or stale fact stops design, creates a new Context Pack and linked candidate Intent Snapshot version, and requires exact owner confirmation before design resumes or the decision settles. | Read and test both canon schema copies; confirm the stop, version, binding, and owner-confirmation rule are identical. | SE-3, SE-7 |
| OUT-6 | Every edit lands in all four places at once — canon templates, canon `schema.yaml` instructions, and the mirrored `openspec/schemas/goalrail-intent/` copies — with the byte-identity test still green, and the canon test stops promising a validation it never runs. | `go test ./internal/harness/` green; the misleading comment gone or the validation actually run. | SE-2, SE-3 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No schema version bump. It pins nothing and would only look like pinning. | SE-5 |
| NG-2 | No per-decision Context Pack artifact. The minimal citation binding in OUT-5 is the whole of it. | SE-3 |
| NG-3 | No claim of schema enforcement anywhere. These are author-side forcing functions and reviewer anchors; validation does not check sections, and saying otherwise is the unprovable-claim class. | SE-2 |
| NG-4 | Other already-written changes are not rewritten: their six-column Context Packs remain readable and they continue to pin the schema by name. | SE-2, SE-5, SE-7 |
| NG-5 | PR 56 does not introduce platform-specific atomic directory exchange, a recovery journal, or a claim/install filesystem transaction. Its concurrency guarantee is the bounded optimistic final check in OUT-1. | SE-8 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The first real canon change upgrades cleanly. | On baseline and goalrail: doctor reports behind (not edited), `gr update` succeeds without flags, doctor then reports current. | SE-4 |
| SIG-2 | The ratchet's pending promotion closes. | `input-space` promotion status moves from partially-pending to active with the template as its landing place. | SE-6 |
| SIG-3 | The class trend is measurable. | Post-promotion `input-space` occurrences are countable by `ratchet.py`; recurrence escalates per the existing rule. | SE-6 |
| SIG-4 | The reviewed late-change path fails closed at the deterministic final-check seam. | Tests cover behind, missing, and initially-current files; without discard the latest bytes remain, while discard records and backs up the latest observed edit within three attempts. | SE-7, SE-8 |
| SIG-5 | This change obeys the artifact grammar it ships. | After owner confirmation, the production loader reads `canon-template-gaps-v0` with Context Pack v3, Intent Snapshot v3, structured evidence, and retained recipes. | SE-7 |

> The exact wording of template prose is design-level. The opponent's exact
> amendments are the starting text and may be edited for tone, not meaning.

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** Vitaly D.
- **Confirmed at:** 2026-08-03T04:57:25Z
- **Verification action:** The owner read the Russian owner-facing view of `INT-canon-template-gaps-v0 v3`, including its optimistic non-atomic update boundary, artifact compatibility, context-refresh gate, non-goals, and success signals, and replied "да" on 2026-08-03.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
