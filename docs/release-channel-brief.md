# Release channel brief — owner discussion, 2026-07-30

Input for the next session's Context Pack. This is a discussion record, not a
confirmed intent: the release-channel change starts from its own Context Pack and
its own owner confirmation.

## Why now

`README.md` documents installation as `go install`, which requires Go on the
user's machine. The landing page being planned installs Goalrail by handing a
prompt to an agent, and that prompt should not require a Go toolchain. A release
channel is what closes the gap.

It also satisfies a revisit condition already on record. Confirmed non-goal NG-1
of `harness-init-v0` reads: "No network release check and no self-update of the
`gr` binary. There is no tag, no release pipeline, and no channel to query…
Revisit when a release channel exists." Creating the channel is what makes that
question askable — and it is item 3 of the harness-init brief, where the owner
decided the only version a user sees is Goalrail's own.

## Scope

One narrow slice: continuous integration, a release workflow producing binaries
for the platforms that build today, a version that comes from the tag rather than
a constant, and the README's install section updated to name the binaries.

Explicitly **not** in this slice, each for its own reason:

- **The update-availability check in `gr doctor`.** This slice creates the channel;
  querying it is a separate change with its own intent, now unblocked.
- **Windows.** See decision 2.
- **Signing, notarization, SBOM, Homebrew tap, self-update.** No demand yet, and
  each carries its own decision.

## Owner decisions, 2026-07-30

1. **A workflow of our own, not goreleaser.** Four static binaries, no CGO, no
   signing, no package managers — that is four `go build` invocations plus a
   checksum file, around sixty lines of YAML. goreleaser would add a
   configuration DSL and an external action for work that does not exist here.
   Revisit condition: a Homebrew tap or artifact signing becomes wanted, where
   goreleaser earns its keep.
2. **Windows is unsupported, with the reason recorded rather than the gap left
   silent.** `internal/boundedio` opens the escalation artifact with
   `O_NOFOLLOW|O_NONBLOCK` — flags Windows lacks — and those flags are how the
   promoted hygiene rule is actually implemented: `ambient-connect` requires that
   "a stale artifact that is oversized, non-regular, or a link outside the
   repository is cleared and noted without being followed, blocking on, or
   copied" (`openspec/specs/ambient-connect/spec.md`). A build-tagged
   stub would compile and silently drop a guarantee the specs make, which is the
   class of quiet breakage this product exists to remove. Supporting Windows means
   deciding how to keep that guarantee there; the codebase already has the
   platform-split pattern (`internal/evidence/file_lock_unix.go` and
   `file_lock_unsupported.go`). Revisit condition: a real user on Windows.
3. **This goes through the full OpenSpec cycle, kept to a narrow slice.** It is not
   a YAML file in `.github`: it changes what a version means, satisfies a recorded
   revisit condition, and creates the distribution boundary the product is
   installed through.

## Verified facts, so the next session does not re-derive them

Each was checked on 2026-07-30 against `main` at `6e85a5b`.

| Fact | How it was established |
|---|---|
| `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64` all build with `CGO_ENABLED=0`. | Cross-compiled each to `/dev/null`. |
| `windows/amd64` does not build: `syscall.O_NOFOLLOW` is undefined there. | Same, and the compiler names `internal/boundedio/bounded_file.go:27`. |
| The repository has no `.github` directory at all — no CI of any kind. | Directory listing. |
| GitHub Actions is enabled for the repository, all actions allowed. | `gh api repos/heurema/goalrail/actions/permissions`. |
| There are zero tags. | `git tag`. |
| `harness.Version` is the constant `"0.1.0"` in `internal/harness/doctor.go`. | Read. |
| `go install github.com/heurema/goalrail/cmd/gr@latest` works today and yields a binary reporting `version 0.1.0` with the canon digest. | Run against a clean module cache in a scratch GOPATH. |
| The repository is public and MIT-licensed. | `gh api`, and `LICENSE` merged in #36. |

## Tripwires the next session should carry into its Context Pack

- **The version has three sources and they disagree.** A tagged release should
  stamp the version through `-ldflags`; a binary from `go install module@v0.1.0`
  already carries it in build info; a `go build` from a checkout has neither. A
  cascade — ldflags, then `debug.ReadBuildInfo`, then a plain `devel` — is the
  honest shape. Reporting `0.1.0` from a constant after the second release would
  be a lie the tool tells about itself.
- **CI will silently skip the canon check.** `TestPinnedCLIAcceptsTheEmbeddedCanon`
  now probes the npm cache with `--offline` and skips loudly when the package is
  absent — a deliberate fix so an ordinary `go test ./...` never reaches the
  network. On a fresh CI runner the cache is empty, so the check that the embedded
  canon is one the stock CLI accepts would not run at all, and the release would
  be green without it. CI has to prime the cache explicitly, or the guarantee
  ships unverified.
- **Unsigned macOS binaries meet Gatekeeper.** A user who downloads one through a
  browser gets a warning that `go install` never produces. Not a blocker, but the
  install documentation should say it rather than let the user discover it.
- **Artifact names outlive this change.** Whatever scheme ships gets baked into
  the documentation, the landing page, and any install script. Proposal to confirm:
  `gr_<version>_<os>_<arch>.tar.gz` plus a single `checksums.txt`.

## Boundaries

- Creating the first tag and publishing a release are external actions. Each keeps
  its own owner gate; neither is granted by this brief or by confirming the intent
  that follows it.
- Nothing here authorizes touching the owner's working scaffold configuration, and
  the deferred branch `codex/dogfood-run-v0` stays untouched.

## Open questions for the intent

1. Does the first tag match the constant the code already carries (`v0.1.0`), or
   does the first published release start elsewhere?
2. Is the artifact naming scheme above confirmed, given it will be quoted in the
   documentation and the landing page?
3. Should CI run on pull requests as well as on the release tag? Running it on
   pull requests is what would have caught the Windows build failure before it
   became a release-time discovery.
