# Harness-init brief — owner discussion, 2026-07-30

Input for the next session's Context Pack. This is a discussion record, not a
confirmed intent: the harness-init change starts from its own Context Pack and
its own owner confirmation.

## Scope

`gr init` grows from marking a repository into installing the whole harness:
OpenSpec with the Goalrail schema, optional observability wiring. `gr doctor`
grows out of `gr health` into full diagnosis. Updating becomes one command.

## Update model (owner-refined)

- The only version the user sees is **Goalrail's own**. OpenSpec is an internal
  dependency: when a new OpenSpec version appears, compatibility with the
  Goalrail schema is verified on our side, and the new pin ships inside a `gr`
  release.
- `gr doctor` reports "a Goalrail update is available" and can show what is
  inside, including an OpenSpec bump. `gr update` applies it: rewrites the pin,
  re-validates, shows what changed. Reversible.
- Never silent. A zero-action update could change strict-validation behaviour
  under a working agent, which is exactly the class of silent breakage this
  project exists to remove. Seamless means one command, not zero.
- The pinned invocation (`OPENSPEC_TELEMETRY=0 npx --yes
  @fission-ai/openspec@<pin>`) is owned by `gr`; the user never manages it.

## Custom OpenSpec

- The customization is **not a fork**. The CLI is stock; everything Goalrail
  adds is the `goalrail-intent` schema plus config, laid down as repository
  files. Only the overlay is maintained.
- `gr init` on a clean repository scaffolds `openspec/` with the schema. On a
  repository where OpenSpec is already present, it adds the schema and switches
  the config without touching existing specs and changes, then validates and
  reports what changed.
- The canonical schema is embedded in the `gr` binary and materialized into
  each repository. `gr doctor` compares digests and reports drift; otherwise
  per-repository copies diverge silently.

## Observability

- Principle: **Goalrail works fully with no observability backend**. The local
  state root is the source of truth. Langfuse, Phoenix, and anything else are
  exporters — removable attachments, never dependencies.
- v0 is bring-your-own only: an existing instance (endpoint plus keys) gets
  wired; absence gets one non-nagging `gr doctor` line — "observability: not
  configured (optional)".
- One exporter interface; the existing Langfuse adapter is its first
  implementation; Phoenix and others slot in later if demanded.

## Where the hook gets registered

Today `gr connect` writes to user scope only — one path, the scaffold's user
settings. The promoted requirement already records why that is a limitation
rather than a preference, and that it holds for the first scaffold alone: there,
project-local hooks are silently skipped by an external defect, so user scope is
forced. The second scaffold's project-level hooks are an ordinary merging
settings layer, so the route is open there and is named in the requirement as
the first thing to revisit.

Project scope is better, and for a structural reason. Under user scope the hook
is invoked in every session the user starts anywhere and switches itself off by
checking for the repository marker — the boundary holds because our first act
enforces it. Registered per repository, the hook is never invoked in unrelated
sessions at all, and the boundary is a property of where it is installed.

That makes `gr init` its natural home: marking a repository and registering the
hook in it are one act, and for the second scaffold a separate `connect` step
then has no reason to exist. Which is squarely this work's territory.

Consequence for the open approval question: **do not answer it with a
user-scope registration.** It is a throwaway — the registration would be
installed in the owner's working configuration, read once, removed, and then
superseded by the project-scope arrangement anyway. Answer it inside this work,
in the configuration that will actually ship.

## Owner decisions, 2026-07-30

1. **Hosted multi-tenant Langfuse: deferred.** Traces carry prompts and user
   code; hosting them makes Goalrail a data processor — accounts, tenant
   isolation, privacy. That is a separate product decision, not an init
   feature. Revisit condition: a user who wants monitoring but will not
   self-host.
2. **v0 observability is bring-your-own**, and none at all is a fully working
   configuration.
3. **The update surface is `gr`'s own version.** OpenSpec compatibility is
   verified internally before a release; the user is never asked to reason
   about OpenSpec versions.
4. **Registration scope belongs to this work, and the first real attachment
   waits for it.** The owner asked why a live check would use user scope when
   project scope is the direction, and the answer was only that the code has
   nothing else today — which is a reason to build it, not a reason to install a
   registration that will be replaced. So: no user-scope registration in the
   owner's own configuration for the sake of a one-off observation, and the
   approval question is answered here, once, in the shipping arrangement.
