# Intent Snapshot

- **Intent ID:** intent-claude-code-matcher-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-claude-code-matcher-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Verify the second scaffold live; study its documentation first, because project-level hooks appear to work there.
- **SE-2 (external):** That scaffold's session-start matcher takes one exact value per entry, and omitting it makes the hook fire on every session-start occurrence including resumption, clearing, compaction, and forking.
- **SE-3 (repository):** The shipped connection routine for that scaffold writes no matcher, so the registration it produces is exactly the fire-on-everything shape, while the promoted contract requires the announcement to accompany only the event that opens a run.
- **SE-4 (external):** Project-level hooks for that scaffold are an ordinary merging settings layer, not a per-hook review gate, so the promoted limitation about user-scope registration was generalised from the other scaffold and does not hold here.
- **SE-5 (repository):** Live verification of that scaffold was attempted and abandoned: an isolated home does not satisfy its login state, and the remaining routes required writing into the owner's working configuration or copying an account file containing unrelated project history.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Register one session-start entry per matcher value that opens a session, so the announcement cannot accompany resumption, clearing, compaction, or forking. The promoted single-delivery rule must hold on every supported scaffold, not only the one whose transport happens to enforce it. | Connect on a stand and read the produced configuration: one entry per opening value, none that fires unconditionally. | SE-2, SE-3, CTX-1, CTX-2, CTX-4 |
| OUT-2 | Keep removal exact after the shape changes: disconnection removes every entry the connection added across all matcher values and leaves foreign entries for the same event untouched. | Connect, disconnect, and compare the configuration byte for byte with its original; repeat with a foreign entry present. | SE-3, CTX-3 |
| OUT-3 | Correct the recorded limitation so it names the scaffold its evidence came from, and state that the other scaffold's project-level hooks are an ordinary merging settings layer rather than a blocked route. A true observation about one provider must not be written as a property of attachment in general. | Read the requirement and confirm the limitation is attributed rather than universal. | SE-4, CTX-5, CTX-6 |
| OUT-4 | Record that the second scaffold has never been exercised live, and that this fix rests on its published matcher contract together with the owner's own working configuration, which agree with each other. | Read the recorded limit and confirm it names what remains unobserved. | SE-5, CTX-7 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not adopt project-level registration in this change; correcting the record is not the same as changing where hooks are installed, and that move deserves its own decision. | SE-4, CTX-5 |
| NG-2 | Do not write into the owner's working scaffold configuration, and do not copy account files containing unrelated history, to obtain a live run. | SE-5, CTX-7 |
| NG-3 | Do not change the announcement text, retention, intent binding, fail-quiet posture, health reporting, or the wrapper lifecycle. | SE-3, CTX-4 |
| NG-4 | Do not claim the second scaffold is verified end to end. | SE-5, CTX-7 |
| NG-5 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, provider run, or external effect. Each remains a separate owner gate. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The single-delivery rule holds on both scaffolds. | The produced registration carries one entry per opening matcher value and none that fires unconditionally; a test pins it. | SE-2, CTX-1 |
| SIG-2 | Removal stays exact under the new shape. | Connect-then-disconnect restores the configuration byte for byte, and a foreign entry for the same event survives both operations. | SE-3, CTX-3 |
| SIG-3 | The limitation is attributed, not universal. | The requirement names the scaffold its evidence came from and states the other's project layer merges ordinarily. | SE-4, CTX-6 |
| SIG-4 | What is unobserved is named. | The record states that no live session has exercised the second scaffold. | SE-5, CTX-7 |

## Ambiguities and Unknowns

None blocking. One recorded limit: the second scaffold's end-to-end behaviour —
announcement delivery, question retention, and whether it prompts for any
approval — has not been observed, and this change does not claim otherwise.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-30T07:15:00Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — one registration per opening matcher value so the announcement cannot repeat inside a session, exact removal under the new shape, correcting the recorded limitation to name the scaffold its evidence came from rather than generalising it, and recording that the second scaffold has never been exercised live — and explicitly confirmed the snapshot.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
