# Context Pack

- **Context Pack ID:** CTXP-pre-pr-review-v0
- **Version:** 2
- **Previous version:** 1
- **Started at:** 2026-08-01T11:55:00Z
- **Completed at:** 2026-08-01T12:50:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | Automated review found 14 defects across two pull requests on one day, all after the pull request was opened; three were P1 and two would have shipped wrong behaviour | `docs/portfolio/REVIEW-FINDINGS.jsonl` in the Heurema workspace | 2026-08-01T11:30:00Z | The problem is timing and independence, not reviewer quality. The same reviewer run earlier would have caught the same defects before they were public |
| CTX-2 | repository | Grouped by cause, those 14 findings collapse to six classes; two already crossed a three-occurrence threshold on the first day of counting | same journal | 2026-08-01T11:40:00Z | Reviewer instructions are not static. Whatever runs the review must accept instructions that grow as classes are promoted |
| CTX-3 | external | `codex review` is a first-class non-interactive subcommand. It accepts `--base <BRANCH>`, `--uncommitted` and `--commit <SHA>`, takes custom review instructions as an argument or on stdin, and overrides configuration including the model with `-c` | `codex review --help`, installed at `/opt/homebrew/bin/codex` | 2026-08-01T12:10:00Z | The pre-pull-request review primitive already exists and is designed for exactly this. `--base` reviews a branch before a pull request exists |
| CTX-4 | external | `claude -p/--print` runs non-interactively with `--output-format json`, `--allowed-tools` and `--disallowed-tools`, `--append-system-prompt`, `--max-turns` and `--model` | `claude --help`, installed at `/Users/vi/.local/bin/claude` | 2026-08-01T12:15:00Z | The second provider is drivable headlessly with a bounded tool surface and a bounded turn budget, which is what keeps a review from becoming an open-ended agent run |
| CTX-5 | repository | Initialization already detects both scaffolds and the diagnosis already reports their state — `claude-code: active (repository scope)`, `codex: not active (user scope)` | `gr doctor` on Baseline and on a scratch repository | 2026-08-01T08:35:00Z | The registry of available reviewers is not new work. What is missing is what to do with it |
| CTX-6 | repository | Goalrail already spawns external processes: `exec.Command("git", ...)` in `internal/ambient/scope.go` and `internal/localrun/git_observer.go`, and `cmd/goalrail-canary` launches a configured command | source | 2026-08-01T12:20:00Z | Spawning a process is not a new capability. Spawning a non-deterministic, metered one is |
| CTX-7 | repository | Goalrail deliberately never executes the pinned OpenSpec CLI. It owns the version string, materializes the overlay, and reports availability; the user or the agent runs it | `harness-init` spec, "Installing and maintaining the harness needs no Node runtime"; `gr doctor` output line `openspec cli: available (needed for ...)` | 2026-08-01T12:20:00Z | This is the strongest existing precedent and it points away from execution. Whether review is the exception, and why, is the central design question |
| CTX-8 | external | nitpicker v0.8.2 supports Anthropic by API key only, with no path for a Claude subscription; its OpenAI subscription path reuses the Codex CLI OAuth client and its own documentation calls that arguably outside OpenAI's terms; its Gemini keyring path is documented as having caused account suspensions in 2026 | `https://github.com/arsenyinfo/nitpicker` README; recorded in `docs/portfolio/EVIDENCE.jsonl` | 2026-08-01T12:10:00Z | The obvious third-party option was evaluated and rejected. The installed vendor CLIs use the same subscriptions as their vendors intend, which is both cheaper and cleaner |
| CTX-9 | repository | The workspace contract now requires a bounded pre-pull-request adversarial review in a fresh context, cross-provider by default, with every finding dispositioned before merge and a three-occurrence promotion ratchet | Heurema `AGENTS.md`, "Independent review and the findings ratchet" | 2026-08-01T11:45:00Z | The policy exists and is currently prose. This change is what would make it executable, and prose disciplines are exactly what the ratchet says lag |
| CTX-10 | repository | Every review consumes a metered subscription, and the global rules require reading a usage guard before substantial Codex work and permit reduced modes when it reports `DRAIN` or `STOP` | `~/.local/state/vicc/codex-usage-guard/status.json` per the global operating rules | 2026-08-01T12:20:00Z | A review command that ignores the budget would be the first thing in this workspace to spend without a gate |
| CTX-11 | repository | The findings journal lives in the Heurema workspace, which is not a Git repository, while Goalrail installs into one repository at a time | `docs/portfolio/REVIEW-FINDINGS.jsonl`; workspace `AGENTS.md`, "Workspace shape" | 2026-08-01T11:50:00Z | Where a repository-scoped tool records findings, and how those reach a workspace-level ratchet, is unresolved by anything that exists today |
| CTX-12 | repository | The workspace contract assigns the reviewer a provider different from the author's, but the diagnosis reports only which scaffolds are installed and active — nothing records which one produced a given change | `gr doctor` output; workspace `AGENTS.md` | 2026-08-01T12:20:00Z | "Cross-provider" is not derivable from any record — but see CTX-13: it is derivable from the invoking environment |
| CTX-13 | repository | A session's own scaffold is visible in its environment: a Claude Code session carries `CLAUDECODE` and a `CLAUDE_CODE_*` family, and a Codex session carries its own `CODEX_*` variables — measured live from inside a Claude Code session on this machine | `env` probe, 2026-08-01 | 2026-08-01T12:45:00Z | Authorship needs no flag on the ordinary path: the process that invokes the review knows what it is. An explicit override remains for the odd path; refusal is only for the case where nothing is detectable and nothing is stated |

## Material Unknowns

None blocking. Two questions are design-level rather than context-level and are
recorded so they are not mistaken for settled: whether Goalrail runs the
reviewer or only reports how to (CTX-7), and where findings are recorded so a
repository-scoped tool can feed a workspace-level ratchet (CTX-11).
