## 1. Context and Intent Foundation

- [x] 1.1 Add provider-neutral Context Pack types, bounded validation, and deterministic tests for sufficient, unresolved, exhausted, malformed, and sensitive-payload cases.
- [x] 1.2 Link Context Pack/item references into Intent Snapshot validation and wording-only provenance checks without changing the three semantic intent groups.
- [x] 1.3 Add `context.md`, its template and parser, migrate `goalrail-intent` to schema version 2 with `context -> intent`, and backfill this active change so OpenSpec status remains coherent.

## 2. Canary Version and Evidence Lifecycle

- [x] 2.1 Add and validate manifest version 2, preserve the immutable version 1 builder, and add the frozen non-activated v2 runtime manifest.
- [x] 2.2 Make evidence/store/operator boundaries manifest-version aware, retain historical v1 compatibility, and isolate v2 under `events-v2.jsonl` with explicit CLI selection.
- [x] 2.3 Add append-only flow-phase, context/basis, and telemetry evidence payloads with transition, correction, privacy, projection, and mixed-version rejection tests.

## 3. Langfuse Reconciliation and Overhead

- [x] 3.1 Add a provider-neutral trace-source port and deterministic reconciliation that validates session identity, groups observation envelopes by trace, applies the flow window, deduplicates, and returns explicit unavailable/conflict outcomes.
- [x] 3.2 Implement the bounded stdlib Langfuse HTTP reader for the verified observations-v2 contract with cursor pagination, response limits, timeout injection, strict decoding, credential redaction, and `httptest` coverage.
- [x] 3.3 Add v2 operator/CLI reconciliation and derived overhead recording for flow and baseline while retaining separate machine and owner-review components and forbidding caller-authored v2 totals.

## 4. Structured Owner Assessment

- [x] 4.1 Add assessment-basis and item-judgment domain validation with deterministic aggregate consistency for `match`, `partial`, and `miss`.
- [x] 4.2 Extend v2 operator and CLI assessment flows, append-only correction behavior, inspect/report projection, and tests for incomplete, duplicate, foreign, inconsistent, and valid judgments.

## 5. Durable Documentation and Proof

- [x] 5.1 Record the context/evaluation decision in an ADR and add one project lifecycle document identifying artifact ownership, transitions, and owner gates.
- [x] 5.2 Publish current v2 operator guidance while preserving v1 history and documenting explicit Langfuse credential/use and missing-telemetry behavior.
- [x] 5.3 Add one deterministic synthetic v2 flow case and one baseline case with retained JSONL/report/validation artifacts and rollback assertions proving no real activation or external Langfuse write.
- [x] 5.4 Run focused tests, `go vet ./...`, `go test -race ./...`, `git diff --check`, and strict OpenSpec validation; record any unsupported provider shape without weakening fail-closed behavior.
