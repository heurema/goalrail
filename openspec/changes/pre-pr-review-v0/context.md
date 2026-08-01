# Context Pack

- **Context Pack ID:** CTXP-pre-pr-review-v0
- **Version:** 4
- **Previous version:** 3
- **Started at:** 2026-08-01T11:55:00Z
- **Completed at:** 2026-08-01T15:10:00Z
- **Outcome:** sufficient

> **Every claim below carries the command that reproduces it.** Version 3 and
> earlier did not, and one of them — that the installed `claude` accepts
> `--max-turns` — was simply invented, then used to justify calling the review
> bounded when nothing bounded it. A Context Pack is the evidence a whole change
> is compiled from; an unverifiable line in it is worse than a missing one,
> because everything downstream inherits it as fact. Facts with checks, or
> nothing.

## Context Items

| ID | Kind | Claim | Verification (run this) | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The findings journal holds 14 entries from 2026-08-01, every one recorded after its pull request was already open | `wc -l < ~/personal/heurema/docs/portfolio/REVIEW-FINDINGS.jsonl` → 14 at the time of writing; `jq -r 'select(.phase != "post-pr")' <file>` → empty for those 14 | 2026-08-01T11:30:00Z | The problem is timing and independence, not reviewer quality. The same reviewer run earlier would have caught the same defects before they were public |
| CTX-2 | repository | Grouped by cause those findings collapse to six classes, two of which crossed a three-occurrence threshold on the first day of counting | `jq -r .class <file> \| sort \| uniq -c \| sort -rn` → `5 input-space`, `3 governing-docs`, then four classes below three | 2026-08-01T11:40:00Z | Reviewer instructions are not static. Whatever runs the review must accept instructions that grow as classes are promoted |
| CTX-3 | external | `codex review` is a non-interactive subcommand taking custom instructions as an argument or on stdin. Its scope flags `--base`, `--uncommitted` and `--commit` **cannot be combined with those instructions** | `codex review --base main - </dev/null` → `error: the argument '--base <BRANCH>' cannot be used with '[PROMPT]'` | 2026-08-01T14:35:00Z | The reviewed range has to travel in the prose, because the flag that would carry it excludes the instructions. **Corrected in v4:** v1–v3 recorded the flags from `--help` without running them together, and the first live review died on exactly that |
| CTX-4 | external | `claude -p` runs non-interactively with `--output-format`, `--allowed-tools`, `--disallowed-tools`, `--append-system-prompt`, `--effort` and `--model`. It has **no** `--max-turns`; the only cost-shaped bound is `--max-budget-usd` | `claude --help \| grep -c max-turns` → 0; `claude --help \| grep -c -- --max-budget-usd` → 1; `claude --version` → 2.1.215 | 2026-08-01T14:40:00Z | A turn budget is not available, so any ceiling must come from something that is not a vendor flag. **Corrected in v4:** v1–v3 asserted `--max-turns` existed. It was never checked and does not exist. A review built on it would have been unbounded while claiming otherwise |
| CTX-5 | repository | Initialization already detects both scaffolds and the diagnosis already reports each one's state | `gr doctor \| grep -cE '^(codex\|claude-code):'` → 2 | 2026-08-01T14:55:00Z | The registry of available reviewers is not new work. What is missing is what to do with it |
| CTX-6 | repository | Goalrail already spawns external processes | `grep -rl "exec.Command" --include="*.go" internal/ cmd/ \| grep -v _test \| wc -l` → 5 | 2026-08-01T14:55:00Z | Spawning a process is not a new capability. Spawning a non-deterministic, metered one is |
| CTX-7 | repository | The harness is required to work with no Node runtime, which is why Goalrail never executes the pinned OpenSpec CLI | `grep -c "needs no Node runtime" openspec/specs/harness-init/spec.md` → 1 | 2026-08-01T14:55:00Z | The strongest existing precedent points away from execution. Whether review is the exception, and why, is the central design question |
| CTX-8 | external | nitpicker v0.8.2 supports Anthropic by API key only, with no path for a Claude subscription; its OpenAI subscription path reuses the Codex CLI OAuth client and its own README calls that arguably outside OpenAI's terms; its Gemini keyring path is documented as having caused account suspensions in 2026 | Read `https://raw.githubusercontent.com/arsenyinfo/nitpicker/main/README.md` and search for `auth = "codex"` and `Research only`; the quoted lines are recorded in `docs/portfolio/EVIDENCE.jsonl` under `nitpicker-subscription-auth-2026-08-01` | 2026-08-01T12:10:00Z | The obvious third-party option was evaluated and rejected. The installed vendor CLIs use the same subscriptions as their vendors intend |
| CTX-9 | repository | The workspace contract requires a bounded pre-pull-request adversarial review in a fresh context, cross-provider, with every finding dispositioned and a three-occurrence promotion ratchet | `grep -c "Independent review and the findings ratchet" ../AGENTS.md` → 1 | 2026-08-01T14:55:00Z | The policy exists and is prose. This change is what would make it executable, and prose disciplines are what the ratchet says lag |
| CTX-10 | repository | A machine-readable usage guard exists and the global rules require reading it before substantial Codex work | `jq -r '.level, .operating_mode' ~/.local/state/vicc/codex-usage-guard/status.json` → `OK`, `NORMAL` | 2026-08-01T14:20:00Z | A review command that ignores the budget would be the first thing here to spend without a gate. It is also one user's file, so it belongs in configuration rather than in a released binary |
| CTX-11 | repository | The findings journal lives in the Heurema workspace, which is not a Git repository, while Goalrail installs into one repository at a time | `git -C ~/personal/heurema rev-parse --git-dir` → fails | 2026-08-01T14:55:00Z | Where a repository-scoped tool records findings, and how those reach a workspace-level ratchet, is unresolved by anything that exists today |
| CTX-12 | repository | Nothing records which provider produced a given change; the diagnosis reports only which scaffolds are installed and active | `gr doctor \| grep -c "author"` → 0 | 2026-08-01T14:55:00Z | Authorship is not derivable from any stored record — but see CTX-13 |
| CTX-13 | repository | A session's scaffold is visible in its own environment, and a Claude Code session on this machine simultaneously carries a Codex **companion** variable | `env \| grep -cE '^CLAUDECODE=\|^CODEX_COMPANION_SESSION_ID='` → 2, run inside a Claude Code session | 2026-08-01T14:55:00Z | Authorship needs no flag on the ordinary path. But a prefix match on `CODEX_*` would read that companion as authorship and hand the review back to its own author, so only a primary session marker may count |

## Material Unknowns

None blocking. Two questions are design-level rather than context-level and are
recorded so they are not mistaken for settled: whether Goalrail runs the
reviewer or only reports how to (CTX-7), and where findings are recorded so a
repository-scoped tool can feed a workspace-level ratchet (CTX-11).

## Version history

- **v4 (2026-08-01):** Every item gained the command that reproduces it, after
  the owner stopped the work over an invented claim. CTX-4 corrected: the
  installed `claude` has no `--max-turns`. CTX-3 corrected: `codex review`
  refuses its scope flags together with custom instructions, which the first
  live review discovered by failing.
- **v3:** Reproduction commands added to CTX-1 and CTX-2 only.
- **v2:** CTX-13 added after measuring the session environment.
