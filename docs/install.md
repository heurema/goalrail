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

Work in a temporary directory, and keep it until placement is verified:

```sh
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

```sh
[ "$(wc -c < "${archive}")" = "${size}" ] || { echo "size disagrees with metadata" >&2; exit 1; }

sha256sum --ignore-missing -c checksums.txt   # Linux
shasum -a 256 --ignore-missing -c checksums.txt   # macOS

sha256sum "${archive}" | grep -q "^${digest} " || { echo "digest disagrees with metadata" >&2; exit 1; }
```

Extract and check the binary before choosing a durable destination. The archive
contains only `gr` and `LICENSE`, and `gr version` must report the version the
metadata named:

```sh
tar -xzf "${archive}"
./gr version   # {"canon":"sha256:…","version":"<version>"}
```

Place it at a durable user-local path, creating the directory if it does not
exist and preserving any previous binary until the new one is verified:

```sh
mkdir -p ~/.local/bin
cp gr ~/.local/bin/gr
~/.local/bin/gr version
```

`~/.local/bin` is not on `PATH` on every machine, and putting it there is a
separate explicit user configuration decision. Until it is made, invoke `gr` by
its full path.

Delete the temporary directory only after verification and durable placement.

If any name, size, digest, platform, version, or archive member disagrees, stop.
Do not try another unplanned source or execute the bytes.

## The other route: `go install`

On a machine that already has a Go toolchain:

```sh
go install github.com/heurema/goalrail/cmd/gr@latest
```

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

```sh
manifest="$(printf '%s' "${entry}" | jq -r .setup_manifest.name)"
curl --fail --location --proto '=https' --proto-redir '=https' --output "${manifest}" \
  "https://github.com/heurema/goalrail/releases/download/${version}/${manifest}"

gr setup plan --repo . \
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
plan_digest="sha256:$(sha256sum plan.json | cut -d' ' -f1)"
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

```sh
bundle_archive="$(printf '%s' "${entry}" | jq -r .setup_archive.name)"
curl --fail --location --proto '=https' --proto-redir '=https' --output "${bundle_archive}" \
  "https://github.com/heurema/goalrail/releases/download/${version}/${bundle_archive}"
sha256sum --ignore-missing -c checksums.txt

mkdir -p bundle && tar -xzf "${bundle_archive}" -C bundle

gr setup verify-plan --repo . --bundle bundle \
  --plan plan.json --authorization authorization.json \
  --release-metadata current-release.json
```

Verification returns `"verified":true` with the plan, authorization and manifest
digests it bound. Only then:

```sh
gr setup apply --repo . --bundle bundle \
  --plan plan.json --authorization authorization.json \
  --release-metadata current-release.json \
  --continuation-ref 'task:<bounded id of the request this interrupted>'
```

Apply performs only the enumerated mutations, captures the setup receipt and the
private recovery locator, runs the diagnosis, and returns to the original
request at the Intent Snapshot gate. Setup success confirms no intent and
authorizes no implementation.

The receipt's terminal status reflects the diagnosis taken by the binary that
applied it. A development build reports a version that corresponds to no
published bundle, so an apply performed by one records a failure even where every
mutation succeeded.

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

The planning runtime and compiler are established by inspection: the installed
bundle manifest supplies each component's path, recorded version and digest, and
those files are hashed. Nothing is executed to establish readiness, so a machine
with no Node runtime on its lookup path diagnoses exactly as one that has it.

Local hooks are advisory. A prepared workflow file is not activation evidence.
Managed project identity does not prove provider trust or protected admission.
Provider authentication, trust confirmation, workflow activation, required-check
configuration, and branch protection are separate owner-authorized external
actions.

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
gr setup rollback \
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
