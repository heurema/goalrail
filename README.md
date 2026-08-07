<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/goalrail-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="assets/goalrail-light.svg">
  <img src="assets/goalrail-dark.svg" width="100%" alt="Goalrail keeps agent delivery on track from confirmed intent through linked work and protected admission">
</picture>

**A coding-agent workflow that keeps project intent and delivery evidence connected.**

</div>

Goalrail makes workflow participation a property of the repository, not of one
checkout. A committed `.goalrail/project.json` identifies the same managed
project in every clone and linked worktree. Local runtimes, scaffold attachment,
provider trust, and protected admission remain separately observed facts.

## What happens after a clean clone

On a machine with only a supported coding agent installed:

1. The agent sees the committed project declaration and bootstrap guidance.
2. Before material code work, it diagnoses local readiness read-only. If the
   compatible Goalrail/OpenSpec bundle is absent, it pauses the feature request.
3. It resolves one exact setup plan with component versions, SHA-256 identities,
   destinations, configuration changes, network access, rollback, and zero
   project-code writes.
4. It asks the owner to authorize that exact plan digest. It installs nothing
   until the owner explicitly agrees.
5. After a successful setup receipt, it returns to the original request,
   prepares a versioned Intent Snapshot, and waits for confirmation of that
   exact intent version before implementation.
6. A work unit then links the change, run/session, commits, pull request,
   reviews, checks, terminal receipt, and owner decisions as bounded evidence.

The owner should not have to discover tools or translate an error into an
installation recipe. The supported agent explains the exact enabling action,
requests permission, performs only the authorized plan, verifies it, and resumes
the original request.

Local hooks are advisory early feedback. A prepared workflow file is not
activation evidence. Managed project identity does not prove that integration
is protected. Goalrail reports `fully_enforced` only after a protected shared
admission check is independently verified on the exact target. Without that
evidence, it reports the project as advisory and does not make an ironclad claim.

## Lifecycle

```text
init or migrate -> doctor -> exact setup consent -> confirmed intent/change
-> lineage begin -> bounded work -> lineage attach/inspect -> verify-lineage
-> protected admission
```

- `gr init` creates portable v0.2 governance for a new managed project. Where a
  supported scaffold keeps its settings inside the repository, it also registers
  the session hooks there and adds this clone's own ignore rule so no commit can
  carry them to a teammate. It does not install local dependencies, attach a
  scaffold that registers at user scope, create a commit, or enable a remote
  required check.
- `gr migrate` is the explicit one-project transition from exact v0.1.8
  evidence. There is no automatic migration and no `gr init` per worktree.
- `gr doctor --json` keeps managed identity, local readiness, lineage readiness,
  and shared-admission activation separate.
- `gr setup plan` is read-only. `verify-plan`, `apply`, and `rollback` operate on
  exact canonical artifacts and preserve the authorization boundary.
- `gr lineage begin|attach|inspect` maintains and resolves the append-only work
  graph. Exact bounded evidence may be replicated by content digest.
- `gr verify-lineage` evaluates one frozen Git range and admission packet. The
  same canonical input has the same result locally and in a shared adapter. It
  is advisory: a packet read from disk carries no authenticated provider
  observation, so an owner-gated relation stays missing until the shared check
  observes the provider and verifies the range in one invocation.

Run `gr help` or `gr <command> --help` for the public command surface. Internal
hook and provider collectors are intentionally not operator commands.

## Install and setup bundles

The v0.2 release contract publishes two archive classes for macOS and Linux on
`amd64` and `arm64`:

| Bundle | Name | Intended use |
|---|---|---|
| Minimal | `gr_<version>_<os>_<arch>.tar.gz` | Standalone `gr` and `LICENSE`; diagnosis and explicit manual operation |
| Setup | `goalrail-setup_<version>_<os>_<arch>.tar.gz` | Exact private Goalrail, Node, and OpenSpec bundle selected by an authorized setup plan |

`current-release.json` is an anonymous discovery pointer, not an integrity
identity. A plan pins the exact release asset name, size, immutable release URL,
manifest, and SHA-256 digest, then verifies them against `checksums.txt` before
execution. Executables go to durable user-local storage; temporary download
directories are not installation locations.

Do not pipe a download into a shell. Browser-downloaded macOS artifacts may be
quarantined; inspect and resolve that trust decision explicitly after checksum
verification. See [docs/install.md](docs/install.md) for the manual and
agent-assisted contracts, rollback, platform notes, and exact trust boundary.

## Project files and ownership

Initialization or migration renders three ownership classes:

| Ownership | Examples | Update rule |
|---|---|---|
| Canon-owned | policy, bootstrap, setup profile, adapter descriptors, prepared admission adapter | Exact known bytes; drift is diagnosed and never silently overwritten |
| Managed block | Goalrail block inside `AGENTS.md` or `CLAUDE.md` | Only the uniquely delimited block is managed; surrounding repository content is preserved |
| Repository-owned | project selections, intent/change artifacts, ordinary code | Goalrail validates declared references and policy; it does not claim whole-file ownership |

The legacy `.goalrail/ambient.json` marker may be retained as migration evidence,
but it is no longer project identity. Local machine state and credentials never
belong in `.goalrail/project.json`.

The full schema, reason-code, exit-code, ownership, replication, and compatibility
contract is in [docs/contracts-v0.2.md](docs/contracts-v0.2.md). The lifecycle
source-of-truth map is in [docs/project-lifecycle.md](docs/project-lifecycle.md).

## Migration from v0.1.8

v0.2 intentionally changes identity and diagnosis semantics. Run `gr migrate`
once in the selected legacy project, review every generated path, and commit the
migration so all clones and worktrees inherit the declaration. Local setup is a
separate machine/scaffold concern, and a prepared admission workflow remains
inactive until external protection is independently observed.

See [docs/migration-v0.2.md](docs/migration-v0.2.md) for the owner procedure and
rollback boundaries.

## Commands

| Command | What it does |
|---|---|
| `gr init` | Create committed v0.2 project governance; register a scaffold whose settings live in this repository; no other local setup or external activation |
| `gr migrate` | Convert one exact v0.1.8 project to committed v0.2 governance |
| `gr doctor` | Diagnose project identity, governance, local readiness, lineage, and independently observed shared admission |
| `gr setup plan|verify-plan|apply|rollback` | Plan and execute exact authorized local setup with receipts and recovery |
| `gr lineage begin|attach|inspect` | Create, extend, and resolve the append-only work-unit graph |
| `gr verify-lineage` | Produce canonical admission for an explicit base/head range and packet |
| `gr update` | Reconcile known repository canon without hiding schema migration or owner edits |
| `gr connect` / `gr disconnect` | Manage supported local scaffold attachment under explicit consent |
| `gr prepare|inspect|start|finish` | Run the bounded owner-driven wrapper lifecycle |

`gr update` updates repository harness files, not the `gr` executable. Commit,
push, pull-request creation, branch-protection changes, deploy, and release keep
their own owner gates.

## Compatibility and current status

The v0.2 line writes only current project, setup, WorkSpec, lineage, and
admission schemas. Readers retain legacy WorkSpecs, receipts, and exact v0.1.8
evidence for one migration window; they never relabel legacy evidence as
normally valid. The deprecated doctor field `initialized` mirrors `managed` for
one release and no longer reads the local marker.

This v0.2 implementation is under development. Release publication, real
required-check activation, and migration of any existing external project are
separate owner-authorized actions. Binaries are not signed or notarized, and
Windows is not supported because the current safe-file boundary depends on Unix
semantics that must not be silently weakened.

## How this repository is developed

Every significant change starts from bounded evidence, becomes a versioned
Intent Snapshot the owner actively confirms, and only then compiles into a
proposal, specs, design, and tasks. Confirmed intent describes the result; it
never grants permission to commit, push, merge, activate, deploy, or publish.

The accepted contract lives in [`openspec/specs/`](openspec/specs). Change
rationale and evidence live in [`openspec/changes/`](openspec/changes).

## Pilot

Goalrail is founder-led and looking for small product teams already using coding
agents in real development. Contact [Vitaly on Telegram](https://t.me/vitnm) to
discuss a bounded first experiment.

## License

MIT — see [LICENSE](LICENSE).
