# Migrating one Goalrail v0.1.8 project to v0.2

Goalrail v0.2 is a deliberate governance migration, not a local re-init. The
owner migrates one project once, reviews and commits the portable declaration,
then every clone and linked worktree sees the same managed project identity.

Do not run `gr init` in each worktree. Do not copy a local marker into other
checkouts. Local readiness remains separately diagnosed on each machine and
supported scaffold surface.

## Breaking semantic changes

| v0.1.8 behavior | v0.2 behavior |
|---|---|
| `.goalrail/ambient.json` in one checkout implied initialization | Committed root `.goalrail/project.json` is the only project identity claim in every clone/worktree |
| `initialized` mixed project presence and local attachment | Deprecated `initialized` mirrors only `managed` for one release; use the four explicit readiness fields |
| `gr init` could be understood as harness plus local attachment | `gr init` creates portable project governance only; setup, trust, and activation are separate actions |
| OpenSpec overlay/config could be treated as Goalrail presence | OpenSpec is a replaceable compiler; copied schema files never establish Goalrail identity |
| A local hook could appear to enforce the workflow | Local hooks are advisory; only independently verified protected shared admission can block integration authoritatively |
| A committed workflow could appear active | A prepared workflow file is not activation evidence |
| Legacy WorkSpecs/receipts could be reused without current bindings | Legacy evidence is inspection/migration input only and is never relabeled `VALID` |

The one-release reader window supports exact v0.1.8 marker/overlay evidence,
legacy WorkSpecs, and receipts. It does not silently backport v0.2 validity.

## Preconditions

Before migration:

1. select the exact repository/worktree to migrate;
2. use a v0.2-compatible `gr` binary and run `gr doctor --json --repo <path>`;
3. require category `migration_required`, which means Goalrail found the exact
   known v0.1.8 overlay and acceptable bounded legacy marker evidence;
4. inspect the Git worktree and preserve unrelated owner changes;
5. resolve any foreign custom OpenSpec schema or conflicting reserved path with
   the owner instead of overwriting it; and
6. treat commit, push, pull-request creation, merge, setup, and provider
   activation as separate actions.

A similarly named overlay, malformed/secret-shaped marker, foreign schema,
unsafe path, already-current declaration, or unsupported state is not migration
authority. Stop on the exact diagnostic instead of forcing the command.

## Migrate the project

Run one explicit project-scoped command:

```sh
gr migrate --repo '<project-worktree>'
```

Optionally record the scaffold that later setup should plan for:

```sh
gr migrate --repo '<project-worktree>' --scaffold codex
```

`--scaffold` records a requested later setup target only. It does not install a
runtime, edit user configuration, register trust, or attach/activate a provider.

Migration renders current declaration-bound governance, OpenSpec overlay,
supported-agent guidance, adapter descriptors, and a prepared shared-admission
integration. It preserves repository-owned surrounding content and writes the
declaration last so a partial run cannot establish a valid project claim.

The JSON report names every created or changed path, its ownership class, the
project ID, requested later setup, and the required commit. Re-running the exact
migration is idempotent; a conflicting current/foreign state stops instead of
being overwritten.

## Review and commit

Before committing:

1. review the exact migration report and `git diff`;
2. verify `.goalrail/project.json` contains no user identity, credential,
   absolute machine path, provider session, or checkout-local state;
3. verify its policy, bootstrap, and setup-profile paths and digests match the
   committed bytes;
4. verify managed blocks are unique and all surrounding `AGENTS.md` / `CLAUDE.md`
   content is unchanged;
5. run `gr doctor --json --repo '<project-worktree>'`; `managed` must now be true,
   while missing local setup or external activation must remain visibly
   separate; and
6. run the repository tests and review the intended commit scope.

Migration does not create a commit. The project does not become portable until
the owner separately authorizes and creates a commit containing the complete
migration. Push, PR, merge, and protected-check activation each retain their own
gate.

After that commit reaches another clone or linked worktree, Goalrail discovers
the same project ID from the committed declaration. No marker and no second
`gr init` are required. `gr doctor` may still report setup required for that
machine/scaffold; this does not erase managed identity.

## Local setup after migration

The migration commit is governance, not installation. On a machine lacking the
declared Goalrail/OpenSpec bundle or supported attachment, the agent must keep
material code work paused and follow [install.md](install.md):

1. inspect read-only;
2. present one complete canonical setup plan and digest;
3. request explicit authorization for exactly that plan;
4. verify and apply only the authorized local mutations;
5. capture the receipt and private recovery locator; and
6. return to the original request at the candidate Intent Snapshot gate.

A successful local setup does not confirm intent, authorize implementation,
create lineage, or activate a remote check.

## Prepared versus active admission

Migration may commit a prepared shared-admission adapter. That file proves only
that repository bytes are ready for a later provider action.

Local hooks are advisory. A prepared workflow file is not activation evidence.
Managed project identity does not prove protected enforcement. Goalrail reports
`shared_admission_active=true` only after a read-only observer independently
verifies the configured provider/boundary, exact target, required
`goalrail/lineage` check, evidence source, observation time, and current
freshness.

Missing access is `unknown` or `access_denied`; expired evidence is `stale`.
Until independent verification succeeds, the correct aggregate category is
`locally_ready_advisory`, not `fully_enforced`.

Enabling a workflow, granting credentials, changing branch protection, or
making a check required is an external owner-authorized action. Migration and
local setup do not authorize it.

## Rollback

Rollback boundaries are intentionally separate:

### Before the migration commit

Use the migration report and reviewed Git diff to restore only the exact
uncommitted paths created or changed by that attempt. Preserve unrelated owner
changes and retained legacy evidence. Goalrail does not delete files or reset
the worktree automatically.

### After the migration commit

Revert the exact migration commit through the repository's normal Git review
flow while the one-release legacy reader remains available. Do not rewrite
retained WorkSpecs, receipts, lineage, or legacy evidence. An older binary must
not silently interpret an unsupported v1 declaration as unmanaged; use a
compatible binary or a complete project-commit revert.

### Local setup

Use `gr setup rollback --recovery <private-recovery.json>` for the exact setup
attempt. It restores only recorded prior bytes and installed paths proven to
belong to that plan. It does not change the migration commit.

### External protected admission

Disable or change provider protection only through a separately authorized
provider rollback. Goalrail then diagnoses the observed inactive/advisory state;
it never mutates branch protection as a side effect of project rollback.

No rollback path automatically commits, pushes, merges, deletes releases,
destroys evidence, removes unrelated local dependencies, or changes external
protection.
