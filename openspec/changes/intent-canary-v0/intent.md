# Intent Snapshot

- **Intent ID:** `INTENT-CANARY-V0`
- **Version:** 1
- **Status:** confirmed
- **Owner:** Goalrail owner
- **Run references:** `change_id=intent-canary-v0`; automatic `run_id` and root Codex `session_id` capture is implemented and evidenced by the correlation probe and synthetic receipt

## Source Evidence

- **SE-1 — Owner statement:** If an agent misunderstands intent, successful downstream execution is not useful; intent must therefore be the first-class upstream artifact.
- **SE-2 — Owner statement:** OpenSpec should be customized by strengthening its exploration stage with explicit intent handling, while CodeSpeak, Codeplain, and similar work are idea sources rather than dependencies.
- **SE-3 — Owner statement:** Correlation must be automatic rather than maintained through manual copying, because manual process creates avoidable omissions and errors.
- **SE-4 — Owner statement:** Instrumentation should begin with broad useful capture and be reduced with evidence; the current setup should keep one shared Langfuse project with project tags.
- **SE-5 — Owner decision:** Use a small baseline comparison, avoid over-engineering, and start the first bounded Goalrail step now.
- **SE-6 — Project constraint:** This change plans the first slice only. It must not silently discard the retained runtime architecture hypotheses or turn OpenSpec into Goalrail's canonical runtime model.
- **SE-7 — Owner decision:** A Goalrail-launched Codex App task may use the immutable `threadId` returned by the provider-authoritative `create_thread` response as its root session identity. CLI runs continue to use lifecycle-hook identity; manually created App tasks remain `UNLINKED` and ineligible for the v0 canary.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Every flow change begins with a versioned Intent Snapshot that separates source evidence from the agent's interpretation. | Inspect the artifact and its lifecycle fields before proposal generation. | SE-1, SE-2 |
| OUT-2 | The owner actively confirms desired outcomes, non-goals, and observable success signals before they can compile into a proposal. | Attempt proposal creation before and after confirmation; record the owner's assessment action. | SE-1, SE-2 |
| OUT-3 | A change is joined automatically to its run and root Codex session, with missing links exposed as `UNLINKED` and no guessed identity. | Exercise start, resume, subagent, missing-context, and concurrent-change cases. | SE-3 |
| OUT-4 | A bounded real-work canary compares the intent-first flow with a baseline and records enough evidence to judge intent match, process cost, and whether misunderstandings were prevented. | Complete 15 assigned changes using the fixed comparison protocol and review the resulting evidence. | SE-4, SE-5 |
| OUT-5 | The first slice remains replaceable and narrow: it improves Goalrail's own team workflow without committing the runtime to OpenSpec, Langfuse, or an infrastructure provider. | Review dependencies and boundaries before apply and after the canary. | SE-5, SE-6 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | This first planning cycle stops after producing and validating the OpenSpec artifacts; applying them and running the 15-change canary require separate owner approval. | SE-5, SE-6 |
| NG-2 | It does not finalize Temporal versus Restate, OpenFGA versus SpiceDB, or the sandbox ladder. | SE-6 |
| NG-3 | It does not fork the Langfuse Codex plugin, add a dashboard, automate retention, or split projects and credentials. | SE-3, SE-4, SE-5 |
| NG-4 | It does not introduce shadow mutations, a custom workflow engine, or a parallel evidence ledger. | SE-5, SE-6 |
| NG-5 | A 15-change directional canary is not treated as general causal proof. | SE-5 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Lineage is reliable. | At least 13 of 15 changes have a verified `change_id → run_id → session_id` join, with zero wrong joins. | SE-3, SE-4 |
| SIG-2 | Intent match moves in the useful direction. | The flow group's `partial + miss` rate is lower than the baseline group's rate; compare rates, not raw counts. | SE-1, SE-5 |
| SIG-3 | The flow prevents at least one real misunderstanding. | At least one material misunderstanding is documented at the time it is corrected and before delivery. | SE-1, SE-4 |
| SIG-4 | The process remains tolerable for a small team. | Median flow overhead is at most 30 minutes and at least two-thirds of flow changes receive `repeat_optin=yes`. | SE-5 |

Stop or reshape the current flow implementation if any of these signals occur:

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-5 | Process cost is excessive. | Median flow overhead exceeds 30 minutes, or at least 3 flow changes are abandoned because of the process. | SE-5 |
| SIG-6 | Automatic lineage is not dependable. | At least 2 links remain unresolved after one bounded escalation, or any wrong join occurs. | SE-3 |
| SIG-7 | The added flow shows no useful movement. | It prevents no material misunderstanding, its non-match rate is not lower than baseline, and it costs more. | SE-1, SE-5 |
| SIG-8 | Evidence integrity is compromised. | Existing evidence is rewritten retroactively instead of adding a new correction event. | SE-4 |

These stop signals reject or reshape the current flow implementation; they do not by themselves disprove the importance of intent.

## Ambiguities and Unknowns

None.

## Lifecycle Notes

- The bounded probe selected two provider-boundary mechanisms: lifecycle-hook identity for CLI runs and the authoritative `create_thread` launch receipt for Goalrail-launched App tasks. Existing or manually created App tasks without such a receipt remain `UNLINKED`.
- The concrete queue of 15 real changes will be selected when the canary starts. Assignment order and assessment rules are fixed in this change so selection cannot tune the result afterward.

## Confirmation

- **Confirmed by:** Goalrail owner
- **Confirmed at:** 2026-07-21
- **Verification action:** The owner reviewed the proposed first-step framing, accepted the bounded baseline approach and automatic-correlation requirement, then explicitly instructed Codex to write and start this first goal.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
- **Mechanism clarification:** On 2026-07-21 the owner explicitly confirmed the App launch-receipt mechanism after the live probe. This resolves an implementation ambiguity without changing the confirmed outcomes, non-goals, or success signals, so the intent version remains 1.
