# Intent Snapshot

- **Intent ID:** INT-artifact-conformance-contract-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-artifact-conformance-contract-v0 v1
- **Run references:** owner session `eba4d3c0-28c5-4861-91aa-9f7aa05617d5`; local implementation session, 2026-08-02

## Source Evidence

- **SE-1 (owner, 2026-08-02):** requested implementation of all previously
  recommended ideas, explicitly preferred a versioned contract, and called
  format-only failures unreasonable: "можешь реализовать все предложенные тобой
  идеи, мне нравится версия контракта очень странно получать ошибки формата".
- **SE-2 (referenced proposal, same owner session):** the recommended set was a
  repo-owned versioned conformance contract and stable provenance corpus,
  production-composition conformance, historical negative controls and
  metamorphic relations, differential legacy replay, and evidence-backed review
  dispositions. Generic mutation testing and panic-only fuzzing were explicitly
  deferred rather than recommended for implementation.
- **SE-3 (repository):** the #56 result has a surviving disagreement between its
  actual Context Pack declaration and the strict production parser, while the
  adapter and ambient suites remain green. See CTX-2 through CTX-5.
- **SE-4 (repository):** the accepted seven-column Context Pack loses the
  verification recipe in the domain model, ambient discovery suppresses malformed
  active intents, and update has a compare-before-write race. See CTX-4, CTX-6,
  and CTX-7.
- **SE-5 (repository and review):** one external-review assertion was disproven
  by code and shuffled tests, and the existing review design is advisory. See
  CTX-8 and CTX-10.
- **SE-6 (confirmed predecessor boundary):** #56 deliberately added author-side
  forcing functions without machine enforcement and preserved existing changes;
  this enforcement layer is a new dependent intent. See CTX-9.

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Goalrail owns an explicit, versioned machine contract for the Context Pack and Intent Snapshot it consumes. The contract version is separate from each artifact's own ID/version, new canon emits one unambiguous form, and a documented legacy mode continues to accept already-written supported forms instead of turning harmless spelling differences into breakage. | Generate a new pair from canon, identify its contract version without interpreting prose, and load both it and the pinned legacy corpus successfully. | SE-1, SE-2, SE-3, SE-6, CTX-2, CTX-9 |
| OUT-2 | When an artifact is invalid, the user sees an actionable diagnostic naming the file, contract version or legacy mode, field or section, observed value, accepted expectation, and a concrete repair hint. Local-run and ambient paths surface the failure; neither reduces it to an opaque format complaint or silently skips it. | Run the malformed corpus through each applicable production path and compare the structured error plus rendered message with the case manifest. | SE-1, SE-3, SE-4, CTX-2, CTX-4 |
| OUT-3 | A stable repository-owned conformance corpus carries provenance and expected semantics. It contains canon-generated contract-v1 positives, pinned pre-existing legacy positives, and historical negatives; no fixture generated only by the implementation under test is allowed to be the sole oracle for its own acceptance. | Inspect the corpus manifest for source/provenance, contract mode, expected outcome, and applicable lanes, then run the corpus test without regenerating expected values. | SE-2, SE-3, CTX-1, CTX-3 |
| OUT-4 | Conformance exercises the real compositions separately: local-run intent verification through `IntentResolver.Verify`, the development compiler through `LoadChange`, and ambient active-intent discovery. Each active change is also checked individually as a smoke input, so one valid artifact or aggregate count cannot hide another invalid artifact. | Run the conformance command and confirm every case reports a result for every declared lane, including each current active change by name. | SE-2, SE-3, SE-4, CTX-3, CTX-4, CTX-5 |
| OUT-5 | Historical negative controls and semantic relations pin every confirmed #56 failure relevant to this slice: cross-file Context Pack ID/version mismatch, legacy declaration variants, old/new table headings, verification-recipe preservation, malformed active-intent visibility, and update compare-before-write safety. Accepted aliases must produce equal domain meaning; identity/version mutations must be rejected; preserved fields must round-trip byte-for-byte; and a post-inspection edit must never be overwritten. | Run each named regression and relation as an independently addressable test; temporarily reverting its corresponding fix must make that named control fail without changing the corpus expectation. | SE-2, SE-3, SE-4, CTX-2, CTX-4, CTX-6, CTX-7 |
| OUT-6 | Legacy compatibility is checked by differential replay against a pinned known-good Goalrail reference. Any intentional new behavior is an explicit versioned delta with rationale and owner confirmation; silently updating a golden fixture cannot redefine compatibility. | Replay the legacy corpus against the pinned reference and candidate, then confirm the report contains no unexplained difference and every allowed difference names its contract delta. | SE-2, SE-6, CTX-9 |
| OUT-7 | External review remains defense in depth rather than an oracle. Every finding relied on by this change has an `accepted`, `refuted`, or `deferred` disposition, a local evidence reference, and the resulting action or reason; a reviewer result alone neither changes the contract nor becomes an automatic gate. | Validate the disposition register, reproduce the evidence for the refuted `previousCanons` allegation, and confirm no contract expectation is sourced only to an undisposed review claim. | SE-2, SE-5, CTX-8, CTX-10 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Contract v1 is limited to Context Pack and Intent Snapshot semantics plus the minimum supporting fixtures required to exercise their consumers; it does not define a universal Markdown language or version every OpenSpec artifact. | SE-1, SE-2, CTX-5 |
| NG-2 | No parser generator, schema service, database, workflow engine, reviewer provider, or new runtime dependency is introduced. | Project scope discipline; SE-2 |
| NG-3 | Generic mutation testing and panic-only fuzzing are not part of this slice. They may later measure suite sensitivity or parser robustness, but neither supplies the missing oracle or production wiring. | SE-2 |
| NG-4 | External model review does not become a correctness oracle, owner, veto, or mandatory merge gate. | SE-5, CTX-8, CTX-10 |
| NG-5 | Current active changes are smoke inputs, not the normative contract; their agreement cannot replace the stable corpus and explicit contract version. | SE-2, SE-3, CTX-3, CTX-4 |
| NG-6 | The confirmed history of `canon-template-gaps-v0` is not rewritten. Compatibility is added in this dependent change, and any artifact migration remains explicit and reviewable. | SE-6, CTX-9 |
| NG-7 | This planning or implementation work grants no permission to commit, push, modify PR #56, merge, publish, release, or deploy. | Goalrail action boundary |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The surviving #56 format split is gone without stranding legacy artifacts. | The real `canon-template-gaps-v0` Context Pack/Intent pair passes `IntentResolver.Verify`; every pinned legacy positive passes; a new canon pair declares and passes contract v1. | OUT-1, OUT-4 |
| SIG-2 | Format failures are specific and visible. | Every malformed case returns its expected typed diagnostic with path, mode/version, location, observed value, expectation, and hint; ambient reports the invalid change by name rather than skipping it. | OUT-2, OUT-4 |
| SIG-3 | The conformance oracle is traceable and runs through production wiring. | One deterministic test command reports 100% of manifest cases across all declared lanes, and every case has non-empty provenance not limited to `generated-in-this-change`. | OUT-3, OUT-4 |
| SIG-4 | All known regressions are pinned semantically. | The named controls for declaration binding, heading compatibility, recipe preservation, malformed ambient input, and update concurrency pass; no accepted field is silently lost and the concurrency test observes no overwrite. | OUT-5 |
| SIG-5 | Legacy behavior changes only by an explicit delta. | Differential replay reports zero unexplained candidate/reference differences; each intentional difference points to a versioned, owner-confirmed contract delta. | OUT-6 |
| SIG-6 | Review evidence is auditable without becoming authoritative. | Every finding used by this change has exactly one valid disposition and evidence; the known false positive is `refuted`, and no unresolved or undisposed claim is used as a contract expectation. | OUT-7 |
| SIG-7 | The repository remains healthy after the bounded slice. | Contract/conformance tests, targeted adapter/ambient/harness tests, `go test -race` for touched packages, full `go test ./...`, OpenSpec validation, and a clean diff audit all pass before handoff. | OUT-1 through OUT-7 |

## Ambiguities and Unknowns

None. The owner confirmed that "all proposed ideas" means exactly OUT-1 through
OUT-7 with NG-1 through NG-7. Exact syntax, compatibility aliases, diagnostic
types, corpus layout, pinned reference selection, and the safe-write mechanism
remain proposal/design choices constrained by this intent rather than
unrecorded intent guesses.

## Confirmation

- **Confirmed by:** Vitaly D.
- **Confirmed at:** 2026-08-02
- **Verification action:** The owner read the Russian plain-language view that
  covered the contract, legacy behavior, diagnostics, corpus, three conformance
  lanes, historical controls, differential replay, review dispositions, all
  exclusions, and the success checks, then answered "да" to the exact question
  confirming `INT-artifact-conformance-contract-v0 v1`.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
