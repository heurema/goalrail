## Context

Two disciplines the ratchet keeps paying for exist only as reviewer prose, and
the canon templates never ask for them (CTX-1, CTX-3, CTX-6). An adversarial
exchange (evidence/adversarial-exchange-2026-08-02.md) amended every proposed
edit and found the blocking defect: `previousCanons` is empty, so the first
canon change ever shipped would read as user edits everywhere (SE-4).

## Goals / Non-Goals

Goals are OUT-1..OUT-6 of the confirmed intent. Non-goals that bind the
implementation: no version bump (refuted — the pin does not exist), no
per-decision artifact, no enforcement claims, no touching existing changes.

## Decisions

**Upgrade history before content, as separate commits.** (CTX-5, SE-4)
`previousCanons` gains the current canon digest and its files before any byte
changes; the migration test materializes that canon and proves `behind`, not
`edited`. Landing both in one commit would test the transition against itself.

**The opponent's amended text is the starting text.** (SE-3) The Consumed
Inputs table, the exact empty form, the recipe column with its
non-reproducible clause, and the decision-citation rule are taken as amended;
edits are tone-only. Their rationale lives in the exchange record, not
restated here.

**Instructions and templates change together.** (CTX-6) The schema.yaml
instruction for design gains the Consumed Inputs and citation requirements;
context's instruction gains the recipe requirement; intent's description
becomes candidate-or-confirmed. Generation follows instructions; a
template-only edit would produce artifacts that ignore the new sections.

**The canon CLI test stops overpromising.** (OUT-6) Its comment claims strict
validation; the test never runs `openspec validate`. Either the comment
describes what runs, or validation actually runs. The cheap honest fix is the
comment — measured leniency (SE-2) means running validate would not verify
sections anyway.

## Consumed Inputs

This slice reads: the previous canon's bytes (embedded at build time, trusted,
single fixed state — the migration test refuses any other digest); the four
template/schema files it edits (repository content, byte-identity enforced by
existing tests). No user-supplied, environment, or runtime input handling is
introduced or changed.

## Correlation and Evidence

Every claim traces to the Context Pack or the exchange record; the exchange is
preserved verbatim in evidence/. Decisions above cite their CTX/SE items
inline per OUT-5's own rule, applied to this change first.

## Measurement and Stop Conditions

Done when: migration test pins the real transition (SIG-1 locally); both
harnesses upgrade clean; `input-space` promotion flips to active with the
template as its landing place (SIG-2); ratchet.py counts post-promotion
occurrences (SIG-3). Stop: if the migration test cannot be made to pass
without weakening the edited-file refusal, stop and report — the refusal
protects real user edits and is not negotiable.

## Risks / Trade-offs

- The forcing function is prose to an LLM; authors can still write hollow
  tables. Accepted: the falsifiable empty form and reviewer contradiction are
  the mitigation, and SIG-3 measures whether it works.
- previousCanons grows per canon change; acceptable at this cadence, revisit
  if it ever holds more than a handful.

## Rollback

Content edits revert cleanly; previousCanons is additive and harmless to keep.
Adopters who already updated keep the new overlay; nothing depends on it
beyond prose.

## Open Questions

None.
