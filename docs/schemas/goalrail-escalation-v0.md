# `goalrail.escalation/v0`

The shape of the payload a run writes when the work item cannot be completed as
specified. It is the content of the reserved escalation path,
`.goalrail/blocked.md`.

## Boundary

Goalrail treats this payload as opaque bytes. It validates retention hygiene
only — regular file, non-empty, bounded size, valid UTF-8, no unsupported
control characters, no secret-shaped content — and retains the exact bytes
append-only. It never parses, interprets, executes, or grades the payload,
because Goalrail runs no checks.

Semantic validation belongs to downstream consumers: a kata oracle, a review
surface, or any tool that reads the retained bytes. A payload that fails
downstream validation does not retroactively change the terminal status of the
run that produced it. The receipt records what was asked; whether it was a good
question is judged separately and recorded separately — on the answering intent
version, as a disposition.

The format names no provider, model, harness, or vendor, and depends on no
Goalrail runtime type. A payload stays valid evidence independently of the tool
that produced it.

## Document shape

The artifact is a Markdown document with a YAML front matter block. Prose after
the front matter is free-form context for a human reader and carries no
machine-checked meaning.

```markdown
---
schema: goalrail.escalation/v0
block_class: conflicting-requirements
sources:
  - ref: docs/spec.md:14
    interpretation: retention is 30 days
  - ref: README.md:52
    interpretation: retention is 90 days
separating_example: A record created 45 days ago is either purged or kept.
question: Which retention period governs when the two documents disagree?
---

Optional free-form context for the reader.
```

### Fields

| Field | Required | Meaning |
|---|---|---|
| `schema` | yes | Exactly `goalrail.escalation/v0`. |
| `block_class` | yes | What kind of block this is. See below. |
| `sources` | yes | One or more references the block was derived from. Each entry has a `ref`; `interpretation` is optional. |
| `separating_example` | no | A concrete case the candidate readings decide differently. |
| `question` | yes | Exactly one question, answerable by the owner. |

`sources[].ref` is a repository-relative path, optionally suffixed with `:line`.

### Block classes

| Class | Two sources | Separating example |
|---|---|---|
| `conflicting-requirements` | expected | expected |
| `ambiguous-requirement` | expected | expected |
| `missing-dependency` | not expected | not expected |
| `broken-environment` | not expected | not expected |
| `insufficient-access` | not expected | not expected |

A block class without two sources or without a separating example is valid
without those fields. The format deliberately does not force the shape of a
requirements conflict onto a missing dependency — that over-fitting was the
defect the original design carried.

## Constraints

- Exactly one question. A payload that asks several questions is one payload
  too coarse to answer; split the block or ask the question that unblocks the
  work.
- The payload is written once, before the run ends. It is not a log.
- No credentials, tokens, secrets, transcripts, or copied source bodies. Cite a
  source by reference, not by pasting it.

## Chain

A blocked receipt records the reserved path, the digest of the retained bytes,
and the frozen intent reference. The intent version that answers the question
may record the run identifier, the same escalation digest, and a disposition of
`answered`, `spurious`, or `withdrawn`.

That is the whole chain, and it is what the design delivers: a question and a
decision are versioned artifacts, machine-linked. It is not a guarantee that the
same conflict cannot recur — nothing prevents a later WorkSpec from referencing
a pre-answer intent version, and no registry of answered questions exists.
