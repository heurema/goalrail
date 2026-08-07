# Intent Snapshot

- **Intent ID:** bounded-command-failures-v0
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** confirmed
- **Owner:** role:repository-owner
- **Context Pack:** bounded-command-failures-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 — Owner statement:** The owner asked for the raw Git error in `gr init`
  to be corrected, as issue #78.
- **SE-2 — Repository fact:** It is not one command's slip. `init`, `migrate`
  and `update` all relay Git's own message and exit status when run outside
  version control (CTX-1).
- **SE-3 — Repository fact:** `doctor` behaves correctly only because it was
  repaired, and that repair recorded no requirement, so nothing carries the
  behaviour to the other three or to a command written later (CTX-2).
- **SE-4 — Repository fact:** The repository already requires bounded failures
  for two particular surfaces and for no command in general (CTX-3).
- **SE-5 — Repository fact:** A relayed message travels with the wrong exit
  status, and the command surface already distinguishes a check that did not run
  from a state it found (CTX-4).
- **SE-6 — Repository fact:** The underlying classification needed to report the
  condition without losing the failure already exists (CTX-5).
- **SE-7 — Repository fact:** A machine consumer already depends on this: the
  release workflow reads a diagnosis run in a throwaway directory, and a relayed
  error there is an empty result (CTX-6).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | A command that meets a path outside version control says so in Goalrail's own words, naming the condition, and relays no message, exit status or stack trace produced by another program. | Run every command with `--repo` pointing at a directory that is not a repository; read each first line. | SE-2, SE-3 |
| OUT-2 | The rule is recorded as a requirement rather than living in whichever commands happen to have been repaired, so a command written later inherits it. | Read the specification and find the requirement stated for commands generally. | SE-3, SE-4 |
| OUT-3 | Reporting the condition does not hide a genuine failure: a discovery that broke — Git absent, a directory refused — remains distinguishable from a path that is simply not a repository, in wording and in exit status. | Run a command with no Git on the lookup path and again outside a repository; compare both. | SE-5, SE-6 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not suppress the cause. A bounded message says what happened in Goalrail's terms; it does not become silence or a generic failure. | SE-6 |
| NG-2 | Do not change the exit statuses a caller already reads for situations that are not this one. | SE-5 |
| NG-3 | Do not alter commands that already behave correctly, and do not restate the fix already made in the diagnosis as new work. | CTX-1, CTX-2 |
| NG-4 | Do not extend this to every error a command can produce. The scope is a foreign program's message reaching the user, not error wording in general. | SE-4 |
| NG-5 | Do not make the requirement about Git specifically. The subject is any program Goalrail invokes on the user's behalf; Git is the one that appears today. | SE-4 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | No command relays a foreign message outside a repository. | The survey across every command contains no `fatal:` and no `exit status`; today three of eight do. | CTX-1 |
| SIG-2 | The requirement exists in the specification and names commands generally rather than one of them. | One requirement, with scenarios covering a command that reports the condition and one that would relay. | CTX-3 |
| SIG-3 | A broken discovery is still distinguishable from an absent repository. | Two runs, two different messages and two different exit statuses. | CTX-5 |
| SIG-4 | A regression fails against the current code for each command that relays today. | Three failures before, none after. | CTX-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** role:repository-owner
- **Confirmed at:** 2026-08-07
- **Verification action:** The owner was shown a plain-language view of this
  exact version in the conversation language before any implementation existed,
  covering the measured scope across every command, the three outcomes including
  the one that keeps a broken discovery distinguishable, and every boundary. They
  confirmed version 1 by name. No code had been written when they answered.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
