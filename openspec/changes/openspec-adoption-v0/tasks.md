## 1. Read a schema well enough to compare it, and know when you cannot

- [ ] 1.1 Add a reader that extracts, from a `schema.yaml`, the top-level `artifacts:` sequence as an ordered list of entries carrying `id`, the `requires` list, and the raw byte span of `instruction`. Stdlib only — the module has no dependencies (`go.mod`) and the existing change-schema reader is a line scanner (`internal/adapters/openspec/adapter.go:192`). Do not add a YAML library and do not shell out to the pinned CLI: `harness-init` already requires the harness to need no Node runtime.
- [ ] 1.2 Give the reader an explicit "cannot follow this" outcome, returned rather than logged: no top-level `artifacts:` key, an entry with no `id`, a duplicate `id`, or indentation it cannot track. Uncertainty must be representable, because the whole defence of a hand-rolled reader is that it refuses instead of guessing.
- [ ] 1.3 Compare two read schemas into a difference carrying: ids present in only one, ids whose `requires` sets differ, and ids whose instruction spans differ by digest. Digest the span; never interpret it. Reporting *that* an instruction differs is the requirement, and not reading it is what keeps 1.1 defensible.
- [ ] 1.4 Test against deliberately awkward but legal YAML: quoted ids, `>` and `|` block scalars, comments between entries, trailing whitespace, CRLF. Assert that every shape either compares correctly or reports as uncomparable — a confident wrong diff is the one failure mode that matters.
- [ ] 1.5 Test the pair that will actually be compared in practice: the embedded canon against Baseline's `intent-driven`. Assert `context` is reported as added and `design` as having gained dependencies.

## 2. Extract the rules block without owning it

- [ ] 2.1 Add extraction of the configuration's rules block: from a `rules:` key at column zero to the next column-zero key or end of file. Apply the same column-zero discipline `schemaAssignment` already applies to `schema:` (`internal/harness/config.go:139-173`) — an indented `rules:` belongs to something else.
- [ ] 2.2 Return the span verbatim and a count of the sequence items inside it. The count is of what a reader sees; do not interpret any rule's content.
- [ ] 2.3 Digest the extracted span alone. Not the file: a digest over the whole configuration would be reset by an edit to `context:`, and the advisory's stop condition would stop meaning what it says.
- [ ] 2.4 Test a configuration with no `rules:` key, one with an empty block, one with an indented `rules:` elsewhere in the file, and one where `rules:` is the final key with no trailing newline.

## 3. Count what still pins the replaced schema

- [ ] 3.1 Count changes naming a given schema across `openspec/changes/*/.openspec.yaml` and `openspec/changes/archive/*/.openspec.yaml`, reusing `readChangeSchema` (`internal/adapters/openspec/adapter.go:192`) rather than adding a second reader of the same file.
- [ ] 3.2 Report active and archived counts separately in the data, even though the requirement only obliges a total; whether the thing pinning a schema is finished work or work in flight is the first question anyone asks next.
- [ ] 3.3 Test a repository where nothing names the schema, one where only the archive does, and one with no changes directory at all.

## 4. Put the adoption section in the initialization report

- [ ] 4.1 Extend the configuration outcome (`internal/harness/config.go:34-41`) to carry the rules span, its count and its digest alongside the `PreviousSchema` it already reports. Populate it only on a switch that replaced another schema.
- [ ] 4.2 Add the adoption section to the initialization report (`cmd/gr/ambient.go:111-123`): the schema difference from 1.3, the rules disclosure from 2.2, and the counts from 3.1. Omit the section entirely where no schema was replaced.
- [ ] 4.3 Include the fixed statement that the rules were written against the replaced schema and that Goalrail neither interprets nor edits them. Assert in a test that no per-rule verdict word appears anywhere in the section.
- [ ] 4.4 Make every part of this fail-open: a failure in 1.x, 2.x or 3.x becomes a notice naming what could not be produced, and never fails initialization or changes its exit status. By the time this runs the overlay is written and the configuration is switched; aborting here would recreate the half-installed state `unshareable-registration-v0` already removed once.
- [ ] 4.5 Assert the configuration is byte-identical apart from the schema line after a run that produced a full adoption section.

## 5. Record the adoption where Goalrail keeps its own state

- [ ] 5.1 Add an optional `adoption` object to the marker (`internal/ambient`): replaced schema name, adoption time, rules digest. Optional by construction — every marker written so far lacks it.
- [ ] 5.2 Write it during initialization only when a switch replaced another schema, and never into the user's configuration.
- [ ] 5.3 Test that a marker with no adoption record reads as "never replaced a schema" and that `gr doctor` and `gr update` report no fault for it. `gr update` reads the marker on every run (`cmd/gr/harness.go:155-159`), so a required field would break every existing installation.

## 6. Carry the advisory in the diagnosis

- [ ] 6.1 Add one diagnosis line reported while the marker records an adoption and the configuration's current rules digest still matches the recorded one, naming the replaced schema and the adoption time (`internal/harness/doctor.go`).
- [ ] 6.2 Report it as a fact, not a fault: no effect on the overall verdict, no effect on exit status, no next action. Follow the observability line already in that report, which exists for exactly this shape.
- [ ] 6.3 Stop reporting it when the digests differ. No flag, no acknowledgement command, no stored dismissal — the user's own edit is the stop condition.
- [ ] 6.4 Never reprint the rules in the diagnosis; initialization discloses them once, when they can still be compared against what was replaced.
- [ ] 6.5 Test the full sequence on one repository: switch, line present; edit an unrelated part of the configuration, line still present; edit the rules block, line gone.

## 7. Prove it on the repository it was found on

- [ ] 7.1 Run the whole path against a copy of Baseline and record the adoption section and the diagnosis line as evidence. Baseline is where the defect was observed, and a fixture that agrees with the implementation proves less than the case that produced the requirement.
- [ ] 7.2 Assert Baseline's pin count is above zero and the report says the replaced schema directory must remain — it is pinned by the archive and by one open change, so this repository can never report zero.
- [ ] 7.3 Assert that initialization and diagnosis produce unchanged machine-readable shape and exit status with no terminal attached, so nothing here became interactive.
