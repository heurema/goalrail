<div align="center">

# Goalrail

### The open-source AI agent framework and meta-harness for all your AI agents.

Goalrail is an open-source **AI agent framework** and meta-harness that gives you a common orchestration layer over Claude Code, Codex, Cursor, Kimi Code, Pi, and the agents you write yourself: swap or combine harnesses without rewriting, enforce policies and sandboxing, and collaborate in real time from any device.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
![Status: alpha](https://img.shields.io/badge/status-alpha-orange.svg)
[![Python 3.12+](https://img.shields.io/badge/python-3.12%2B-blue.svg)](#1-install)

[goalrail.dev](https://goalrail.dev) · [Download desktop for macOS Apple Silicon](https://github.com/heurema/goalrail/releases/download/desktop-v0.1.1/Goalrail-0.1.1-arm64.dmg)

</div>

---

## Why Goalrail?

Goalrail lets you:

- **📱 Work with agents from any device, including your phone.** Sessions
  follow you: start in your terminal, continue in the browser, pick it up on
  your phone. Messages, sub-agents, terminals, and files stay in sync.

- **🤖 Supervise multiple agents.** Use Claude Code, Codex, Pi, and custom
  agents (defined in YAML) together in the same session. Ask one agent to
  review another's work, or split a task across agents that are each good at
  different things.

- **🔌 Use any model.** A first-party API key, a Claude/ChatGPT subscription,
  or any compatible gateway. All first-class.

- **🤝 Collaborate.** Share a session so teammates can chat with your agent
  and watch it work live, co-drive it on your machine, or fork the
  conversation to continue on their own.

- **☁️ Run agents in cloud sandboxes.** No laptop required: run sessions in
  disposable [Modal](https://modal.com), [Daytona](https://www.daytona.io),
  [Islo](https://islo.dev), [E2B](https://e2b.dev),
  [CoreWeave](https://docs.coreweave.com/products/sandboxes),
  [Kubernetes](https://kubernetes.io), [OpenShell](https://github.com/NVIDIA/OpenShell),
  or [Boxlite](https://github.com/boxlite-ai/boxlite) sandboxes, launched from the
  CLI or provisioned by the server per session (*managed hosts*).

- **🛡️ Govern your agents.** Create
  [policies](#6-govern-your-agents-with-policies) to pause for your approval
  before risky actions, cap spend, or limit which tools an agent reaches.
  They apply to the whole server, one agent, or a single chat.

---

## Quick start

### 1. Install

One command installs the current Goalrail build and everything it needs:

```bash
curl -fsSL https://raw.githubusercontent.com/heurema/goalrail/main/scripts/install_oss.sh | sh
```

<details>
<summary>Prefer to install manually?</summary>

Goalrail currently uses the `goalrail` Python package and needs **Python 3.12+**:

```bash
uv tool install goalrail        # or: pip install "goalrail"
```

Or with the current legacy [Homebrew tap](https://github.com/heurema/homebrew-tap):

```bash
brew install heurema/tap/goalrail
```

Or install straight from the repo:

```bash
uv tool install -q --python 3.12 git+https://github.com/heurema/goalrail.git
```

</details>

<details>
<summary>Toolchain and prerequisites (if the installer reports a missing tool)</summary>

- **`uv`** (required). https://docs.astral.sh/uv/getting-started/installation/
  The installer offers to set this up for you.
- **`git`** (required).
- **Node.js 22 LTS or newer** with **`npm`**, for the Claude, Codex, and Pi
  coding harnesses. `goalrail run` installs the harness CLI you pick.
  https://docs.npmjs.com/downloading-and-installing-node-js-and-npm
- **Kiro CLI** (optional), for `goalrail kiro`: install with
  `curl -fsSL https://cli.kiro.dev/install | bash`, then sign in with Kiro.
- **`tmux`**, required by the native `goalrail claude` / `goalrail codex` /
  `goalrail kiro`
  wrappers (`brew install tmux` / `apt install tmux`; the installer offers
  to install it for you).
- **`bubblewrap`** (`bwrap`), **Linux only**. The native `goalrail claude` /
  `goalrail codex` / `goalrail kiro` and `goalrail pi` harnesses wrap each agent
  terminal in a `bwrap` OS-sandbox; on Linux that isolation is mandatory, so a
  missing `bwrap` binary makes those terminals fail to start
  (`apt install bubblewrap`; the installer offers to install it for you). macOS
  uses the built-in `seatbelt` sandbox and needs nothing extra.

</details>

<details>
<summary>Windows (native)</summary>

The current Goalrail build runs natively on Windows in a degraded mode. The `install_oss.sh`
bootstrap is POSIX-only, so install with `uv` directly:

```powershell
uv tool install --python 3.12 goalrail
# or from the repo:
uv tool install --python 3.12 git+https://github.com/heurema/goalrail.git
```

What works on Windows: `goalrail server`, the web UI, and the SDK-based
harnesses (`goalrail run <agent.yaml>` with the claude-sdk / cursor / copilot
/ codex harnesses). Agents run under a Windows **Job Object** for process-tree
containment.

What is **not** available on Windows (use Linux/macOS, or WSL, for these):

- the native `goalrail claude` / `goalrail codex` / `goalrail cursor`
  tmux/PTY terminal wrappers (run an SDK harness or the web UI instead);
- `bwrap`/`seatbelt` filesystem & network sandboxing and the L7 egress proxy
  — the Job Object backend contains the process tree and enforces resource
  limits but does **not** isolate the filesystem or network.

</details>

<details>
<summary>Updating to a new release</summary>

When a newer release is on PyPI, the CLI shows a one-line notice (once per
release) pointing here. To update:

```bash
goalrail upgrade            # detects how you installed, drains & stops the local
                            # server, then runs the matching upgrade command
goalrail upgrade --check    # just report whether a newer release is available
```

`goalrail upgrade` waits for in-flight agent sessions to finish before stopping
the local server (pass `--force` to stop them immediately); the next `goalrail`
command brings the server back up on the new version. Source checkouts update with
`git pull` instead. Silence the notice with `GOALRAIL_NO_UPDATE_CHECK=1`.

The check queries your configured package index — honoring `UV_INDEX_URL` /
`PIP_INDEX_URL` and your `uv.toml` / `pip.conf` (default PyPI), so private
mirrors work out of the box; override with `GOALRAIL_INDEX_URL` if needed.

</details>

### 2. Start your first agent

`goalrail` picks a model with you and starts a session in your terminal. It
also launches a local web UI at `http://localhost:6767` that shows the same
session in the browser, or on a phone on your network (step 4). The macOS
Apple Silicon desktop app is available from the
[Goalrail desktop release](https://github.com/heurema/goalrail/releases/download/desktop-v0.1.1/Goalrail-0.1.1-arm64.dmg);
mobile downloads are not published yet.

> [!NOTE]
> The install puts `goalrail` on your PATH.

> [!TIP]
> On first run, Goalrail picks up model credentials already in your
> environment (an `ANTHROPIC_API_KEY` / `OPENAI_API_KEY`, or a `claude` /
> `codex` CLI you're logged into) and offers one as the default.

```bash
goalrail
```

Or launch a specific agent runtime, or your own agent:

```bash
goalrail claude                      # Claude Code, in a session your team can join
goalrail codex                       # Codex
goalrail kiro                        # Kiro CLI
goalrail kimi                        # Kimi Code (https://kimi.com), headless
goalrail run path/to/agent.yaml      # your own agent (see "Write your own agent")
```

#### 🐙 Polly, 🟠🔵 Debby, and ✍️ Scribe

Three example agents ship with the repo, and they make good first sessions:

```bash
goalrail run examples/polly/
goalrail run examples/debby/
goalrail run examples/scribe/

# Run an orchestrator on a different harness (sub-agents keep their own):
goalrail run examples/polly/ --harness pi
goalrail run examples/debby/ --harness openai-agents
goalrail run examples/polly/ --harness cursor  # Cursor CLI (needs cursor-agent + CURSOR_API_KEY)
goalrail run examples/polly/ --harness copilot # GitHub Copilot SDK (needs a GitHub token w/ Copilot, e.g. GH_TOKEN)
```

**🐙 Polly** is a multi-agent coding orchestrator who writes no code herself.
She's the tech lead: she plans, delegates the work to coding sub-agents
(Claude Code, Codex, or Pi) in parallel git worktrees, then routes each diff
to a reviewer from a different vendor than the one that wrote it. You merge.

**🟠🔵 Debby** is a brainstorming partner with two heads, one Claude and one GPT.
Every question you ask goes to both heads, and she lays the two answers out
side by side. Type `/debate` and the heads critique each other for a few
rounds before converging. (She needs both a Claude and an OpenAI credential;
see step 3.)

**✍️ Scribe** is a documentation orchestrator, the docs counterpart to Polly.
She turns git diffs, commit history, and PRs into release notes, changelogs, and
migration guides. She authors the prose herself and delegates only read-only
code investigation to a researcher sub-agent, then can route a draft through an
independent different-vendor reviewer to fact-check its claims before it ships.
(The cross-model fact-check needs an OpenAI credential; the rest runs on one.)

**Prefer the browser?** Start a server and register your machine as a host:

```bash
goalrail server start   # start the local server and web UI in the background
goalrail host           # (separate terminal) register this machine as a host
```

In the web UI, hit **New Chat**, pick your machine, and go. Check status with
`goalrail server status`; stop everything with `goalrail stop`.

### 3. Choose & switch models

```bash
goalrail setup
```

Add a credential, set a default, or remove one, grouped by agent. Goalrail
works with four kinds of credentials:

| | Kind | What it is |
|---|---|---|
| 🔑 | **API key** | A first-party vendor key for Anthropic, OpenAI, and similar providers |
| 🎟️ | **Subscription** | A Claude Pro/Max or ChatGPT plan, via the official `claude` / `codex` CLIs |
| 🌐 | **Gateway** | Any OpenAI- or Anthropic-compatible `base_url` and key (OpenRouter, LiteLLM, Ollama, vLLM, Azure) |

Defaults are per agent, so a Claude default and a Codex default coexist. You
can also switch models in the middle of a session with the `/model` command.

<details>
<summary>Gateway base URLs (OpenRouter, Ollama)</summary>

When you add a **Gateway** credential, `goalrail setup` asks for a base URL
and a key. The base URL depends on which agent you point it at:

| Provider | For | Base URL | Key |
|---|---|---|---|
| **OpenRouter** | Claude Code | `https://openrouter.ai/api` | your OpenRouter key (`sk-or-…`) |
| **OpenRouter** | Codex / OpenAI agents | `https://openrouter.ai/api/v1` | your OpenRouter key (`sk-or-…`) |
| **Ollama** (local) | Codex / OpenAI agents | `http://localhost:11434/v1` | any value (Ollama ignores it) |

For Claude Code, point at OpenRouter's Anthropic-compatible endpoint
(`…/api`, **not** `…/api/v1`). For Codex and the OpenAI-agents harness, use
the OpenAI-compatible `…/api/v1`.

</details>

### 4. Deploy a server (and use it from your phone📱)

Run Goalrail on a server with a stable URL
([`deploy/README.md`](https://github.com/heurema/goalrail/blob/main/deploy/README.md) is the full guide) and your sessions
become reachable from anywhere, including your phone. The web UI is built for
mobile, so you get the same chat, sub-agents, terminals, and files, in sync
with your laptop.

One `docker compose up` runs the server on any host you have (a VPS, a home
server); Render deploys with one click; Fly.io, Railway, Hugging Face Spaces,
and Modal are covered too. The server can also provision a cloud sandbox per
session (*managed hosts*), so no laptop has to stay online. The full menu of
targets, the database options, and the sandbox setup live in
[`deploy/README.md`](https://github.com/heurema/goalrail/blob/main/deploy/README.md).

Once the server is up, sign in and register your laptop as a host:

```bash
goalrail login https://your-host    # sign in once; run / attach / host reuse the token
goalrail host  https://your-host    # new sessions can now run on this machine
```

> [!TIP]
> On your own network you don't need a deploy. Open your machine's LAN
> address on your phone (e.g. `http://192.168.x.x:6767`).

### 5. Collaborate with your team

Goalrail supports **multi-user accounts**, controlled by one environment
variable:

```bash
GOALRAIL_AUTH_ENABLED=1 goalrail server start
```

The **Docker deploy in [step 4](#4-deploy-a-server-and-use-it-from-your-phone)
turns it on for you** (`GOALRAIL_AUTH_ENABLED` defaults to `1` there).

#### Invite your teammates

Open the web UI (`http://localhost:6767` locally, or your host's URL) and
sign in as `admin`; first run prints the password and saves it locally. Then
open **Admin → Members → Invite** to create a single-use invite link, no
email server needed. Send it over; your teammate opens it, sets a password,
and they're in. Signup is invite-only.

<!-- TODO: screenshot of Admin → Members → Invite. -->

> [!NOTE]
> Teammates need to be able to reach the server. A local server is only
> reachable on your network; for anyone off it, deploy an always-on host
> (see [step 4](#4-deploy-a-server-and-use-it-from-your-phone)).

#### Code together

- **Share a live session.** Hit **Share** in the web UI and send the link;
  teammates watch your agent work and chat with it in real time.
- **Co-drive.** A teammate co-attaches to your running session; their
  messages execute on **your** machine. Great for pairing or handing the
  keyboard to a domain expert mid-investigation.

  ```bash
  goalrail attach <session_id>
  ```

- **Fork.** Clone a conversation onto your own machine and continue
  independently from the fork point.

  ```bash
  goalrail run --fork <session_id>
  ```

> [!TIP]
> Want your team to sign in with the logins they already have (**Google,
> GitHub, Okta, Microsoft**)? Set `GOALRAIL_OIDC_ISSUER` plus a client ID
> and secret on your deployed server and restart. The full walkthrough,
> domain allowlists, and the proxy-only `header` auth mode are covered in
> [`deploy/README.md#auth`](https://github.com/heurema/goalrail/blob/main/deploy/README.md#auth).

### 6. Govern your agents with policies

**Policies** decide what an agent may do: run shell commands, edit files,
spend tokens. They check every action and either allow it, block it, or pause
to ask you first.

- **In the web UI**: open a session's info panel to browse the available
  policies and toggle them on or off.
- **In chat**: ask. *"Add a policy that asks me before running shell
  commands."* The agent sets it up for you.

Want defaults that apply to everyone, or to a specific agent? Define them in
your server config or an agent's YAML:

```yaml
policies:
  approve_shell:
    type: function
    handler: goalrail.policies.builtins.safety.ask_on_os_tools   # ask before shell / file writes
  cap_calls:
    type: function
    handler: goalrail.policies.builtins.safety.max_tool_calls_per_session
    factory_params:
      limit: 50                    # cap how many tools one session can call
  budget:
    type: function
    handler: goalrail.policies.builtins.cost.cost_budget
    factory_params:
      max_cost_usd: 5.00           # hard spend cap...
      ask_thresholds_usd: [3.00]   # ...with a soft warning on the way
```

Policies stack across three levels, **server-wide** (admin), **per-agent**
(developer), and **per-session** (you), with the stricter session rules
checked first. Spend caps and access limits ship as builtins.

See the [policy guide](https://github.com/heurema/goalrail/blob/main/docs/POLICIES.md) for the full catalog and trust model.

---

## Write your own agent

An agent is a short YAML file: your prompt, your tools, and optional helper
sub-agents a supervisor can delegate to. You don't have to write it by hand:
agents can build agents, so describe the agent you want in any Goalrail chat
and it authors the file for you.

```yaml
name: my_agent
prompt: You are a helpful data analyst.

executor:
  harness: claude-sdk          # or: claude-native, codex, codex-native, cursor, cursor-native, kiro-native, openai-agents, pi, pi-native, antigravity, qwen, kimi, copilot

tools:
  # A local Python function (schema auto-generated from the signature)
  word_count:
    type: function
    callable: mypackage.mymodule.word_count

  # A sub-agent the supervisor can delegate to
  researcher:
    type: agent
    prompt: Search for relevant information and summarize it.
    tools:
      word_count: inherit
```

Run it with:

```bash
goalrail run path/to/my_agent.yaml
```

The same file can declare sub-agents and reviewers. For a fuller example, see
Polly at [`examples/polly/`](https://github.com/heurema/goalrail/tree/main/examples/polly/), and the
[Agent YAML spec](https://github.com/heurema/goalrail/blob/main/docs/AGENT_YAML_SPEC.md) for the full schema.
