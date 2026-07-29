# Retained extract: the A/B measurement and the kata's in-task channel

The measurement that motivates the escalation channel was produced in a separate
repository and is not resolvable from this one. The claims this change depends on
are therefore retained here verbatim, so the Context Pack can be audited without
reaching outside the repository that holds it.

Source of record: `katas/account-cleanup-v1/evidence/ab-measurement-2026-07-29.md`
on branch `codex/kata-account-cleanup-authoring` of the separate
`baseline-kata-authoring` repository, commit `e923dff`, read 2026-07-29.

## The result

One task, one conflict between two normative documents, one variable between
branches: whether a question-shaped return is expressible and machine-accepted.
Six fresh agents per branch (3 × Opus 5, 3 × Fable 5), neutral prompt.

| | branch A: channel present | branch B: no channel |
|---|---|---|
| asked | 6/6 | 0/6 |
| patched | 0/6 | 6/6 |
| oracle verdict | 6 × PASS | 6 × NEEDS_WORK |

## The caveat this change is designed around

Recorded by the measurement's own authors, verbatim in substance:

> Branch A was loud: the channel occupied about a third of the task statement,
> plus a leak in the version string (`…-clarification-v1`, quoted by one agent).
> 6/6 in branch A may be an overstatement; the 100→0 delta remains a delta.

They also record that branch B leaked dead clarification functions that four of
six agents noticed, that the run was headless, and that n=6 on one scenario in
one day is not publication-grade.

This is why the announcement introduced by this change is deliberately minimal.
A loud announcement would reproduce branch A's confound, and a later measurement
could not separate "the environment accepts a question" from "the task told the
agent to ask one".

## The kata's in-task channel

In the measured kata the channel consisted of two parts: a documented return
format (`docs/return-format.md`) and an accepting branch in the return script
(`scripts/prepare-return.mjs`), which required a clean tree and hashed the
returned question. The control copy differed only by their absence.

Both responsibilities are now duplicated by the harness — admission, the
clean-scope condition, the digest, and append-only retention. Removing them from
the kata is only safe once the harness announces the channel itself; otherwise
the capability becomes unreachable and the measurement unrepeatable.
