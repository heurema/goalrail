# Context Pack

- **Context Pack ID:** review-stall-isolation-v0
- **Version:** 1
- **Previous version:** pending
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Started at:** 2026-08-05T19:05:00Z
- **Completed at:** 2026-08-06T07:30:45Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
|---|---|---|---|---|---|---|
| CTX-1 | repository | A full pre-PR review with the Codex reviewer consumed its entire deadline twice, at 25 and 50 minutes, and produced no finding and no receipt | `gr review --full --author claude-code --base origin/main` on `codex/admission-hardening-after-pr62` | Not independently reproducible — wall-clock runs against a live provider; retained evidence: the two captured logs, each beginning `the codex reviewer did not finish within the review deadline` | 2026-08-05T19:30:19Z, 2026-08-05T20:20:54Z | Establishes that the reviewer failure is total rather than partial: nothing survives to disposition |
| CTX-2 | repository | In both runs the reviewer recorded zero events for the remainder of the deadline after issuing one codebase-memory MCP call | Codex rollout files under `$CODEX_HOME/sessions/2026/08/05/` | Not independently reproducible — session rollouts of past runs; retained evidence: rollout `019fd350-e059-…` last event `19:07:33Z` against a `19:30Z` kill, and rollout `019fd368-4d9c-…` last event `19:31:35Z` against a `20:20Z` kill | 2026-08-06T06:50:00Z | Distinguishes a blocked reviewer from a slow one; a slow reviewer keeps emitting events |
| CTX-3 | repository | One `trace_path` call issued by Codex never returns | `printf '<prompt naming trace_path>' \| codex -c model_reasoning_effort=low -c sandbox_mode=read-only review -` | Run that command with an external watchdog; it emits `mcp: codebase-memory-mcp/trace_path started` and never completes, so the watchdog terminates it | 2026-08-06T07:05:00Z | Reduces the failure to one reproducible call, independent of diff size or review scope |
| CTX-4 | repository | The same tool, project, and arguments answer immediately for a different MCP client | `trace_path(project="goalrail", function_name="goalrail.internal.admission.Verify", direction="inbound", mode="calls")` issued by this session's MCP client | Issue that call from any non-Codex MCP client against the same server binary; it returns three callers | 2026-08-06T07:02:00Z | Excludes the server, the index, and the arguments as the cause; the fault is in the Codex pairing |
| CTX-5 | repository | The block is unaffected by approval policy | Same probe with `-c approval_policy=never` | Run both probes with a watchdog; both stop at `trace_path started` | 2026-08-06T07:05:00Z | Excludes an unanswerable interactive approval prompt, which a non-interactive `review -` invocation would otherwise invite |
| CTX-6 | repository | The block is unaffected by the reviewer's read-only sandbox | Same probe without `-c sandbox_mode=read-only` | Run the probe with the flag removed and a watchdog; it stops at `trace_path started` | 2026-08-06T07:12:00Z | Excludes the read-only boundary added by commit `a8634a4`, which predates the last successful Codex review |
| CTX-7 | repository | Disabling the named MCP server by configuration override lets the same Codex invocation finish | `codex -c mcp_servers.codebase-memory-mcp.enabled=false review -` over a bounded git question | Run that invocation; it exits 0 and answers from git within the watchdog window | 2026-08-06T07:20:00Z | Establishes that removing the reviewer's inherited MCP surface is sufficient to make the reviewer usable |
| CTX-8 | repository | An empty-table override does not clear the configured MCP servers | `codex -c 'mcp_servers={}' review -` | Run that invocation; the tool remains callable and the run blocks at `trace_path started` | 2026-08-06T07:28:00Z | Rules out a generic one-key removal, so isolation cannot be expressed by clearing the table |
| CTX-9 | external | The Codex CLI exposes no global switch that disables all MCP servers | `codex --help`, `codex review --help` | Run both; the only configuration surfaces are `-c/--config`, `--enable`, and `--disable <FEATURE>` | 2026-08-06T07:29:00Z | Forces reviewer isolation to be expressed as caller-supplied arguments rather than a vendor flag |
| CTX-10 | repository | Goalrail builds the reviewer invocation itself and offers no way to add provider arguments | `internal/review/run.go` `reviewCommand`, and `command.Env = strippedEnvironment(...)` | Read both; the argument list is fixed in code and the environment passes through minus author markers | 2026-08-06T07:26:00Z | Explains why the verified workaround cannot be applied today without editing the owner's machine configuration |
| CTX-11 | repository | A blocked reviewer is reported only as a deadline overrun, with no distinct outcome and no receipt | `internal/review/run.go` deadline handling; `$XDG_STATE_HOME/goalrail/reviews` | Read the deadline branch, then list the receipt directory after a blocked run; no receipt is written for it | 2026-08-06T06:45:00Z | Names the second defect: the caller cannot tell a slow reviewer from a stopped one, and pays the full deadline either way |
| CTX-12 | repository | The Codex reviewer path last succeeded on 2026-08-02 and has 24 recorded receipts | `$XDG_STATE_HOME/goalrail/reviews/*.json` | List the receipts and group by `reviewer`; 24 name `codex`, the newest dated 2026-08-02 | 2026-08-06T06:40:00Z | Shows the path is a working one that regressed rather than an unused one |
| CTX-13 | repository | Reviewer output is captured incrementally, so liveness is observable while a run is in flight | The retained logs of both blocked runs contain streamed reviewer output up to the moment of the block | Inspect either captured log; it holds partial reviewer output followed by the final MCP line | 2026-08-06T07:00:00Z | Establishes that a progress signal already exists to watch, without adding a new channel |

## Material Unknowns

None.

<!-- Which side of the Codex/codebase-memory-mcp pairing owns the deadlock is
unresolved, but it is not material here: both defects in scope are Goalrail's
own, and neither remedy depends on that answer. -->
