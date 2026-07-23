## Context

Goalrail already has a canary-specific Go operator, immutable Codex run context, provider-authoritative lifecycle correlation, frozen verification checks, and append-only evidence. Those mechanisms prove useful invariants, but the canary manifest and evidence model are not a general contract for one coding-agent task.

This change adds the smallest separate local-run path:

```text
intent artifact -> WorkSpec preparation -> explicit start -> provider adapter
                -> provider observation -> operator verification -> receipt
```

The canonical WorkSpec and receipt remain provider-neutral. Codex is the first adapter only. Its sandbox and interactive approval UI remain the enforcement boundary for a later authorized real run.

The target repository may already contain unrelated local changes. The flow therefore needs a read-only worktree baseline before launch so that later evidence can identify run-time changes without requiring a clean worktree, copying source content, or performing Git writes.

Planning and implementation remain inert. The shipped v0 must not invoke a real provider until a later owner-authorized activation change deliberately wires that path.

## Goals / Non-Goals

### Goals

- Define one strict, versioned JSON WorkSpec for exactly one trusted local repository run.
- Produce a deterministic digest from provider-neutral semantic content.
- Validate confirmed intent, pinned repository state, path containment, bounded content, checks, stop conditions, and the fixed v0 posture before preparation succeeds.
- Keep preparation inspectable and free of provider launches or target-worktree mutations.
- Make one successful start claim permit at most one adapter invocation, including under concurrent start attempts or process failure.
- Preserve provider enforcement and obtain root-session lineage only from a provider-authoritative receipt.
- Produce a bounded terminal receipt with exact check results and a worktree-delta reference.
- Reuse proven Codex correlation rules without coupling the new canonical domain to canary or provider types.

### Non-Goals

- No Temporal, Restate, queue, scheduler, daemon, background continuation, retry, repair loop, or multi-run orchestration.
- No OpenFGA, custom authorization layer, Effect Gateway, credential broker, credential injection, or external effects.
- No separate sandbox selection or implementation.
- No automatic branch, stage, commit, push, pull-request, merge, rebase, deploy, publish, or release action.
- No automatic check executor in v0; the operator or provider runs checks, and Goalrail records bounded structured results.
- No additional provider adapter and no general provider control plane.
- No refactor or migration of the existing canary flow.
- No real Codex launch in this change.

The excluded workflow, authorization, effect-enforcement, credential, and stronger-sandbox layers remain possible target-architecture components. This slice neither selects nor rejects them.

## Decisions

### 1. Use a strict canonical JSON WorkSpec

The canonical artifact is UTF-8 JSON decoded into closed Go structs with unknown fields rejected. Markdown and OpenSpec artifacts may compile into this representation, but they are development inputs rather than canonical runtime types.

The minimal semantic shape is:

```json
{
  "schema": "goalrail.work-spec/v0",
  "id": "work-...",
  "version": 1,
  "repository": {
    "root": "/absolute/canonical/repository/root",
    "base_revision": "<full commit id>"
  },
  "intent": {
    "id": "intent-...",
    "version": 3,
    "artifact_ref": "openspec/changes/.../intent.md",
    "digest": "sha256:..."
  },
  "task": "<bounded canonical task statement>",
  "paths": ["path/inside/repository"],
  "checks": [
    {
      "id": "check-...",
      "argv": ["go", "test", "./..."]
    }
  ],
  "stop_conditions": [
    {
      "id": "stop-...",
      "description": "<bounded observable condition>"
    }
  ],
  "posture": "trusted-local-provider-enforced-v0"
}
```

The repository root is resolved to an absolute symlink-free Git top level. The base revision is resolved to a full commit ID before freezing. The intent reference binds ID, version, repository-relative artifact path, and content digest; preparation resolves the artifact and requires its status to be `confirmed`. The initial resolver may understand the project-local Goalrail intent artifact, but OpenSpec names and types do not enter the WorkSpec.

Check commands are argument vectors, never shell fragments. They are frozen verification instructions, not permission for Goalrail to execute them. The single posture identifier expands to fixed code-level invariants: one trusted local repository, provider sandbox and approvals authoritative, no credentials, external effects, Git writes, retry, background continuation, or additional agents. A fixed posture avoids configurable authority fields that v0 cannot enforce.

Every string, list, and argument vector has a small explicit size/count limit. Validation rejects unknown fields, absolute or escaping scoped paths, control characters, high-confidence credential and private-key patterns, raw provider prompt/transcript fields, and unsupported posture values. These checks are retention hygiene, not a replacement for provider sandboxing or secret-management controls.

### 2. Canonicalize before hashing and never mutate a frozen instance

Canonicalization:

1. normalizes IDs and UTF-8 text according to one documented rule;
2. cleans repository-relative paths and rejects duplicates or containment escapes;
3. sorts fields whose semantics are sets (`paths`, checks by ID, stop conditions by ID);
4. preserves order only inside an argument vector;
5. serializes the closed struct with the standard library JSON encoder; and
6. computes `sha256:<lowercase hex>` over those canonical bytes.

Preparation stores exactly those bytes. A later read recomputes and verifies the digest. Any semantic change requires a new WorkSpec version and therefore a new prepared instance.

### 3. Keep operational state outside the target worktree

`gr` stores local-run state under a user state root, never under the target repository. Resolution order is an explicit CLI state directory, `GOALRAIL_STATE_HOME`, then the platform user-state default. Tests always inject a temporary directory.

The store uses small write-once JSON files rather than a database or workflow engine:

```text
<state-root>/
  prepared/<work-spec-digest>/
    work-spec.json
    preparation.json
    launch-claim.json          # absent until an activated or fixture start
  runs/<run-id>/
    provider-observation.json
    terminal-receipt.json
```

Directories are owner-only and files are non-executable owner-readable/writable. Each artifact is written through create-exclusive or temporary-file-plus-atomic-rename operations. Existing files are verified, never overwritten.

`launch-claim.json` is created atomically before the adapter call and contains the generated run ID and WorkSpec digest. It is the concurrency and retry boundary: only one concurrent caller can create it, and any later start is rejected even if the process crashes or the provider call fails. A claim without a provider observation remains visibly `launch_attempted_unknown`; it is never retried or guessed complete.

The activation check happens before run-ID generation and before the launch claim. Therefore the v0 `ACTIVATION_REQUIRED` result neither launches a provider nor consumes the prepared WorkSpec.

### 4. Separate preparation, start, and verification

The operator surface has four conceptual operations:

- `gr prepare --file <work-spec.json>` validates and freezes the WorkSpec, captures a read-only worktree baseline, and prints the exact canonical snapshot and digest.
- `gr inspect --digest <digest>` or `gr inspect --run <run-id>` prints bounded stored state without changing it.
- `gr start --digest <digest> --adapter <name> -- <adapter args>` is the separate explicit one-shot start. Adapter selection and arguments are operational input and never enter the WorkSpec digest.
- `gr finish --run <run-id> ...` records one explicit result for every frozen check and produces the terminal receipt. It does not execute checks or perform Git writes.

The package-level service implements these operations independently of CLI parsing so deterministic tests can inject a fixture adapter and fixture worktree observer. The production CLI is wired with real activation disabled. There is no CLI flag, environment variable, or fixture name that enables the real Codex launch path.

The lifecycle is:

| State | Entry | Allowed next state |
|---|---|---|
| `prepared` | Valid frozen WorkSpec and baseline exist | `launch_attempted` |
| `launch_attempted` | Atomic claim exists | `awaiting_verification`, `launch_failed`, `unlinked`, or `launch_attempted_unknown` |
| `awaiting_verification` | Provider completed and lineage is verified | `passed`, `failed`, or `verification_incomplete` |
| terminal | Valid terminal receipt exists | none |

A provider launch error is terminal `launch_failed`. Missing, malformed, or conflicting authoritative identity is terminal `unlinked`. A non-zero provider outcome is bounded failure evidence and is never converted into an authorization decision. Missing or unavailable required check results produce `verification_incomplete`, never `passed`.

### 5. Use a narrow provider adapter boundary

The local-run service owns a small adapter interface whose input contains the immutable run ID, frozen WorkSpec, canonical repository root, and process I/O handles. Provider-specific executable discovery, arguments, environment, sandbox behavior, approval interaction, and receipt transport belong to the adapter implementation.

The adapter returns only a bounded observation:

- adapter name and version reference;
- launch outcome category and exit result;
- provider-authoritative root-session receipt, when available; and
- bounded reason codes.

It never returns or persists a rendered prompt, transcript, hook payload, approval contents, credentials, or source bodies.

The Codex adapter follows the existing proven patterns:

- a versioned immutable context envelope is passed by value for this process;
- the envelope binds WorkSpec digest, run ID, and canonical repository root;
- lifecycle-hook data is validated against that envelope;
- the first valid provider-authoritative root identity is sticky;
- missing or conflicting identity stays explicitly unlinked; and
- no manual ID, stdout parsing, transcript parsing, or heuristic recovery is accepted.

The new envelope is adapter-local and versioned separately. It may reuse correlation helpers and reason codes, but it does not change the existing canary `RunContext` schema or make Codex types part of the canonical domain.

The real Codex adapter may be implemented and unit-tested as an unwired component. The production local-run service returns `ACTIVATION_REQUIRED` before invoking it until a later owner-authorized change alters that wiring.

### 6. Fingerprint the worktree without retaining source content

At preparation, a read-only Git observer verifies the pinned revision and captures a baseline observation:

- current `HEAD`;
- normalized `git status --porcelain=v1 -z --untracked-files=all` metadata;
- sorted repository-relative paths reported as dirty or untracked;
- content hashes and file-mode metadata for those already-dirty paths; and
- a digest of the bounded observation.

The observer enforces a maximum number of reported paths and fails preparation if the observation cannot be kept bounded. It stores no patch, file body, diff body, or copied worktree content.

At finish, the observer captures the same bounded metadata and compares it with the baseline. A worktree-delta reference contains the baseline digest, terminal digest, and sorted paths whose state or content hash changed during the run. This distinguishes run-time edits from pre-existing unrelated changes, including further modification of an already-dirty file.

Any newly changed path outside the frozen effective scope is reported as a scope violation and prevents `passed`. Read-only Git inspection is allowed; all Git writes remain forbidden.

### 7. Make the receipt the bounded review handoff

The terminal receipt is a provider-neutral closed struct containing:

- WorkSpec ID, version, and digest;
- run ID;
- provider adapter reference;
- verified root-session reference, or an explicit unlinked reason;
- pinned base revision and observed terminal `HEAD`;
- effective path scope;
- the exact frozen check IDs, argv references, and one result per check;
- terminal status and bounded categorical reasons;
- worktree baseline and terminal observation digests;
- sorted changed-path and scope-violation references; and
- timestamps for preparation, launch attempt, provider observation, and terminal recording.

The receipt stores no prompt, transcript, hook payload, credential, secret, raw source body, patch, or command output. Check output remains outside canonical state; a result may contain only status plus an optional bounded evidence reference and digest.

The service writes at most one terminal receipt. After writing it, `gr` returns control to the operator and performs no follow-on action.

## Correlation and Evidence

The lineage chain is:

```text
confirmed intent artifact digest
  -> frozen WorkSpec digest
  -> generated run ID in atomic launch claim
  -> provider-authoritative root-session receipt
  -> provider observation
  -> exact check results + worktree-delta reference
  -> terminal receipt
```

The WorkSpec digest is the stable semantic key. The run ID represents the single launch attempt. The provider session reference is evidence, not canonical task identity.

Preparation failures produce bounded errors only and create no run ID. Activation denial produces no claim or provider evidence. Once a claim exists, later absence of evidence is represented explicitly rather than repaired. All retained references are validated for size, allowed characters, repository containment where applicable, and absence of prohibited raw content.

## Measurement and Stop Conditions

Implementation of this design is acceptable only when deterministic local tests show:

- equivalent set ordering yields byte-identical canonical JSON and one digest;
- unknown fields, provider/OpenSpec fields, invalid intent status, stale intent digest, missing revision, path escapes, unsafe payloads, empty checks, empty stop conditions, and unsupported posture fail before preparation;
- preparation records no provider call and performs no target-worktree write;
- two concurrent starts yield one launch claim and at most one fixture-adapter call;
- launch failure, provider denial, missing identity, and conflicting identity remain explicit and cause no retry;
- a pre-existing dirty path is not attributed to the run unless its state or content changes after baseline;
- a new out-of-scope change prevents a passing receipt;
- missing check results produce `verification_incomplete`;
- terminal state performs no Git write, external effect, second launch, background continuation, or follow-on operation; and
- the production command rejects every real adapter start with `ACTIVATION_REQUIRED` and records zero real provider launches.

Stop this slice and return to design if any implementation requires provider fields in WorkSpec, an activation bypass, automatic command or Git execution, raw prompt/transcript/source retention, a second launch path, an additional service or dependency, or modification of the existing canary contract.

## Risks / Trade-offs

- **Operator-supplied check results are less automated.** This is deliberate: automatic execution would introduce a new effect surface. A later slice can add a bounded check runner when its authority and evidence rules are explicit.
- **Write-once files do not provide workflow-engine recovery.** A crash after the launch claim can leave `launch_attempted_unknown`. This is safer than an implicit retry and sufficient for one trusted local run.
- **Secret-pattern detection is incomplete.** Closed schemas, low size limits, reference-first evidence, and prohibited-field validation reduce accidental retention, but provider sandboxing and operator discipline remain necessary.
- **Filesystem atomicity is host-local.** The v0 store assumes one local filesystem and does not claim distributed coordination.
- **Worktree hashing adds read cost.** The bounded Git-reported path set avoids copying content or scanning a database; preparation fails rather than silently accepting an unbounded dirty set.
- **Provider selection is outside the WorkSpec digest.** This preserves provider-neutral intent semantics. The selected adapter and version are still captured in the launch claim and receipt for audit.
- **Codex correlation reuse may expose helper-level coupling.** Reuse is limited to adapter-local validation patterns and reason codes; the canary context schema and canonical domain remain unchanged.

## Rollback

Before real activation, rollback is local and clean:

1. remove or disable the new `cmd/gr`, local-run service/store, WorkSpec domain types, and unwired Codex adapter additions;
2. leave the existing canary command, domain, evidence store, and artifacts unchanged; and
3. delete the external local-run state directory only when the operator explicitly chooses to remove those inert local receipts.

No database migration, external cleanup, credential revocation, Git history rewrite, or provider-session termination is required because this change creates no real provider session or external effect.

## Open Questions

None for this slice. Real activation, automated check execution, durable recovery, stronger authorization/effect enforcement, credential handling, additional providers, and stronger sandboxing require separate intent and design decisions.
