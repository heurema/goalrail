# Context Pack

- **Context Pack ID:** context-observe-claude-code-live-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-30T10:30:00Z
- **Completed at:** 2026-07-30T10:40:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The promoted requirement states that the second supported scaffold's end-to-end behaviour has not been observed in a live session, naming announcement delivery, question retention, and whether it prompts for any approval, and that its registration shape rests on the provider's published matcher contract plus an agreeing working configuration. | repo:openspec/specs/ambient-connect/spec.md | 2026-07-30T10:30:00Z | This is the record a live run makes partly obsolete; leaving it unchanged would understate what is now known. |
| CTX-2 | repository | An earlier attempt at live verification was abandoned on the belief that the scaffold's login state is tied to `HOME`, leaving only routes that wrote into the owner's working configuration or copied an account file holding unrelated project history. | repo:openspec/changes/archive/2026-07-30-claude-code-matcher-v0/context.md | 2026-07-30T10:31:00Z | The recorded blocker is what made the behaviour unobserved; if it was wrong, the limitation outlived its cause. |
| CTX-3 | external | The scaffold's credentials are held in the operating system keychain, confirmed by finding the credential entry present while no credential file exists beside the settings. The account file is separate from authentication. | local:keychain-probe-2026-07-30 | 2026-07-30T10:32:00Z | The recorded blocker was wrong: authentication does not depend on `HOME`, so the earlier abandonment rested on a mistaken cause. |
| CTX-4 | external | The provider's CLI accepts `--settings <file-or-json>` to load additional settings for one run, and `--setting-sources` to choose which of user, project, and local settings are read. | local:claude-cli-2.1.215-help | 2026-07-30T10:33:00Z | Hooks can therefore be exercised for a single run without registering anything in the owner's configuration, which removes the objection that blocked the earlier attempt. |
| CTX-5 | external | One non-interactive run on a synthetic contradictory task delivered the announcement, produced a question at the reserved path with the source untouched, and retained that question outside the repository with its own identifier, digest, session reference, and an explicit unbound reason. The retained payload was byte-identical to the file the agent wrote. | repo:openspec/changes/observe-claude-code-live-v0/evidence/live-run-2026-07-30.md | 2026-07-30T10:36:46Z | Announcement delivery and question retention are now observed on this scaffold rather than inferred. |
| CTX-6 | external | The run was non-interactive, and the provider's own help states that the workspace trust dialog is skipped in that mode. The hooks were also supplied per-run rather than registered in the user configuration. | local:claude-cli-2.1.215-help | 2026-07-30T10:37:00Z | No approval prompt appeared, but both differences bear on precisely that question, so the run says nothing about an interactive session with a registered hook. |
| CTX-7 | repository | The owner's configuration was checked after the run and held zero managed handlers. | local:post-run-configuration-check | 2026-07-30T10:37:30Z | The observation was obtained without the write that the earlier attempt refused to make, so the recorded non-goal was honoured. |

## Material Unknowns

One, unchanged by this run and now the only part of the recorded limitation that
still holds: whether this scaffold gates a registered hook behind user approval.
Answering it needs an interactive session with a registration in the user
configuration, which is the arrangement the earlier attempt declined to create
while the owner works inside that same scaffold.
