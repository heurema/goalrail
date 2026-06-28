---
name: harness-integration-guide
description: Reference guide for building new Goalrail harness integrations — covers SDK/subprocess harnesses and native harnesses as separate tracks, each with their own feature matrix, implementation patterns, and prioritized checklist.
---

# Harness integration guide

This skill describes the **feature matrix** every Goalrail harness must
consider. Use it when planning, reviewing, or implementing a new harness.

Goalrail has two distinct harness tracks with different architectures and
feature sets:

- **SDK/subprocess harnesses** — run the vendor model directly (in-process SDK,
  CLI subprocess, or ACP subprocess). They own the model lifecycle.
- **Native harnesses** — wrap a vendor's own TUI or server and mirror its
  output into Goalrail. They observe and relay, rather than drive.

---

## Part 1 — SDK / subprocess harnesses

These harnesses run the vendor model directly and bridge Goalrail tools into
the vendor's tool-calling interface.

### Capability matrix

| Capability | What it means |
|---|---|
| **Connects to Goalrail MCP** | Harness exposes/consumes tools via the MCP protocol (in-proc SDK MCP server) |
| **Model override** | User can select a model via `--model` / config; some harnesses are vendor-locked (e.g. Claude-only, GPT-only, Gemini-only) |
| **Auth** | How credentials are obtained — API key, gateway token, vendor CLI login, OAuth, etc. |
| **Streaming** | Harness forwards token-level or delta-level streaming to the Goalrail forwarder |
| **Goalrail policies** | Harness enforces Goalrail-side tool policies — must support ALLOW, ASK, and DENY verdicts for both tool calls and tool results |
| **Native elicitation** | When a policy verdict is ASK, the harness surfaces the approval request in the Goalrail web UI so the user can approve or deny |
| **Interrupt** | User can cancel a running turn mid-stream |
| **Live queue (concurrent)** | Multiple turns can be queued and processed concurrently |
| **Tool-boundary steer** | Goalrail can inject steering text at tool-call boundaries |
| **Resume/fork from Goalrail transcript** | Rebuild a conversation from a stored Goalrail transcript (replay history, seed prompt, or vendor session ID) |
| **Compaction** | Long conversations are compacted; harness surfaces `CompactionComplete` events |
| **Reasoning** | Model reasoning/thinking tokens are forwarded |
| **Images** | Image content (screenshots, diagrams) is forwarded — full binary, path reference, or text-flattened |
| **Cost tracking** | Harness reports token usage and cost data back to Goalrail for each turn |

### MCP connectivity

The harness must bridge Goalrail's builtin MCP tools so the model can call
them. These tools provide session management, agent orchestration, policy
control, and web access:

- `sys_session_get_info`, `sys_session_list`, `sys_session_get_history`
- `sys_agent_get`, `sys_agent_list`, `sys_agent_download`
- `sys_call_async`, `sys_cancel_async`, `sys_cancel_task`
- `sys_read_inbox`
- `sys_add_policy`, `sys_policy_registry`
- `load_skill`
- `list_comments`, `update_comment`
- `web_fetch`, `web_search`

### Goalrail policies

The harness must support the Goalrail policy engine's three verdicts at two
checkpoints:

| Checkpoint | ALLOW | ASK | DENY |
|---|---|---|---|
| **Tool call** (before execution) | Proceed silently | Surface approval request to user (via elicitation) | Block the call and return a policy-denied error to the model |
| **Tool result** (after execution) | Return result to model | Surface result for user review before returning | Suppress the result and return a policy-denied error to the model |

### Native elicitation

When a policy verdict is ASK, the harness must surface the pending tool call
or tool result in the Goalrail web UI as an approval card, then relay the
user's approve/deny decision back to the harness to continue or block
execution.

### Resume / fork strategies

| Strategy | How it works |
|---|---|
| Full history replay | Replays the entire message history into a fresh thread/session |
| History prefix replay | Replays a prefix of the history into a fresh session |
| Text-prefix replay | Injects a text summary/prefix of prior history |
| Prompt seeding | Seeds prior history into the system prompt on rebuild |
| Vendor session ID | Relies on the vendor's own session persistence (no Goalrail-side rebuild) |

### Auth patterns

| Pattern | Description |
|---|---|
| API key / OpenAI-compatible gateway | Direct API key or routed through a OpenAI-compatible gateway |
| Vendor API key (direct) | Vendor-specific API key (e.g. Cursor, Gemini) |
| Vendor CLI login / config file | Credentials stored in a vendor config file or managed via vendor CLI login |
| OAuth / GitHub token | OAuth flow or platform token (e.g. GitHub PAT) |
| Gateway + fallback | Primary gateway with fallback to vendor-native auth |

### Checklist for a new SDK/subprocess harness

All capabilities are **required** for a complete harness integration:

- [ ] Connects to Goalrail MCP (in-proc SDK MCP server or vendor-specific bridge)
- [ ] Model override works (or document vendor lock-in)
- [ ] Auth is configured and documented (setup flow in `goalrail setup`)
- [ ] Streaming forwards to the Goalrail forwarder
- [ ] Goalrail policies enforce tool-use rules
- [ ] Native elicitation surfaces tool-approval requests to web UI
- [ ] Interrupt cancels the running turn
- [ ] Live queue supports concurrent turns
- [ ] Tool-boundary steering injects correctly
- [ ] Resume/fork rebuilds conversation from Goalrail transcript
- [ ] Compaction is surfaced (`CompactionComplete` events)
- [ ] Reasoning tokens are forwarded
- [ ] Images are forwarded (full binary preferred; path or text-flattened acceptable)
- [ ] Cost tracking reports token usage and cost per turn
- [ ] Unit tests cover tool bridging, auth, model routing
- [ ] Mock LLM tests cover the happy path without real API calls

---

## Part 2 — Native harnesses

Native harnesses wrap a vendor's own TUI or server and mirror output into
Goalrail. They relay the vendor's conversation into the Goalrail session.

### Capability matrix

| Capability | What it means |
|---|---|
| **Transport** | How the native harness communicates — tmux TUI, app server, HTTP/SSE, file-inject TUI |
| **Connects to Goalrail MCP** | Whether the native harness connects to the Goalrail MCP server |
| **Model override** | User can select a model at launch or per-prompt |
| **Auth** | Vendor login / config / token |
| **Streaming (forwarder)** | `deltas` (token-level) vs `complete-only` (full response after completion) |
| **Goalrail policies** | Whether the native harness enforces Goalrail-side tool policies — must support ALLOW, ASK, and DENY verdicts for both tool calls and tool results |
| **Native elicitation** | When a policy verdict is ASK, the native harness surfaces the approval request in the Goalrail web UI so the user can approve or deny |
| **Interrupt** | User can abort a running turn |
| **Bidirectional sync (TUI->Goalrail)** | TUI output mirrors into the Goalrail conversation |
| **In-harness session-cmd sync** | Supports `clear`, `fork`, `resume`, `switch` commands from Goalrail |
| **Resume/fork from Goalrail transcript** | Can rebuild conversation from Goalrail transcript (native rebuild, or fresh launch) |
| **Compaction** | Vendor-internal compaction status |
| **Reasoning** | Model reasoning/thinking tokens are forwarded |
| **Images** | Image content is forwarded — path reference, full binary, or text-flattened |
| **Cost tracking** | Native harness reports token usage and cost data back to Goalrail for each turn |

### Checklist for a new native harness

All capabilities are **required** for a complete native harness integration:

- [ ] Transport chosen and implemented (tmux TUI, app server, HTTP/SSE)
- [ ] Connects to Goalrail MCP
- [ ] Model override works (or document vendor lock-in)
- [ ] Auth configured (vendor login / config)
- [ ] Streaming forwarder works (deltas preferred; complete-only acceptable)
- [ ] Goalrail policies enforce tool-use rules
- [ ] Native elicitation surfaces tool-approval requests to web UI
- [ ] Interrupt aborts the running turn
- [ ] Bidirectional sync mirrors TUI output into Goalrail conversation
- [ ] Session commands (clear, fork, resume) work from Goalrail
- [ ] Resume/fork rebuilds from Goalrail transcript
- [ ] Compaction status is surfaced
- [ ] Reasoning tokens are forwarded
- [ ] Images are forwarded (path preferred; binary or text-flattened acceptable)
- [ ] Cost tracking reports token usage and cost per turn
- [ ] Unit tests cover forwarder, auth, transport
- [ ] Mock LLM tests cover the happy path without real API calls
