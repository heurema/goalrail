# Intent Canary v0 Repository Evidence Privacy Audit

- **Audited:** 2026-07-21
- **Scope:** every current production caller that can append
  `canary/events.jsonl`
- **Verdict:** `PASS` for the internal, non-sensitive v0 eligibility boundary
- **Not covered:** client, sensitive, production, or externally retained content;
  those remain ineligible under manifest v1

## Writer inventory

Code-graph inbound tracing found one repository writer:
`internal/evidence.Store.Append`. The only production event constructors that
reach it are:

| Constructor | Stored projection |
|---|---|
| `operator.Service.Start` | Assignment ordinal, variant, manifest/intent version, generated run ID, and synthetic flag |
| `operator.Service.appendCorrelation` | Validated lineage projection and bounded Langfuse identity references |
| `operator.Service.Deliver` | Terminal category, check references, green flag, and optional numeric flow overhead |
| `operator.Service.Abandon` | Terminal category, categorical reason, process flag, and optional numeric flow overhead |
| `operator.Service.Assess` | Owner ID, categorical intent outcome, derived green/material flags, timestamp, and optional repeat choice |
| `operator.Service.CorrectAssessment` | Same bounded assessment projection plus categorical reason and superseded event ID |
| `operator.Service.RecordMaterialCorrection` | Owner ID, categorical reason, timestamp, and bounded source reference |
| `operator.Service.Disable` | Canary ID/version, categorical stop reason, timestamp, actor, and bounded source reference; no change or session identity |

No other production `os.OpenFile` caller writes the evidence path. Tests may
write temporary files to verify tamper detection; they are not emission paths.

## Approved serialized schema

The JSONL envelope contains only:

- schema version, sequence, previous digest, and digest;
- canonical event, canary, change, actor, run, session, and correction IDs;
- enumerated event, lineage, identity-source, terminal, outcome, and variant
  values;
- non-zero UTC timestamps, booleans, bounded counters, and finite non-negative
  overhead;
- a lowercase SHA-256 context digest;
- field-specific references described below.

Assignment, lineage, terminal, and assessment are closed typed payloads. The
reader rejects unknown JSON fields. An event kind accepts exactly its permitted
payload, so no generic metadata map or free-form text field crosses the store
boundary.

### Reference allowlist

Repository event sources accept only these schemes with a canonical metadata
identifier:

```text
codex-app, codex-hook, git, github, hook, launch-receipt,
openspec, owner-review, request, review
```

Check references accept only `check`, `ci`, or `test`, again with a canonical
metadata identifier. Observation references accept only:

- `langfuse-session:<canonical-session-id>`;
- `langfuse-trace:<32-lowercase-hex-trace-id>`.

Paths, URLs, whitespace, multiline data, arbitrary schemes such as `prompt` or
`transcript`, secret-shaped fragments, and overlong values fail before append.
Reason and actor fields are canonical categorical tokens, not prose.

## Provider-boundary verification

Codex lifecycle-hook and App launch inputs are decoded into a fixed identity
projection. Raw provider JSON is never attached to an `EvidenceEvent`.

The serialized-output tests inject transcript paths, prompts, authorization or
token-shaped fields, titles, and other source content into both provider
surfaces. They verify that the exact JSONL bytes retain only the authoritative
session/thread identity and contain none of the injected fields or values.

## Test evidence

| Boundary | Test evidence |
|---|---|
| Credential, whitespace, raw payload, and free-form reason rejection | `internal/evidence.TestStoreRejectsSensitiveOrRawPayload` |
| Per-field source/check/observation allowlists | `internal/evidence.TestStoreAllowsOnlyApprovedRepositoryReferenceKinds` |
| Unknown stored field rejection | `internal/evidence.TestStoreRejectsUnknownStoredFields` |
| Hook projection excludes prompt and transcript fields | `internal/adapters/codex.TestHookPayloadAndTranscriptFieldsAreNotRetained` |
| Full CLI lifecycle JSONL excludes raw hook payload | `cmd/goalrail-canary.TestCommandRunsSyntheticLifecycleWithoutManualSessionID` |
| Rollback marker is append-only and repository-local | `internal/integration.TestDisableIsRepositoryLocalAndLeavesUnrelatedAdaptersUsable` |
| App JSONL excludes title, prompt, transcript, and token fields | `internal/operator.TestLaunchReceiptRepositoryEvidenceKeepsOnlyApprovedIdentity` |
| Record-size and digest-chain enforcement | `internal/evidence` store suite |

## Residual boundary

Software cannot determine whether a human intentionally mislabels prose as a
short canonical ID. V0 therefore combines structural enforcement with a narrow
operator contract: references name existing evidence; they do not contain it.
The manifest separately excludes sensitive work and shared-retention boundary
violations. Widening eligibility requires a new privacy decision, not a looser
identifier parser.
