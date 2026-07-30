<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/goalrail-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="assets/goalrail-light.svg">
  <img src="assets/goalrail-dark.svg" width="100%" alt="Goalrail keeps agent delivery on track from product-team intent through controlled work, checks, repair, evidence, and human review">
</picture>

**A coding agent that asks instead of guessing.**

</div>

Goalrail installs a harness into one repository. After that the agent you already
use — in the tool you already use — works against intent you confirmed, and when
the work cannot be done as specified it writes the question down and stops
instead of inventing an answer.

You run no Goalrail command per task. You start sessions the way you always do.

## Install

```sh
go install github.com/heurema/goalrail/cmd/gr@latest
```

Then, in the repository you want it in:

```sh
gr init
```

That is the whole installation for scaffolds whose settings layer allows it. `gr
init` prints exactly what it created, what it changed, and what it left alone.

Prebuilt binaries are not published yet — see [Status](#status).

### Or hand it to your agent

Goalrail is meant to be installed by the agent, not by hand. Paste this:

> Install Goalrail in this repository. Run `go install
> github.com/heurema/goalrail/cmd/gr@latest`, then `gr init` in the repository
> root. If it refuses to register the session hooks because the settings path is
> not ignored by git, read the reason it prints and re-run with
> `gr init --fix-gitignore`. Finally run `gr doctor` and show me its output
> verbatim. Do not edit any file outside this repository.

The last step is the point: the agent proves the installation rather than
claiming it.

## What `gr init` writes

Nothing is installed anywhere you did not name, and nothing lands in a file a
commit could hand to a teammate.

| Path | What it is |
|---|---|
| `openspec/schemas/goalrail-intent/**` | The workflow schema and its templates, materialized from a copy inside the binary |
| `openspec/config.yaml` | Created if absent; afterwards only its `schema:` key is managed |
| `.goalrail/ambient.json` | The marker that says this repository participates — git-ignored, per clone |
| `.claude/settings.local.json` | The session hooks, in the per-user project file, **only if git ignores it** |

A registration a commit could carry would run in every teammate's session on your
consent alone, so `gr init` refuses to write one and tells you what would make it
possible. It never touches your user-level scaffold configuration.

To remove the attachment: `gr disconnect --scaffold <name>`. The files above are
ordinary repository content — deleting them is your act, not ours.

## The loop

1. A session opens. The agent is told, in one fixed line, that this repository
   accepts one escalation and where to write it.
2. The agent works. If the task cannot be completed as specified — contradictory
   requirements, a decision that is not its to make — it writes the question to
   `.goalrail/blocked.md` and changes nothing else.
3. The session ends. The question is retained outside the repository, with its own
   identity and a link to the intent it belongs to, so deleting the file loses
   nothing.
4. You answer by confirming a new version of the intent. The next session works
   from that.

There is no second answering mechanism, no dialogue, no resume.

## Commands

| Command | What it does |
|---|---|
| `gr init` | Install the harness here, and register the session hooks where the scaffold allows it |
| `gr doctor` | Report whether the harness is intact: attachment, overlay drift per file, toolchain, observability. Exits `0` healthy, `1` a problem you can act on, `2` the check itself did not run |
| `gr update` | Bring this repository's harness up to what the installed `gr` carries. Stops rather than overwriting a local edit; keeps a recoverable copy |
| `gr connect` | Attach a scaffold that can only register at user scope; needs `--yes` |
| `gr disconnect` | Remove every registration, in whichever scope it lives |
| `gr version` | This binary's version and the overlay it carries |

Run `gr help` for the rest.

## Supported scaffolds

**Claude Code** — `gr init` is the whole attachment. The hooks are registered in
the repository, so they are never invoked in unrelated sessions at all. A live
session confirmed a registered hook runs with no approval step.

**Codex** — registering inside a repository is blocked there by an external
defect, so it registers at user scope: `gr connect --scaffold codex --yes`, then
review and trust the hooks with `/hooks` inside Codex. Until you do, nothing runs.
`gr doctor` names that state, which otherwise looks exactly like a broken install.

## Status

Early, and honest about it.

**Works today:** installation, the diagnosis, the update path, the escalation loop
end to end on Claude Code, and the intent-first workflow this repository is
itself developed with.

**Not there yet:** no published release binaries — `go install` is the only
route; observability is bring-your-own detection only, with no exporter; and
`gr update` refreshes this repository's harness, never the `gr` binary.

Requires Go to install. The harness itself needs no Node, but validating and
archiving changes uses the stock [OpenSpec](https://github.com/Fission-AI/OpenSpec)
CLI, which does — `gr doctor` reports whether it is available and what its absence
costs, and never fails because of it.

## How this repository is developed

Every significant change here starts from bounded evidence, becomes a versioned
intent snapshot the owner actively confirms, and only then compiles into a
proposal, spec deltas, a design, and tasks. Confirmed intent describes the
requested result; it never grants permission to commit, push, merge, or run
anything.

The accepted contract lives in [`openspec/specs/`](openspec/specs). The reasoning
behind each change, including what was observed and what was not, lives in
[`openspec/changes/archive/`](openspec/changes/archive).

## Pilot

Goalrail is founder-led and looking for small product teams already using coding
agents in real development. If the loop above sounds like a problem you have,
message [Vitaly on Telegram](https://t.me/vitnm) — we can look at your setup and
decide whether there is a useful first experiment.

## License

MIT — see [LICENSE](LICENSE).
