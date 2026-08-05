# Intent Snapshot

- **Intent ID:** intent-admission-hardening-after-pr62
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-admission-hardening-after-pr62-v1 version 1
- **Run references:** initiating Codex task; GitHub PR #62; review `PRR_kwDOTewKTc8AAAABIg7nZA`; audit `task:admission-hardening-after-pr62-audit`

## Source Evidence

- **SE-1 — Owner correction:** The merge of PR #62 was premature because material review comments arrived after the merge; the comments must be checked rather than treated as informational.
- **SE-2 — Verified review result:** CTX-1 through CTX-13 establish that all nine unresolved review findings apply to current `main`: five P1 authorization or admission failures and four P2 correctness failures.
- **SE-3 — Owner scope decision:** After receiving the nine-finding audit and the recommendation to stop release and activation, the owner explicitly approved creation of a candidate Intent Snapshot for one hotfix covering all nine findings.

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Shared admission accepts an owner decision only when the decision is bound to authenticated provider evidence or an equivalently unforgeable artifact. A branch-controlled lineage event, actor label, or repository-root JSON file alone cannot exercise owner authority. | Evaluate equivalent work-unit graphs with a forged repository event, an unauthenticated provider observation, and an authenticated provider decision; require denial for the first two and authorization only for the last. | SE-2, CTX-3 |
| OUT-2 | The effective shared-admission evidence graph incorporates authenticated provider packet evidence for pull requests, reviews, checks, and owner decisions before lifecycle completeness is evaluated. An ordinary GitHub PR can become valid from real provider observations without precommitting or fabricating provider results. | Run the generated GitHub-admission flow against bounded fixtures with valid provider evidence, missing evidence, stale evidence, and branch-controlled substitutes; require `VALID` only for the complete authenticated fixture. | SE-2, CTX-4 |
| OUT-3 | Semantic evidence is checked before a lineage relation counts as satisfied. Confirmed-intent relations require a valid frozen Intent Snapshot with active confirmed status, and terminal-receipt relations require a valid receipt whose terminal status is `passed`. | Substitute candidate, malformed, and confirmed intent replicas plus failed, incomplete, launch-failed, and passed terminal receipts; require only confirmed intent and passed receipt to satisfy their relations. | SE-2, CTX-5, CTX-7 |
| OUT-4 | Durable governance remains declaration-bound across a non-bootstrap change. Project identity cannot change outside an explicit bootstrap or migration path, and policy, bootstrap guidance, and setup profile must all match the paths and digests recorded by the head declaration before governance is valid. | Evaluate non-bootstrap fixtures that change `project_id` or mutate each bound artifact without updating its declaration reference; require canonical denial. Verify a valid unchanged identity and an explicitly authorized bootstrap or migration path remain distinguishable. | SE-2, CTX-6, CTX-10 |
| OUT-5 | Project policy path rules apply to the paths they declare. A prefix matcher rooted at `.` matches every normalized repository path and retains its configured classification, rule ID, priority, and required evidence kinds. | Classify root and nested add, modify, rename, delete, mode, and submodule fixtures under a `.` prefix rule; require the rule to win according to existing priority semantics with no hard-coded fallback. | SE-2, CTX-11 |
| OUT-6 | Admission verification is total for every packet accepted by domain validation: missing trusted evaluation time or incompatible provenance yields a stable canonical non-valid result and never a panic. | Evaluate bounded packets with provenance and nil, zero, stale, future, unauthenticated, and trusted evaluation times; require deterministic reason codes and no panic. | SE-2, CTX-9 |
| OUT-7 | A setup plan marked complete contains only mutations that the verified apply path can execute and recover under the exact inspected pre-state. An existing executable that cannot be safely replaced and restored is reported as a planning conflict before owner authorization. | Plan against absent, already-current, mode-only, safely recoverable, same-size different-digest, and unsafe executable destinations; require complete plans only where apply and rollback can honor the represented mutation. | SE-2, CTX-8 |
| OUT-8 | Each of the nine post-merge findings has a focused regression test at the narrowest owning boundary, and the combined shared-admission and setup behavior remains deterministic across repeated local runs. | Run the owning package suites plus exact adversarial fixtures for all nine findings; require every regression test to fail on merge commit `6a896355` and pass after remediation without weakening existing valid cases. | SE-2, CTX-2, CTX-13 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | This change does not activate a Goalrail GitHub workflow, enable a required check or ruleset, publish a release, deploy, install anything on an owner machine, mutate Deltacue, or use credentials beyond bounded read-only verification. Each remains a separate owner-authorized action after remediation. | SE-3, CTX-12 |
| NG-2 | The hotfix does not treat branch-controlled actor labels, repository files, commit authorship, or local hooks as authenticated owner authority. | SE-2, CTX-3 |
| NG-3 | The hotfix does not weaken required lineage, accept failed or incomplete receipts, reinterpret candidate intent as confirmed, or hide missing provider evidence merely to make the shared check pass. | SE-2, CTX-4, CTX-5, CTX-7 |
| NG-4 | No new central service, database, workflow engine, authorization product, provider-specific canonical domain, or broad setup redesign is introduced. The fix stays within the existing pure verifier, bounded collectors, Git reader, policy matcher, and setup plan/apply contracts. | SE-3, CTX-3, CTX-4, CTX-8, CTX-10 |
| NG-5 | The archived PR #62 planning and proof artifacts are not rewritten to erase the post-merge findings. This change adds a new versioned remediation record and preserves the original evidence. | SE-1, SE-2, CTX-1, CTX-2 |
| NG-6 | Candidate intent does not authorize proposal compilation, implementation, commit, push, pull-request creation, review-thread replies or resolution, revert, merge, release, or activation. | SE-3 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Branch-authored owner decisions cannot authorize admission. | Forged `owner_decision` fixtures return a stable non-valid owner-authority reason; the authenticated provider fixture is the only allowed case. | CTX-3 |
| SIG-2 | Real provider evidence can complete the shared graph. | One ordinary authenticated GitHub packet supplies the pull-request, review, check, and owner-decision relations used by completeness without a branch commit containing those observations. | CTX-4 |
| SIG-3 | Intent and terminal evidence retain their semantics. | Candidate or malformed intent and every non-`passed` terminal receipt remain missing or invalid; exact confirmed intent plus a passed receipt satisfies the corresponding relations. | CTX-5, CTX-7 |
| SIG-4 | Project identity is stable outside explicit migration. | Every non-bootstrap base/head `project_id` mismatch is denied before admission, while unchanged identity remains valid. | CTX-6 |
| SIG-5 | Every declaration-bound governance artifact is verified. | Byte changes to policy, bootstrap, or setup profile without matching declaration digests each set head governance invalid and produce canonical denial. | CTX-10 |
| SIG-6 | Root policy rules really cover the repository. | A canonical `path: "."` prefix rule matches all normalized fixture paths and preserves its rule ID and required evidence kinds. | CTX-11 |
| SIG-7 | Accepted packet shapes cannot crash admission. | Nil-time provenance and all other negative time fixtures return canonical results under normal tests and a panic-detecting regression test. | CTX-9 |
| SIG-8 | Setup authorization never promises a deterministic failure. | A same-size different-digest executable produces an incomplete conflict plan, while current and mode-only repair cases retain their intended behavior. | CTX-8 |
| SIG-9 | All nine review threads have traceable proof. | The remediation change maps each GitHub discussion URL to at least one failing-before/passing-after test and a resulting code or contract change. | CTX-1, CTX-3, CTX-4, CTX-5, CTX-6, CTX-7, CTX-8, CTX-9, CTX-10, CTX-11 |
| SIG-10 | Existing behavior remains green without masking the defects. | `go test ./internal/admission ./internal/admissiongit ./internal/githubadmission ./internal/setup` and the repository-wide verification suite pass after the new adversarial tests are included. | CTX-13 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-08-05T16:22:10Z
- **Verification action:** The repository owner reviewed the complete Russian plain-language view of `intent-admission-hardening-after-pr62` version 1 and replied `да` to the exact-version confirmation question in the initiating Codex task.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
