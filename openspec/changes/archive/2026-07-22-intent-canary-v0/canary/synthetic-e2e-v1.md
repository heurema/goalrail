# Synthetic End-to-End Receipt v1

## Scope

- Fixture: `synthetic-e2e-v1`
- External effects: none
- Real canary assignments consumed: none
- Real 15-change canary started: no

This receipt covers a deterministic local integration fixture. Its lifecycle
identity comes from a provider-shaped synthetic hook payload; it is not evidence
of a new live Codex or Langfuse run.

## Preserved evidence

- Append-only event chain: `internal/integration/testdata/synthetic-e2e-v1/events.jsonl`
- Aggregate report: `internal/integration/testdata/synthetic-e2e-v1/report.json`
- Validation receipt: `internal/integration/testdata/synthetic-e2e-v1/validation.json`

The fixture preserves a verified `change_id -> run_id -> root_session_id` join,
a green selected-check result, an independent owner `match` assessment, and a
`yes` repeat opt-in. It contains approved metadata and evidence references only.

## Synthetic overhead

The manifest-v1 timestamp formula receives one synthetic agent interval of 10
seconds and one required synthetic owner-review interval of 5 seconds. The
preserved result is 0.25 minutes. This is formula-validation evidence, not live
Langfuse timing.

## Report outcome

The aggregate verdict is `PENDING`, as required before 15 terminal assignments.
The synthetic fixture contributes one flow assignment and does not activate or
consume an ordinal from the real canary.

## Reproduction

```sh
mkdir -p /tmp/goalrail-go-cache /tmp/goalrail-go-tmp
GOCACHE=/tmp/goalrail-go-cache \
  GOTMPDIR=/tmp/goalrail-go-tmp \
  go test -v ./internal/integration \
  -run '^TestSyntheticEndToEndEvidenceMatchesPreservedArtifacts$'
```

The test generates the complete run in an isolated temporary store and compares
the event chain, report, and validation receipt with the preserved files
byte-for-byte.
