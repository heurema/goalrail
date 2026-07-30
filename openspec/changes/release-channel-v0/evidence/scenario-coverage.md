# Scenario coverage — release-channel-v0

Every scenario in the two delta specs, and what exercises it. Three kinds of
evidence appear here, kept apart on purpose:

- **test** — a Go test in this repository.
- **local run** — a command run in this session against a scratch clone, a fake
  home, or a temporary directory, recorded with what it produced.
- **after the first tag** — cannot be observed until a release exists. Listed so
  the gap is visible rather than implied; task group 11 performs them.

The local runs used a scratch clone of this worktree, tagged `v0.1.0` locally and
never pushed. Nothing here touched the owner's scaffold configuration.

## release-channel

### Every change is verified automatically before it can become a release

| Scenario | Evidence |
|---|---|
| A supported platform stops compiling | local run: `CGO_ENABLED=0 GOOS=… GOARCH=… go build` over the four published platforms succeeds and `windows/amd64` fails at `internal/boundedio/bounded_file.go:27` — the failure the matrix would surface on a pull request. The workflow runs the same loop. |
| A pull request is opened | after the first pull request carrying `.github/workflows/ci.yml`; the trigger is `pull_request` with no branch filter. |
| The default branch moves | after the first push to `main` carrying the workflow; the trigger is `push` on `main`. |
| A claimed platform is not compiled | local run: the platform list in `ci.yml`, in `release.yml`, and in the README install table is the same four. |

### A tag publishes one complete, verifiable set of artifacts

| Scenario | Evidence |
|---|---|
| A tag is pushed | local run: the release job's build, packaging, and checksum steps executed against the tagged scratch clone produced `gr_v0.1.0_{darwin,linux}_{arm64,amd64}.tar.gz` plus `checksums.txt`, each archive holding `gr` and `LICENSE`. |
| One platform fails to build | local run: `set -euo pipefail` in every step plus `needs: [test, canon]` on the publishing job; a failing build exits the step before any `gh release create` runs. Observable in full only after a real failed release. |
| A downloader verifies what they fetched | local run: `shasum -a 256 --ignore-missing -c checksums.txt` with one archive of four present exits `0`; without `--ignore-missing` it exits `1`. |
| The current release must be found without knowing its version | local run: `curl -sI https://github.com/cli/cli/releases/latest/download/checksums.txt` returns `302` to the versioned asset (CTX-21), and the release publishes `checksums.txt` under that fixed name. Confirmed against this repository's own release after the first tag. |
| The stable name would carry the version | local run: the packaging step writes `checksums.txt` with no version in the name; the archives carry the tag. |

### A binary reports the version it was built from, verbatim

| Scenario | Evidence |
|---|---|
| A released binary reports itself | local run: built at the local tag `v0.1.0` in the scratch clone, `gr version` reports `v0.1.0`. Test: `TestVersionIsReportedExactlyAsTheBuildCarriesIt`. |
| A binary is built between releases | local run: one commit after that tag, `gr version` reports `v0.1.1-0.20260730163543-9f1d30e6ec5d`. |
| A second release follows a first | test: the resolver reads the build rather than a constant, so no version string exists to forget; `TestVersionIsReportedExactlyAsTheBuildCarriesIt` pins each shape. Fully observable at the second real release. |
| The build carries no version at all | test: `TestVersionSaysUnknownWhenTheBuildCarriesNone`, over absent build information, a nil reader result, and an empty version string. |

### A release refuses an artifact whose stamped version is not its tag

| Scenario | Evidence |
|---|---|
| An artifact carries a different version | local run: an untracked file added to the tagged scratch clone made the next build stamp `v0.1.0+dirty`, and the gate's comparison rejected it. |
| An artifact is built for another platform | local run: `go version -m` read the stamp out of the `linux/amd64` and `linux/arm64` artifacts on a darwin host without executing them. |
| Build output lands inside the checkout | local run: the first build into `./dist` stamped `v0.1.0` and every later build stamped `v0.1.0+dirty`, including one written outside the tree while the earlier artifact remained inside it. The workflow builds into `$RUNNER_TEMP` and refuses to start from a tree `git status --porcelain` reports as non-empty. |

### The canon check runs before a release instead of skipping into a green result

| Scenario | Evidence |
|---|---|
| The check is skipped during a release | local run: `go test -json -short -run '^TestPinnedCLIAcceptsTheEmbeddedCanon$'` exits `0` while the gate counts zero passes and fails. |
| The check does not run at all | local run: the same gate over a pattern matching no test — `go test` exits `0`, the gate counts zero passes and fails. |
| The check runs during a release | local run: with the npm cache primed, the gate counts exactly one pass. |
| A contributor has no runtime | test: `TestPinnedCLIAcceptsTheEmbeddedCanon` skips loudly under `-short`, with no `npx`, or with an unprimed cache; the suite passes. Unchanged by this change. |
| The pin moves | local run: the priming step derives `@fission-ai/openspec@1.6.0` from `gr doctor -json`, so no second copy exists to drift; a mismatch would leave the cache unprimed, the check skipped, and the gate red. |

### The documented install route works without a Go toolchain

| Scenario | Evidence |
|---|---|
| A user without Go installs | local run: with `PATH=/usr/bin:/bin:/usr/sbin:/sbin` and no `go` resolvable, the README's own commands discovered the archive name from `checksums.txt`, verified it, extracted `gr` into `~/.local/bin`, and `gr version` reported `v0.1.0`. |
| Only one archive was downloaded | local run: as above, one archive of four present, `--ignore-missing` exits `0`. |
| The documentation names an artifact | local run: the four names in the README install table match the four the packaging step produced. Confirmed against the published asset list after the first tag. |
| A user downloads through a browser | after the first tag: the browser path stamps the quarantine attribute that the command-line path does not (CTX-18), and the documented remedy is `xattr -d com.apple.quarantine`. The claim in the README is scoped to that path precisely because the command-line route was verified not to produce it. |
| A remedy no longer works | local run: the README documents `xattr -d` and the System Settings route, and explicitly does not document Control-click, which stopped overriding Gatekeeper in macOS Sequoia (CTX-18). |

### The agent-facing prompt carries what an agent cannot guess

| Scenario | Evidence |
|---|---|
| An agent installs from the prompt | after the first tag, and behind the provider-run gate (task 10.6). The human walkthrough of the same steps, with no Go on the path, is the local run above; what it cannot exercise is an agent following the prompt. |
| The download location is cleaned afterwards | local run: initializing from `~/.local/bin/gr` and then removing the download directory leaves `gr doctor` reporting `harness: working` and exiting `0`; initializing from a temporary copy and then removing it makes the diagnosis report "the registered Goalrail executable is missing at …". Both states observed. |
| The prompt forbids the placement it requires | local run: the rewritten prompt names `~/.local/bin` as expected and permits it explicitly, restricting only other writes outside the repository. |

## harness-update

| Scenario | Evidence |
|---|---|
| The user reads the command's help | unchanged behaviour; the promoted scenario is carried through the `MODIFIED` delta untouched, and `gr update -h` states that the binary is not updated. |
| A release lookup would be attempted | unchanged behaviour: no `gr` command reaches the network. |
| A release channel exists | local run: the change adds a release channel and no lookup; nothing in `internal/harness` or `cmd/gr` gained a network call, and the requirement's reason now rests on the boundary rather than on the channel's absence. |
