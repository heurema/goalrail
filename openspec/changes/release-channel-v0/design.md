## Context

The repository has no `.github` directory, no tags, and no release pipeline. Its
whole verification contract is the Go toolchain — `go vet ./...` clean and
`go test ./...` green across 13 packages in about 12 seconds, with no module
dependency to download. The version is the constant `"0.1.0"` in
`internal/harness/doctor.go`, reported by `gr version` and by `gr doctor`.

Four platforms build with `CGO_ENABLED=0`; `windows/amd64` does not, because
`internal/boundedio` uses `syscall.O_NOFOLLOW`, which is how a promoted hygiene
guarantee is implemented.

Two facts shape almost every decision below, both verified in this change's
Context Pack:

- Go already stamps a version into every build from version control (CTX-4), and
  every stamped shape carries a leading `v`. A shallow single-ref checkout at a
  tag still stamps that tag.
- Any untracked file in the work tree — including a build's own output — marks
  the build as modified, so artifacts written into the checkout stamp `+dirty`
  onto every binary built after them (CTX-16).

## Goals / Non-Goals

**Goals:**

- One verification workflow and one release workflow, both readable end to end
  without a configuration DSL.
- A version that follows the build with no hand-maintained constant and no
  transformation between what the build carries and what the tool prints.
- A release that cannot publish a mislabelled, incomplete, or unverified set.
- Install documentation, including the agent prompt, that works with no Go
  toolchain.

**Non-Goals:**

- goreleaser or any release framework; signing, notarization, SBOM, provenance,
  a tap, an install script, a container image; a Windows binary or a stub for it.
- Any network lookup, update check, or self-update inside `gr`.
- Any new switch, environment contract, or surface inside the harness for the
  release's benefit.
- Branch protection or required status checks: a repository settings change is a
  separate decision with its own owner gate.

## Decisions

### D1 — Two workflows, and the release re-runs verification itself

`ci.yml` triggers on `pull_request` and on `push` to `main`. `release.yml`
triggers on `push` of tags matching `v*`.

A tag event carries no dependency on any earlier run: a tag can point at a commit
that never saw a pull request. So the release workflow runs the full verification
on the tagged commit as a `needs:` prerequisite of building, rather than trusting
that CI ran somewhere. The duplication is about twelve seconds of test time and
is the only thing that makes "nothing is published unless the checks passed" true.

*Rejected:* `workflow_run` chaining, which would introduce ordering the release
cannot verify.

### D2 — The version is build information, read once, reported verbatim

`internal/harness.Version` stops being a `const` and becomes a package-level
`var` initialized from `debug.ReadBuildInfo()`:

- build information present and `Main.Version` non-empty → that string, verbatim;
- otherwise → `"unknown"`.

Every shape Go produces passes through untouched: `v0.1.0`, `v0.1.1-0.<ts>-<sha>`,
`v0.0.0-<ts>-<sha>`, either with `+dirty`, and `(devel)`. Keeping it a `var` with
the same name and type means the three existing call sites and both tests that
reference it compile unchanged.

*Rejected:* `-ldflags -X` injection. Verified unnecessary — a checkout at the tag,
including the shallow single-ref checkout `actions/checkout` performs, already
stamps the tag — and it would create a second source that can disagree with the
first. *Rejected:* stripping the leading `v`, normalizing `(devel)`, or mapping a
pseudo-version to something friendlier: each is a transformation that makes the
printed string something other than what the binary carries, and the release gate
then has to compare two dialects.

Note for the implementer: a `go test` binary reports `(devel)` in every Git state
(CTX-17). The unit test therefore exercises a resolver over injected
`debug.BuildInfo` values; it cannot observe real stamped shapes from inside a test.

### D3 — Artifacts are built outside the work tree, and the tree is asserted clean

Each `go build` writes to `"$RUNNER_TEMP"/dist`, and the archives and checksums
file are assembled there. Before the first build the job asserts
`git status --porcelain` is empty and fails loudly if it is not.

Writing outside the tree is necessary but not sufficient on its own: what dirties
the stamp is anything untracked *inside* the tree, so the assertion is what
actually holds the guarantee (CTX-16 verified both halves).

*Rejected:* adding `dist/` to `.gitignore`. It would work, but it makes the
repository carry a permanent accommodation for a directory that only exists on a
runner, and it hides exactly the class of accident the assertion catches.

### D4 — The publish gate reads stamps, it does not run binaries

For each of the four artifacts the release runs `go version -m <artifact>`, takes
the `mod` line's version field, and requires string equality with
`${GITHUB_REF_NAME}`. Verified: this reads a `linux/arm64` binary from a `darwin`
host without executing it (CTX-17), which is what makes a single builder possible.

Equality is exact. A tree dirtied by anything stamps `v0.1.0+dirty`, which a
prefix or substring match would happily publish.

The checkout stays at `actions/checkout`'s default single-ref, no-tags fetch.
Verified: a tag-following fetch that pulls a second tag on the same commit stamps
the higher tag (CTX-17), so fetching more tags would make the stamp depend on
what else happens to point at that commit.

Optionally the runner-native binary is executed once as a smoke check — that is a
convenience, not the gate.

### D5 — The canon check is enforced from the workflow, through the test's own JSON

`go test` exits 0 and prints `ok` when `TestPinnedCLIAcceptsTheEmbeddedCanon`
skips; `go test -json` marks that test `skip` in the same run (CTX-23). The
release therefore runs that test with `-json` and requires a test-level
`"Action":"pass"` whose `"Test"` is exactly `TestPinnedCLIAcceptsTheEmbeddedCanon`.
It fails on `skip`, on `fail`, and on absence.

The gate is stated positively on purpose. A gate that only looks for a skip passes
when the stream carries no verdict for that test at all — a rename, a build tag, a
mistyped `-run` pattern — which is the same green-by-omission failure the whole
requirement exists to remove. The run selects the test with an anchored pattern so
the gate and the run cannot disagree about which test was meant.

Nothing in the harness changes: no environment variable, no build tag, no new
flag. A contributor without Node still gets today's loud local skip, which is the
behaviour a deliberate earlier change installed.

The package to prime is derived from the binary rather than duplicated in YAML,
as an implementation choice rather than a promoted rule: the job runs
`go run ./cmd/gr doctor -json -repo "$(mktemp -d)"` and extracts the package from
the `invocation` field of the report. That field carries `PinnedNewChange`, which
is `PinnedInvocation` plus ` new change <name> --schema goalrail-intent`, so the
package is the token after `npx --yes` rather than the whole field — comparing the
field to `PinnedInvocation` would never match. Two further details, both verified:
`gr init` has no JSON flag, so the diagnosis is the machine-readable surface; and
the diagnosis
exits non-zero against an empty scratch repository, by design, while still
emitting the full report on standard output. The step therefore ignores the exit
status and fails instead when the extraction finds no package — the failure that
actually matters, since an empty result would prime nothing and silently return
the run to a skipped check.

*Rejected:* hard-coding `@fission-ai/openspec@1.6.0` in YAML. It would drift from
`internal/harness/canon.go` on the next pin bump; the drift would be loud rather
than silent, because the primed cache would miss and the skip gate would fail the
job — but a second copy of a version that exists to be authoritative in one place
is what the harness's own digest discipline exists to avoid. *Rejected:* grepping
the constant out of the Go source, which reintroduces a parser for a value the
product already reports.

### D6 — Publication uses the preinstalled `gh`, whose default is already all-or-nothing

`gh release create "$GITHUB_REF_NAME" <artifacts...>` creates the release as a
draft, uploads every asset, and publishes only afterwards — its own documentation
states this, and `gh` is preinstalled on the runners (CTX-22). A failed upload
therefore leaves an unpublished draft, not a partial public release.

The publish job re-runs idempotently over a leftover draft from a failed attempt:
it reads the release's state for the tag and deletes it only when it is still a
draft, aborting when a published release already exists. The refusal is the
workflow's own, not the platform's — release immutability is a repository setting,
and this repository has not enabled it, so an unconditional delete would remove a
good published release and its assets. The policy stands either way: a published
release is not repaired in place, and a defective one is superseded by a new tag.

The publishing step runs with the job token in its environment. `contents: write`
on the job sets what the token may do; `gh` reads the credential itself and fails
without it. It is the platform's own token, not a stored secret, so NG-7 holds.

*Rejected:* raw REST calls, which would have to reimplement the draft-then-publish
sequence, and third-party release actions, which NG-3 excludes.

### D7 — Permissions are declared per workflow and per job

`ci.yml` declares `permissions: contents: read`. `release.yml` declares
`contents: read` at the top and raises `contents: write` only on the job that
publishes. The repository's default workflow permission is currently `write`
(CTX-24), which is exactly why the workflows declare their own rather than
inherit it.

Building and publishing are the same job. That keeps the artifacts on one runner
— no upload-and-download hand-off, nothing to lose between jobs — at the cost of
that job holding write while it builds. The alternative trade is worse: a
separate publishing job would need the artifacts passed through the platform's
artifact store, which is more moving parts around the one act that is externally
visible.

Actions used: `actions/checkout` and `actions/setup-go`, both first-party, pinned
to their major tags. `setup-go` resolves the toolchain with
`go-version-file: go.mod` so the workflow never carries a version string that can
drift from `go.mod` (CTX-13).

### D8 — Runners: Linux for everything, macOS for the tests it can actually prove

Cross-compilation of all four platforms and the whole test suite run on
`ubuntu-latest`. The suite additionally runs on `macos-latest`, because half the
published platforms are darwin and the file-locking and bounded-IO paths are
platform-specific — compiling for darwin proves nothing about running there.
Public-repository runner minutes are free (CTX-11), so the second job costs wall
clock only.

### D9 — Archive layout: flat, binary plus licence

`gr_<tag>_<os>_<arch>.tar.gz` contains `gr` and `LICENSE` at the archive root, and
`checksums.txt` lists the four archives with SHA-256 digests. `<tag>` is the tag
verbatim, `v` included, so the string in the filename is the string the binary
reports and the string the gate compares.

Flat rather than a versioned subdirectory: the documented extraction is
`tar -xzf <archive> gr`, which takes only the binary and cannot overwrite a
`LICENSE` in the working directory, while the licence still travels inside the
archive as the MIT terms require.

### D10 — What the install documentation must carry

- A platform table mapping `uname -s`/`uname -m` to the archive name.
- Verification with `shasum -a 256 --ignore-missing -c checksums.txt` on macOS and
  `sha256sum --ignore-missing -c checksums.txt` on Linux. Verified: without
  `--ignore-missing` the command exits 1 when the file lists archives the user did
  not download (CTX-20).
- Discovery without knowing the version: `checksums.txt` is the only asset whose
  name does not change between releases, so
  `https://github.com/heurema/goalrail/releases/latest/download/checksums.txt`
  is the stable entry point; it names the four current archives, and each of those
  names then resolves under the same `releases/latest/download/` prefix. Verified
  live against a public repository (CTX-21).
- A durable location — `~/.local/bin/gr` — before `gr init` runs. A registration
  records the symlink-resolved absolute path of the executable that wrote it and
  `gr doctor` fails when nothing runnable remains there (CTX-19), so a binary left
  in a temporary directory produces an attachment that breaks on the next cleanup.
  The agent prompt permits this one write outside the repository explicitly.
- macOS: the command-line download carries no quarantine flag and produces no
  warning; a browser download extracted in Finder does. The remedy is
  `xattr -d com.apple.quarantine gr`, or System Settings → Privacy & Security →
  Open Anyway. Not `Control`-click → Open, which stopped overriding the gatekeeper
  in macOS Sequoia. The wording is "not notarized": the darwin/arm64 binary is
  ad-hoc signed by the Go linker (CTX-18).
- Windows: unsupported, with the reason and the revisit condition.
- `go install` as the second route, noting that after the first tag `@latest`
  resolves to that tag rather than the branch tip.

### D11 — Nothing post-processes a published binary

No `strip`, no `upx`, no re-signing, no rewriting. The Go linker's ad-hoc
signature on darwin/arm64 is load-bearing: a modified binary is killed on Apple
Silicon rather than merely warned about (CTX-18). What the linker produced is what
ships.

## Correlation and Evidence

The tag is the correlating identity across the whole slice: it names the release,
it is stamped into every artifact, it appears in every artifact name, and it is
what the publish gate compares against. Nothing else needs to agree with anything
else.

Evidence of a release is the release itself — its asset list, the checksums file,
and the stamps inside the artifacts — plus the workflow run that produced it. No
new record, store, or telemetry is introduced; nothing about a release enters a
repository's state or the local state root.

Failure handling is uniform: any failed step publishes nothing and leaves the
previous release untouched. A published release is never repaired in place.

## Measurement and Stop Conditions

Pass signals for this slice:

- A pull request shows a check result, and a deliberately broken platform build
  fails it.
- A tag produces four archives plus `checksums.txt`, and a downloaded archive
  verifies against that file with the documented command.
- Each artifact's stamp equals the tag; a seeded mismatch stops the release.
- The canon check reports as executed in the release run; removing the priming
  step fails the run.
- An agent handed the prompt on a machine without Go reaches a healthy
  `gr doctor`.

Stop and reshape if: the verification takes long enough that contributors start
skipping it; the release workflow needs a stored secret (it must not); or the
version resolver needs a special case for a build situation the Context Pack did
not record — that would mean the single-source assumption is wrong, and the
resolver should be revisited before more rules accumulate inside it.

## Risks / Trade-offs

- **The release re-runs verification.** Duplicated work in exchange for a
  guarantee a tag event cannot otherwise carry. Accepted; the suite is seconds.
- **No branch protection.** A red check does not stop a merge. Accepted for this
  slice: the workspace contract already forbids direct pushes to `main`, and
  turning on protection is a repository settings change with its own gate.
- **The documentation lands before the first release exists.** For the length of
  one release run the README names archives nobody can fetch. Accepted by
  confirmed intent (AMB-4); the mitigation is to tag immediately after merge.
- **`(devel)` and pseudo-versions are user-visible.** A user who builds from a
  checkout sees a long string rather than a friendly number. Accepted: it is the
  truth about their binary, and the alternative is a version that lies.
- **`macos-latest` and `ubuntu-latest` move under us.** A runner image change can
  break a run that has nothing to do with the change under test. Accepted;
  pinning images creates its own staleness, and the failure is loud.

## Rollback

Delete the two workflow files: continuous integration and releases stop, and
nothing else in the product changes. Published releases remain downloadable and
are unaffected.

The version resolver is independent of the workflows: if it were reverted to a
constant, `gr` keeps working and only the reported string becomes stale again.
The documentation change is likewise independent — reverting it restores the
`go install`-only instructions.

Nothing in this slice writes to a repository's state, the local state root, or a
scaffold's configuration, so there is nothing to clean up on rollback.

## Open Questions

None that can materially change implementation. The four questions this change
opened — the first tag's version, the artifact naming, the trigger set, and
whether the documentation may precede the first release — were answered in
confirmed intent version 2.
