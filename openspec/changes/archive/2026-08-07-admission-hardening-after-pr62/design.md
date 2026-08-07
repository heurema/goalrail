# Design: Admission hardening after PR #62

## Context

PR #62 reached `main` before its automated review completed. The reviewed tree is byte-equivalent to the squash merge, so all nine unresolved findings apply to the current implementation: five authorization or admission defects and four correctness defects (CTX-1, CTX-2). Existing package tests remain green, and the prepared shared-admission workflow is not active in this repository, which makes a bounded code-and-contract remediation possible before activation or release (CTX-12, CTX-13).

This design implements confirmed Intent Snapshot `intent-admission-hardening-after-pr62` version 1 and the four delta specifications compiled from it. It does not amend the confirmed outcomes or authorize implementation, activation, release, installation, or another external effect.

### Decision question

How should the pure, provider-neutral admission verifier combine committed lineage with current authenticated provider observations, validate intent and receipt semantics, and preserve an unforgeable owner-authority boundary without adding a service, database, workflow engine, provider-specific canonical domain, or second verifier?

### Constraints

- The admission core remains deterministic and side-effect free. It does not read a filesystem, clock, network, credential, provider API, or mutable repository state (CTX-4, CTX-9).
- The committed work-unit graph remains the canonical repository record. Provider observations are current evaluation inputs and are never persisted or represented as branch-authored lineage (CTX-3, CTX-4).
- OpenSpec remains a replaceable artifact compiler/provider. Provider parsing may occur at an adapter boundary, but OpenSpec types do not enter the canonical Goalrail domain.
- The current change uses existing packages and standard-library mechanisms. It introduces no runtime dependency or infrastructure.
- Base governance is the authority for a non-bootstrap range. A head change cannot mint a new project identity or owner authority for itself (CTX-6, CTX-10).

### Options considered

| Option | Description | Trade-off | Decision |
|---|---|---|---|
| A | Build one bounded, non-persistent effective evidence view from the committed graph, exact semantic projections of committed artifacts, and same-invocation authenticated provider observations. | Requires a stricter composition boundary and makes a serialized packet insufficient to complete authoritative provider relations by itself. | Selected. It preserves the canonical graph, closes the authority bypass, and lets an ordinary pull request become complete from live evidence (CTX-3, CTX-4). |
| B | Encode provider observations as synthetic lineage events carried in the packet and evaluate the augmented graph. | Conflates point-in-time provider evidence with canonical lineage and adds a second event-authenticity model. | Rejected because it creates ambiguity about whether packet events are canonical and makes replay or persistence unsafe (CTX-3, CTX-4). |
| C | Require reviews, checks, and owner decisions to be committed before admission. | Creates a cycle for ordinary pull requests and encourages fabricated or stale provider evidence. | Rejected because it cannot satisfy the intended shared flow (CTX-4). |

### Independent critique receipt

| Field | Receipt |
|---|---|
| Requested role | One isolated security and architecture critique of Option A |
| Requested model | `claude-fable-5` |
| Actual provider/model | `firstParty/claude-fable-5`; model usage also reported an auxiliary `claude-haiku-4-5` routing call of 16 output tokens |
| Status | `success`; verdict `ACCEPT_WITH_CHANGES` |
| Degraded or fallback path | None for the requested critique; `modelUsage` contained the requested Fable model |
| Cost exposed by provider | USD 0.507342 |

The critique's material objections are incorporated below: same-invocation provider authentication, base-policy authority, head-bound latest owner decisions, raw-byte semantic binding, canonical conflict handling, unconditional trusted time on the authoritative provider path, and explicit legacy-packet denial. Its suggestion to add codec pins to project policy is not adopted: exact artifact schema IDs plus the versioned verifier codec registry provide the required deterministic pin without widening `goalrail.policy/v1` for this hotfix. A future policy-level codec registry requires separate intent and a policy schema migration.

## Goals / Non-Goals

### Goals

- Close all nine reviewed defects with fail-closed, canonical results and focused regressions (CTX-1 through CTX-11, CTX-13).
- Make provider observations usable for current lifecycle completeness without writing them into repository lineage (CTX-4).
- Make owner authority depend on current provider authentication and committed base policy, never on branch-controlled labels or files (CTX-3, CTX-6).
- Count confirmed intent and terminal receipt relations only after exact-artifact semantic validation (CTX-5, CTX-7).
- Verify the full declaration-bound governance set and stable project identity before any policy or authority decision (CTX-6, CTX-10).
- Preserve deterministic policy classification, total verifier behavior, and executable setup plans (CTX-8, CTX-9, CTX-11).

### Non-Goals

- No shared-workflow activation, required-check or ruleset mutation, release, deploy, installation, credential setup, or Deltacue change (CTX-12).
- No cryptographic provider-attestation service, signing infrastructure, database, workflow engine, or authorization product.
- No claim that a local hook, `prek`, a packet file, or a direct library call is independently unbypassable.
- No rewriting of the archived PR #62 artifacts or review history (CTX-1, CTX-2).
- No broad rewrite of lineage, setup, or provider schemas beyond the minimum behavior described here.

## Decisions

### 1. Authoritative provider evidence is same-invocation capability, not packet data

The shared GitHub admission entrypoint SHALL perform provider collection and core verification in one trusted invocation. The existing GitHub adapter reads the current provider state, authenticates the observation through its configured API path, freezes the exact base/head range, and constructs a bounded in-memory provider-candidate set. It then invokes the same pure admission core used by local checks. A canonical admission packet may still be emitted for audit and reproducibility, but its serialized `authenticated` field, actor fields, or authority claims SHALL NOT by themselves create an authoritative candidate (CTX-3, CTX-4).

The authoritative composition contract is:

1. Freeze immutable base and head revisions and collect both governance states.
2. Read current provider state for that exact range through the registered GitHub adapter.
3. Construct provider candidates in memory from the authenticated response; do not reconstruct them from a packet file.
4. Supply the exact provider observation time as the verifier evaluation time.
5. Verify governance, semantic projections, candidates, and completeness in the fixed order defined below.
6. Emit the canonical result and optional audit packet only after the verdict is fixed.

The prepared shared workflow SHALL use this composite path. The standalone packet-verification path remains useful for deterministic inspection but treats serialized provider candidates as absent for authoritative lifecycle and owner-decision relations. Local and hook entrypoints always use an empty authenticated-candidate set. This is an intentional compatibility break: a legacy or hand-authored packet cannot authorize an owner-gated relation and returns `owner_decision_missing` or the exact missing late relation rather than `VALID` (CTX-3, CTX-4).

This boundary does not claim that an arbitrary caller of a Go function is trusted. The enforcement claim belongs only to the separately verified protected integration entrypoint; local results remain advisory.

### 2. The verifier builds one ephemeral effective evidence view

The pure core SHALL construct an immutable effective view for the current call. It starts with verified committed graph relations, adds semantically satisfying committed relations, then adds authenticated provider candidates according to a closed mapping:

| Provider artifact kind | Effective relation | Required extra binding |
|---|---|---|
| `pull_request` | `pull_request` | Exact repository, pull-request identity, base SHA, and head SHA |
| `review_index` | `review_index` | Exact pull request and current head; deterministic empty index is allowed when policy permits |
| `check_set` | `check_set` | Exact current head and bounded selected checks |
| `owner_decision` | `owner_decision` | Exact current head plus authenticated actor, authority, outcome, and observation time |
| `closure` | `closure` | Exact pull request and integration or rejection outcome; never required for admission-ready state |

Unknown kinds, unsupported schemas, range mismatches, untrusted candidates, or candidates without the required binding fail closed with `packet_invalid`, `provenance_untrusted`, or the exact missing relation. Candidate ordering is canonical by relation, identity, digest, actor, and observation time. Multiple distinct candidates for a single-valued relation produce `lineage_conflict`; no map iteration, arrival order, or first-wins behavior can select a winner (CTX-4).

The effective view exists only in memory for the duration of `Verify`. It SHALL NOT be written back to the packet, cached as lineage, emitted as synthetic events, or used to rewrite the committed graph. Completeness and exception evaluation consume this effective view rather than the committed graph alone (CTX-4, CTX-5).

### 3. Owner decisions use authenticated current-head semantics and base policy

Committed `owner_decision` events remain ordinary repository evidence and never satisfy owner authority, even when their `actor_ref` matches policy, their target count is one, or their target is under `repo:root/` (CTX-3). Repository-root trust proves committed-byte integrity only; it never proves external identity, permission, or approval.

For the GitHub path, the adapter derives decisions only from the latest provider review state for each actor at the exact head SHA. A later dismissal, neutral state, or `CHANGES_REQUESTED` state shadows an earlier approval. The in-memory candidate carries the provider actor reference, provider-derived authority reference, normalized outcome, review identity, head SHA, and observation time. The core then requires:

- head SHA equality with the frozen range;
- authenticated observation in the same invocation;
- actor and authority binding produced by the registered adapter;
- authority membership in `basePolicy.OwnerDecision.AuthorityRefs`;
- an outcome supported by the base policy;
- outcome `allow` before normal admission can succeed.

A valid `reject` observation is retained as negative evidence but cannot satisfy the admission-ready owner decision. Any base/head project-identity conflict or invalid base governance denies before owner policy is consulted (CTX-3, CTX-6, CTX-10).

### 4. Exact committed artifacts produce bounded semantic projections

The Git collector SHALL read semantic artifacts from the immutable evaluated revision using their repository references and existing size/path bounds. It hashes the exact raw bytes and requires the digest to match the exact lineage target before parsing. Parse failure, non-canonical content where canonical encoding is required, identity/version mismatch, or unsupported codec produces no satisfying projection (CTX-5, CTX-7).

The collector creates a non-persistent, provider-neutral projection containing only:

- artifact kind, identity, version, and raw-byte digest;
- exact codec/schema identity recognized by the current verifier version;
- the admission-relevant state (`confirmed` for intent or terminal status for a receipt);
- a valid/invalid parse result without copied semantic body content.

For intent, the current Goalrail intent adapter parses the exact retained artifact into `domain.IntentSnapshot`; `domain.ValidateIntentSnapshot` must succeed, identity and version must match the reference, status must be exactly `confirmed`, confirmation must be present, and ambiguities must be empty. OpenSpec-specific parser types stay outside the domain and the effective graph (CTX-7).

For terminal receipts, the existing bounded canonical receipt decoder validates the entire exact replica, including schema and structural invariants; the projected status must be exactly `passed`. `failed`, `blocked`, `unlinked`, `launch_failed`, `launch_attempted_unknown`, and `verification_incomplete` remain non-satisfying (CTX-5).

The pure core independently rechecks projection kind, identity, version, digest, known codec identity, allowed state enum, uniqueness, and target relation. A projection cannot satisfy another digest or relation. The projection is never serialized or persisted. This preserves exact-artifact binding while keeping provider/compiler codecs out of the pure core. The verifier version pins the accepted codec/schema set; unknown codec identities fail closed.

Semantic failure maps to existing stable reason codes: `intent_unconfirmed` for malformed, mismatched, candidate, or otherwise unconfirmed intent; `receipt_missing` for a missing, malformed, mismatched, or non-`passed` terminal receipt. Evidence references distinguish the exact target. No new reason code is required for this slice (CTX-5, CTX-7).

### 5. Governance is fully declaration-bound before authority selection

Git collection SHALL load the declaration plus policy, bootstrap, and setup profile from both immutable revisions. For every declared artifact it validates safe repository-relative path, bounded regular-file bytes, declared schema, exact raw-byte digest, canonical representation where the artifact contract requires one, and project binding where the artifact carries a project ID (CTX-10).

The verifier order is fixed:

1. Validate input structures, versions, and exact base/head range.
2. Validate complete head governance.
3. If base has a valid declaration, validate complete base governance and require `head.ProjectID == base.ProjectID`.
4. Resolve non-bootstrap policy and owner authority from the valid base governance.
5. Only for an explicit no-base-declaration bootstrap or migration classification may complete head governance establish the project identity.
6. Continue to semantic binding, provider candidates, policy classification, and completeness.

There is no fallback from malformed, missing, drifted, or identity-conflicting governance to unmanaged or head-defined authority. All such cases return `declaration_invalid` or the existing bootstrap/policy conflict classification before lineage can allow admission (CTX-6, CTX-10).

### 6. Repository root is an explicit prefix sentinel

For `PathMatcherPrefix`, normalized matcher path `.` is handled before ordinary prefix concatenation and matches every valid normalized repository-relative path. Other prefixes retain the existing exact-or-`prefix/` boundary behavior. Rule priority, rule ID, classification, and required evidence kinds remain unchanged; the fix does not add a special fallback materiality rule (CTX-11).

Focused tests cover root and nested add, modify, rename, delete, mode, and submodule paths, plus a neighboring prefix that must not match.

### 7. Trusted time is required before provenance is inspected

Domain packet validation SHALL reject provenance without a non-nil, non-zero evaluation time and a bounded time-authority reference. The verifier SHALL defensively check time again before every provenance or time-bounded evidence access and return `trusted_time_missing`; it never dereferences absent time (CTX-9).

The authoritative provider path always supplies one caller-injected authenticated observation time. It never reads the wall clock inside the core. Provider-free advisory evaluation may continue using the work-unit creation time only when it has no provider candidate, provenance, or time-bounded exception. Stale, future, or post-evaluation observations fail with the existing trusted-time or provenance reason. Equivalent explicit inputs therefore remain byte-identical (CTX-9).

### 8. Setup planning advertises only executable, recoverable mutations

`inspectExecutableDestination` SHALL distinguish three existing-file states after safe-path, size, mode, and digest inspection (CTX-8):

- desired digest and desired mode: no mutation;
- desired digest and a safely repairable different mode: emit the existing digest-bound mode repair and preserve the prior mode for rollback;
- different digest at an existing destination, including a same-size file: emit `EXECUTABLE_DESTINATION_CONFLICT`, mark the plan incomplete, and emit no `select-executable` mutation.

This intentionally chooses the smallest safe behavior instead of adding executable backup storage or a new replacement protocol. A future recoverable replacement design requires separate intent, explicit retained-byte bounds, and rollback tests.

### 9. Canonical reason mapping and evaluation order are stable

The hotfix reuses the existing reason registry:

| Condition | Classification | Stable reason |
|---|---|---|
| Invalid or drifted declaration-bound governance, including project-ID change | `INVALID` | `declaration_invalid` |
| Missing trusted evaluation time | `MISSING` | `trusted_time_missing` |
| Unauthenticated or stale provider candidate | `INVALID` | `provenance_untrusted` |
| Unsupported candidate, schema, codec, or range binding | `INVALID` | `packet_invalid` or `range_mismatch` |
| Conflicting single-valued candidates | `AMBIGUOUS` | `lineage_conflict` |
| Candidate or invalid confirmed-intent artifact | `MISSING` | `intent_unconfirmed` |
| Invalid or non-passed terminal receipt | `MISSING` | `receipt_missing` |
| Missing authenticated owner approval, including legacy serialized claims | `MISSING` | `owner_decision_missing` |

The fixed high-level order is structural validation, governance validation, base authority resolution, graph and raw-artifact binding, semantic projection validation, provider-candidate validation with owner decision last, materiality classification, and completeness over the effective view. A failure at an earlier boundary cannot be repaired by evidence at a later boundary (CTX-3 through CTX-11).

## Consumed Inputs

| Input | Source and trust | Accepted variants | Refused variants | Mutation / race policy | Verification |
|---|---|---|---|---|---|
| Base and head revisions | Explicit immutable Git object IDs supplied by the entrypoint | Existing valid base plus valid head; explicit no-declaration base for bootstrap | Moving refs, range mismatch, missing object, non-bootstrap identity replacement | Read only from immutable revisions; checkout state is irrelevant | Exact packet/range equality and full Git-object reads (CTX-2, CTX-6) |
| Project declaration | Canonical repository path in each revision | Supported canonical schema and complete bound refs | Missing-at-head, malformed, non-canonical, unsafe path, unsupported schema | No fallback to local marker or unmanaged state | Decode/freeze equality and exact digest (CTX-6, CTX-10) |
| Policy, bootstrap, setup profile | Paths and digests in each revision's declaration | Exact bounded bytes; canonical form where required; matching project binding | Missing, substituted, digest drift, schema drift, path escape | All three are read before authority; no automatic repair | Independent ref/path/digest/schema checks in base and head (CTX-10) |
| Committed work-unit graph | Exact head-revision work-unit anchor, events, and replicas | Canonical bounded graph with deterministic conflicts | Branch authority claims, malformed events, unavailable required targets | Read only; never rewritten by verification | Existing graph binding plus semantic projection checks (CTX-3, CTX-5, CTX-7) |
| Intent artifact bytes | Exact `IntentRef.SourceRef` at the evaluated revision | Domain-valid exact confirmed snapshot with matching ID/version/digest | Candidate, malformed, ambiguous, passive confirmation, mismatch | Read once from immutable Git object; no normalized digest substitution | Raw-byte digest, adapter parse, domain validation, projection rebind (CTX-7) |
| Terminal receipt replica | Exact content-addressed repository replica | Fully valid canonical receipt with status `passed` | Failed, blocked, unlinked, incomplete, launch-failed, malformed, mismatch | Read once; no partial projection or status-only JSON parse | Existing codec validation, canonical equality, projection rebind (CTX-5) |
| Live GitHub snapshot | Authenticated registered adapter in the protected same invocation | Exact current PR/range, latest current-head reviews, selected checks, explicit empty indexes | Serialized reconstruction, stale head, unavailable authority, unauthenticated response, prohibited bodies | Provider read occurs before core call; no persistence as lineage | Adapter range/time/auth checks plus core candidate checks (CTX-3, CTX-4) |
| Provider candidate set | In-memory output of the live adapter call | Closed known kinds with exact range/time/provenance bindings | Packet-at-rest candidates, unknown kinds, duplicates, conflicting slots | Exists only for the call; sorted before evaluation; never cached or emitted as events | Closed mapping, uniqueness, base-policy authority, canonical conflict denial (CTX-3, CTX-4) |
| Canonical admission packet | Seed plus bounded audit references | Structurally valid packet with explicit time when provenance exists | Secret-shaped content, nil-time provenance, unsupported version, claims used as standalone authority | May be emitted for audit; cannot mint authoritative candidates | Domain validation and equality checks; authority fields ignored without same-invocation capability (CTX-3, CTX-9) |
| Normalized changed paths | Immutable Git diff | Valid repository-relative add/modify/rename/delete/mode/submodule paths | Absolute, escaping, malformed, or ambiguous paths | No writes; deterministic normalization | Root-sentinel and boundary matcher tests (CTX-11) |
| Executable destination pre-state | Read-only local setup inspection | Absent, already current, or same digest with repairable mode | Non-regular, unsafe, unreadable, oversized, or different digest | Plan binds exact inspected state; apply revalidates before mutation | Plan/apply/rollback fixtures including same-size collision (CTX-8) |

## Correlation and Evidence

- The remediation trace SHALL map each of the nine PR #62 discussion URLs to its owning requirement, implementation diff, and at least one regression test. The reviewed and merged tree identity is recorded by CTX-1 and CTX-2.
- Provider observations retain only bounded reference identities and digests in the audit packet. Raw review, comment, source, prompt, transcript, token, or credential content is neither projected nor persisted (CTX-3, CTX-4).
- The effective view preserves source attribution: committed events remain marked committed; semantic projections name their exact raw-byte digest and codec; provider candidates name their adapter, provider reference, exact range, and observation time.
- Canonical results retain the existing stable reason and evidence-reference structure. Semantic failures point to the exact intent or receipt target; provider failures point to the exact candidate identity; governance failures point to the exact declaration-bound artifact.
- Test names and fixture IDs SHALL include the corresponding review-finding identifier so failing-before and passing-after evidence can be reconstructed without rewriting PR #62 artifacts.

## Measurement and Stop Conditions

Implementation verification is complete only when all of the following are true:

1. Each of the nine focused regression tests fails against merge commit `6a896355958874a4989eec131c6cc20a24921995` for the reviewed reason and passes on the remediation branch (CTX-2, CTX-13).
2. Forged branch owner decisions and serialized `authenticated=true` packets cannot satisfy owner authority; one same-invocation authenticated current-head approval can.
3. One complete GitHub fixture reaches `VALID` through the ephemeral provider view, while missing, stale, conflicting, and unauthenticated variants fail with canonical reasons (CTX-3, CTX-4).
4. Candidate or malformed intent and every non-`passed` receipt remain semantically unsatisfied (CTX-5, CTX-7).
5. Policy, bootstrap, setup-profile, and non-bootstrap project-ID drift each deny before policy or owner authority is evaluated (CTX-6, CTX-10).
6. Root policy fixtures cover all declared Git change kinds and preserve rule identity, classification, priority, and evidence kinds (CTX-11).
7. Nil, zero, stale, future, and untrusted time fixtures return canonical non-valid results under a panic-detecting test (CTX-9).
8. Same-size different-digest executable planning is incomplete; absent, current, and mode-only cases retain their intended behavior (CTX-8).
9. `go test ./internal/admission ./internal/admissiongit ./internal/githubadmission ./internal/setup` and the repository-wide verification suite pass with repeated canonical-result equality checks (CTX-13).
10. The prepared workflow remains inactive, no required check or ruleset is changed, and no release or installation occurs (CTX-12).

The work stops at the first owner gate after local implementation proof. Activation, release, repository-setting changes, installation, and a real external canary remain separate decisions.

## Risks / Trade-offs

- **Composite-path compatibility:** a serialized packet can no longer make a standalone verifier authoritative for live provider relations. This is a deliberate security break; tooling that depended on the old two-step authority interpretation must use the same-invocation shared entrypoint.
- **Semantic projection trust:** parsing occurs outside the pure core. Raw-byte digest binding, exact known codec IDs, full existing validators, projection rebinding, and protected composition minimize the trust gap without duplicating parsers in the core.
- **Provider eventual consistency:** exact-head and latest-review checks may temporarily deny a legitimate admission while GitHub state converges. The safe behavior is retrying a fresh authoritative invocation, never accepting stale evidence.
- **Stricter governance:** repositories whose bootstrap or setup profile already drifted will newly fail closed. Diagnostics must name the exact artifact; automatic repair remains forbidden.
- **Setup conservatism:** an existing different executable now requires an explicit future replacement design or manual owner action. This removes an impossible plan rather than adding backup semantics silently.
- **Implementation coupling:** `admissiongit` must invoke the current intent and receipt validators to produce neutral projections. The projection contract remains provider-neutral so a future artifact provider can replace the parser without changing admission semantics.

## Rollback

- The safe rollback floor is deny-all for provider-derived owner decisions while keeping the prepared shared workflow inactive. Never restore branch-controlled owner authority or accept serialized `authenticated=true` as proof.
- The ephemeral-view implementation is isolated from persisted lineage, so it can be reverted without repository-data migration. Committed graph formats remain unchanged.
- Semantic projections are non-persistent; reverting their consumer leaves no stored shadow artifacts to clean up.
- Governance and root-matcher fixes are independently revertible in code, but a rollback that reopens a reviewed defect is not releaseable and must keep shared activation stopped.
- Setup planner rollback is a code revert only; no new executable backup or state format is introduced.

Revisit the selected design if an authoritative path is found deserializing candidates or projections from packet storage, or if exact same-invocation head binding repeatedly false-denies valid provider state due to measured API consistency behavior. Until a replacement contract is proven, the fallback remains deny-all owner decisions.

## Open Questions

None material. Private symbol names and the exact CLI spelling of the composite shared entrypoint may follow existing package conventions during implementation, but the trust boundary, input order, compatibility behavior, and stop conditions above are fixed.
