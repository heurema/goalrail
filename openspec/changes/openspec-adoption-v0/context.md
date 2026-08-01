# Context Pack

- **Context Pack ID:** CTXP-openspec-adoption-v0
- **Version:** 1
- **Previous version:** none
- **Started at:** 2026-08-01T08:20:00Z
- **Completed at:** 2026-08-01T09:10:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | `EnsureConfig` rewrites only the `schema:` line and leaves every other byte of the configuration alone; the edit is line-level precisely so project prose and rules survive | `internal/harness/config.go:53-113` | 2026-08-01T08:20:00Z | Any adoption surface must keep this boundary; a parse-and-reserialize merge would reformat content Goalrail does not own |
| CTX-2 | repository | Initialization checks the configuration before the first write, so a refusal leaves the repository as it was | `cmd/gr/ambient.go:117-122` | 2026-08-01T08:20:00Z | A richer decision surface must not become a wizard that edits while it asks |
| CTX-3 | repository | Only `spec-driven` counts as a stock schema; every other named schema is foreign and requires `-confirm-schema-switch` | `internal/harness/config.go:23,94` | 2026-08-01T08:20:00Z | Defines exactly which repositories ever reach this problem |
| CTX-4 | repository | Measured on Baseline: switching `intent-driven` → `goalrail-intent` changed exactly one line; the `context:` block and every `rules:` entry survived byte-for-byte; `openspec validate --all` then passed 7 of 7 | Baseline `git diff openspec/config.yaml`; validate run | 2026-08-01T08:35:00Z | The mechanics are sound. The gap is disclosure, not damage |
| CTX-5 | repository | Baseline's surviving rules now disagree with the schema they sit on: one names status `draft`, which `goalrail-intent` does not define (it uses `candidate`/`confirmed`); others require the `$grilling` skill and an atomic human-intent ledger, neither of which appears anywhere in the `goalrail-intent` instruction | Baseline `openspec/config.yaml` vs `openspec/schemas/goalrail-intent/schema.yaml` | 2026-08-01T08:35:00Z | This is the concrete harm: rules keep instructing agents after the schema they were written for is gone |
| CTX-6 | repository | `openspec validate --all` passed with those contradictory rules present, so rules are not validated against the active schema | Baseline validate run | 2026-08-01T08:45:00Z | Nothing downstream will ever catch stale rules. If Goalrail does not report them, nothing does |
| CTX-7 | repository | Rules exist only in `config.yaml`. They are not materialized into any instruction file and appear in no other artifact in either repository | Full-text search for exact rule strings across Baseline and Goalrail | 2026-08-01T09:05:00Z | Stale rules mislead by being read, not by being compiled. The remedy is disclosure, not regeneration |
| CTX-8 | repository | Every change pins its own schema in its `.openspec.yaml`; Baseline's open change still names `intent-driven` and validated after the switch | `internal/adapters/openspec/adapter.go:165`; Baseline `changes/own-image-receipt-migration/.openspec.yaml` | 2026-08-01T08:45:00Z | Deleting a superseded schema directory breaks every change that still names it, so "delete it" is usually wrong advice |
| CTX-9 | external | OpenSpec 1.6.0 `schema` offers `which`, `validate`, `fork` and `init`. There is no archive, deprecate or delete | `openspec schema --help`, pinned CLI 1.6.0 | 2026-08-01T09:00:00Z | There is no supported "archive the old schema" operation to call. Whatever Goalrail recommends has to be expressible as files a user already owns |
| CTX-10 | external | `openspec schema fork <source> [name]` copies an existing schema into the project for customization | `openspec schema --help`, pinned CLI 1.6.0 | 2026-08-01T09:00:00Z | A user who wants Goalrail's artifacts plus their own rules already has a supported path. The recommendation may be to fork rather than to build a merge feature |
| CTX-11 | repository | The initialization report named what it created and what it switched, and said nothing about rule drift or about `context` becoming a required artifact | `gr init` report on Baseline | 2026-08-01T08:32:00Z | This is the observable defect the change exists to remove |
| CTX-12 | repository | `gr update` brings a repository's overlay up to the installed binary's canon; it does not update the binary and makes no network request | `cmd/gr/harness.go:119-125`; measured `already_current: true` on Baseline | 2026-08-01T08:58:00Z | If adoption reporting should recur rather than happen once, `doctor` and `update` are the surfaces that already run again |
| CTX-13 | repository | `goalrail-intent` adds a required `context` artifact, stable intent IDs, `Intent Coverage` in the proposal, and per-requirement `**Intent IDs:**` citations; it drops the `$grilling` instruction and the human-intent ledger, and changes the status vocabulary | `diff` of `intent-driven` and `goalrail-intent` schema definitions | 2026-08-01T08:15:00Z | This diff is exactly what a user needs told at adoption time, and is derivable mechanically from the two schema files |

## Material Unknowns

None.
