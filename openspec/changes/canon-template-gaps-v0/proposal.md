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
  a migration test pins the actual transition. Only then do canon bytes change.
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
  observation.
- Every edit lands in the canon templates, the canon `schema.yaml`
  instructions, and the mirrored repository schema copies at once, byte-identical
  where the tests demand it; the canon CLI test stops promising a validation it
  does not run.

## Intent Coverage

| Change | Intent IDs | Boundaries preserved |
|---|---|---|
| Previous-canon history + migration test | OUT-1, SIG-1 | No version bump rides along (NG-1) |
| Consumed Inputs section + empty form | OUT-2, SIG-2, SIG-3 | Not claimed as enforced (NG-3) |
| Neutral intent headings + description | OUT-3 | Existing changes untouched (NG-4) |
| Verification recipe column | OUT-4 | Not claimed as enforced (NG-3) |
| Decision-to-context citation rule | OUT-5 | No per-decision artifact (NG-2) |
| Four-place mirroring + honest test | OUT-6 | — |

## Capabilities

**New Capabilities**

- `canon-content` — what the goalrail-intent canon templates ask of authors,
  and how the canon's own copies stay coherent.

**Modified Capabilities**

- `harness-update` — an overlay matching a previous canon is behind, not
  edited.

## Impact

- `internal/harness/overlay.go` — previous-canon history; the classification
  seam the opponent found.
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
- No rewriting of any existing change's artifacts.
