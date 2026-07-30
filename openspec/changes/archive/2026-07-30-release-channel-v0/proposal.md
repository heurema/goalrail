## Why

Installing Goalrail requires a Go toolchain, because `go install` is the only
route the project offers. The prompt a user hands to an agent inherits that
requirement, which is the gap a release channel closes. The repository also has
no automated check of any kind: the Windows build has been broken for as long as
`internal/boundedio` has used `O_NOFOLLOW`, and nothing would have said so before
a release tried to produce a binary. And the version the tool reports is a
constant, so the second release would report the first release's number unless
somebody remembered to edit a line.

Confirmed intent creates the channel: verification that runs on every change, a
tag that publishes a complete and verifiable set of artifacts, a version that
follows the build rather than a constant, and install documentation that works
without Go — including the agent-facing prompt this change exists for. It also
satisfies a recorded revisit condition: `harness-init-v0`'s NG-1 deferred the
update-availability question until a release channel existed. Creating the
channel is not querying it; that stays a separate change.

## What Changes

- Continuous integration runs on every pull request and every push to `main`:
  `go vet` and the full test suite on both operating systems the project ships
  binaries for, and a compile of every platform the documentation claims. What it
  promises is a visible failed check — this repository has no branch protection,
  and configuring one is its own decision.
- Pushing a `v*` tag publishes a release: four archives, one per supported
  platform, each carrying the `gr` binary and the licence, plus one `checksums.txt`
  covering them. The verification runs again on the tagged commit as a
  prerequisite of building, and publication happens only after every artifact
  exists and every check passed.
- **BREAKING** for anything that read `harness.Version` as a compile-time
  constant: the version becomes what the build carries, reported verbatim. A
  released binary reports its tag, `v` and all; a build between tags reports the
  next patch version as a pseudo-version naming the commit; a modified tree says
  so; a build with no version control reports the toolchain's development marker.
  Nothing normalizes, strips, or rewrites the string on the way out.
- The release refuses to publish an artifact whose stamped version is not exactly
  the tag. The stamp is read out of each artifact without executing it, which is
  what makes the check possible for platforms the runner cannot run.
- Artifacts are built without leaving anything untracked in the work tree,
  because an artifact inside the checkout marks every later build as modified and
  would stamp three of the four binaries `+dirty`.
- The canon check stops skipping into a green result: the release makes the pinned
  package available, and the run fails whenever the check did not execute —
  skipped or never run at all. Nothing inside the harness changes: the enforcement
  lives in the workflow, and a contributor without the runtime still gets a loud
  local skip. Where the workflow gets the package from is a design decision, not a
  promoted rule.
- The README's install documentation is rewritten around the binaries: which
  archive per platform, the verification command that works when only one archive
  was downloaded, a macOS note that states what is actually true, and Windows
  named as unsupported with its reason. `go install` remains documented, with the
  note that after the first tag it installs the release rather than the branch tip.
- The agent-facing prompt carries the three steps an agent cannot guess: how to
  find the current release without knowing its version, where to put the binary so
  the attachment it installs keeps working, and the checksum command that succeeds
  on a partial download. The prompt permits that one write outside the repository
  explicitly.
- The promoted rule that `gr` performs no network lookup for a newer release
  survives unchanged, but stops resting on "there is no channel" and rests on the
  boundary instead.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Verification runs on every pull request and push to `main`, over both operating systems and every claimed platform. | OUT-1, SIG-1 | NG-2, NG-7 |
| What the check promises is visibility, not a blocked merge. | OUT-1 | NG-7 |
| A `v*` tag publishes four archives and one checksums file, each archive carrying the binary and the licence. | OUT-2, SIG-3 | NG-3, NG-4 |
| The checksums file keeps one name across releases, because discovery without a known version depends on exactly one asset whose address does not move. | OUT-2, OUT-7, SIG-6 | NG-4 |
| Publication happens only after every build and every check succeeded, through tooling that stages a draft and publishes last. | OUT-2, SIG-3 | NG-3, NG-7 |
| Artifacts are produced without dirtying the tree they are built from. | OUT-2, OUT-3, SIG-2 | NG-6 |
| The reported version is what the build carries, verbatim, in every build situation. | OUT-3, SIG-2 | NG-5 |
| The release rejects an artifact whose stamp is not exactly the tag, read without executing it. | OUT-2, OUT-3, SIG-2 | NG-5 |
| The same string is the tag, the reported version, and the version inside every archive name. | OUT-3, SIG-5 | NG-5 |
| The canon check is primed and its skip fails the run, with no new switch inside the harness. | OUT-4, SIG-4 | NG-6, NG-7 |
| Install documentation names the archives, the working verification command, the macOS truth, and the unsupported platform. | OUT-5, SIG-3, SIG-5 | NG-2, NG-4 |
| `go install` stays documented, with what `@latest` resolves to after the first tag. | OUT-5, SIG-6 | NG-1 |
| The agent prompt and the status section move to the binary route and carry discovery, placement, and verification. | OUT-7, SIG-6 | NG-4, NG-9 |
| The prohibition on a release lookup survives with its reason restated, and the archived condition is recorded as satisfied without editing the archive. | OUT-6, SIG-7 | NG-1 |

No proposed change lies outside this table.

## Capabilities

### New Capabilities

- `release-channel`: what a change must pass before it can become a release, what
  a tag publishes, how a binary's version is established and checked against the
  tag it claims, and what the documented install route must carry for a user or
  an agent that has no Go toolchain.

### Modified Capabilities

- `harness-update`: the prohibition on querying a release channel stops being
  justified by the channel's absence and is justified by the boundary — the
  channel now exists, and asking it a question is a separate change.

## Impact

- New `.github/workflows/**`: one verification workflow and one release workflow.
  The repository has no `.github` directory today.
- `internal/harness`: `Version` stops being a constant and becomes what the build
  carries. `internal/harness/doctor_test.go` compares the diagnosis against that
  value and continues to hold; `cmd/gr/harness_test.go` reads the version through
  the command surface and continues to hold.
- `README.md`: the install section, the agent-facing prompt, and the status
  section.
- `openspec/specs/harness-update/spec.md`: one requirement's justification, after
  archival.
- No new module dependency, service, credential, or stored secret. No change to
  the canon, the overlay, the escalation loop, the diagnosis, or how currency is
  judged.
- Nothing in the harness learns to reach the network.

## Non-Goals

- No update-availability check and no self-update; creating the channel is not
  querying it.
- No Windows binary and no build-tagged stub for the platform-specific flags.
- No goreleaser or other third-party release framework.
- No signing, notarization, SBOM, build provenance, Homebrew tap, install script,
  package manager, or container image — and no post-link processing of a
  published binary.
- No change to how a repository's state is judged: digests decide, and the
  version remains information about the binary.
- No change to how a registration records the executable it names.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, tag, release publication, or other external effect. Each remains a
  separate owner gate, recorded before the action.
