# Intent Snapshot

- **Intent ID:** intent-blocked-terminal-outcome-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-blocked-terminal-outcome-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Move the escalation channel out of the kata and into Goalrail itself, as a terminal outcome rather than a mid-run dialogue, and represent it as a first-class `blocked` status with explicit receipt versioning (variant A).
- **SE-2 (repository):** Measured behaviour: agents return a structured question when the environment accepts one machine-checkably, and guess otherwise; prose caveats block nothing.
- **SE-3 (repository):** The five-lens review reshaped the design: no WorkSpec change, a reserved escalation path, eager append-only retention of the question, opaque-bytes hygiene validation, and explicit state rules; the recurrence claim was cut as undeliverable.
- **SE-4 (repository):** The receipt today has no schema field and its validator whitelists five statuses, so a new status requires an explicitly versioned receipt.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Add a terminal run state `blocked` to the local-run contract: the provider run ended without an implementation because the work item cannot be completed as specified, and an owner decision is required. Blocked is terminal — answering it never resumes the run; the answer creates a new intent version and a new one-shot run. | Inspect the state set and a blocked receipt; confirm no resume path exists. | SE-1, SE-2, CTX-1, CTX-5, CTX-8 |
| OUT-2 | Reserve one constant repository-relative escalation path (no WorkSpec schema change). At `gr prepare`, fail closed when the path already exists in the frozen baseline observation. At provider observation in `Start`, capture the artifact in the same worktree snapshot that decides the delta, validate hygiene only (regular file, size cap, UTF-8, non-empty, secret-shape scan), and retain its exact bytes append-only in the run store. | Confirm `goalrail.work-spec/v0` is byte-identical, a frozen v0 fixture digest is unchanged, and a blocked receipt references retained bytes rather than a mutable worktree file. | SE-3, CTX-3, CTX-4 |
| OUT-3 | Define the state rules: `blocked` requires a valid escalation artifact AND an empty in-scope delta; artifact plus in-scope edits yields `failed` with an explicit reason while still retaining the artifact; `unlinked`, `denied` and `launch_failed` take precedence over `blocked` and record the question digest as evidence without minting a blocked status; supplied check results are recorded verbatim and the status is never `passed` while the artifact exists. | Review deterministic tests covering each rule, including the hedge case and each precedence case. | SE-3, CTX-1, CTX-4, CTX-6 |
| OUT-4 | Version the terminal receipt explicitly as `goalrail.terminal-receipt/v1`: add a schema field, an escalation block (path, digest, retained reference), and the frozen intent reference (id, version, digest). Legacy receipts without the schema field remain readable as v0. | Inspect a v1 receipt and confirm an existing v0 receipt still validates. | SE-4, CTX-2, CTX-8 |
| OUT-5 | Make the question-to-decision chain machine-followable: the blocked receipt carries the intent reference and the escalation digest; the answering intent version may carry a `resolves` record (run id, escalation digest) and a disposition (`answered`, `spurious`, `withdrawn`) parsed by the existing OpenSpec adapter. | Follow the chain from a blocked receipt to the answering intent version using recorded identifiers only. | SE-3, CTX-6, CTX-7 |
| OUT-6 | Publish the escalation payload shape as a small provider-neutral schema (`goalrail.escalation/v0`: block class, sources, optional separating example, optional per-source interpretations, one question) that downstream tooling validates; Goalrail itself treats the artifact as opaque bytes beyond hygiene. | Confirm Goalrail performs no semantic validation and the schema exists as a spec artifact consumable by the kata oracle. | SE-3, CTX-6 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No mid-run dialogue, no control plane, no session resume, no background continuation. Blocked is terminal and the loop closes only through a new intent version and a new run. | CTX-5 |
| NG-2 | No change to `goalrail.work-spec/v0`: no new WorkSpec field, no digest impact on frozen WorkSpecs. | SE-3, CTX-3 |
| NG-3 | No recurrence registry and no claim that an answered conflict cannot recur; the delivered property is a machine-followable chain, not prevention. | CTX-7 |
| NG-4 | No semantic validation of the question inside Goalrail (no source parsing, no example execution); hygiene checks only, consistent with Goalrail never running checks. | SE-3, CTX-6 |
| NG-5 | Blocked is never success-adjacent: it cannot be recorded as `passed`, cannot suppress supplied check results, and is not a fourth per-check result state; the `gr finish` result grammar is unchanged. | SE-3, CTX-6 |
| NG-6 | This snapshot authorizes no implementation, commit, push, PR, merge, provider run, or external effect; each remains a separate owner gate. | CTX-5 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A blocked run is fully reconstructible from append-only evidence. | The receipt alone yields the frozen WorkSpec digest, intent reference, escalation digest, and the retained question bytes; deleting the worktree file after the run changes nothing. | SE-3, CTX-2, CTX-4 |
| SIG-2 | The contract cannot be closed as done while the decision pends. | With a valid retained artifact and clean in-scope delta, `gr finish` yields status `blocked` for any supplied inputs; no input combination yields `passed`. | SE-2, CTX-1 |
| SIG-3 | Gaming paths are closed by construction. | Deterministic tests: pre-existing artifact fails `prepare`; artifact plus edits yields `failed`; unlinked precedence holds; verbatim check results are preserved on a blocked run. | SE-3, CTX-6 |
| SIG-4 | Existing evidence remains valid. | A checked-in frozen v0 WorkSpec fixture keeps its digest byte-for-byte; an existing v0 receipt still validates; strict OpenSpec validation passes across all specs. | SE-4, CTX-2, CTX-3 |
| SIG-5 | The chain is followable without conventions. | From a blocked receipt, recorded identifiers alone lead to the answering intent version and its disposition. | SE-3, CTX-7 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-29T12:30:50Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — the terminal blocked outcome with no resume path, the reserved escalation path that leaves the WorkSpec schema untouched, the state and precedence rules, the explicitly versioned receipt, the machine-followable question-to-decision chain, and the cut recurrence claim — and explicitly confirmed the snapshot by name and version.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
