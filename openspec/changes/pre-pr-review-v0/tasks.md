# Reconciliation Ledger

The previous 50-item checklist described several superseded intent versions and
was no longer a trustworthy apply queue. This ledger preserves its disposition
without relabeling failed historical proof as success. Only the checkboxes below
this table are active implementation tasks.

| Disposition | Previous task IDs | Reconciliation result |
|---|---|---|
| Implemented | 1.1, 1.4, 3.1–3.4, 4.1–4.5, 5.2–5.4, 6.1–6.5, 7.1–7.4, 9.1–9.6, 10.1–10.4, 11.1–11.5 | Existing code and tests implement these behaviors; do not reopen them except where an active repair task below requires a narrow edit |
| Superseded | 1.3, 1.5, 2.1–2.3, 5.1 | Later confirmed intent replaced author refusal, installed-only selection, and the unusable original Codex invocation contract |
| Partial | 1.2, 7.5 | Primary-marker detection exists but misses `CODEX_THREAD_ID`; receipt-state unit coverage exists but not the public command flow |
| Historical proof failed | 8.1–8.4 | The Baseline run did not finish with a current receipt on the PR's final head before publication; retain it as failed evidence and use the replacement canary below |

## 1. Repair authorship detection

- [x] 1.1 Replace the one-marker-per-provider table with registered marker sets: keep `CLAUDECODE`, add `CODEX_THREAD_ID`, retain `CODEX_SESSION_ID` as a compatibility marker, and keep companion variables non-author evidence.
- [x] 1.2 Collapse multiple non-empty markers for the same provider to one identity, preserve `unknown` for zero or multiple detected providers, preserve unconditional valid-override precedence, and strip every registered and companion marker from reviewer child environments.
- [x] 1.3 Add table tests for real Codex desktop detection without `--author`, both Codex markers together, the Claude-plus-companion case, empty markers, cross-provider ambiguity, invalid and valid overrides, and the complete stripped-marker set.

## 2. Resolve the review base safely

- [x] 2.1 Remove the semantic `main` default from `--base` and add a resolver result that carries the selected human-readable ref and its immutable commit while distinguishing omitted input from an explicit value.
- [x] 2.2 Make explicit `--base` authoritative; otherwise accept exactly one locally resolvable `refs/remotes/*/HEAD` symbolic target, prefer it over a same-named local branch, and return an input error naming `--base` for zero or multiple targets without fetch or hosting-service access.
- [x] 2.3 Resolve repository root, attached branch, selected base, immutable base/head commits, and a non-empty range before instructions materialization, the gate, or any provider invocation; use immutable commits for every reviewed range while retaining the selected ref in the receipt.
- [x] 2.4 Add Git fixtures for explicit precedence, divergent local and remote-tracking defaults, absent and multiple remote HEADs, unresolvable refs, and invalid input invoking neither a gate nor reviewer; make the offline case fail fast even when the configured remote URL is unreachable.

## 3. Give the refuter the exact reviewed diff

- [x] 3.1 Render the canonical reviewed-range diff once after commits are frozen and reuse those exact bytes for its digest, the Claude reviewer input where applicable, and every refute prompt without changing Codex's native custom-instruction invocation.
- [x] 3.2 Extend refute tests so the stub receives the first report plus byte-identical canonical diff content whose digest is stored in the receipt, while both reports remain verbatim and no finding field or decision is derived.

## 4. Prove the public no-terminal contract

- [x] 4.1 Add a command-level fixture that drives initialization, `gr review`, and diagnosis through the real `gr` dispatcher with a real Git repository, stub provider executables, buffered streams, and no terminal; decode every successful result as JSON and assert no prompt.
- [x] 4.2 In that fixture, prove review → current, commit → stale, re-review → current, and corrupted receipt → unreadable while the doctor verdict and exit status remain determined only by independent repository health.
- [x] 4.3 Cover malformed and ambiguous base input through the same no-terminal command surface, asserting the same input-error semantics, no prompt, and no gate or provider invocation.

## 5. Complete deterministic verification

- [x] 5.1 Run formatting and the focused review/CLI suites with an explicit five-minute Go test timeout, record their elapsed time, and fix any deterministic failure without widening the change.
- [x] 5.2 Run `go test -timeout 5m ./...` and `go vet ./...`; stop at the first conclusive failure rather than starting the real canary.
- [x] 5.3 Run `git diff --check` and strict pinned OpenSpec validation, inspect the final diff for unrelated or generated files, and mark only tasks whose stated verification actually passed.

## 6. Run the owner-gated replacement canary

- [ ] 6.1 Before any commit or paid provider call, obtain separate owner authorization for the exact validated head and record in `evidence/reconciliation-canary-2026-08-04.md`: branch, head, commands, maximum three `gr review` rounds, maximum four provider invocations, final-full-pass requirement, and the no-PR failure stop.
- [ ] 6.2 Run the first ordinary review from the real Codex desktop environment without `--author` or `--base`; record non-sensitive receipt fields, duration, selected author/reviewer/mode, base ref/commit, head, and report digests without copying raw reports or local absolute paths into repository evidence.
- [ ] 6.3 If findings exist and the caller is about to act, run at most one `--refute` command before fixing and disposition its two verbatim reports outside Goalrail; if no findings exist, skip the refute command and record why no provider invocation was spent.
- [ ] 6.4 Apply only confirmed in-scope findings, rerun deterministic verification, and obtain separate commit authorization for any changed final head; a real intent divergence or hole stops for the single owner question, while an improvement idea is recorded outside this change.
- [ ] 6.5 Run the final explicit `--full` review on the committed final head and record success only when its receipt is current and the caller finds no material issue left; reaching either ceiling or retaining a material finding records failed canary evidence and stops before push or PR.
