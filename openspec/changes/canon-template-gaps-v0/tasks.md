## 1. The upgrade path learns its own history (OUT-1) — first, separately

- [x] 1.1 Record the current canon (digest + files) as a previous canon in `internal/harness/overlay.go` before any content changes, so the classification seam the opponent found closes against the real bytes.
- [x] 1.2 Classify an overlay file byte-identical to a previous canon as behind, never edited; `gr update` replaces it without `--discard-local-edits`.
- [x] 1.3 Migration test: materialize the previous canon by its recorded digest in a scratch repository, diagnose (behind), update (clean), diagnose (current). It must fail if the previous canon's digest is wrong.
- [x] 1.4 Verify the edited-file refusal still stands: a genuinely edited file still stops the update and names the flag.

## 2. Canon content, all four places at once (OUT-2..OUT-5)

- [x] 2.1 design template: Consumed Inputs table with the amended columns and the exact falsifiable empty form; design instruction: the same requirement plus the decision-to-context citation rule with the pack-version condition.
- [x] 2.2 intent template: "Outcome" and "Boundary" headings; schema description: candidate-or-confirmed.
- [x] 2.3 context template: Verification recipe column with the bounded/read-only/prerequisites/expected-observation rule and the non-reproducible form; context instruction updated to match.
- [x] 2.4 Mirror every edit into `openspec/schemas/goalrail-intent/`; byte-identity test green.
- [x] 2.5 Tests asserting the mandatory texts exist in the canon (section headings, empty-form sentence, recipe column), so a later edit cannot silently drop them.

## 3. The canon CLI test tells the truth (OUT-6)

- [x] 3.1 Fix the comment in `canon_pinned_cli_test.go` that promises strict validation the test never runs — or run the validation and state what it actually checks (measured: not sections).

## 4. Prove it on the adopters (SIG-1, SIG-2)

- [x] 4.1 Build, then cross live where a crossing exists: baseline (held the shipped overlay) → behind, `gr update` clean, current. goalrail had no crossing to make — its mirror is edited by this branch; that leg is covered by the migration test against shipped bytes.
- [ ] 4.2 Flip the `input-space` promotion to active in PROMOTIONS.jsonl with the template as its landing place; goalrail#49 gets the resolution comment and closes.
