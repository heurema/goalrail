# `goalrail.doctor/v2`

`gr doctor --json` reports project identity, local readiness, lineage-policy
readiness, and protected shared admission as independent layers.

The complete schema and reason registry is in
[`../contracts-v0.2.md`](../contracts-v0.2.md). Installation and migration
procedures are intentionally separate from this read-only report.

The aggregate `working` field is true only when all four required layers are
verified. A committed valid `.goalrail/project.json` sets `managed=true` in
every clone and linked worktree. Missing runtimes, compilers, scaffold
attachments, or provider trust leave that identity intact and produce
`GOALRAIL_SETUP_REQUIRED` where setup is the applicable remedy.

## `initialized` compatibility window

`initialized` is deprecated for one release. In v2 it mirrors `managed`, which
means valid committed declaration presence. It no longer reads
`.goalrail/ambient.json` and no longer means that a checkout-local hook or
runtime is ready. Consumers must migrate to `managed`, `locally_ready`,
`lineage_ready`, and `shared_admission_active` before the field is removed.

The report carries `initialized_deprecated=true` and an
`initialized_semantics` description so a machine consumer cannot silently
retain the old marker interpretation.

## Shared admission

A prepared repository adapter or workflow file is not activation evidence.
`shared_admission_active=true` requires current read-only evidence naming the
provider or boundary, required check, protected target, evidence source,
observation time, and freshness. Missing access is `unknown` or
`access_denied`; expired evidence is `stale`.

## Exit categories

| Exit | Category |
| ---: | --- |
| 0 | `fully_enforced` |
| 3 | `unmanaged` or `migration_required` |
| 4 | `declared_invalid` or `governance_invalid` |
| 5 | `setup_required` |
| 6 | `local_not_ready` |
| 7 | `locally_ready_advisory` |
| 2 | Diagnosis did not run |

Every non-working required layer also emits a stable reason code and an exact
next action. Optional observability, review, and release facts do not change the
aggregate verdict.
