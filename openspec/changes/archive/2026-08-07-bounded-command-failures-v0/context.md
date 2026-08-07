# Context Pack

- **Context Pack ID:** bounded-command-failures-v0
- **Version:** 1
- **Previous version:** pending
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Started at:** 2026-08-07T16:00:00Z
- **Completed at:** 2026-08-07T16:35:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
|---|---|---|---|---|---|---|
| CTX-1 | repository | Three commands relay Git's own message when run outside version control: `init`, `migrate` and `update` each print `not a Git repository: fatal: not a git repository (or any of the parent directories): .git`. | `gr` v0.2.1 | In a directory that is not a Git repository, run each of `init`, `migrate`, `update`, `doctor`, `health`, `connect`, `verify-lineage`, `review` with `--repo .`; expect the first three to contain `fatal:` and the rest not to. | 2026-08-07T16:20:00Z | Establishes the scope as a class across commands rather than one command's slip. |
| CTX-2 | repository | `doctor` alone reports the same situation in its own words, and it does so because of a fix rather than a recorded rule: the change that corrected it altered no specification. | `c9cb73f` | Read the commit's file list; expect no path under `openspec/`. | 2026-08-07T16:25:00Z | The behaviour exists in one command by accident of a repair, so nothing carries it to the others or to a command not yet written. |
| CTX-3 | repository | The specifications require bounded failures for particular surfaces but for no command in general: the release check must surface no error, stack trace or text from its source, and run preparation must fail with a bounded reason. | `openspec/specs/harness-doctor/spec.md:175`; `openspec/specs/local-run/spec.md:31` | Read both; then grep `openspec/specs` for a requirement covering command failure generally and expect none. | 2026-08-07T16:28:00Z | The principle is already the repository's, applied twice; what is missing is the general statement, which is why this is a new requirement rather than a restoration. |
| CTX-4 | repository | The command surface distinguishes, by exit status, a diagnosis that found problems from a check that did not run, so that an unattended caller can tell them apart. | `cmd/gr/harness.go:52-60` | Read `doctorFailure`; expect exit 2 documented as "the check itself broke". | 2026-08-07T16:30:00Z | A relayed foreign message usually accompanies the wrong status too, so the two questions are one. |
| CTX-5 | repository | Only Git's own verdict now means "not a repository": every other failure of the discovery command surfaces as the discovery failing. | `internal/project/root.go` | Read `ResolveWorktreeRoot`; expect the classification to require a command that ran and printed Git's verdict. | 2026-08-07T16:31:00Z | The distinction a bounded message must preserve already exists underneath, so a command can report the condition without losing the failure. |
| CTX-6 | repository | A machine consumer already depends on a command answering with a report rather than an error in this exact situation: the release workflow reads the pinned planning invocation from a diagnosis run in a throwaway directory, and every tagged release since the diagnosis was rewritten would have stopped there. | `.github/workflows/release.yml`; `c9cb73f` | Read the step that derives the pinned package, and the commit that made the diagnosis answer. | 2026-08-07T16:33:00Z | Shows the cost is not cosmetic: a relayed error is an empty result for whoever parses it. |

## Material Unknowns

None.
