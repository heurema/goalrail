# Goalrail v0.2 installation and setup

Goalrail separates a binary download from managed project setup. Installing a
standalone `gr` does not install the declared planning runtime, attach a
scaffold, create provider trust, or activate protected admission.

## Release assets

Each v0.2 release publishes four supported platforms:

- `darwin_amd64`
- `darwin_arm64`
- `linux_amd64`
- `linux_arm64`

Windows is not currently supported. The release contains:

| Asset | Contents and use |
|---|---|
| `gr_<version>_<os>_<arch>.tar.gz` | Minimal archive containing `gr` and `LICENSE` |
| `goalrail-setup_<version>_<os>_<arch>.tar.gz` | Plan-selected private Goalrail, Node, and OpenSpec runtime bundle |
| `goalrail-setup_<version>_<os>_<arch>.manifest.json` | Exact per-file, component, license, provenance, and binary identities for the setup archive |
| `current-release.json` | Bounded anonymous discovery metadata for the current compatible release |
| `checksums.txt` | SHA-256 coverage for every published release asset |

The setup archive is not a convenience archive to unpack globally. It is an
input to the exact-plan authorization and verification flow.

## Discovery and integrity

The public discovery URL is:

```text
https://github.com/heurema/goalrail/releases/latest/download/current-release.json
```

Reading it requires no login, credential, repository identity, or machine
registration. It names the release version, supported platforms, asset names,
sizes, SHA-256 digests, binary identity, compatibility facts, and checksum
artifact.

`latest` and `current-release.json` are mutable discovery pointers. They are not
integrity identities and cannot remain in an authorized plan. After selecting a
compatible version, use only exact release URLs:

```text
https://github.com/heurema/goalrail/releases/download/<version>/<asset-name>
```

Verify all of the following before execution or durable placement:

1. selected OS and architecture;
2. exact release version and asset name;
3. downloaded byte size;
4. metadata SHA-256;
5. the same SHA-256 in the exact release's `checksums.txt`;
6. for setup bundles, manifest digest, release/platform identity, every file,
   embedded binary version, component source, license, and provenance identity.

Never use curl-to-shell, execute from a pipe, trust only the mutable `latest`
redirect, or replace a missing checksum with a transport-success assumption.

## Manual minimal installation

This route installs only standalone `gr`. It is useful for diagnosis and manual
operator commands; it does not satisfy a managed project's full setup profile.

1. Download `current-release.json` to a temporary directory without executing
   anything:

   ```sh
   curl --fail --location --proto '=https' \
     --output current-release.json \
     https://github.com/heurema/goalrail/releases/latest/download/current-release.json
   ```

2. Inspect the bounded JSON, select the entry matching the machine, and record
   its exact `<version>`, minimal archive `<name>`, `<size>`, and `<sha256>`.
3. Download that archive and the checksum file from the exact immutable release
   URL. Do not substitute `latest`:

   ```sh
   curl --fail --location --proto '=https' --output '<archive-name>' \
     'https://github.com/heurema/goalrail/releases/download/<version>/<archive-name>'
   curl --fail --location --proto '=https' --output checksums.txt \
     'https://github.com/heurema/goalrail/releases/download/<version>/checksums.txt'
   ```

4. Compare the actual byte size with metadata, then verify SHA-256 against both
   metadata and the matching single line from `checksums.txt`:

   ```sh
   shasum -a 256 '<archive-name>'       # macOS
   sha256sum '<archive-name>'           # Linux
   ```

5. Extract to the temporary directory and verify `gr version` before selecting
   a durable destination. The archive contains only `gr` and `LICENSE`.
6. Place `gr` at a durable user-local path such as `~/.local/bin/gr`, preserving
   any previous binary until the new one is verified. Add `~/.local/bin` to
   `PATH` only as a separate explicit user configuration decision.
7. Delete the temporary directory only after verification and durable
   placement.

If any name, size, digest, platform, version, or archive member disagrees, stop.
Do not try another unplanned source or execute the bytes.

## macOS quarantine

Goalrail binaries are not currently signed or notarized. A browser, Finder, or
machine policy may attach `com.apple.quarantine`; command-line downloads often
do not, but the download method alone is not proof. Inspect the extracted binary:

```sh
xattr -p com.apple.quarantine ~/.local/bin/gr
```

Exit status indicating no such attribute means there is no quarantine flag.
When it is present, first complete checksum/version verification. Then choose an
explicit trust route: use macOS Privacy & Security / Open Anyway, or deliberately
remove only that attribute from the exact verified binary:

```sh
xattr -d com.apple.quarantine ~/.local/bin/gr
```

Removing quarantine changes a local security decision. An agent must list it in
the setup plan and receive authorization; it must not clear the attribute merely
because execution failed.

## Agent-assisted clean-machine setup

A managed repository already contains `.goalrail/project.json`, its referenced
`.goalrail/bootstrap.md`, setup profile, and supported-agent instructions. A
feature request is not permission to install anything.

The supported agent must:

1. recognize the committed managed declaration in the clone or worktree;
2. make zero material project-code writes while setup is unresolved;
3. inspect OS, architecture, proposed durable destinations, existing bytes,
   scaffold configuration, and prerequisites read-only;
4. read only bounded non-executable public release metadata and the selected
   setup manifest during planning;
5. emit one canonical `goalrail.setup-plan/v1` with `project_code_writes: 0`;
6. show the exact plan and digest in an owner-readable explanation;
7. ask exactly: `Do you authorize Goalrail setup plan <plan_digest> exactly as shown?`;
8. on refusal, silence, ambiguity, or an incomplete plan, make no mutations and
   keep the original feature request paused;
9. after exact authorization, download only the plan-pinned immutable assets to
   a temporary directory, verify them, run `gr setup verify-plan`, then run
   `gr setup apply` with the exact artifacts and a bounded continuation ref;
10. perform only enumerated local mutations, never manufacture provider trust,
    capture the setup receipt and private recovery locator, run diagnosis, and
    return to the original request at the Intent Snapshot gate.

A new dependency, changed destination byte, login, credential action, global
setting, privilege escalation, trust record, external mutation, or other
unlisted prerequisite invalidates authorization. The agent must stop, create a
new complete plan, and ask again for that new digest.

The committed bootstrap contract contains the full protocol. An agent that
cannot or will not follow it must not be allowed to improvise installation or
continue material work.

## Durable placement

An authorized setup plan normally places the versioned bundle below:

```text
~/.local/share/goalrail/bundles/<version>/<os_arch>/
```

It selects the verified executable at a stable user-local path such as:

```text
~/.local/bin/gr
```

The exact paths are plan data, not universal permission. Existing files,
symlinks, special files, unreadable configuration, or unexpected byte changes
make the plan incomplete or invalidate apply. Temporary archives never become a
durable installation, and setup never writes project code.

## Verification and trust boundaries

After apply, `gr doctor --json` must report each layer independently:

- `managed`: committed project declaration is valid;
- `locally_ready`: exact bundle, compiler/runtime, project canon, schema, and at
  least one supported attachment are ready;
- `lineage_ready`: governing lineage policy is valid;
- `shared_admission_active`: current independent evidence proves the required
  protected check on the target.

Local hooks are advisory. A prepared workflow file is not activation evidence.
Managed project identity does not prove provider trust or protected admission.
Provider authentication, trust confirmation, workflow activation, required-check
configuration, and branch protection are separate owner-authorized external
actions.

## Rollback

### Authorized setup

Every apply attempt retains one private bounded `goalrail.setup-recovery/v1`
artifact. Roll back only that attempt:

```sh
gr setup rollback \
  --repo '<repository>' \
  --home '<planned-home>' \
  --recovery '<private-recovery.json>'
```

Rollback rechecks the exact current bytes, restores only recorded prior state,
and removes only paths proven to belong to that plan. It never deletes unrelated
runtimes or configuration and never changes Git history or external protection.

### Manual minimal installation

Manual installation has no Goalrail recovery artifact. Preserve the previous
destination before replacement. Manual rollback is the explicit restoration of
that known previous file, or removal of the exact newly installed binary when
the destination was previously absent. Do not recursively remove
`~/.local/share`, `~/.local/bin`, a home directory, or an unresolved variable.

## No automatic external effects

Setup does not commit, push, open a pull request, enable a provider workflow,
change branch protection, deploy, publish, log in, issue credentials, or grant
trust. A successful setup receipt returns control to the initiating request; it
does not authorize any of those effects or confirm project intent.
