# Intent Snapshot

- **Intent ID:** intent-verify-intent-context-binding-v0
- **Version:** 2
- **Previous version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-verify-intent-context-binding-v0 version 2
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Fix every confirmed review finding on this change, starting with the snapshot that wrongly declared it specification-only, then stop before merge, archival, and deployment.
- **SE-2 (repository):** The change introduces preparation-time Context Pack verification. Version 1 of this snapshot listed implementation as a non-goal while the change shipped exactly that implementation, and the earlier implementation commit is not in the ancestry of `main`.
- **SE-3 (repository):** Preparation applied no schema-aware rule, so a current project-schema change that omitted both the Context Pack declaration and the artifact bypassed verification, unlike `changeRequiresContext`.
- **SE-4 (repository):** Preparation freezes the WorkSpec before invoking the verifier, so version 1 of the requirement described a boundary that does not hold.
- **SE-5 (repository):** A bounded artifact read that opens a FIFO blocks indefinitely, so preparation hangs instead of failing with a bounded reason.
- **SE-6 (repository):** A Context Pack source cell is one evidence reference and admits no space or semicolon, so two rows of version 1 of the pack were invalid.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Introduce preparation-time verification that reads the confirmed intent together with the sibling Context Pack it declares, with both artifacts resolving inside the canonical repository root under the same bounded size limit, and cover that implementation by this snapshot rather than treating it as pre-existing. | Read the requirement and confirm this snapshot's outcomes describe the shipped implementation. | SE-2, CTX-1, CTX-9 |
| OUT-2 | Fail preparation before any prepared state is persisted, and before any run ID, launch claim, or provider invocation exists, whenever the Context Pack is missing when required, unreadable, malformed, mismatched, oversized, non-regular, or resolving outside the canonical repository root. | Read the failure scenarios and compare them with the regression tests. | SE-2, SE-5, CTX-1, CTX-12 |
| OUT-3 | Require the Context Pack using the same schema-aware distinction the change loader already applies: a current change under the project intent schema always requires one, an archived change requires one only when its intent declares it, and a change under another schema that declares none remains valid. | Confirm preparation and the change loader agree, and that omitting both the declaration and the artifact no longer bypasses verification. | SE-3, CTX-10 |
| OUT-4 | State the boundary that actually holds in the implementation rather than an aspirational one, so the promoted requirement and preparation cannot drift. | Compare the requirement wording with the order of operations in preparation. | SE-4, CTX-11 |
| OUT-5 | Keep the change provider-neutral: add no Context Pack fields to the canonical WorkSpec and depend on no proposal, specification, design, task, provider, or activation artifact. | Confirm the canonical WorkSpec and receipt schemas are unchanged and the requirement names no provider. | SE-2, CTX-5, CTX-6 |
| OUT-6 | Give the behaviour a spec home under a new intent identity and keep this change's own Context Pack valid under the verification it introduces, leaving merged evidence untouched. | Confirm every pack source cell is one valid reference and that no merged archive file is edited. | SE-6, CTX-4, CTX-13 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not change the canonical WorkSpec or receipt schemas, `gr` command behaviour, JSON results, or error contracts. | SE-2, CTX-1 |
| NG-2 | Do not reuse, re-version, amend, or edit `intent-activate-dogfood-run-v0`, any archived change, or any other merged evidence. Version 1 of this snapshot is retained unmodified. | SE-1, CTX-4, CTX-9 |
| NG-3 | Do not restate the intent-level Context Pack semantics already promoted in `intent-snapshot` and `context-pack`; state only preparation-time enforcement. | CTX-5, CTX-6 |
| NG-4 | Do not reorder preparation to make an aspirational boundary true; state the boundary that holds. | SE-4, CTX-11 |
| NG-5 | This snapshot authorizes no merge, archival, spec promotion, deployment, provider run, or external effect. Each remains a separate later owner gate. | SE-1, CTX-8 |
| NG-6 | Do not add a workflow engine, authorization system, effect gateway, credential broker, sandbox platform, queue, daemon, database, dashboard, or general activation surface. | CTX-8 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The promoted `local-run` capability describes preparation-time Context Pack enforcement at the boundary that holds. | One modified requirement covers the accepted path, the schema-aware requirement rule, and every rejection class; strict OpenSpec validation passes. | SE-2, SE-4, CTX-2, CTX-11 |
| SIG-2 | Omitting both the declaration and the artifact no longer bypasses verification. | A current project-schema change without either is rejected; an archived change that declares none, and a change under another schema, still succeed. | SE-3, CTX-10 |
| SIG-3 | A non-regular artifact is rejected without blocking. | A FIFO supplied as the intent artifact or the Context Pack fails with a bounded reason instead of hanging. | SE-5, CTX-12 |
| SIG-4 | This change's own Context Pack passes the verification it introduces. | Every source cell matches the accepted evidence-reference pattern. | SE-6, CTX-13 |
| SIG-5 | The change stays provider-neutral and narrow. | Canonical WorkSpec and receipt schemas are byte-unchanged; the diff touches only this change's artifacts and the intent resolver. | SE-2, CTX-5, CTX-6 |
| SIG-6 | Every scenario maps to a regression test. | Each stated scenario has a named test, and the full suite plus race detector pass. | SE-2, CTX-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-26T09:47:09Z
- **Verification action:** Owner reviewed the owner-facing summary of candidate version 2, together with the fix and evidence for each confirmed review finding, and explicitly confirmed the snapshot by name and version in the current Goalrail task.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
