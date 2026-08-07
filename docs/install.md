# Goalrail v0.2 installation and setup

This document is written for a supported agent. The owner is not expected to
download an archive, compare a digest, or place a binary: the agent performs the
installation and the owner makes one decision — authorizing a single plan — and
then sees a verdict.

Goalrail separates a binary download from managed project setup. Installing a
standalone `gr` does not install the declared planning runtime, attach a
scaffold, create provider trust, or activate protected admission.

Obtaining `gr` itself is the one part that cannot be delegated: before it exists
there is nothing to delegate to. Everything after that is a `gr` command.

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

## Prerequisites

This document assumes `curl`, `tar`, `jq`, and a SHA-256 tool. They are the
agent's own prerequisites, not Goalrail components: a machine missing one cannot
reach the setup plan at all, and discovering that later is exactly the kind of
unlisted prerequisite that invalidates an authorization. Check first and report
the missing one rather than installing it:

```sh
for tool in curl tar jq; do
  command -v "${tool}" >/dev/null || { echo "missing prerequisite: ${tool}" >&2; exit 1; }
done
```

The SHA-256 tool differs between the supported platforms — `sha256sum` on Linux,
`shasum` on macOS — so bind it once and use it everywhere below:

```sh
if command -v sha256sum >/dev/null; then
  sha256() { sha256sum "$@"; }
elif command -v shasum >/dev/null; then
  sha256() { shasum -a 256 "$@"; }
else
  echo "missing prerequisite: a SHA-256 tool" >&2; exit 1
fi
```

## Resolving this machine's platform

The metadata names an operating system and an architecture; the machine reports
different words for the same thing. `uname -s` gives `Darwin` or `Linux`, which
lower-case to `darwin` and `linux`. `uname -m` gives `arm64` or `aarch64`, both
of which are `arm64`, and `x86_64` or `amd64`, both of which are `amd64`.

```sh
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64)  arch=amd64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
```

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

Digests are written two ways and must be compared as the same value. The
metadata carries an algorithm prefix — `sha256:2141dd…` — while `checksums.txt`
and the `sha256sum` and `shasum` tools print the bare hexadecimal. Strip the
prefix before comparing, or compare against `checksums.txt`, which is already
bare.

Pin the protocol across the redirect as well as on the first request. A release
asset URL redirects to a different host, and `--proto` governs only the request
it is given; `--proto-redir` governs where a redirect may go.

Never use curl-to-shell, execute from a pipe, trust only the mutable `latest`
redirect, or replace a missing checksum with a transport-success assumption.

## Obtaining `gr`

This installs only standalone `gr`. It does not satisfy a managed project's
setup profile; it is what makes the commands below available.

Two paths are used throughout. `repo` is the managed clone being set up; `work`
is a temporary directory that holds downloads until placement is verified. Every
command below names one of them explicitly, because the shell stays in `work`
and `.` would name the wrong one:

```sh
repo="$(pwd)"          # the managed clone, before changing directory
work="$(mktemp -d)"
cd "${work}"
```

Read the discovery metadata:

```sh
curl --fail --location --proto '=https' --proto-redir '=https' \
  --output current-release.json \
  https://github.com/heurema/goalrail/releases/latest/download/current-release.json
```

Select this machine's entry and record the exact version, archive name, size and
digest:

```sh
version="$(jq -r .release_version current-release.json)"
entry="$(jq --arg os "${os}" --arg arch "${arch}" \
  '.supported_platforms[] | select(.platform.os == $os and .platform.arch == $arch)' \
  current-release.json)"
archive="$(printf '%s' "${entry}" | jq -r .minimal_archive.name)"
size="$(printf '%s' "${entry}" | jq -r .minimal_archive.size_bytes)"
digest="$(printf '%s' "${entry}" | jq -r .minimal_archive.sha256 | sed 's/^sha256://')"
```

These values come from a mutable pointer that nothing has verified yet, so treat
every one of them as untrusted input before it reaches a filesystem path or a
URL. A name carrying a slash or a leading dot would make the download below
write wherever it pointed, long before any checksum could object:

```sh
case "${archive}" in
  ''|*/*|.*) echo "the metadata names an unusable archive: ${archive}" >&2; exit 1 ;;
esac
case "${version}" in v[0-9]*) ;; *) echo "unusable version: ${version}" >&2; exit 1 ;; esac
expr "${size}" + 0 >/dev/null 2>&1 || { echo "unusable size: ${size}" >&2; exit 1; }
printf '%s' "${digest}" | grep -Eq '^[0-9a-f]{64}$' || { echo "unusable digest" >&2; exit 1; }
```

Download that archive and the checksum file from the exact immutable release
URL. Do not substitute `latest`:

```sh
curl --fail --location --proto '=https' --proto-redir '=https' --output "${archive}" \
  "https://github.com/heurema/goalrail/releases/download/${version}/${archive}"
curl --fail --location --proto '=https' --proto-redir '=https' --output checksums.txt \
  "https://github.com/heurema/goalrail/releases/download/${version}/checksums.txt"
```

Compare the byte size with the metadata, then the digest with both the metadata
and `checksums.txt`. The checksum file covers every published asset, so verify
only what was downloaded — otherwise the command fails over archives that were
never fetched:

`--ignore-missing` skips the entries whose files are absent, which is what makes
the command usable here — but it also succeeds when the file being verified has
no entry at all, as long as some other listed file does. Isolate the archive's
own line so its absence is a failure rather than a pass:

```sh
[ "$(wc -c < "${archive}")" = "${size}" ] || { echo "size disagrees with metadata" >&2; exit 1; }

grep -F " ${archive}" checksums.txt > archive.sha256 || { echo "no checksum line for ${archive}" >&2; exit 1; }
[ "$(wc -l < archive.sha256)" = "1" ] || { echo "ambiguous checksum lines for ${archive}" >&2; exit 1; }
sha256 -c archive.sha256 || exit 1

sha256 "${archive}" | grep -q "^${digest} " || { echo "digest disagrees with metadata" >&2; exit 1; }
```

Extract and check the binary before choosing a durable destination. The archive
contains only `gr` and `LICENSE`, and `gr version` must report the version the
metadata named:

```sh
tar -xzf "${archive}"
./gr version   # {"canon":"sha256:…","version":"<version>"}
```

Placing it is a durable change to the machine, and it happens before any setup
plan exists, so it cannot be covered by the plan authorization asked for later.
It is its own owner decision: ask before creating or replacing the destination,
and do not replace an existing installation because setup was merely discovered.

Place it at a durable user-local path, creating the directory if it does not
exist and keeping any previous binary until the new one is verified. Without
that copy there is nothing for the rollback below to restore:

A destination that is a symbolic link would be written through, changing a file
outside the directory this decision named. Refuse it rather than following it:

```sh
mkdir -p ~/.local/bin
if [ -L ~/.local/bin/gr ]; then
  echo "~/.local/bin/gr is a symbolic link; resolve it before replacing it" >&2; exit 1
fi
if [ -e ~/.local/bin/gr ] && [ ! -f ~/.local/bin/gr ]; then
  echo "~/.local/bin/gr is not a regular file" >&2; exit 1
fi
[ -f ~/.local/bin/gr ] && cp -p ~/.local/bin/gr "${work}/gr.previous"
cp gr ~/.local/bin/gr
~/.local/bin/gr version   # if this disagrees, restore "${work}/gr.previous"
```

`~/.local/bin` is not on `PATH` on every machine, and putting it there is a
separate explicit user configuration decision. Every command below therefore
invokes `~/.local/bin/gr` by its full path: a bare `gr` would fail, or resolve
some other installation that happens to be on `PATH`.

Keep `work` until the managed setup below has completed: it holds the discovery
metadata, the checksum file and the previous binary, all of which the later steps
and the rollback still need.

If any name, size, digest, platform, version, or archive member disagrees, stop.
Do not try another unplanned source or execute the bytes.

## The other route: `go install`

On a machine that already has a Go toolchain:

```sh
GOBIN=~/.local/bin go install github.com/heurema/goalrail/cmd/gr@latest
```

`GOBIN` is set because `go install` otherwise writes to `$GOPATH/bin` or
`$HOME/go/bin`, while every command below invokes `~/.local/bin/gr`.

Once a release tag exists this installs that release rather than the branch tip.
It performs no checksum verification of its own beyond the Go module checksum
database, and it produces no setup bundle: the managed setup below is still
required.

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

## Managed setup, step by step

A managed repository already contains `.goalrail/project.json`, its referenced
`.goalrail/bootstrap.md`, setup profile, and supported-agent instructions. A
feature request is not permission to install anything.

The agent must:

1. recognize the committed managed declaration in the clone or worktree;
2. make zero material project-code writes while setup is unresolved;
3. inspect OS, architecture, proposed durable destinations, existing bytes,
   scaffold configuration, and prerequisites read-only;
4. read only bounded non-executable public release metadata and the selected
   setup manifest during planning;
5. present the exact plan and its digest in an owner-readable explanation;
6. ask exactly: `Do you authorize Goalrail setup plan <plan_digest> exactly as shown?`;
7. on refusal, silence, ambiguity, or an incomplete plan, make no mutations and
   keep the original feature request paused.

A new dependency, changed destination byte, login, credential action, global
setting, privilege escalation, trust record, external mutation, or other
unlisted prerequisite invalidates authorization. The agent must stop, create a
new complete plan, and ask again for that new digest.

### Resolve a complete plan

`gr setup plan` reads what it is given and never fetches. Supplying nothing
produces an incomplete plan naming what is missing, so fetch the two inputs
first. The release metadata is the file already downloaded above; the setup
manifest is named by this machine's entry in it:

The scaffold must be named. It defaults to `codex`, so omitting it while setting
up for another agent plans and attaches the wrong one and then reports it ready:

```sh
scaffold=claude-code   # or codex — the agent performing this setup

manifest="$(printf '%s' "${entry}" | jq -r .setup_manifest.name)"
curl --fail --location --proto '=https' --proto-redir '=https' --output "${manifest}" \
  "https://github.com/heurema/goalrail/releases/download/${version}/${manifest}"

~/.local/bin/gr setup plan --repo "${repo}" --scaffold "${scaffold}" \
  --release-metadata current-release.json \
  --setup-manifest "${manifest}" > plan.json
```

A usable plan reports `"state":"complete"` and `"project_code_writes":0`. An
incomplete one names its reason under `incomplete_reason_ids`;
`RELEASE_METADATA_UNAVAILABLE` and `SETUP_MANIFEST_UNAVAILABLE` mean the
corresponding flag above was not supplied.

### Ask the owner

The plan digest is the SHA-256 of the plan's own canonical bytes, which is what
`gr setup plan` wrote:

```sh
plan_digest="sha256:$(sha256 plan.json | cut -d' ' -f1)"
```

Present the plan's components, mutations, destinations, network access and
rollback in the owner's language, then ask the exact question above.

### Record the authorization

An authorization is a `goalrail.plan-authorization/v1` document naming the
project, that exact plan digest, and the owner's decision. It must be canonical
JSON: the exact field order below, and **no trailing newline** — a newline makes
it non-canonical and the verification refuses it without explaining why.

```sh
printf '{"schema":"goalrail.plan-authorization/v1","project_id":"%s","plan_digest":"%s","decision_ref":"%s","actor_ref":"%s","authorized_at":"%s"}' \
  "$(jq -r .project_id plan.json)" \
  "${plan_digest}" \
  "decision:<bounded reference to the owner's recorded decision>" \
  "owner:<bounded owner reference>" \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > authorization.json
```

### Verify, then apply

Download the setup archive the plan pinned, extract it to a temporary directory,
and verify it against the plan and the authorization before anything durable
happens:

The archive is verified against both identities before it is opened. The checksum
file alone would not catch a publication where it and the metadata disagree, and
`verify-plan` receives the extracted directory rather than the archive, so it
cannot see the difference either:

```sh
bundle_archive="$(printf '%s' "${entry}" | jq -r .setup_archive.name)"
bundle_size="$(printf '%s' "${entry}" | jq -r .setup_archive.size_bytes)"
bundle_digest="$(printf '%s' "${entry}" | jq -r .setup_archive.sha256 | sed 's/^sha256://')"
case "${bundle_archive}" in ''|*/*|.*) echo "unusable setup archive name" >&2; exit 1 ;; esac

curl --fail --location --proto '=https' --proto-redir '=https' --output "${bundle_archive}" \
  "https://github.com/heurema/goalrail/releases/download/${version}/${bundle_archive}"

[ "$(wc -c < "${bundle_archive}")" = "${bundle_size}" ] || { echo "setup archive size disagrees" >&2; exit 1; }
sha256 "${bundle_archive}" | grep -q "^${bundle_digest} " || { echo "setup archive digest disagrees" >&2; exit 1; }
grep -F " ${bundle_archive}" checksums.txt > bundle.sha256 || { echo "no checksum line" >&2; exit 1; }
sha256 -c bundle.sha256 || exit 1

mkdir -p bundle && tar -xzf "${bundle_archive}" -C bundle

~/.local/bin/gr setup verify-plan --repo "${repo}" --scaffold "${scaffold}" --bundle bundle \
  --plan plan.json --authorization authorization.json \
  --release-metadata current-release.json
```

Verification returns `"verified":true` with the plan, authorization and manifest
digests it bound. Only then:

```sh
~/.local/bin/gr setup apply --repo "${repo}" --scaffold "${scaffold}" --bundle bundle \
  --plan plan.json --authorization authorization.json \
  --release-metadata current-release.json \
  --continuation-ref 'task:<bounded id of the request this interrupted>'
```

Apply performs only the enumerated mutations, captures the setup receipt and the
private recovery locator, runs the diagnosis, and returns to the original
request at the Intent Snapshot gate. Setup success confirms no intent and
authorizes no implementation.

The receipt's terminal status reflects the diagnosis taken after the mutations,
not whether the mutations succeeded. Two ordinary situations produce a failure
status over a setup that did everything it was authorized to do: a scaffold whose
own trust step the user has not yet completed, which Goalrail is forbidden from
performing on their behalf; and an apply run by a development build, whose
version corresponds to no published bundle. Read the receipt's mutation statuses
and the diagnosis reasons rather than the terminal status alone, and report the
pending user action rather than an installation failure.

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

After apply, the diagnosis must report each layer independently. It takes the
repository explicitly, because the shell is still in `work` and its default would
diagnose that directory instead:

```sh
~/.local/bin/gr doctor --repo "${repo}" --json
```


- `managed`: committed project declaration is valid;
- `locally_ready`: exact bundle, compiler/runtime, project canon, schema, and at
  least one supported attachment are ready;
- `lineage_ready`: governing lineage policy is valid;
- `shared_admission_active`: current independent evidence proves the required
  protected check on the target.

The planning runtime and compiler are established by inspection: the installed
bundle manifest supplies each component's path, recorded version and digest, and
those files are hashed. Nothing is executed to establish readiness, so a machine
with no Node runtime on its lookup path diagnoses exactly as one that has it.

Local hooks are advisory. A prepared workflow file is not activation evidence.
Managed project identity does not prove provider trust or protected admission.
Provider authentication, trust confirmation, workflow activation, required-check
configuration, and branch protection are separate owner-authorized external
actions.

## Cleaning up

Only once apply has completed and its receipt is captured:

```sh
cd "${repo}"
rm -rf "${work}"
```

Keep the recovery artifact apply produced for as long as the installation may
need rolling back; it does not live in `work`.

## What the owner sees

One question — authorize this exact plan digest or not — and then one verdict:
the components and versions installed, where they were placed, which scaffold
attachments were registered and which still need the scaffold's own trust step,
and what remains before the project is fully enforced. Everything above is the
agent's work, and a failure at any step is reported as that step's exact
blocker, not as an installation recipe for the owner to follow.

## Rollback

### Authorized setup

Every apply attempt retains one private bounded `goalrail.setup-recovery/v1`
artifact. Roll back only that attempt:

```sh
~/.local/bin/gr setup rollback \
  --repo '<repository>' \
  --home '<planned-home>' \
  --recovery '<private-recovery.json>'
```

Rollback rechecks the exact current bytes, restores only recorded prior state,
and removes only paths proven to belong to that plan. It never deletes unrelated
runtimes or configuration and never changes Git history or external protection.

### The `gr` binary itself

Obtaining `gr` has no Goalrail recovery artifact. Preserve the previous
destination before replacement. Rollback is the explicit restoration of that
known previous file, or removal of the exact newly installed binary when the
destination was previously absent. Do not recursively remove `~/.local/share`,
`~/.local/bin`, a home directory, or an unresolved variable.

## No automatic external effects

Setup does not commit, push, open a pull request, enable a provider workflow,
change branch protection, deploy, publish, log in, issue credentials, or grant
trust. A successful setup receipt returns control to the initiating request; it
does not authorize any of those effects or confirm project intent.
