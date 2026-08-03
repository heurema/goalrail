## Why

Goalrail's current reader accepts only one Context Pack binding spelling while
repository-confirmed changes contain another, and no artifact contract declares
which forms are supported. A valid pair can therefore fail on spelling alone,
and ambient discovery can hide a malformed confirmed claimant entirely; this
change makes compatibility explicit and actionable without rewriting artifacts.

## What Changes

- Introduce a repository-owned artifact contract for Context Pack and Intent
  Snapshot pairs, separate from artifact identity/version and OpenSpec schema
  version. New canon emits contract v1; artifacts with no declaration use only
  a finite pinned legacy mode; declared unknown or malformed contracts fail
  closed without legacy fallback.
- Return one safe structured diagnostic contract for pair contract selection,
  format, Context binding, and referenced-evidence failures, with stable codes
  and actionable repair hints.
- Add a deterministic repository-owned conformance corpus with independently
  pinned semantic or diagnostic expectations, provenance, input mode, and an
  explicit compiler, local-run, and ambient lane matrix.
- Strengthen local-run preparation so Context/Intent contract failures return
  the structured diagnostic before any prepared state, run identity, or
  provider action exists.
- Strengthen ambient binding so an active artifact that claims confirmed status
  but fails the contract is retained by change name as diagnostic evidence and
  leaves the question unbound, without blocking or delaying the session.
- Preserve semantic compatibility baselines for the supported declaration
  spellings, six/seven-column Context Items, old/new headings, recipe retention,
  and real repository pairs while keeping already completed fixes unchanged.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Select explicit contract v1 or a finite legacy mode before interpreting a pair | OUT-1, SIG-1, SIG-2, SIG-3 | NG-1, NG-6, NG-8, NG-9 |
| Return safe structured diagnostics from direct consumers | OUT-2, SIG-3 | NG-2, NG-3 |
| Retain malformed confirmed ambient evidence and leave binding unbound | OUT-2, OUT-4, SIG-4 | NG-1, NG-7 |
| Define a stable corpus and explicit three-lane matrix | OUT-3, OUT-4, SIG-5 | NG-3, NG-4, NG-5, NG-9 |
| Pin semantic relations and existing regression baselines | OUT-5, SIG-2, SIG-3 | NG-6, NG-10 |
| Verify the bounded slice without external effects | SIG-7 | NG-2, NG-7 |

## Capabilities

### New Capabilities

- `artifact-conformance`: Contract selection, finite legacy compatibility,
  structured diagnostics, canon declaration, deterministic corpus cases, lane
  applicability, and semantic conformance relations for Context/Intent pairs.

### Modified Capabilities

- `local-run`: Pair verification failures during preparation return the shared
  structured artifact diagnostic before any state or provider effect.
- `ambient-connect`: Active confirmed artifacts that fail pair conformance are
  named in retained diagnostic evidence and make binding unbound while the
  background helper remains fail-quiet toward the session.

## Impact

- Affects the OpenSpec pair reader, local-run `IntentResolver`, ambient active
  intent discovery, canon Context/Intent templates, and their tests and fixtures.
- Adds repository-owned diagnostic and corpus structures but no runtime
  dependency, service, database, workflow engine, or provider integration.
- Existing artifacts remain byte-identical. Only explicitly pinned legacy forms
  are promised compatibility; unpinned historical syntax is not admitted by
  observation, and no migration or archive rewrite is introduced.

## Non-Goals

- The contract covers only Context Pack and Intent Snapshot pairs and their
  direct consumers, not a universal Markdown language or other OpenSpec
  artifacts.
- Legacy support is a finite allowlist, not permissive parsing, typo correction,
  alias discovery, or fallback from a declared contract failure.
- The slice adds no parser generator, schema service, database, workflow engine,
  provider, external review oracle, review-disposition registry, mutation
  testing program, or new runtime dependency.
- Active or archived change trees are not dynamic normative corpora; older
  Goalrail binaries are not executable compatibility oracles.
- Existing artifacts are not rewritten, migrated, committed, pushed, merged,
  published, released, or deployed by this change.
- Harness update concurrency, previous-canon migration, Verification recipe
  retention, Previous version lineage, and completed review dispositions remain
  unchanged and protected by their existing requirements and tests.
