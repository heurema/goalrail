## Why

PR #62 landed before nine material review findings arrived, leaving current `main` with authorization, lineage-completeness, governance-integrity, policy-matching, verifier-totality, and setup-planning defects. These defects must be corrected before Goalrail's prepared shared-admission workflow is activated or the affected behavior is released.

## What Changes

- **BREAKING** Harden shared admission so branch-controlled owner-decision claims cannot exercise owner authority; only authenticated provider evidence or an equivalently unforgeable decision artifact can satisfy the owner boundary.
- **BREAKING** Construct lifecycle completeness from the committed work-unit graph plus bounded authenticated provider observations, allowing a normal GitHub pull request to become valid without precommitting or fabricating reviews, checks, or owner decisions.
- **BREAKING** Require semantic validation of confirmed-intent and terminal-receipt replicas before their lineage relations count as present; candidate intent and every non-`passed` receipt remain non-admissible.
- **BREAKING** Reject non-bootstrap project-identity changes and treat policy, bootstrap, and setup-profile digest mismatches as invalid head governance.
- Make repository-root prefix rules match every normalized repository path while preserving existing priority, classification, rule identity, and evidence requirements.
- Make admission verification total for every domain-valid packet: absent or incompatible trusted-time evidence produces a canonical non-valid result instead of a panic.
- Make setup planning reject executable replacements that the represented apply and rollback contracts cannot safely execute and recover.
- Add focused adversarial regression coverage for all nine PR review threads, with failing-before evidence against merge commit `6a896355` and passing-after verification at the owning package boundaries.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Authenticate owner-decision authority independently of branch-authored lineage | OUT-1, SIG-1 | NG-1, NG-2, NG-4 |
| Incorporate bounded provider evidence into effective completeness | OUT-2, SIG-2 | NG-1, NG-3, NG-4 |
| Validate confirmed-intent and successful terminal-receipt semantics | OUT-3, SIG-3 | NG-3, NG-4 |
| Preserve project identity and verify every declaration-bound governance artifact | OUT-4, SIG-4, SIG-5 | NG-4, NG-5 |
| Correct repository-root policy matching | OUT-5, SIG-6 | NG-3, NG-4 |
| Return canonical denial for missing or incompatible trusted time | OUT-6, SIG-7 | NG-3, NG-4 |
| Align setup plan completeness with executable apply and rollback capability | OUT-7, SIG-8 | NG-1, NG-4 |
| Add traceable regressions and run the complete verification suite | OUT-8, SIG-9, SIG-10 | NG-1, NG-5, NG-6 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `lineage-admission`: tighten owner authority and provider provenance, incorporate authenticated provider observations into admission completeness, validate semantic evidence before normal validity, make packet-time handling total, and make `.` a true repository-root prefix.
- `work-unit-lineage`: require semantically valid confirmed intent, successful terminal execution evidence, and authenticated late provider relations before a work unit becomes admission-ready.
- `project-governance`: keep project identity immutable outside explicit bootstrap or migration and verify every declaration-bound governance artifact at the evaluated revision.
- `managed-setup`: make a setup plan incomplete when an inspected executable replacement cannot be safely applied and recovered under the represented contract.

## Impact

- Affected packages: `internal/admission`, `internal/admissiongit`, `internal/githubadmission`, `internal/domain`, `internal/localrun`, and `internal/setup`, plus their focused tests and conformance fixtures.
- Affected behavior: some ranges or setup states previously accepted, counted complete, or marked executable will now fail closed with canonical evidence.
- Public contract impact: admission and setup behavior becomes stricter; delta specs must preserve stable machine-readable outcomes and define any reason-code additions before implementation.
- Dependencies and infrastructure: no new runtime dependency, service, database, provider, workflow engine, or deployment plane.
- Operations: the prepared Goalrail workflow remains inactive in this repository; activation, release, and branch-protection changes remain separate owner gates.

## Non-Goals

- Do not activate a shared workflow or required check, change a ruleset, publish a release, deploy, install on an owner machine, mutate Deltacue, or perform another external effect.
- Do not weaken required lineage, accept branch-controlled authority, count candidate intent or failed receipts as valid, or hide missing provider evidence to make admission pass.
- Do not add a central service, database, authorization product, workflow engine, provider-specific canonical domain, or broad setup redesign.
- Do not rewrite the archived PR #62 change, proof, or review evidence.
- This proposal does not authorize implementation, commit, push, pull-request creation, review-thread mutation, revert, merge, release, or activation.
