## 1. Freeze the conformance oracle

- [x] 1.1 Define the hand-authored `internal/conformance/testdata/artifact-contract-v1/manifest.json` schema and add contract-v1 plus minimized `openspec-adoption-v0`/`pre-pr-review-v0` binding positives with fixed valid Proposal inputs, provenance, modes, lanes, per-file SHA-256, and exact semantic projections.
- [x] 1.2 Add pinned preservation positives for six/seven-column Context Items, old/current headings, answering-intent `Resolves`/`Disposition`, Verification recipe, Previous version, and the repaired real pair with independently written expectations.
- [x] 1.3 Add pinned negative fixtures for partial/malformed/mismatched/unknown contracts, Context identity/version and referenced-evidence failures, hostile diagnostic content, and malformed-confirmed ambient coexistence without deriving expectations from consumer output.
- [x] 1.4 Add manifest self-validation tests that verify every fixture digest before lane execution and reject duplicate case IDs, unknown modes or lanes, unsafe fixture paths, missing provenance or expectations, tampered fixtures, and non-deterministic ordering; provide no golden updater.

## 2. Implement pair contract selection and diagnostics

- [x] 2.1 Add the exact `goalrail-context-intent` version 1 declarations, `legacy-v0` selection metadata, and a table-driven selector that accepts only equal complete contract-v1 declarations or complete absence in both artifacts and rejects partial, malformed, mismatched, unknown, duplicated, or unlisted metadata without fallback while preserving all supported lifecycle keys including `Previous version`, `Run references`, `Resolves`, and `Disposition`.
- [x] 2.2 Add the finite `ArtifactDiagnostic` code registry, explicit `unselected` contract mode for selector failures, structured fields, `error`/`Unwrap` behavior, deterministic rendering, safe logical-path validation, bounded observation escaping, and secret-shaped redaction with hostile-input unit tests.
- [x] 2.3 Refactor Context/Intent parsing behind one bounded pair API that selects a profile before semantic parsing, removes error-driven profile fallback, returns contract selection separately from unchanged domain values, and preserves the existing legitimate intent-only reader path.
- [x] 2.4 Add focused pair tests proving contract-v1 parsing, finite legacy parsing, `vN`/`version N` semantic equality, Context identity/version rejection, no declared-contract downgrade, six/seven-column and old/current-heading behavior, and lossless retained fields.

## 3. Wire compiler and local-run lanes

- [x] 3.1 Route `LoadChange` through the bounded pair API while preserving context-required detection, confirmed-status validation, proposal loading, and proposal-coverage validation; expose an optional `ConformedPair` in `CompiledChange`, return pair diagnostics unchanged, and keep compiler evidence distinct from runtime evidence.
- [x] 3.2 Add `IntentResolver.Resolve` to return a resolved intent with optional `ConformedPair` after the existing containment, regular-file, size, digest, status, and reference checks; make `Verify` delegate to it without changing the local-run verifier interface, and leave archived/foreign-schema intent-only behavior unchanged.
- [x] 3.3 Add local-run service tests proving a pair diagnostic is inspectable with `errors.As`, no prepared state, run ID, launch claim, or provider invocation exists after failure, and valid contract-v1 and pinned legacy pairs still prepare normally.

## 4. Make ambient invalid-confirmed evidence visible but nonblocking

- [x] 4.1 Replace ambient's unconditional intent-only discovery with bounded envelope-aware discovery and an `IntentResolution` value carrying reference, unbound reason, and sorted binding diagnostics; require a sibling pair for the Goalrail schema, declared Context binding, present Context, or any artifact-contract field, preserve the existing intent-only path when none applies, and use exact top-level confirmed-status inspection so an invalid confirmed claimant forces `INVALID_CONFIRMED_INTENT` while candidate or incomplete WIP is not promoted.
- [x] 4.2 Advance newly written question records to `goalrail.ambient-question/v1`, add optional structured `binding_diagnostics` correlated by change name, and copy discovery evidence into unbound records without rewriting existing v0 records or minting run outcomes.
- [x] 4.3 Add ambient resolver, `StopSession`, serialization, and hook tests proving malformed confirmed input cannot hide behind a valid intent, multiple diagnostics are deterministic and redacted, valid single intent still binds, WIP behavior is preserved, and every diagnostic or internal failure remains invisible and nonblocking to the session.

## 5. Execute all corpus lanes independently

- [x] 5.1 Implement `TestArtifactConformanceCorpus` with separate adapters for `LoadChange`, `IntentResolver.Resolve`, and ambient discovery; materialize pair and fixed Proposal fixtures in temporary repositories and compare each declared case-lane result to its hand-authored semantic projection or complete structured diagnostic.
- [x] 5.2 Make the corpus verify every pinned fixture digest before lanes and fail on tampering, a missing or skipped declared lane, unexpected pass, changed diagnostic, changed preserved field, or dynamic dependence on unrelated active/archive changes, then verify two runs at one revision produce the same ordered matrix.

## 6. Emit contract v1 from current canon

- [x] 6.1 Before editing current canon, copy its exact overlay into a new append-only previous-canon fixture, append its computed identity and per-file digests after the existing entry, and add a migration test while leaving `internal/harness/testdata/canon-v1` and its recorded digests byte-identical.
- [x] 6.2 Add the two exact contract rows to current Context and Intent templates in `openspec/schemas/goalrail-intent` and `internal/harness/canon`, then update both mirrored schema instructions and forcing-function assertions.
- [x] 6.3 Verify current canon byte identity, pinned OpenSpec CLI acceptance, deterministic canon digest, every previous-canon transition, and generation of a contract-v1 pair that produces equal compiler and local-run semantics.

## 7. Verify the bounded slice

- [x] 7.1 Run formatting, targeted adapter/conformance/local-run/ambient/harness tests, race tests for every touched package, `go vet ./...`, and `go test ./...`; fix failures without widening legacy syntax or weakening diagnostics.
- [x] 7.2 Run strict pinned OpenSpec validation and diff audit, confirm all manifest cases and lanes pass, confirm no runtime dependency or external effect was added, and confirm active/archive artifacts plus historical `canon-v1` remain byte-identical.
