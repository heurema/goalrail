## Why

Goalrail currently treats participation as checkout-local setup, so a linked worktree or fresh clone can appear unmanaged and material work can bypass the intent-to-receipt flow. The confirmed intent requires managed-project identity, friendly consented setup, and deterministic shared admission for one reconstructable work-unit lineage.

## What Changes

- Add a versioned committed Goalrail project declaration and thin human/Codex/Claude bootstrap guidance so project identity survives clones and worktrees.
- Add a read-only setup planner and a consent-bound setup executor contract that installs only an enumerated integrity-verifiable bundle, reports every mutation and rollback, emits a setup receipt, and returns the agent to the original intent gate.
- Add a provider-neutral work-unit lineage record that references existing intent, change, WorkSpec, run/session, commit, pull-request, review/comment, check, terminal-receipt, and owner-decision evidence without copying their semantic content.
- Add one deterministic bounded-diff lineage verifier with categorical reasons, committed materiality and exception policy, local early-check integration, and a shared-admission command suitable for a protected required check.
- Change initialization, diagnosis, update, ambient discovery, WorkSpec binding, CLI help, and release guidance to use the committed project contract and distinguish managed identity, local readiness, lineage readiness, and shared-admission activation.
- Add one explicit migration from the v0.1.8 checkout-local marker layout; retained legacy evidence remains readable, while new conformant work uses the new project and WorkSpec contracts.
- **BREAKING:** redefine `initialized` from a worktree-local marker state to committed managed-project identity, version the structured diagnosis contract, and require new WorkSpecs to bind one managed project and one current Goalrail change. Compatibility aliases and legacy readers remain bounded migration aids, not a conformant verdict.
- Do not activate external CI requirements or branch protection automatically; Goalrail provides and diagnoses the shared check, while enabling an external protected boundary remains separately authorized.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Committed declaration, clone/worktree recognition, and supported-agent bootstrap | OUT-1, OUT-2, OUT-9, OUT-10; SIG-1, SIG-2, SIG-11, SIG-12 | NG-1, NG-7, NG-8 |
| Exact read-only setup plan, explicit permission, bounded execution, setup receipt, and resume | OUT-3, OUT-4; SIG-2, SIG-3, SIG-4 | NG-4, NG-6, NG-8 |
| Single Goalrail flow authority and managed-project/change binding | OUT-5; SIG-5 | NG-3, NG-5, NG-7 |
| Provider-neutral work-unit lineage with late backward references | OUT-6; SIG-6 | NG-3, NG-7 |
| Deterministic local/shared verifier and honest enforcement health | OUT-7, OUT-10; SIG-5, SIG-7, SIG-10, SIG-12 | NG-1, NG-2, NG-6, NG-8 |
| Committed materiality and explicit exception classifications | OUT-8; SIG-8, SIG-9 | NG-5 |
| One project-scoped legacy migration with retained historical evidence | OUT-9; SIG-11 | NG-3, NG-8 |
| Versioned update and drift handling for governing files | OUT-10; SIG-12 | NG-4, NG-7 |

## Capabilities

### New Capabilities

- `project-governance`: Committed managed-project identity, policy version, supported-agent bootstrap contract, clone/worktree recognition, and project-scoped migration state.
- `managed-setup`: Read-only setup planning, exact permission scope, integrity verification, bounded installation and attachment, rollback evidence, setup receipts, and return to the original intent gate.
- `work-unit-lineage`: Provider-neutral work-unit identity and append-only references across project policy, intent, change, WorkSpec, runs/sessions, Git and pull-request evidence, reviews/checks, receipts, exceptions, and owner decisions.
- `lineage-admission`: Deterministic materiality classification and bounded-diff verification, categorical verdicts, optional local early checks, shared-admission behavior, and enforcement-health truthfulness.

### Modified Capabilities

- `harness-init`: Replace checkout-local project identity with a committable project declaration and generate the supported bootstrap/admission payload; make local attachment a reported setup result rather than the source of project identity.
- `harness-doctor`: Report managed identity, local setup, governing-file drift, lineage readiness, and shared-admission activation independently, with exact remedies and a versioned machine contract.
- `harness-update`: Update canon-owned governance/bootstrap/admission files without overwriting repository-owned policy, and preserve an auditable rollback path.
- `ambient-connect`: Scope ambient behavior by the committed managed-project declaration rather than a per-worktree marker while remaining inert in unrelated repositories.
- `work-spec`: Require new WorkSpecs to bind the managed-project declaration and exact current Goalrail change in addition to confirmed intent and pinned repository state.
- `operator-cli-help`: Replace the claim that ordinary work needs no per-task Goalrail flow with discoverable managed setup, intent/change, lineage verification, and admission guidance.
- `release-channel`: Make the agent-facing installation route produce an exact integrity-verifiable setup plan with durable user-local placement and no unlisted dependency or credential action.

## Impact

- Affected surfaces: `cmd/gr`, ambient initialization and health, embedded canon/templates, domain WorkSpec validation, local run preparation, release/install documentation, and new project/setup/lineage/admission packages.
- Public contracts: new committed declaration and policy files, setup plan/receipt, work-unit record, verifier JSON and exit codes, diagnosis schema version, and a new WorkSpec schema version.
- Repository integrations: generated thin bootstrap guidance and a shared-check adapter/configuration; external required-check or branch-protection activation remains manual and owner-gated.
- Dependencies: keep the `gr` runtime self-contained; reuse published release checksums and existing standard-library conventions. `prek` is optional. No central service, database, or mandatory new local framework is introduced.
- Migration: v0.1.8 markers remain detectable for one explicit migration path; existing immutable intent/run/receipt evidence is referenced, not rewritten.

## Non-Goals

- Physically prevent arbitrary local filesystem writes in an uncontrolled clone or introduce a controlled write gateway/sandbox.
- Treat local hooks, commit trailers, bootstrap prose, or `prek` as an authoritative admission boundary.
- Accept parallel intent/spec sources or copy existing canonical artifacts into a second lineage source of truth.
- Install or change any unlisted dependency, login, credential, global configuration, or external service under general feature/setup consent.
- Add broad small/docs/generated/tool-based exemptions; uncertain materiality remains non-valid until policy or an explicit bounded exception resolves it.
- Add a central server, database, dashboard, workflow engine, authorization product, or provider-specific canonical type.
- Implement, install, mutate Deltacue, commit, push, create a pull request, enable CI/branch protection, release, publish, deploy, or use credentials merely because this proposal exists.
