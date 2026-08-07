# Goalrail v0.2 contracts

This document is the operator registry for schemas, stable machine-readable
reasons, exit categories, ownership, evidence replication, and the v0.2
compatibility window. Human descriptions may improve; identifiers and their
semantic boundaries are the automation contract.

## Schema registry

### Project governance

| Schema | Owner and purpose |
|---|---|
| `goalrail.project/v1` | Repository-owned project identity and digest-bound references to policy, bootstrap, and setup profile |
| `goalrail.bootstrap/v1` | Canon-owned human/agent bootstrap contract stored as exact committed Markdown bytes |
| `goalrail.policy/v1` | Repository policy for materiality, required lineage, authority, exceptions, and bootstrap behavior |
| `goalrail.setup-profile/v1` | Repository-selected compatible Goalrail bundle, exact planning compiler/runtime, scaffold adapters, and prepared shared-admission adapter |
| `goalrail.project-canon/v1` | Binary-owned manifest of known canon and managed-block inputs; it does not claim repository-owned surrounding content |

### Setup and release

| Schema | Owner and purpose |
|---|---|
| `goalrail.release-metadata/v1` | Public bounded current-release discovery metadata; discovery only, never an artifact integrity identity by itself |
| `goalrail.setup-source-lock/v2` | Release-build source lock for runtime/compiler inputs and supported platforms, the reviewed dependency closure (package count, install-script count, digest), and the upstream publication and adoption dates of each pin. Supersedes v1, which carried neither the closure nor the dates; the document is a build input and is not published in a release |
| `goalrail.setup-bundle-manifest/v1` | Immutable per-release/per-platform setup archive manifest with component, file, license, provenance, and binary identities |
| `goalrail.setup-plan/v1` | Canonical read-only plan binding exact components, mutations, prerequisites, trust steps, network, rollback, verification, and zero project-code writes |
| `goalrail.plan-authorization/v1` | Explicit owner decision bound to one complete setup-plan digest |
| `goalrail.setup-plan-verification/v1` | Re-planning and temporary-bundle equality report immediately before apply |
| `goalrail.setup-apply-result/v1` | Bounded apply-attempt result and recovery locator |
| `goalrail.setup-receipt/v1` | Durable semantic setup receipt, including plan, authorization, mutation, diagnosis, status, and continuation references |
| `goalrail.setup-receipt-emission/v1` | Canonical receipt plus its digest at the CLI emission boundary |
| `goalrail.setup-diagnosis-summary/v1` | Bounded doctor facts retained by a setup receipt without copying an unbounded report |
| `goalrail.setup-recovery/v1` | Private exact prior-state set for one setup attempt; never repository evidence or a public artifact |
| `goalrail.setup-rollback-result/v1` | Result of applying one exact private recovery set |

### Work and lineage

| Schema | Owner and purpose |
|---|---|
| `goalrail.work-spec/v1` | Frozen new-run assignment bound to project, confirmed intent/change, and work unit |
| `goalrail.work-unit/v1` | Stable anchor for one bounded piece of work and its required relations |
| `goalrail.lineage-event/v1` | Immutable typed relation between bounded content-addressed references |
| `goalrail.lineage-exception/v1` | Explicit scoped and expiring bootstrap, exemption, or break-glass authority evidence |
| `goalrail.change-snapshot/v1` | Deterministic digest manifest for a bounded Git change-artifact tree |

### Admission and provider evidence

| Schema | Owner and purpose |
|---|---|
| `goalrail.admission-packet/v1` | Temporary collector output containing frozen range identity and bounded evidence/provenance references; never policy authority |
| `goalrail.admission-result/v1` | Canonical provider-neutral classification, allow/deny outcome, stable reasons, and missing/conflicting references |
| `goalrail.github-pull-request/v1` | Canonical bounded GitHub pull-request identity replica |
| `goalrail.github-review-index/v1` | Canonical bounded GitHub review-state replica without review bodies |
| `goalrail.github-check-set/v1` | Canonical bounded required-check replica without logs |
| `goalrail.github-owner-decision/v1` | Canonical bounded GitHub owner-decision reference without comment bodies |
| `goalrail.github-closure/v1` | Canonical bounded pull-request closure/merge reference |
| `goalrail.doctor/v2` | Read-only layered diagnosis of project claim, governance, local readiness, lineage readiness, and protected shared admission |

Provider evidence schemas are adapter outputs, not Goalrail domain authority.
GitHub types do not enter project, WorkSpec, lineage-event, or admission-result
schemas.

## Stable reason codes

### Project declaration claim

The `goalrail.doctor/v2` `claim.reason` field uses:

- `DECLARATION_UNREADABLE`
- `DECLARATION_UNSAFE_PATH`
- `DECLARATION_NOT_REGULAR`
- `DECLARATION_TOO_LARGE`
- `DECLARATION_MALFORMED`
- `DECLARATION_NON_CANONICAL`
- `DECLARATION_CHANGED`

An absent canonical declaration has claim state `unmanaged` and no declaration
error reason. Any present but invalid reserved path fails closed as
`declared_invalid`.

### Setup-plan incompleteness

`goalrail.setup-plan/v1.incomplete_reason_ids` uses:

- Release/platform: `RELEASE_METADATA_UNAVAILABLE`,
  `RELEASE_METADATA_INVALID`, `RELEASE_INCOMPATIBLE`, `PLATFORM_UNSUPPORTED`.
- Manifest: `SETUP_MANIFEST_UNAVAILABLE`, `SETUP_MANIFEST_DIGEST_MISMATCH`,
  `SETUP_MANIFEST_INVALID`, `SETUP_MANIFEST_RELEASE_MISMATCH`,
  `SETUP_MANIFEST_COMPONENT_INVALID`.
- Bundle destination: `BUNDLE_DESTINATION_CONFLICT`,
  `BUNDLE_DESTINATION_UNSAFE`.
- Executable destination: `EXECUTABLE_DESTINATION_CONFLICT`,
  `EXECUTABLE_DESTINATION_UNREADABLE`, `EXECUTABLE_DESTINATION_UNSAFE`.
- Scaffold: `SCAFFOLD_UNSUPPORTED`, `SCAFFOLD_CONFIG_INVALID`,
  `SCAFFOLD_TARGET_UNAVAILABLE`, `SCAFFOLD_TARGET_UNSAFE`.

Any reason makes the plan `incomplete` and therefore non-executable. Detail text
is deliberately outside the authorization-bound reason registry.

### Admission

`goalrail.admission-result/v1.reasons[].code` uses:

- `ACTIVATION_UNVERIFIED`
- `BOOTSTRAP_RANGE`
- `CHANGE_MISMATCH`
- `CHECK_MISSING`
- `DECLARATION_INVALID`
- `EXCEPTION_APPLIED`
- `EXCEPTION_EXPIRED`
- `EXCEPTION_SCOPE_MISMATCH`
- `GENERATED_EVIDENCE_MISSING`
- `INTENT_UNCONFIRMED`
- `LINEAGE_CONFLICT`
- `MATERIAL_PATH_UNBOUND`
- `OWNER_DECISION_MISSING`
- `PACKET_INVALID`
- `POLICY_CONFLICT`
- `PROVENANCE_UNTRUSTED`
- `PULL_REQUEST_MISSING`
- `RANGE_MISMATCH`
- `RECEIPT_MISSING`
- `REVIEW_MISSING`
- `RUN_SESSION_MISSING`
- `TRUSTED_TIME_MISSING`
- `WORK_SPEC_MISSING`

Admission classifications are `VALID`, `MISSING`, `AMBIGUOUS`, `INVALID`,
`EXEMPTED`, `BREAK_GLASS`, and `BOOTSTRAP`. Classification and outcome are
separate: `VALID` is normal conformance; explicitly authorized exception
classes may allow integration but never become `VALID`.

### Doctor

Top-level doctor reasons are:

- Identity/setup: `PROJECT_UNMANAGED`, `GOALRAIL_PROJECT_MIGRATION_REQUIRED`,
  `PROJECT_DECLARATION_INVALID`, `GOALRAIL_SETUP_REQUIRED`.
- Governing artifacts: `GOVERNING_ARTIFACT_INVALID`, `PROJECT_CANON_DRIFT`.
- Canon states: `PROJECT_CANON_MISSING`, `PROJECT_CANON_BEHIND`,
  `PROJECT_CANON_DRIFT`.
- Overlay states: `HARNESS_OVERLAY_MISSING`, `HARNESS_OVERLAY_BEHIND`,
  `HARNESS_OVERLAY_EDITED`, `HARNESS_OVERLAY_SUPERSEDED`.
- Planning: `PLANNING_SCHEMA_NOT_READY`, `PLANNING_COMPONENT_MISSING`,
  `PLANNING_COMPONENT_INCOMPATIBLE`,
  `PLANNING_COMPONENT_UNVERIFIED_INTEGRITY`, `PLANNING_COMPONENT_INVALID`.
- Attachment: `ATTACHMENT_INVALID`, `ATTACHMENT_UNREADABLE`,
  `ATTACHMENT_NOT_READY`.
- Shared admission: `SHARED_ADMISSION_INACTIVE`, `SHARED_ADMISSION_UNKNOWN`,
  `SHARED_ADMISSION_STALE`, `SHARED_ADMISSION_ACCESS_DENIED`.

Each emitted reason includes its layer, observed state, optional exact path, and
an actionable next step. Optional update, review, and observability facts do not
change the aggregate verdict.

## Exit categories

### `gr doctor`

| Exit | Category | Meaning |
|---:|---|---|
| 0 | `fully_enforced` | Managed, locally ready, lineage-ready, and protected shared admission independently verified |
| 3 | `unmanaged`, `migration_required` | No current declaration, or exact legacy evidence needs explicit migration |
| 4 | `declared_invalid`, `governance_invalid` | A reserved claim or declaration-bound governance artifact is invalid |
| 5 | `setup_required` | A declared local component or supported attachment requires planned setup |
| 6 | `local_not_ready` | Setup is present but another required local readiness layer is not ready |
| 7 | `locally_ready_advisory` | Local and lineage layers are ready, but protected shared admission is not proven |
| 2 | no diagnosis | Usage, read, or internal failure prevented a trustworthy report |

### `gr verify-lineage`

| Exit | Meaning |
|---:|---|
| 0 | Canonical admission result has outcome `allow` |
| 3 | Canonical admission result has outcome `deny` |
| 2 | Usage, packet collection, Git-range freezing, verification, or output failed; no admission conclusion may be inferred |

`gr verify-lineage` reads a packet at rest. A serialized packet cannot carry an
authenticated provider observation, so this path is advisory for every
provider-owned relation and returns `OWNER_DECISION_MISSING` where a live
approval is required. The authoritative shared path observes the provider and
verifies the frozen range inside one trusted invocation; the packet it emits is
audit evidence, never the authority.

Other public commands currently use conventional success/error exits and must
not be treated as stable admission categories.

## Ownership boundaries

| Artifact/state | Owner | Boundary |
|---|---|---|
| `.goalrail/project.json` and repository selections | Repository | Portable claim committed once; contains no user identity, credential, absolute path, or local attachment state |
| Policy, bootstrap, setup profile, adapter descriptors, prepared workflow | Goalrail canon plus declared repository selection | Exact known bytes or bounded managed blocks; surrounding owner content is preserved |
| Intent/change artifacts and project code | Repository and owner-confirmed workflow | Goalrail validates bindings; it does not infer intent or gain write/commit authority from confirmation |
| Setup plan and authorization | Planner, then owner | Plan is read-only; authorization covers only one exact complete digest and one bounded attempt |
| Local executable/runtime/scaffold config | Local user/machine | Never committed as project identity; writes require exact setup consent and just-in-time byte checks |
| Work-unit events | Repository Git history | Append-only canonical evidence; conflicts stay visible |
| Local run store | Local Goalrail runtime | Original canonical WorkSpec/run/receipt ownership; not directly available to shared CI |
| Content-addressed replica store | Repository Git history | Exact validated copies only; replica does not become a new semantic owner |
| Admission packet | Registered collector | Evidence input only; cannot supply project policy or rewrite repository authority |
| Admission result | Pure Goalrail verifier | Deterministic conclusion over frozen input; provider adapters cannot change semantics |
| Provider protection/trust | External provider and owner | A local file can prepare an adapter but cannot prove activation or grant provider trust |

## Content-addressed replica rule

`gr lineage attach` may copy a bounded canonical receipt or evidence object into
`.goalrail/evidence/sha256/<digest-component>` only when:

1. the exact bytes decode under the declared supported schema;
2. their SHA-256 digest equals the declared digest;
3. the lineage event references that digest;
4. the payload contains no raw prompt, transcript, review/comment/source body,
   credential, token, secret-shaped value, or unsupported structure; and
5. an existing object at that digest is byte-identical.

The replica retains the original artifact identity. Goalrail never summarizes,
rewrites, relabels, or stores two payloads for one digest. Identical attachment
is idempotent; mismatch is a visible conflict. Historical event and replica
bytes are immutable—rollback uses a superseding/closure event or a Git revert,
not in-place editing.

## One-release compatibility window

The v0.2 writer emits only current project/setup/WorkSpec/lineage/admission
schemas and doctor v2. Readers retain exact v0.1.8 marker/overlay evidence,
legacy WorkSpecs, and legacy receipts for migration and inspection. Legacy
evidence is never rewritten or reported as normally `VALID`; a committed
bootstrap policy may classify an exact retained range as `BOOTSTRAP`.

`goalrail.doctor/v2.initialized` is deprecated for one release and mirrors only
`managed`, meaning a valid committed declaration. It never reads
`.goalrail/ambient.json` and says nothing about local readiness or protected
admission. Consumers must move to `managed`, `locally_ready`, `lineage_ready`,
and `shared_admission_active` before the alias is removed.

There is no silent auto-migration. Reverting code after v1 project artifacts
are committed may require a compatible binary or an explicit project-commit
revert; an older binary must not reinterpret an unsupported declaration as
unmanaged.
