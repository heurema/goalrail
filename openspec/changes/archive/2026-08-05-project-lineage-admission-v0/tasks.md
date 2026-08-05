## 1. Canonical contracts and deterministic fixtures

- [x] 1.1 Add `goalrail.project/v1` domain types, bounds, canonical JSON, project-ID generation, digest validation, and malformed/unknown-schema tests.
- [x] 1.2 Add `goalrail.policy/v1` types for exact/prefix path rules, change kinds, explicit priorities, exception authority, owner decisions, and canonical digest tests.
- [x] 1.3 Add `goalrail.setup-profile/v1`, setup-plan v1, setup-receipt v1, and plan-authorization reference types with secret/prohibited-payload validation.
- [x] 1.4 Add Work Unit, lineage-event v1, content-addressed evidence reference, relation-cardinality, and lifecycle-state types with order-independent digest goldens.
- [x] 1.5 Add admission-packet v1 and admission-result v1 types, the complete classification enum, allow/deny outcome, and stable reason-code registry.
- [x] 1.6 Add fixture builders and checked-in canonical JSON goldens for every new schema; prove equivalent semantic inputs produce byte-identical bytes.

## 2. Managed-project discovery

- [x] 2.1 Implement a safe Git worktree-root resolver that ignores caller-selected alternate repositories and handles primary checkouts, linked worktrees, subdirectories, and no-worktree repositories.
- [x] 2.2 Implement declaration claim inspection with bounded `lstat`/regular-file/path-containment checks and distinct unmanaged, managed, and declared-invalid results.
- [x] 2.3 Remove the ambient marker from new project-identity decisions while retaining a read-only v0.1.8 marker/overlay detector for migration diagnosis.
- [x] 2.4 Add substitution and race checks so a declaration changed after validation stops the dependent operation before its first write.
- [x] 2.5 Add a topology test that commits one declaration, creates two independent clones and at least three linked worktrees, and verifies one project ID with zero repeated initialization.
- [x] 2.6 Add unmanaged and invalid-claim tests proving unrelated repositories produce no managed observation/write while reserved malformed declarations fail closed.

## 3. Project canon, initialization, and migration

- [x] 3.1 Add the v1 declaration, default policy, bootstrap guide, setup profile, and prepared-admission templates to the embedded canon with deterministic digests and retained previous canon metadata.
- [x] 3.2 Change fresh `gr init` to generate one random project ID and materialize all repository-owned and canon-owned v1 files without creating a checkout-local identity marker.
- [x] 3.3 Implement managed-block rendering for `AGENTS.md` and `CLAUDE.md`, including absent-file creation, exact known-block update, and refusal on duplicate, malformed, locally edited, or unbounded conflicts.
- [x] 3.4 Preserve existing OpenSpec adoption behavior while binding the selected planning adapter/version in the setup profile and refusing a foreign custom schema without explicit owner resolution.
- [x] 3.5 Separate project initialization from local scaffold readiness so no scaffold, declined attachment, or pending provider trust leaves the project declared but locally not ready.
- [x] 3.6 Add `gr migrate` for the pinned v0.1.8 overlay/marker shape; create only v1 project artifacts, preserve every legacy intent/run/review/receipt byte, and report the required commit boundary.
- [x] 3.7 Add idempotence, no-worktree, unsafe-link, owner-file conflict, foreign-schema, repeated migration, and fresh-clone-after-migration tests.

## 4. Update, diagnosis, and ambient behavior

- [x] 4.1 Extend canon update to reconcile whole canon-owned files and uniquely delimited managed blocks while preserving repository-owned policy and every byte outside a managed block.
- [x] 4.2 Add managed-block drift, previous-canon transition, just-in-time race, cumulative recovery, and explicit-discard tests without weakening the existing overlay rollback discipline.
- [x] 4.3 Implement `goalrail.doctor/v2` with separate managed, locally-ready, lineage-ready, shared-admission-active, aggregate-working, and stable reason fields.
- [x] 4.4 Add a one-release deprecated `initialized` compatibility field mapped only to valid declaration presence and document/test the breaking semantic change.
- [x] 4.5 Report exact planning bundle/runtime/compiler readiness and route every missing or incompatible required component to setup planning without erasing managed identity.
- [x] 4.6 Add a read-only shared-activation evidence interface with active, inactive, unknown, stale, and access-denied states; forbid workflow-file-only inference.
- [x] 4.7 Change the ambient hook's first operation to declaration-claim inspection before payload parsing, retaining zero observation/writes for unmanaged or invalid projects.
- [x] 4.8 Update attachment health to use committed project identity, retain provider trust boundaries, and avoid conflating local attachment with protected enforcement.
- [x] 4.9 Add doctor and ambient fixtures for clean clones, linked worktrees, invalid declarations, missing runtimes, untrusted scaffolds, stale activation receipts, hook failure, and unrelated repositories.

## 5. Integrity-verifiable setup bundle and consent flow

- [x] 5.1 Define the release setup-bundle manifest with exact file paths, component versions, platform/architecture, sizes, SHA-256 digests, binary identity, and bounded license/provenance references.
- [x] 5.2 Pin the private planning runtime and exact compiler dependency tree for each currently supported release platform; stop this milestone if any transitive component cannot be enumerated and integrity-verified.
- [x] 5.3 Extend release packaging to build the optional setup archive alongside the minimal `gr` archive without placing build output in the checkout.
- [x] 5.4 Extend release verification so setup manifest, archive contents, checksums, embedded versions, supported platforms, and current-release metadata must agree before publication.
- [x] 5.5 Implement read-only environment inspection and canonical setup-plan generation covering every destination, local config/hook mutation, network read, prerequisite, provider trust step, rollback, and zero project-code writes.
- [x] 5.6 Implement authorization binding to one complete plan digest; reject missing/refused/stale/general consent and re-resolve the declaration, plan, prerequisites, and targets before the first mutation.
- [x] 5.7 Implement `gr setup verify-plan` for a verified temporary bundle and require semantic equality with the pre-authorized plan before durable installation.
- [x] 5.8 Implement `gr setup apply` with artifact digest verification, durable user-local versioned placement, safe executable selection, bounded configuration edits, just-in-time checks, and no provider trust forgery.
- [x] 5.9 Implement partial-failure reporting and `gr setup rollback` from the exact recovery set without deleting unrelated runtimes, executables, or configuration.
- [x] 5.10 Emit bounded success, failure, and no-write setup receipts with plan/authorization/component/mutation/rollback/doctor/continuation references and no raw request content.
- [x] 5.11 Update generated supported-agent bootstrap instructions to present the exact plan, ask once, stop on any unlisted enabling action, and return to candidate intent only after the receipt.
- [x] 5.12 Add temp-home clean-machine tests for refusal/no writes, successful installation, temporary-directory cleanup, unknown dependency, concurrent target change, partial rollback, trust-required, and shared-admission-unknown states.

## 6. WorkSpec v1 and portable work-unit lineage

- [x] 6.1 Add `goalrail.work-spec/v1` project, policy, change, and work-unit bindings and include every authority field in canonicalization and freezing.
- [x] 6.2 Update preparation validation to require one valid managed project, exact confirmed intent, matching current Goalrail change, open work unit, pinned revision, and contained scope before any launch.
- [x] 6.3 Retain legacy WorkSpec/receipt decoding for inspection while refusing new normal runs from legacy authority; test bounded `BOOTSTRAP` classification without rewriting history.
- [x] 6.4 Implement `gr lineage begin` to create one canonical work-unit anchor and initial immutable declaration/intent/change relations before a managed run.
- [x] 6.5 Implement append-only lineage event storage keyed by semantic digest, with identical-event idempotence and visible conflicts for incompatible single-valued relations.
- [x] 6.6 Implement `gr lineage attach` for validated WorkSpec, run/session, commit, PR, review/comment index, checks, terminal receipt, owner decision, exception, and closure relation types.
- [x] 6.7 Implement exact content-addressed replication of bounded canonical receipts/evidence and reject digest mismatch, schema mismatch, prohibited payload, or two payloads for one digest.
- [x] 6.8 Implement graph resolution and reverse lookup from work unit, intent, change, WorkSpec, run, root session, commit, PR, receipt, and owner decision without storing a second semantic index.
- [x] 6.9 Implement open, admission-ready, and closed completeness evaluation, including explicit empty allowed sets, unavailable provider edges, expired exceptions, and missing late evidence.
- [x] 6.10 Add property and fixture tests for event order independence, late backward references, replica equality, ambiguous bindings, exception visibility, reverse resolution, and raw-content rejection.

## 7. Deterministic materiality and admission core

- [x] 7.1 Implement base-authoritative policy loading and validate the candidate head declaration/policy without allowing the head to weaken its own evaluation.
- [x] 7.2 Implement the exact/prefix/change-kind/priority matcher, default-material classification, evidence-path handling, generated-evidence requirement, and equal-priority ambiguity.
- [x] 7.3 Implement frozen Git base/head diff collection for additions, edits, deletions, renames, modes, submodules, and bounded path sets without absolute-path or locale dependence.
- [x] 7.4 Validate admission packets against the resolved project/range/work unit, trusted adapter contract, bounded references, explicit trusted evaluation time, and prohibited-content rules.
- [x] 7.5 Implement graph verification for declaration, policy, intent, change, WorkSpec, run/session, commits, PR, review/comment index, selected checks, receipt, owner decision, and lifecycle phase.
- [x] 7.6 Implement `VALID`, `MISSING`, `AMBIGUOUS`, `INVALID`, `EXEMPTED`, `BREAK_GLASS`, and `BOOTSTRAP` classification with separate allow/deny admission and stable reason codes.
- [x] 7.7 Implement canonical admission-result JSON and process exit behavior; prove wall clock, map order, checkout path, hidden environment, and network availability cannot affect identical frozen inputs.
- [x] 7.8 Add the public `gr verify-lineage --repo --base --head --packet --json` command with read-only behavior and no provider/network/credential imports in the core package graph.
- [x] 7.9 Build one golden corpus covering valid, missing, ambiguous, malformed, out-of-scope, self-weakened policy, expired exception, permitted exception, migration, and locally bypassed cases.
- [x] 7.10 Run the same corpus through explicit local and simulated shared entrypoints and require byte-identical canonical results and matching exit outcomes.

## 8. Local feedback, GitHub collection, and prepared shared check

- [x] 8.1 Implement thin Goalrail-owned local Git/scaffold shims that call diagnosis or the public verifier and contain no materiality, exception, or verdict logic.
- [x] 8.2 Add optional `prek` adapter generation only when selected in the exact setup profile; prove Goalrail remains functional with no `prek` installation.
- [x] 8.3 Add commit-message trailer validation as an early work-unit index while proving a trailer alone cannot satisfy any missing canonical lineage edge.
- [x] 8.4 Implement the isolated GitHub admission collector that reads bounded pull-request, review/comment index, check, actor/decision, and integration references with read-only credentials and emits provider-neutral packet v1.
- [x] 8.5 Strip and reject raw bodies, tokens, credentials, secret-shaped data, unsupported event shapes, mismatched head/base, unavailable authority, and untrusted timestamps before packet persistence.
- [x] 8.6 Generate a pinned minimal-permission GitHub Actions workflow for pull-request and review re-evaluation that verifies the Goalrail release artifact before invoking the collector and core verifier.
- [x] 8.7 Implement read-only GitHub required-check/ruleset observation with current provenance and explicit unknown on 403, missing auth, unsupported syntax, stale data, or target mismatch.
- [x] 8.8 Add mocked GitHub API/event fixtures for zero comments, multiple reviews, changed head, stale approval, access denial, check recursion avoidance, merge/rejection closure, and credential non-retention.
- [x] 8.9 Add a hook-bypass integration fixture showing that absent/deleted/`--no-verify` local hooks do not alter the prepared shared verdict.
- [x] 8.10 Verify that no init, setup, update, doctor, collector, or verifier path enables branch protection, required checks, comments, commits, pushes, or merges.

## 9. CLI, documentation, compatibility, and migration UX

- [x] 9.1 Add dispatch and built-in flag help for `migrate`, `setup plan|verify-plan|apply|rollback`, `lineage begin|attach|inspect`, and `verify-lineage`; keep internal hooks/collectors hidden.
- [x] 9.2 Rewrite top-level lifecycle help to remove “ordinary work needs no Goalrail command” and explain managed discovery, exact setup consent, confirmed intent/current change, lineage, local advice, and protected admission.
- [x] 9.3 Document every new schema, stable reason code, exit category, ownership boundary, content-addressed replica rule, and the one-release compatibility window.
- [x] 9.4 Update install documentation for minimal versus setup bundles, anonymous immutable metadata, SHA-256 verification, durable user-local placement, macOS quarantine truth, rollback, and no curl-to-shell path.
- [x] 9.5 Add owner-facing migration documentation that states the breaking v0.2.0 semantics, one project-scoped migration, required commit, local setup, prepared-versus-active admission, and exact rollback boundaries.
- [x] 9.6 Add an enforcement-claims check that fails documentation or help text equating local hooks, a workflow file, or managed identity with verified protected admission.

## 10. Integrated verification and owner gates

- [x] 10.1 Run `gofmt` on changed Go files and keep generated/build/temp/runtime artifacts outside the repository and covered by existing or updated ignore rules.
- [x] 10.2 Run `go test ./...` and retain the exact result; stop at the first persistent deterministic failure rather than widening the milestone.
- [x] 10.3 Run `go vet ./...` and the repository's existing canon/conformance checks with the pinned required tooling.
- [x] 10.4 Run `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate project-lineage-admission-v0 --strict` after implementation updates.
- [x] 10.5 Cross-compile every supported platform artifact and verify the setup-bundle manifest/checksum/version matrix without publishing it.
- [x] 10.6 Run one local scratch-repository canary from init through clone/worktree discovery, exact setup simulation, confirmed fixture intent/change, WorkSpec/run receipt, late lineage, local/shared verification, exception visibility, and migration rollback.
- [x] 10.7 Confirm the canary performed no real provider launch, credential use, external settings mutation, commit/push/PR, release, deployment, or Deltacue mutation.
- [x] 10.8 Stop with an implementation and proof receipt for owner review; real GitHub required-check activation, Goalrail self-dogfood, Deltacue migration, commit, push, PR, and release each remain separate explicit owner gates.
