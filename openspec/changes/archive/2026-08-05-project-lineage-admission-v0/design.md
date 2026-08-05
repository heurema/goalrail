## Context

The decision question is: **where can Goalrail place a durable, enforceable project-governance boundary while keeping a clean clone friendly and preserving one reconstructable work-unit chain?**

Verified constraints are that current identity is a per-worktree marker, the current hook is fail-quiet rather than an admission mediator, only tracked content reaches another clone, existing WorkSpec/run/receipt contracts already cover the middle of the lineage chain, installation is separately authorized, and canonical semantics must remain provider-neutral [CTX-1, CTX-2, CTX-3, CTX-4, CTX-6, CTX-11, CTX-13, CTX-14]. The owner confirmed that protected shared admission, not physical prevention of arbitrary local byte writes, is the required ironclad boundary and that clean-machine recovery is an exact permissioned agent-installed bundle [CTX-8, CTX-9, CTX-12]. One independent critique supported that boundary and rejected local hooks as authority [CTX-10].

Material unknowns are none. Exact file formats, verdict labels, adapter boundaries, and first provider integration are implementation decisions already left open by confirmed intent. The decision deadline is this planning cycle; implementation and any external activation remain later gates.

Stakeholders are repository owners, developers cloning a managed project, supported coding agents, reviewers, and operators who must diagnose or reconstruct a work unit.

## Goals / Non-Goals

**Goals:**

- Make managed-project identity committed, versioned, and clone/worktree-stable.
- Give a clean supported agent one read-only setup plan, one exact consent gate, a bounded installation, a setup receipt, and automatic return to candidate intent.
- Extend existing immutable Goalrail evidence into one provider-neutral work-unit graph.
- Evaluate one frozen base/head range with the same deterministic verifier locally and at shared admission.
- Keep exceptions explicit and auditable without calling them normal validity.
- Migrate v0.1.8 projects once without rewriting retained evidence.

**Non-Goals:**

- Prevent arbitrary local filesystem writes in an uncontrolled clone.
- Treat hooks, commit trailers, `prek`, agent prose, or a workflow file as proof of protected enforcement.
- Add a server, database, workflow engine, authorization product, or sandbox.
- Introduce a parallel intent, WorkSpec, receipt, or review source of truth.
- Silently install dependencies, authenticate, modify global configuration, or activate an external required check.
- Migrate Deltacue, publish a release, or perform a real provider/CI canary in this change without separate authorization.

## Decisions

### D1. Put project identity in a committed declaration; keep machine attachment separate

Selected layout [CTX-1, CTX-2, CTX-3, CTX-8, CTX-11]:

- `.goalrail/project.json` — `goalrail.project/v1`, immutable random project ID, contract revision, and digest-bound references to policy, bootstrap, planning, and admission profiles.
- `.goalrail/policy.json` — `goalrail.policy/v1`, repository-owned materiality, exception, evidence, and owner-decision rules.
- `.goalrail/bootstrap.md` — canonical human- and model-legible flow without provider-specific semantics.
- `.goalrail/setup-profile.json` — `goalrail.setup-profile/v1`, compatible Goalrail bundle constraint, planning compiler adapter and exact compiler version, supported scaffold adapters, and prepared shared-admission adapter.
- Root `AGENTS.md` and `CLAUDE.md` managed blocks — thin links to `.goalrail/bootstrap.md` for supported agents.

`project_id` is generated once with `crypto/rand`, encoded without machine or remote identity, and committed. Discovery uses `lstat` at the Git worktree root. A missing reserved path means unmanaged; a present but unsafe or invalid path means declared-invalid. A local marker never decides project identity.

Initialization creates a root adapter file only when absent or updates one uniquely delimited Goalrail block whose digest matches a retained canon. If an owner file exists without a safe managed block, initialization leaves it unchanged, writes the canonical snippet under `.goalrail/`, and reports an owner reconciliation gate. This is less automatic for conflicting repositories but avoids silently overriding existing agent authority.

Alternatives:

1. **Selected: committed declaration plus separate local readiness.** Survives Git topology and is reversible through normal versioned project changes.
2. **Rejected: move the marker into the Git common directory.** Repairs linked worktrees on one machine but not fresh clones or parallel workflow authority.
3. **Rejected: infer Goalrail from `openspec/config.yaml`.** Confuses a replaceable compiler with canonical project governance and cannot distinguish copied schema files from an owner declaration.

### D2. Evaluate policy from the protected base, never from the candidate head alone

The verifier uses the base revision's valid project declaration and policy as authority for the base/head range [CTX-8, CTX-9, CTX-10, CTX-14]. A candidate change to `.goalrail/project.json`, policy, bootstrap, or admission configuration is itself material and cannot weaken the rules used to judge that same change. The head declaration is also validated so a removal, identity substitution, or unsupported migration is visible.

For the first declaration where no base policy exists, only the explicit `BOOTSTRAP` initializer/migration contract can provide bounded authority. After integration, the committed policy governs subsequent changes. Equal-priority overlapping policy rules are `AMBIGUOUS`; the implementation does not guess specificity.

The v1 matcher grammar is deliberately small: exact repository-relative path, literal path prefix, change kind, and explicit integer priority. It has no regex, shell glob, platform path separator, or free-form semantic classifier. All tracked paths are material by default. Canonical Goalrail evidence paths are a distinct evidence class that must validate but do not recursively require a second work unit. A generated-path exemption requires a committed generator/check identity.

Alternative rejected: evaluate only the head policy. It lets one pull request exempt itself and defeats the admission boundary.

### D3. Distribute one integrity-verifiable setup bundle for each supported platform

The release channel adds an optional `goalrail-setup_<os>_<arch>_<version>` archive alongside the minimal `gr` archive [CTX-3, CTX-12, CTX-13]. The setup bundle contains:

- the `gr` binary;
- the private runtime needed by the selected current planning compiler;
- the exact pinned compiler package and its runtime dependency tree;
- a bounded manifest naming every file, version, license/provenance reference, size, and SHA-256 digest.

`gr` remains a native self-contained binary: `init`, `doctor`, `update`, lineage inspection, and verification do not require the private planning runtime. The toolchain is invoked only through the declared replaceable planning adapter. Bundle extraction targets a versioned user-local directory under the platform's Goalrail data root; a stable user-local executable path points to the selected verified `gr`. No package manager, root privilege, system directory, global Node installation, or curl-to-shell script is required.

Before consent, the supported agent reads committed bootstrap/setup metadata, performs local executable and path checks, and reads bounded public release metadata without downloading executable payloads. It presents an exact plan. After consent it downloads the immutable archive and checksum, verifies both, extracts into a temporary directory, runs the bundled `gr setup verify-plan`, and proceeds only if the recomputed plan is semantically identical to the authorized plan. A mismatch requires new permission. The agent then invokes `gr setup apply`; Goalrail owns state checks, durable placement, attachment mutation, rollback set, diagnosis, and receipt. Provider trust remains a user action.

Alternatives [CTX-10, CTX-12, CTX-13]:

1. **Selected: one platform setup bundle plus a verified plan.** Gives one understandable approval and removes package-manager/global-runtime variance.
2. **Rejected: install Goalrail, Node, and OpenSpec independently through whatever package managers exist.** The mutations and transitive dependency graph vary by machine and cannot be consented as one exact bundle.
3. **Rejected: manual commands only.** Preserves authority but fails the confirmed friendly experience.
4. **Deferred: a network bootstrap service or signed remote installer.** Adds infrastructure and a larger supply-chain boundary before measured need.

### D4. Use immutable typed lineage events and content-addressed evidence replicas

The canonical work-unit anchor is `.goalrail/work-units/<work-unit-id>/unit.json`; immutable repository-portable events live under `.goalrail/work-units/<work-unit-id>/events/<sha256>.json` [CTX-6, CTX-9, CTX-10, CTX-14]. IDs are random canonical IDs generated before the first managed run. An event contains one typed relation, source and target identities, actor/adapter reference, observed time, work-unit ID, and semantic digest. There is no mutable sequence counter and no last-write-wins operation; the event set is sorted by digest and relation cardinality is checked by policy.

Existing canonical artifacts keep their identity and ownership. When shared verification needs immutable bytes currently held only in the local run store, `gr lineage attach` places the **exact canonical bytes** in `.goalrail/evidence/sha256/<digest>` after validation. This is a content-addressed replica of one artifact identity, not a rewritten receipt or second semantic model. The event references the digest. Byte inequality, schema mismatch, or a second payload for one digest is invalid.

Late Git/forge evidence is attached without rewriting early artifacts:

- commit messages may carry `Goalrail-Work-Unit: <id>` as an early index, but the trailer is never sufficient evidence;
- a PR adapter owns a bounded managed block containing only stable Goalrail references;
- provider review/comment/check/owner-decision identities enter a provider-neutral admission packet as authenticated references and digests, not copied bodies;
- integration or rejection becomes a later closure event resolved from the forge adapter.

Resolvers merge repository events, content-addressed evidence, the existing local run store, and registered provider references into an in-memory graph. Derived reverse indexes are caches, never canonical state.

Alternatives:

1. **Selected: typed event graph plus exact content-addressed replication.** Accommodates identifiers that do not exist at run time and preserves existing artifact identity.
2. **Rejected: commit trailers and PR labels only.** They are mutable metadata and cannot prove intent, scope, checks, or receipts.
3. **Rejected: one mutable lineage JSON document.** Concurrent late bindings require rewrites and invite last-write-wins ambiguity.
4. **Rejected: central lineage database.** Adds availability, identity, migration, and ownership boundaries not required by this slice.

### D5. Version WorkSpec for project, change, and work-unit binding

`goalrail.work-spec/v1` adds `project`, `policy`, `change`, and `work_unit` references to the existing confirmed-intent and repository bindings [CTX-6, CTX-8, CTX-9, CTX-14]. All authority fields contribute to the WorkSpec digest. Provider session, commits, PRs, reviews, mutable check outcomes, and owner decisions remain later lineage events.

The decoder retains legacy WorkSpecs and receipts as immutable historical evidence. New managed runs require v1. A narrowly committed migration rule can classify exact legacy evidence as `BOOTSTRAP`, never `VALID`; no legacy artifact is rewritten.

Alternative rejected: infer change/work-unit identity from filenames, branch names, or session metadata. It is ambiguous and cannot survive provider/tool changes.

### D6. Keep one pure admission core and thin collection adapters

`internal/admission` accepts only frozen, bounded inputs and has no network or credential access [CTX-4, CTX-6, CTX-10, CTX-13, CTX-14]. The public command is:

```text
gr verify-lineage --repo <path> --base <oid> --head <oid> \
  --packet <goalrail.admission-packet/v1.json> --json
```

It emits canonical `goalrail.admission-result/v1` JSON and an allow/deny process exit. The top-level classifications are `VALID`, `MISSING`, `AMBIGUOUS`, `INVALID`, `EXEMPTED`, `BREAK_GLASS`, and `BOOTSTRAP`. Only `VALID` is normal conformance. The three exception classes may have `admission=allow` only under the exact base policy and always remain visibly non-`VALID`.

Stable reason identifiers are constants, for example `DECLARATION_INVALID`, `POLICY_CONFLICT`, `MATERIAL_PATH_UNBOUND`, `INTENT_UNCONFIRMED`, `CHANGE_MISMATCH`, `RECEIPT_MISSING`, `LINEAGE_CONFLICT`, `EXCEPTION_EXPIRED`, and `ACTIVATION_UNVERIFIED`. Human messages are derived and may evolve; automation keys only on schema, classification, admission, and reasons.

Collection adapters may read a forge or CI environment and emit `goalrail.admission-packet/v1`; they cannot alter policy or verdicts. The first adapter is a replaceable GitHub Actions adapter because a concrete pull-request path is required for the V0 vertical slice. It receives read-only workflow credentials, fetches bounded PR/review/comment/check/actor metadata, strips bodies and tokens, writes one temporary packet, invokes the pure verifier, and deletes the packet after the check. Core packages contain no GitHub types.

If another forge is selected later, it implements the same packet conformance suite. If no supported adapter is configured, local verification still works on repository evidence but protected admission remains inactive.

Alternatives [CTX-10, CTX-14]:

1. **Selected: pure core plus replaceable evidence collectors.** Makes local and shared results identical for identical packets and isolates credentials.
2. **Rejected: embed GitHub API lookups in the verifier.** Network timing, credentials, and provider semantics would make the verdict non-deterministic.
3. **Rejected: separate local and CI validators.** They would drift and turn local success into misleading evidence.

### D7. Treat hooks and `prek` as replaceable early-feedback shims

`gr setup apply` installs small Goalrail-owned shims for supported scaffold/Git events where the exact plan permits it [CTX-4, CTX-10, CTX-11, CTX-12]. Each shim resolves the declared project, freezes its bounded local input, and calls the public verifier or diagnosis path. It contains no policy logic. `prek` integration is generated only when the project already selects it or the setup plan explicitly includes it; Goalrail does not install `prek` as a mandatory second dependency.

Skipping, deleting, or failing a local shim cannot produce a valid receipt. It may delay feedback, but the protected shared check re-evaluates the committed range.

Alternative rejected: enforce through local pre-commit/pre-push alone. `--no-verify`, hook path changes, a fresh clone, or another client bypasses it [CTX-10, CTX-11].

### D8. Make shared activation an independently observed layer

Initialization may prepare a GitHub Actions workflow and declare check name `goalrail/lineage`, but it does not mutate repository settings [CTX-10, CTX-13]. The workflow uses a release artifact pinned by the committed setup profile, verifies its digest, and runs the same collector/core path. It requests only read permissions needed for contents and pull-request evidence; any missing access becomes `MISSING` or activation `UNKNOWN`, never a fallback.

`gr doctor` reports shared admission active only when a registered read-only adapter can verify that the protected integration target requires the exact check. A 403, missing authentication, stale receipt, unsupported ruleset syntax, or no remote yields `unknown`. Enabling or changing rulesets/branch protection is a separate owner-authorized external action with its own rollback.

The workflow listens to pull-request changes and review decisions so an approval learned after the code checks can re-evaluate the same head. Merge/rejection closure remains provider evidence queried by the resolver; Goalrail does not create commits or comments automatically.

### D9. Split diagnosis into identity, readiness, lineage, and enforcement

`goalrail.doctor/v2` reports four required booleans plus categorical detail [CTX-1, CTX-4, CTX-10, CTX-11, CTX-12]:

- `managed` — a valid committed declaration exists;
- `locally_ready` — exact bundle, planning compiler, and selected scaffold attachment verify;
- `lineage_ready` — policy and portable evidence contracts validate;
- `shared_admission_active` — protected required-check activation is independently verified.

`working` is true only when every layer required by policy is true. A one-release deprecated `initialized` field mirrors `managed`, never the legacy marker. Unmanaged, declared-invalid, migration-required, setup-required, advisory, activation-unknown, and enforced have separate stable reason codes and exit behavior.

### D10. Preserve explicit authority boundaries through command separation

Commands are grouped without turning intent confirmation into effect authority [CTX-12, CTX-13, CTX-14]:

- `gr init` — fresh project content only; no commit or external activation;
- `gr migrate` — explicit v0.1.8 project declaration migration;
- `gr setup plan|apply|rollback` — exact local enabling action and receipt;
- `gr doctor` / `gr update` — read-only diagnosis and explicit canon update;
- `gr lineage begin|attach|inspect` — work-unit evidence only;
- existing `prepare|inspect|start|finish` — bounded run lifecycle using WorkSpec v1;
- `gr verify-lineage` — pure admission decision;
- internal hook/collector entry points remain hidden from top-level help.

No command implicitly invokes the next state-changing command. Agent bootstrap may continue the conversation after a successful receipt, but Goalrail itself does not confirm intent, start a run, commit, push, open a PR, enable CI policy, or merge.

### D11. Stage the breaking change behind explicit v1 artifacts and dual readers

The implementation targets Goalrail v0.2.0 semantics [CTX-1, CTX-2, CTX-6]. Writers emit only project/work-unit/setup/admission/WorkSpec v1 and doctor v2. Readers retain v0.1.8 marker, legacy WorkSpec, and receipt support for migration and inspection. New validity is never backported onto legacy evidence.

The old `gr init` command name remains, but its output and identity semantics are breaking and documented. `initialized` remains a one-release diagnosis alias. No silent auto-migration occurs.

### D12. Accept the independent critique and defer the controlled-gateway alternative

The critique's main objection—that hooks plus metadata cannot support an ironclad claim—is accepted [CTX-10]. The design therefore makes the protected shared check authoritative, uses backward-reference events, and preserves exception classes as non-`VALID`. Its suggestion of a controlled write gateway is deferred: it would physically constrain local writes but adds sandbox, credential, and execution-plane boundaries excluded by current intent [CTX-8, CTX-10, CTX-14].

Revisit a gateway only if measured incidents show protected admission and supported-agent fail-closed behavior still allow material unlinked work to integrate, not merely because local bytes can be changed.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| Project declaration | Committed worktree root; repository-owned governance claim | Regular bounded `goalrail.project/v1`, supported version, canonical project ID, in-root refs and valid digests | Absent means unmanaged; empty, unreadable, symlink/special file, oversized, malformed, unknown version, path escape, identity substitution mean declared-invalid | Read with `lstat`, bounded bytes, parse, then just-in-time stat/digest recheck before any dependent write | Unit fixtures plus clone/worktree conformance and substitution races |
| Project policy | Committed path and digest referenced by declaration; base revision is authoritative | Bounded `goalrail.policy/v1`, unique rule IDs/priorities, supported matcher kinds, valid authority/exception fields | Missing, unreadable, oversized, malformed, unknown matcher, equal-priority conflict, head-only self-exemption | Base bytes are frozen by Git OID; head policy is candidate evidence and cannot govern its own range | Golden canonicalization, base/head weakening tests, cross-platform path fixtures |
| Bootstrap/setup profile and managed blocks | Committed canon-owned files plus repository-owned selections | Known schema/canon, unique delimited block, exact compiler/bundle constraints | Missing required adapter, duplicate/nested markers, owner-edited unknown block, unsupported profile | Update only known block after digest recheck; preserve all surrounding bytes; conflict stops | Canon digests, round-trip owner-content tests, live scratch clone fixture |
| Public release metadata/checksums | Documented anonymous release channel; metadata locates immutable artifacts but is not itself the artifact | Supported platform, immutable release, matching version/name/size/SHA-256 and manifest | Empty, malformed, authenticated-only, stale incompatible, digest/version disagreement, oversized | Read-only before consent; authorized plan pins immutable identity; changed current pointer cannot alter plan | Release pipeline cross-checks manifest, archives, checksums, and embedded versions |
| Local machine/setup state | Read-only OS, architecture, executable lookup, planned destinations and scaffold configs; not trusted as authority | Supported OS/arch, absent or exact component, writable safe planned path, supported legal config syntax | Unsafe link, escape, unreadable/malformed config, ambiguous executable, unexpected existing bytes, unsupported platform | Plan performs no writes; apply rechecks each target immediately before mutation and stops on change | Temp-home fixtures, alternate config syntax fixtures, race injection, rollback digest checks |
| Setup authorization | Explicit owner response conveyed by supported agent and bound to plan digest | Current complete plan digest, one bounded attempt | Missing/refused, stale plan, generic feature approval, changed plan, unlisted action | Revalidate before first mutation; new fact invalidates authority; no automatic retry | Consent/no-consent tests and mutation audit against plan |
| Confirmed intent and Goalrail change | Repository artifacts under `goalrail-context-intent`; owner confirmation is lifecycle evidence | Exact confirmed version/digest and one current change compiled from it | Candidate, missing, ambiguous, superseded without exact version, parallel source, digest mismatch | Resolved and hashed before WorkSpec freeze; substitution check before prepare | Existing artifact conformance plus new project/change binding fixtures |
| WorkSpec/run/terminal receipt | Existing Goalrail local run store and content-addressed exact replicas | Current WorkSpec v1 for new runs; legacy read-only; valid bounded receipts | Unsupported/malformed, mismatched project/change/work unit, prohibited payload, conflicting digest | Frozen artifacts immutable; attach exact bytes by digest; never rewrite legacy evidence | Existing local-run suite plus replica equality and migration fixtures |
| Git base/head and diff | Explicit immutable OIDs in target repository | Reachable commits, deterministic rename/change metadata, bounded path set | Missing/replaced OID, unrelated repository, path escape, oversized diff budget, dirty range where frozen input required | Resolve OIDs once and use Git object data; local staged mode freezes its own index tree | Git topology matrix: clone, linked worktree, rename, submodule, mode change, hook bypass |
| Lineage events/evidence blobs | Committed work-unit store, existing local store, registered adapters | Canonical bounded schema, valid digest, supported relation, exact work-unit identity | Raw bodies/secrets, unknown relation, digest collision/mismatch, duplicate incompatible binding, oversized set | Append exact new event; identical event idempotent; no overwrite; conflict retained | Property tests for order independence, cardinality, content-addressed replicas, reverse resolution |
| Admission packet | Temporary output of registered local/shared collector; evidence, never authority | `goalrail.admission-packet/v1`, matching project/range/work unit, bounded authenticated refs, trusted explicit evaluation time | Policy supplied by packet, credentials/raw bodies, unsupported provider claim, stale/mismatched range, untrusted time | Core reads once and never mutates; collector deletes temporary packet after result; same bytes mean same result | Cross-adapter conformance corpus and byte-stable local/shared golden results |
| Shared-activation evidence | Read-only forge/ruleset adapter or bounded activation receipt | Exact target/check identity, current successful query, acceptable freshness | Missing access, 403, stale, malformed, alternate target, workflow-file-only claim | Never mutate external settings; expired evidence becomes unknown | Mock API states plus separately authorized live canary only after implementation |
| Legacy marker/overlay | Current checkout and retained old evidence; migration hint only | Known v0.1.8 marker/overlay shapes within bounds | Secret-shaped/oversized/malformed marker, foreign schema, evidence rewrite request | Read for migration diagnosis; never governs new identity; migration writes new files only after explicit command | Pinned v0.1.8 fixture, repeated migration, fresh-clone result |

## Correlation and Evidence

The correlation spine is:

```text
project_id + policy_digest
  -> intent_id/version/digest
  -> change_id
  -> work_unit_id
  -> workspec_id/version/digest
  -> run_id + provider root_session_ref
  -> terminal_receipt_digest
  -> commit OIDs
  -> pull_request_ref
  -> review/comment/check refs
  -> owner_decision_ref
  -> integration/closure ref
```

Every arrow is a typed immutable event. Required single-valued relations reject competing claims; multi-valued relations use sorted digest sets. Repository evidence owns policy, intent/change bindings, WorkSpec references, and portable receipt bytes. The local run store owns provider-authoritative run/session observations. The forge owns PR/review/comment/check/decision/integration facts. Admission packets carry bounded authenticated references across that boundary and are never persisted with credentials or bodies.

A missing source yields `MISSING`; multiple valid-looking claimants yield `AMBIGUOUS`; malformed, substituted, escaped, or prohibited input yields `INVALID`. None is repaired by guessing. A resolver can begin from any stable bead and reports unavailable external access separately from verified absence.

## Measurement and Stop Conditions

Implementation is accepted only when:

- two independent clones and three linked worktrees resolve one project ID with no marker;
- clean-machine fixtures show zero writes before setup consent, exact mutation equality after consent, rollback, and return to candidate intent;
- every spec scenario has a deterministic test or explicitly gated integration check;
- local and shared core invocations produce byte-identical results for the same corpus;
- `--no-verify` and missing hooks are caught by shared simulation;
- a complete fixture resolves every lineage bead, including late PR/review/owner references;
- exception fixtures remain non-`VALID` and ambiguous materiality denies;
- v0.1.8 migration preserves every retained evidence byte and is idempotent;
- doctor never reports full enforcement from local/configured-only evidence;
- `go test ./...`, `go vet ./...`, canon conformance, OpenSpec strict validation, and release-manifest checks pass.

Hard stop and reshape conditions:

- stop if the platform setup bundle cannot pin and verify the complete compiler dependency tree;
- stop if any verifier path needs network, wall-clock, absolute checkout path, or hidden environment state;
- stop if a supported agent adapter cannot be structurally reconciled without overwriting owner instructions;
- stop if shared activation cannot be distinguished from workflow-file presence;
- stop if portable receipt evidence requires semantic rewriting instead of exact content-addressed replication;
- stop a setup attempt at the first plan/state mismatch with no automatic retry;
- stop release or real-provider canary work until separately authorized.

No retry loop exceeds one recomputation after an explicitly changed setup plan; unchanged failure is terminal.

## Risks / Trade-offs

- **[Bundle size and release complexity]** A private planning runtime makes setup archives much larger → retain the minimal `gr` archive, make the setup bundle profile-specific, verify licenses/digests in CI, and stop if the complete dependency graph cannot be pinned.
- **[Instruction-file conflict]** Existing `AGENTS.md` or `CLAUDE.md` may carry competing authority → never infer semantic safety; create/update only absent or proven Goalrail blocks and require owner reconciliation otherwise.
- **[False enforcement confidence]** A workflow can exist without being required → separate prepared, unknown, and verified-active states; never infer branch protection.
- **[Policy self-weakening]** A head revision could exempt itself → evaluate materiality and exceptions from the protected base policy.
- **[Exception normalization]** Teams may treat allowed break-glass as ordinary success → keep classification and admission separate and render non-`VALID` everywhere.
- **[Provider outage or access loss]** PR/review edges may be unavailable → report unavailable, retain stable references/digests, and deny when current policy requires proof.
- **[Local evidence portability]** CI cannot read the local run store → replicate exact validated receipt bytes by content digest, never a lossy summary.
- **[Supply-chain substitution]** Agent bootstrap runs before Goalrail is installed → show immutable archive/digest and destinations before consent, verify checksum before execution, and have bundled Goalrail recompute the plan before apply.
- **[Breaking diagnosis semantics]** Existing automation may read `initialized`/`working` → publish doctor v2, retain a one-release alias, and provide migration fixtures and release notes.
- **[Forge-specific first adapter]** GitHub details can leak into the core → constrain them to packet collection and run the same provider-neutral conformance corpus; no GitHub type enters project, WorkSpec, event, or result schemas.
- **[Local work remains physically possible]** Unsupported tools can still edit bytes → state the boundary honestly and rely on shared admission; revisit a controlled gateway only after measured integration escapes.

## Migration Plan

1. Add schemas, canonicalization, validators, reason codes, and pure fixtures without changing current writers.
2. Add dual-read project discovery and doctor v2; current projects report migration-required, never unmanaged.
3. Add fresh `gr init` v1 output and explicit `gr migrate`; verify byte-stable scratch clones/worktrees and retained v0 evidence.
4. Add setup bundle packaging, release-manifest verification, setup plan/apply/rollback, and clean-machine simulated fixtures. Publishing remains a separate gate.
5. Add WorkSpec v1, work-unit events, content-addressed evidence replication, and resolvers; keep legacy readers.
6. Add the pure verifier, local shims, and hook-bypass tests.
7. Add the GitHub packet collector and prepared workflow with mocked API fixtures. Do not enable a real required check.
8. Dogfood in a scratch repository. A real Goalrail-repository canary, external activation, and Deltacue migration each require their own explicit authorization and receipt.
9. Release as the documented breaking v0.2.0 line only after all deterministic and release-bundle checks pass.

## Rollback

- Before project artifacts are committed, remove the generated uncommitted files or use the initialization report; no clone has inherited them.
- Setup rollback uses the exact receipt/recovery set to restore changed configuration and remove only installed bundle paths proven to belong to that plan. It never deletes unrelated runtimes or user configuration.
- A migrated project can revert the migration commit while v0 readers remain available; retained intent/run/receipt evidence is untouched. Once shared admission is externally required, disabling it is a separate owner-authorized provider rollback and Goalrail only diagnoses the result.
- Code rollout remains dual-read. Reverting new writers leaves project v1 artifacts declared-unsupported rather than silently unmanaged; a compatible binary or project commit revert is required.
- Work-unit events and content-addressed evidence are append-only. Rollback adds a superseding/closure event or reverts the containing Git commit; it never mutates historical event bytes.
- No rollback path performs commit, push, branch-protection mutation, release deletion, or evidence destruction automatically.

## Open Questions

None for implementation planning. Selecting and activating a non-GitHub forge adapter, enabling a real required check, and deciding whether measured incidents justify a controlled write gateway are later owner decisions with separate evidence and authority gates.
