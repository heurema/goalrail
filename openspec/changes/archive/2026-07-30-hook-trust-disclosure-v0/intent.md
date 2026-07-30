# Intent Snapshot

- **Intent ID:** intent-hook-trust-disclosure-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-hook-trust-disclosure-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Verify the background attachment on a live session, then fix what the verification exposes; prefer a structurally safer arrangement if one exists, and search for a solution before accepting a limitation.
- **SE-2 (repository):** The live verification proved the built chain works once hooks run, and exposed that connection leaves the user with a silent, non-working attachment until an unrelated manual step is performed.
- **SE-3 (external):** A registered hook runs only after its exact definition is reviewed and trusted by hash, managed through the scaffold's own `/hooks` surface.
- **SE-4 (external):** Moving the attachment into the repository is externally blocked: project-local hooks are silently skipped with no trust prompt, the session-start event has passed by the time the user can act, and the defect is open with no workaround.
- **SE-5 (external):** No supported way exists for an installer to obtain hook trust; integrators reproduce the trust hash and write it directly, depending on private implementation details.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Make connection disclose what remains to be done: after registering hooks, the command states in plain terms that the scaffold requires the user to review and trust them, names the surface where that happens, and says the attachment does not act until then. Registration without disclosure delivers silence the user reads as breakage. | Connect and read the output; confirm the trust step and the surface are named. | SE-2, SE-3, CTX-2, CTX-3 |
| OUT-2 | Report attachment health on demand: a diagnostic command tells the user whether the scaffold is connected, whether this repository is initialized, and whether the registered hooks are trusted, naming the next action for whatever is missing. | Run it in each state — unconnected, connected but untrusted, fully working — and confirm each answer and next action. | SE-2, SE-3, CTX-2, CTX-7 |
| OUT-3 | Never forge trust. Goalrail MUST NOT write, compute, or reproduce the scaffold's hook-trust records, even where that is technically possible and practised elsewhere; consent to run a command in every session belongs to the user, through the scaffold's own surface. | Confirm no code path writes trust state, and that the prohibition is stated in the contract. | SE-5, CTX-6 |
| OUT-4 | State the accepted limitation honestly: attachment is registered at user scope, so the hook is invoked in every session and acting only inside initialized repositories is enforced by Goalrail's own first act rather than by absence. Record that the structurally stronger arrangement was attempted and is externally blocked, with the evidence. | Read the recorded limitation and confirm it names the cause and the evidence rather than presenting the arrangement as preference. | SE-4, CTX-4, CTX-5 |
| OUT-5 | Keep the working chain untouched: the announcement, retention, intent binding, and fail-quiet posture proven by the live verification are not modified by this change. | Confirm the diff adds disclosure and diagnosis only. | SE-2, CTX-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not write, compute, or reproduce scaffold trust records, and do not automate the user's approval in any form. | SE-5, CTX-6 |
| NG-2 | Do not adopt project-local hooks while they are silently skipped; do not work around the external defect by other means. | SE-4, CTX-4, CTX-5 |
| NG-3 | Do not change the announcement text, retention, intent binding, fail-quiet posture, or the wrapper lifecycle. | SE-2, CTX-1 |
| NG-4 | Do not add a daemon, control plane, dialogue, or any polling of scaffold state. | CTX-7 |
| NG-5 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, provider run, or external effect. Each remains a separate owner gate. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A newly connected user knows what to do next. | Connection output names the trust requirement, the surface, and the fact that nothing acts until then. | SE-2, CTX-3 |
| SIG-2 | Every failure state has an answer. | The diagnostic distinguishes unconnected, uninitialized, and untrusted, and names the next action for each. | SE-3, CTX-2 |
| SIG-3 | No trust is ever forged. | No code path writes scaffold trust state; a test pins the prohibition. | SE-5, CTX-6 |
| SIG-4 | The limitation is documented with its cause. | The recorded limitation names the attempted alternative, why it fails, and the external evidence. | SE-4, CTX-5 |
| SIG-5 | The verified chain is unchanged. | The announcement text, retention, binding, and fail-quiet behaviour keep their existing tests unmodified. | CTX-1 |

## Ambiguities and Unknowns

None blocking. One recorded limit: the second supported scaffold's trust
behaviour has not been observed; the live verification covered one scaffold.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-29T18:58:00Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — disclosure of the trust step after connection, an on-demand health command, the refusal to forge trust even though other integrators do it, the honestly recorded scope limitation with the externally blocked alternative, and leaving the live-verified chain untouched — together with the recorded limit that the second scaffold's trust behaviour was not observed, and explicitly confirmed the snapshot.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
