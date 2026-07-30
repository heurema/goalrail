## Why

`gr init` marks a repository and installs nothing. The harness a repository
actually needs — the OpenSpec overlay carrying the Goalrail schema, the session
hooks, a way to see whether any of it is intact, and a way to refresh it — is
assembled by hand today, and nothing detects when a repository's copy of the
schema has drifted from the canonical one. Confirmed intent turns
initialization into the act that installs the whole harness, grows `gr health`
into `gr doctor`, and makes refreshing it one command. It also settles where the
session hook is registered: per repository, in the user's own project settings
file, which is what makes the boundary a property of where the hook lives rather
than of Goalrail's first act.

## What Changes

- `gr init` materializes the canonical overlay — the config naming the Goalrail
  schema, the schema, and its six templates — from copies embedded in the `gr`
  binary, alongside the existing marker. It reports what it created, changed,
  and left alone, names the pinned invocation the repository is now driven by,
  and says the new files are the user's to commit. Repeating it changes nothing.
- On a repository that already has OpenSpec, initialization adds the schema and
  switches the config without touching any existing spec or change. A config
  already naming a foreign custom schema stops initialization instead of
  silently rewiring someone else's workflow.
- **BREAKING** for the second scaffold: the session hooks are registered by
  `gr init`, per repository, in that scaffold's per-user project settings file,
  and only after verifying that Git ignores it. `gr connect` for that scaffold
  writes nothing and points at `gr init`. The first scaffold keeps its
  user-scope connection command, because registering inside the repository is
  externally blocked there.
- Registration is refused rather than shared: a settings path a repository could
  supply would install hooks in a teammate's sessions, and their consent is not
  the user's to give.
- Re-running `gr init` repairs a registration that is stale or unscoped, under
  the replace-not-accompany discipline connection already follows. A legacy
  user-scope registration for the repository-scope scaffold is never modified by
  initialization; the diagnosis names it and the consented command that removes
  it.
- The scaffold is selected by detection with a flag override. Where none is
  detected the overlay is still installed and the report names the command that
  registers the attachment later.
- `gr doctor` replaces `gr health` as the diagnosis surface and widens it:
  per-scaffold attachment state as today, overlay presence and per-file digest
  drift, whether the overlay is behind the binary's canon, whether the Node
  runtime the stock OpenSpec CLI needs is present, and one observability line.
  It carries structured output and an exit code that separates healthy from not.
  `gr health` keeps working and names its successor.
- `gr update` brings a repository's overlay up to the installed binary's canon
  in one command, verifies the result by digest, reports every rewritten file,
  and leaves the previous state recoverable. Drift stops it: it names the
  drifted files and overwrites only on an explicit flag. Its help states that it
  does not update the `gr` binary.
- `gr` gains a version of its own, reported by the tool. No repository fact
  depends on reading it: currency is decided by digests. OpenSpec's version
  stays internal.
- No `gr` command executes Node, `npx`, or the stock OpenSpec CLI. The canon is
  validated against that CLI on the development side before it is embedded, so
  a user without Node still gets a standing harness — and `gr doctor` states
  what the missing CLI costs them (validation and archival) rather than
  reporting a fault.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Initialization materializes the embedded overlay beside the marker and reports it. | OUT-1, SIG-1, SIG-2 | NG-4, NG-9 |
| An existing OpenSpec root survives; a foreign custom schema stops initialization. | OUT-2, SIG-3 | NG-4 |
| The hook is registered per repository, in the per-user project file, only when Git ignores it. | OUT-3, SIG-5 | NG-6 |
| Scaffold selection is detection plus a flag override; no scaffold still installs the overlay. | OUT-3, SIG-5 | NG-6 |
| Re-running initialization repairs a stale or unscoped registration. | OUT-3, SIG-6 | NG-7 |
| `gr connect` for the repository-scope scaffold writes nothing and names `gr init`. | OUT-3 | NG-6 |
| Disconnection removes a repository-scope registration with no residue and no collateral. | OUT-4, SIG-6 | NG-8 |
| A legacy user-scope registration is reported, never modified by initialization. | OUT-4 | NG-6 |
| The trust disclosure states what is established for that scaffold, and the interactive observation is recorded. | OUT-5, SIG-11 | NG-7 |
| `gr doctor` reports one diagnosis with a distinct state and next action per failure. | OUT-6, SIG-7 | NG-2 |
| Currency is judged by digests; the binary's version is information only. | OUT-6, OUT-9, SIG-4 | NG-1 |
| Every prescribed remedy resolves to a command the CLI accepts; `gr health` remains as an alias. | OUT-7, SIG-8 | NG-5 |
| `gr update` re-materializes, verifies, reports, and stays reversible. | OUT-8, SIG-9 | NG-1 |
| Drift stops an update; an explicit flag discards local edits. | OUT-8, SIG-4 | NG-8 |
| Observability is detected and reported as optional, with no credential in the repository and no key in any output. | OUT-10, SIG-10 | NG-2, NG-3 |
| No `gr` command invokes Node or the stock CLI; absence is a stated fact. | OUT-1, OUT-6, SIG-12 | NG-10 |

No proposed change lies outside this table.

## Capabilities

### New Capabilities

- `harness-init`: what one initialization installs in one repository, how it
  reports it, where it registers the attachment, and what it refuses.
- `harness-doctor`: the single diagnosis surface for a repository's harness,
  including drift, currency, the checking toolchain, and observability.
- `harness-update`: bringing a repository's overlay to the installed binary's
  canon in one command, with drift as a stop rather than a silent overwrite.

### Modified Capabilities

- `ambient-connect`: registration scope becomes a per-scaffold property —
  repository scope during initialization where the settings layer allows it,
  user scope through the connection command where it does not; consent may never
  be given on a teammate's behalf; the trust disclosure gains a documented state
  distinct from observed and from unknown; attachment health is judged against
  the scope where that scaffold registers, and a registration in the other scope
  is reported rather than silently modified.
- `operator-cli-help`: the help surface presents installing the harness,
  diagnosing it, and updating it; the diagnosis command is discoverable by its
  current name while the previous name keeps working; the hook entry point stays
  hidden.

## Impact

- `cmd/gr`: new `doctor` and `update` commands, `health` retained as an alias,
  `init` grows the overlay and registration work, `connect` redirects for the
  repository-scope scaffold, and a version surface appears.
- `internal/ambient`: registration gains a repository scope with a git-ignore
  precondition, the scope becomes a per-scaffold property, disconnection spans
  both scopes, and the diagnosis grows into its own reporting surface.
- New embedded canon: the overlay files ship inside the binary with a per-file
  digest set, materialized and compared from there.
- `openspec/config.yaml`, `openspec/schemas/goalrail-intent/**`: unchanged as
  repository content; they become the canonical source the embedded copy is
  taken from, so a change to them is a change to the canon.
- No change to the announcement text, the reserved escalation path, retention,
  intent binding, the fail-quiet posture, the wrapper lifecycle, or any
  canonical schema.
- No new dependency, service, credential store, or deployment surface. No Go
  code executes Node or the stock OpenSpec CLI.

## Non-Goals

- No network release check and no self-update of the `gr` binary; there is no
  release channel to query.
- No exporter, span emitter, or trace writer, and no change to the read-only
  Langfuse adapter. v0 observability is detection and reporting.
- No hosted or multi-tenant observability.
- Do not fork, vendor, or re-implement the stock OpenSpec CLI, and do not
  materialize its provider prompt packs. Replacing it to remove Node from the
  agent-facing workflow is a separate decision.
- Do not modify user-level scaffold configuration from initialization, write a
  shared committed settings file, or touch the owner's working configuration or
  deferred branch.
- Do not write, compute, reproduce, or simulate a scaffold trust record.
- Do not delete a repository's committed harness files as part of disconnection.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, provider run, or external effect. Each remains a separate
  owner gate, recorded before the action.
