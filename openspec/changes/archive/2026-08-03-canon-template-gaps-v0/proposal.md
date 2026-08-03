# The canon starts asking the questions the ratchet keeps paying for

## Why

The findings ratchet's largest class — unenumerated input space, 22 of 78
recorded findings — has a promotion that is still pending because it was
supposed to land in the design template and never did. The second-largest
conversational class, unprovable claims, reached a confirmed intent once
already. Both disciplines exist as reviewer prose; prose has been measured
lagging. The canon templates are where an author meets the question before the
work is written, and a reviewer gets a concrete claim to contradict instead of
a judgement to make.

An adversarial exchange with the other provider reshaped this change before it
touched a byte: every original edit was amended, one supporting claim was
refuted, and the opponent found the blocking defect — the overlay upgrade path
has an empty previous-canon history, so the first canon change in this
project's history would misclassify every adopter's overlay as hand-edited and
break clean `gr update`. That fix is now the first outcome, not a footnote.

## What Changes

- **Upgrade history first.** The previous canon's digest joins the overlay's
  known history, an overlay matching it reads as behind rather than edited, and
  a migration test pins the actual transition. Update also rechecks each target
  optimistically immediately before replacement; a visible late change refuses
  without discard, while explicit discard retries at most three times after a
  fresh backup. This deliberately does not claim an atomic filesystem CAS.
- **`design.md`: Consumed Inputs.** A table for every behavior-affecting
  external input the slice introduces or changes — source and trust, accepted
  states, refused states, mutation policy, verification — with an exact,
  falsifiable empty form. The design instruction also binds material decisions
  to Context Pack item IDs, requiring a pack version bump when a decision rests
  on an absent or stale fact.
- **`intent.md`: headings stop asserting confirmation.** "Outcome" and
  "Boundary" replace the "Confirmed …" headings; the schema description says
  candidate-or-confirmed; Status and the Confirmation block remain the only
  places confirmation lives.
- **`context.md`: Verification recipe.** Bounded, read-only, prerequisites and
  expected observation stated; an honest non-reproducible form for historical
  evidence; a refresh command never described as reproducing an earlier
  observation. The runtime reader retains and validates the new field while
  continuing to accept legacy six-column Context Packs without inventing data.
- **Context refresh rebinds owner intent.** A material absent or stale fact
  stops design, creates the next Context Pack and a linked candidate Intent
  Snapshot version, and requires exact owner confirmation before design or the
  affected decision resumes.
- Every edit lands in the canon templates, the canon `schema.yaml`
  instructions, and the mirrored repository schema copies at once, byte-identical
  where the tests demand it; the canon CLI test stops promising a validation it
  does not run.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Previous-canon history, final digest checks, and migration/race tests | OUT-1, SIG-1, SIG-4 | NG-1, NG-5 |
| Consumed Inputs section + empty form | OUT-2, SIG-2, SIG-3 | NG-3 |
| Neutral intent headings + description | OUT-3 | NG-4 |
| Verification recipe column plus retained, validated, legacy-compatible reader data | OUT-4, SIG-5 | NG-3, NG-4 |
| Decision-to-context citation, stop, version, binding, and owner re-confirmation lifecycle | OUT-5 | NG-2 |
| Four-place mirroring + honest test | OUT-6 | NG-1 |

## Capabilities

**New Capabilities**

- `canon-content` — what the goalrail-intent canon templates ask of authors,
  and how the canon's own copies stay coherent.

**Modified Capabilities**

- `harness-update` — an overlay matching a previous canon is behind, not
  edited.

## Impact

- `internal/harness/overlay.go` — previous-canon history; the classification
  seam and per-file optimistic final check the reviews found.
- `internal/harness/update.go`, `cmd/gr/harness.go` — cumulative recovery and
  partial-update reporting across bounded retries.
- `internal/adapters/openspec/`, `internal/domain/` — retain and validate the
  Verification recipe, preserve the six-column reader, and production-load the
  change's own confirmed artifacts.
- `openspec/changes/canon-template-gaps-v0/{context-v2.md,intent-v2.md}` — retain
  the exact predecessor artifacts named by the current v3 lineage.
- `internal/harness/canon/{schema.yaml,templates/}` and
  `openspec/schemas/goalrail-intent/` — the content edits, mirrored.
- `internal/harness/canon_test.go`, `canon_pinned_cli_test.go` — byte-identity
  stays green; the misleading strict-validation comment goes or the validation
  actually runs.
- Adopting repositories see `overlay: behind` once, run `gr update`, and are
  current. Their existing changes pin schemas by name and keep compiling.

## Non-Goals

- No schema version bump; it pins nothing.
- No per-decision Context Pack artifact.
- No claim of schema enforcement anywhere.
- No rewriting of other already-written changes; their legacy artifacts remain readable.
- No platform-specific atomic exchange, recovery journal, or claim/install transaction.
