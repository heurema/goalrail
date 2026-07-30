# Intent Snapshot

- **Intent ID:** intent-observe-claude-code-live-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-observe-claude-code-live-v0 version 1
- **Run references:** one non-interactive provider run on a synthetic fixture, recorded at `evidence/live-run-2026-07-30.md`; owner-authorized before it was made

## Source Evidence

- **SE-1 (owner):** Update the promoted record to match what is now known: move the two behaviours that were observed out of "unverified", keep the third and say why it is still open.
- **SE-2 (repository):** The promoted requirement records all three as unobserved, so it now understates what is known.
- **SE-3 (external):** Announcement delivery and question retention were observed in one live run, with the retained payload byte-identical to what the agent wrote and the source file untouched.
- **SE-4 (external):** The approval question stays open, because the run was non-interactive — where the provider documents that the trust dialog is skipped — and the hooks were supplied per-run rather than registered.
- **SE-5 (external):** The blocker recorded for the earlier abandoned attempt was wrong: credentials live in the operating system keychain, not in a file tied to `HOME`, and a per-run settings flag makes registration unnecessary for observation.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | The promoted record states that announcement delivery and question retention are observed on this scaffold, rather than listing them among what has not been verified. | Read the requirement and confirm neither is described as unobserved. | SE-1, SE-3, CTX-1, CTX-5 |
| OUT-2 | The record keeps the approval question open and says why the run does not settle it: a non-interactive session skips the trust dialog by the provider's own account, and per-run hooks are not a registration. An absent prompt under those conditions is not evidence about an interactive session. | Read the requirement and confirm the remaining gap names both reasons rather than only the outcome. | SE-1, SE-4, CTX-6 |
| OUT-3 | The evidence for the observation is kept in the repository rather than only in a conversation: the invocation, the fixture's shape, the agent's reply, the question, and the retained record. | Read the evidence file and confirm each is present. | SE-3, CTX-5 |
| OUT-4 | The record no longer carries the superseded reason for the earlier abandonment, since authentication does not depend on `HOME` and a per-run settings flag removes the need to register anything. | Read the requirement and confirm no claim rests on the withdrawn blocker. | SE-5, CTX-2, CTX-3, CTX-4 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not claim the approval question is answered, and do not treat the absent prompt in a non-interactive run as evidence about an interactive one. | SE-4, CTX-6 |
| NG-2 | Do not change any behaviour: no code, no registration shape, no announcement text, no retention, no intent binding, no health reporting, no wrapper lifecycle. | SE-1 |
| NG-3 | Do not register hooks in the owner's working configuration, and do not copy account files, to obtain further observation. | SE-5, CTX-7 |
| NG-4 | Do not adopt project-level registration or reopen any other deferred decision. | SE-1 |
| NG-5 | This snapshot authorizes no commit, push, pull request, merge, archival, or further provider run. Each remains a separate owner gate. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The record matches what was observed. | The requirement describes announcement delivery and question retention as observed, and neither appears in the remaining gap. | SE-3, CTX-5 |
| SIG-2 | The remaining gap is stated with its reason. | The requirement names both the non-interactive mode and the per-run supply as why the approval question is unsettled. | SE-4, CTX-6 |
| SIG-3 | The observation is reproducible from the repository alone. | The evidence file carries the invocation, the fixture's shape, the reply, the question, and the retained record. | SE-3, CTX-5 |
| SIG-4 | No behaviour changed. | The change carries no Go diff, and every existing test passes unmodified. | SE-1 |

## Ambiguities and Unknowns

One, deliberately preserved: whether this scaffold gates a registered hook behind
user approval. Answering it needs an interactive session with a registration in
the user configuration — the arrangement the owner has twice declined to create
while working inside that same scaffold.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-30T10:40:00Z
- **Verification action:** Owner was shown the live run's result in plain language — the announcement reached the agent, the agent asked instead of guessing with the source untouched, the question was retained byte-identically outside the repository, and the approval question remains open because a non-interactive run skips the trust dialog — together with the proposed scope in their own terms: move the two observed behaviours out of "unverified", keep the third and say why. The owner confirmed that scope explicitly.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
