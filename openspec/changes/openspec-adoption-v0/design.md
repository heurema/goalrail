## Context

Initialization already switches a schema safely and already refuses to do it
unasked. What it does not do is tell the user what they adopted. Baseline's real
adoption on 2026-08-01 left rules naming a status the adopted schema does not
define, and CTX-6 established that nothing else in the toolchain would ever
report it: rules are not validated against the active schema, and CTX-7 found
they are materialized nowhere, so a stale rule harms purely by being read.

Two constraints shape everything below. The module has **no dependencies at all**
— `go.mod` names none, and the existing reader for a change's schema is a line
scanner rather than a parser. And `harness-init` already requires that installing
and maintaining the harness needs no Node runtime, which forecloses asking the
pinned OpenSpec CLI to read schemas on our behalf.

## Goals / Non-Goals

Goals, from the confirmed intent:

- Report the artifact-level difference between the replaced schema and the
  adopted one (OUT-1).
- Disclose the configuration's rules verbatim, with their count and their
  provenance, and judge none of them (OUT-2).
- Answer whether the replaced schema directory may be removed with a count
  (OUT-3).
- Record the adoption in Goalrail's own state so it outlives the session
  (OUT-4).
- Carry one self-terminating advisory in the diagnosis (OUT-5).

Non-goals restated because they constrain the implementation, not just the
scope: no per-rule verdict (NG-1), no write into the rules block (NG-2), no
interaction (NG-3), no removal of the replaced schema directory (NG-4).

One boundary is asserted here and belongs here rather than in the intent, which
names none about dependencies: **no new dependency, and no YAML round-trip of any
file Goalrail does not own.** The module has none today, and the line-level
rewrite in `EnsureConfig` exists precisely so that a file Goalrail must not
reformat is never parsed and re-serialized. This is a design choice and is
revisable as one — reversing it costs a first dependency, which is worth taking
deliberately rather than acquiring as a side effect of adding a report.

## Decisions

**Compare schemas by scanning, and degrade honestly rather than guess.**

Both sides are read from the repository, and neither is assumed present. The
adopted side is the materialized overlay file, not the embedded canon:
`Materialize(root, false)` reaches `ActionKept` for an overlay file the user has
edited (`internal/harness/overlay.go`), and OpenSpec compiles against whatever is
on disk, so comparing the canon would describe a schema the switch did not
adopt. The replaced side is a file only when the repository materialized it —
a schema shipped with the stock CLI resolves from the installed package
(`openspec schema which spec-driven` reports `Source: package`), and reading it
there would put a Node package path on the harness's critical path, which the
no-Node requirement forbids. That case reports "could not compare" and why; it is
not a defect to be engineered away.

A bounded line-oriented reader extracts, for each entry of the top-level
`artifacts:` sequence, its `id`, its `requires` list, and the raw byte span of
its `instruction`. Comparison then yields: artifacts present in only one schema,
artifacts whose `requires` sets differ, and artifacts whose instruction spans
differ by digest.

Digesting the instruction span rather than reading it is what keeps this within
the boundary: the requirement is to report *that* an instruction differs, never
what the difference means, so the content never has to be understood — which is
also the only reason a hand-rolled reader is defensible here.

Where the reader cannot make confident sense of the replaced schema — no
`artifacts:` sequence, an entry with no `id`, indentation it cannot follow — it
SHALL report that the schemas could not be compared, and the rules disclosure and
the pin count SHALL still be produced. Partial disclosure is worth more than a
confident wrong diff, and CTX-2's ordering guarantee is about refusing before
writing, not about aborting after.

**Extract the rules block textually; do not parse it.**

The block runs from a `rules:` key at column zero to the next column-zero key —
the same column-zero discipline `schemaAssignment` already applies to `schema:`,
and for the same reason: an indented `rules:` belongs to something else. That
span is what gets reproduced verbatim and what gets digested.

Where `rules:` is the last key, the span ends at the last line belonging to its
value rather than at end of file. Trailing column-zero comments and blank lines
are not part of the value, and including them would let an edit to an unrelated
note at the bottom of the configuration change the digest — silencing the
advisory in a case the specification says leaves it standing.

The rule count is the number of sequence items within the span. That holds for
the block style every configuration in evidence uses, and it silently reports
zero for a legal flow mapping such as `rules: {intent: ["a", "b"]}`. The reader
therefore gets the same refusal the schema reader has: a shape it cannot count
confidently is reported as uncountable, and the rules are still reproduced
verbatim. A count is a claim, and a wrong count is worse than an absent one.

**Digest the rules block alone, not the configuration.**

The digest exists to answer one question — have the rules been looked at since
the adoption — and a digest over the whole file would be reset by an unrelated
edit to `context:`. Scoping it to the block is what makes the advisory's stop
condition mean what it says. Accepted weakness, stated to the owner before
confirmation: any edit touching the block silences it, including a trivial one.

**Count pins from the changes already on disk.**

Every change records its schema in its own `.openspec.yaml`, and
`internal/adapters/openspec` already reads exactly that with `readChangeSchema`.
The count covers active changes and the archive, because both keep a schema
alive. Reuse rather than a second reader: two readers of the same file would
eventually disagree, and the one used for reporting would be the one nobody
tested.

Reuse requires hardening it first. `readChangeSchema` trims whitespace and
quotes but never strips an inline comment, so `schema: intent-driven # legacy`
reads as `intent-driven # legacy` and matches nothing
(`internal/adapters/openspec/adapter.go:209-217`). The sibling
`schemaAssignment` in `internal/harness/config.go` already handles comments and
quoting correctly, and its treatment is the one to carry over. Left as it is,
the count would miss a change that pins the schema and the report would say the
directory may be removed — advice that breaks the change it failed to see. This
is the one place where being wrong destroys something, so the count is only
trustworthy after the reader is.

**Store the adoption in the marker, optional by construction.**

`.goalrail/ambient.json` gains an `adoption` object holding the replaced schema
name, the adoption time, and the rules digest. Absence is meaningful and valid —
every marker written before this change lacks it, and CTX-12's update path must
keep reading those without complaint. Nothing about the record is required for
the harness to work; it is evidence, not configuration.

**Adoption reporting is fail-open, always.**

No failure in comparing schemas, reading rules, or counting pins may fail
initialization or change its exit status. Initialization has already written the
overlay and switched the configuration by the time any of this runs; turning a
reporting fault into an aborted command would recreate precisely the half-installed
state this project has already fixed once. A failure is reported as a notice
naming what could not be produced.

**The advisory lives in the diagnosis, not in update.**

CTX-12 established that both `doctor` and `update` recur, so either could carry
it. `doctor` is where standing conditions already live — the observability line
and the newer-release line are both facts-not-faults reported there — and
`update` is an action whose report should describe what it did. One surface, and
it is the one a user runs to ask "is anything wrong here".

## Correlation and Evidence

Every claim the adoption section makes is reproducible from files on disk at the
moment it is made: the two schema files, the configuration, and the changes
directory. Nothing is carried from a previous run and nothing is inferred.

The marker's adoption record is the only durable state introduced, and it holds
one digest and two facts. It is evidence for the advisory's stop condition and
for nothing else; no decision anywhere reads it as authority.

## Measurement and Stop Conditions

The change is done when the five signals in the confirmed intent hold, measured
on a real repository rather than a fixture:

- Baseline's adoption report names `context` as added and `design` as having
  gained dependencies, both checkable by hand against the two schema files
  (SIG-1).
- The rules section reproduces every rule and contains no verdict word (SIG-2).
- Baseline's pin count is above zero and the report says the directory must
  stay; a repository where nothing pins it reports zero (SIG-3).
- The diagnosis line appears after adoption and is gone after an edit to the
  rules block, with no flag involved (SIG-4).
- Initialization and diagnosis produce the same machine-readable shape and exit
  status with no terminal attached (SIG-5).

The stop condition for the schema reader is explicit: it is finished when it can
produce a correct diff for the schemas Goalrail itself ships and a stated
"could not compare" for anything it cannot follow. It is not finished by
becoming a more capable YAML reader, and any pressure in that direction is the
signal to stop and reconsider the approach rather than to keep extending it.

## Risks / Trade-offs

- **A hand-rolled reader on someone else's file.** Mitigated by needing only ids,
  `requires`, and an opaque span; by digesting rather than interpreting; and by a
  first-class "could not compare" outcome. The risk that remains is a schema it
  misreads *confidently*. Tests must therefore include deliberately awkward but
  legal YAML — quoted ids, block scalars, comments between entries — and assert
  that anything not confidently read reports as uncomparable.
- **The digest is coarse.** Accepted before confirmation. A user who reformats
  their rules silences an advisory they never read. The alternative — per-rule
  tracking — requires the per-rule identity NG-1 refuses to invent.
- **Per-clone evidence.** A teammate receiving the switch through a commit sees
  nothing (NG-5). Accepted for this version.
- **A report nobody reads.** The advisory mitigates the one-shot case, but if the
  diagnosis itself is only ever run by an agent, the disclosure still lands
  nowhere. This is a limit of the surface, not of the design, and is worth
  revisiting rather than pretending it is solved.

## Rollback

Purely additive: new report fields, one optional marker object, one diagnosis
line. Reverting the code restores prior behaviour exactly, and a marker carrying
an adoption record stays valid for the reverted binary because the field is
ignored rather than required. No file the user owns is written, so there is
nothing to restore in their repository.

## Open Questions

None blocking. One deferred by the confirmed intent and recorded so it is not
mistaken for an oversight: whether a teammate who receives an adoption through a
committed configuration should ever learn of it (NG-5). Answering it requires a
signal that can live in the repository, which is the one place this change is
forbidden to write.
