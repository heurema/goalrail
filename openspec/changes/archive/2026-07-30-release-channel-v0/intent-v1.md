# Intent Snapshot

- **Intent ID:** intent-release-channel-v0
- **Version:** 1
- **Status:** candidate
- **Owner:** repository owner
- **Context Pack:** context-release-channel-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 (owner):** The README documents installation as `go install`, which requires a Go toolchain on the user's machine. The landing page being planned installs Goalrail by handing a prompt to an agent, and that prompt should not require Go. A release channel is what closes the gap.
- **SE-2 (owner):** The slice is narrow: continuous integration, a release workflow producing binaries for the platforms that build today, a version that comes from the tag rather than a constant, and the README's install section updated to name the binaries.
- **SE-3 (owner):** A workflow of our own, not goreleaser. Four static binaries, no CGO, no signing, no package managers is four `go build` invocations plus a checksum file; goreleaser would add a configuration DSL and an external action for work that does not exist here. Revisit when a Homebrew tap or artifact signing becomes wanted.
- **SE-4 (owner):** Windows is unsupported, with the reason recorded rather than the gap left silent. Supporting it means deciding how to keep the hygiene guarantee its missing flags implement, not adding a matrix entry. Revisit on a real user on Windows.
- **SE-5 (owner):** This goes through the full OpenSpec cycle: it changes what a version means, satisfies a recorded revisit condition, and creates the distribution boundary the product is installed through.
- **SE-6 (owner):** Explicitly outside this slice, each for its own reason: the update-availability check in `gr doctor`, because this slice creates the channel and querying it is a separate change; Windows; and signing, notarization, SBOM, a Homebrew tap, and self-update, which have no demand yet and each carry their own decision.
- **SE-7 (owner):** Creating the first tag and publishing a release are external actions, each keeping its own owner gate; neither is granted by the brief or by confirming this intent. Nothing here authorizes touching the owner's working scaffold configuration, and the deferred branch `codex/dogfood-run-v0` stays untouched.
- **SE-8 (owner):** A user who downloads an unsigned macOS binary through a browser meets a Gatekeeper warning that `go install` never produces. Not a blocker, but the install documentation should say it rather than let the user discover it (CTX-15).
- **SE-9 (repository):** The repository has no continuous integration of any kind, and its whole verification contract is the Go toolchain: `go vet ./...` clean, `go test ./...` green across 13 packages in about 12 seconds, with no dependency to download (CTX-1, CTX-7).
- **SE-10 (repository):** Four platforms build with `CGO_ENABLED=0` and `windows/amd64` does not, because `syscall.O_NOFOLLOW` is undefined there — and those flags are how a promoted hygiene guarantee is implemented, so a build-tagged stub would compile and drop the guarantee silently (CTX-5, CTX-6).
- **SE-11 (repository):** The version is the constant `"0.1.0"`, reported by `gr version` and `gr doctor`, and no verdict about a repository derives from it (CTX-3).
- **SE-12 (repository, verified this session):** Go already stamps a version into every build. A clean checkout at tag `v0.1.0` yields `v0.1.0`, even from a shallow clone of that tag; one commit later yields `v0.1.1-0.<timestamp>-<commit>`; with no tags at all, `v0.0.0-<timestamp>-<commit>`; a dirty tree adds `+dirty`; only a build with VCS stamping off yields `(devel)`. This corrects the brief's assumption that a checkout build carries no version (CTX-4).
- **SE-13 (repository):** The one development-side check that the stock CLI accepts the embedded canon skips — never fails — when the pinned package is not in the npm cache, because it probes with `--offline` so an ordinary `go test` never reaches the network. On a fresh runner the cache is empty, so the check would not run and the release would be green without it (CTX-8).
- **SE-14 (repository):** A promoted requirement forbids `gr update` from querying a release channel and justifies the prohibition by the channel's absence; the archived `harness-init-v0` records "Revisit when a release channel exists" as the condition this change satisfies (CTX-9, CTX-10).
- **SE-15 (repository):** With zero tags, `go install …@latest` installs the branch tip; after the first tag it installs that tag. The README documents that command as the only route, repeats it inside the prompt handed to an agent, and states that no binaries are published (CTX-2, CTX-12).
- **SE-16 (repository):** The repository is public, MIT-licensed, and has Actions enabled with all actions allowed; the licence requires its notice to accompany a distributed copy, and `go.mod` declares the toolchain the workflow must resolve (CTX-11, CTX-12, CTX-13).

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Every pull request and every push to `main` is checked automatically: the project's own verification — vet and the full test suite — runs on both operating systems it ships binaries for, and every platform the documentation claims is compiled. A change that breaks a supported platform's build stops being a release-time discovery and becomes a failed check before merge. | Run the same commands the workflow runs locally and confirm they reproduce today's green baseline and the Windows failure; then read the check result on the first pull request that carries the workflow, which is where the guarantee is actually observable. | SE-2, SE-9, SE-10, CTX-1, CTX-5, CTX-7 |
| OUT-2 | Pushing a version tag publishes a release: one archive per supported platform, each carrying the binary and the licence, and one checksums file covering them all. Nothing is published unless every platform built and the checks passed, so a partial release cannot exist. | Read the workflow's dependency order and, on the first real tag, download the published archives, verify them against the checksums file, and run the extracted binary. | SE-2, SE-3, SE-16, CTX-5, CTX-11, CTX-12 |
| OUT-3 | The version a binary reports is the version it was actually built from, not a constant somebody has to remember to bump. A released binary reports its tag; a binary built from a checkout between tags reports a version that says so, naming the commit; a build with no version control behind it says plainly that it is a development build. A second release never reports the first release's number. | Build the binary in each situation — at a tag, after a tag, from a dirty tree, and with version stamping off — and read what `gr version` prints in each. | SE-2, SE-11, SE-12, CTX-3, CTX-4 |
| OUT-4 | The check that the embedded canon is one the stock CLI accepts actually runs before a release, instead of skipping into a green result. The workflow makes the pinned package available and treats a skipped check as a failure, so a canon can never ship unverified — while a contributor without the runtime still gets a loud skip rather than a broken build. | Read the workflow log for the line that proves the check executed; remove the priming step in a scratch run and confirm the job fails instead of passing quietly; run the suite locally without the runtime and confirm it still skips rather than fails. | SE-13, CTX-8 |
| OUT-5 | Installation stops requiring a Go toolchain and the documentation says exactly how: which archive to download for which platform, how to verify it against the checksums file, that macOS will warn about an unsigned binary and what to do about it, and that Windows is not supported together with the reason. `go install` remains documented as the other route, with the note that after the first tag it installs the release rather than the branch tip. | Follow the README's own download instructions on a machine with no Go installed, verify the checksum, run the binary, and confirm every artifact name the documentation quotes is one the release actually contains. | SE-1, SE-4, SE-8, SE-15, CTX-2, CTX-12, CTX-15 |
| OUT-6 | The promoted requirements stay true the day the channel exists. The rule that `gr` performs no network lookup for a newer release survives exactly as it is, but stops resting on "there is no channel"; the recorded revisit condition is marked satisfied, pointing at the separate change where querying the channel is decided. | Read the updated requirement and confirm the prohibition is unchanged while its stated reason is the deliberate boundary rather than the absence of a channel. | SE-6, SE-14, CTX-9, CTX-10 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No update-availability check and no self-update. No `gr` command queries the channel, resolves a latest version, or replaces the binary. This change creates the channel; asking it a question is the separate change the revisit condition now unblocks. | SE-6, SE-14, CTX-9, CTX-10 |
| NG-2 | No Windows binary and no build-tagged stub for the platform-specific flags. The gap is recorded with its reason and its revisit condition — a real user on Windows — rather than closed by code that compiles while dropping a promoted guarantee. | SE-4, SE-10, CTX-5, CTX-6 |
| NG-3 | No goreleaser and no third-party release framework. Revisit when a Homebrew tap or artifact signing becomes wanted, which is where such a tool earns its keep. | SE-3 |
| NG-4 | No signing, notarization, SBOM, build provenance, Homebrew tap, install script, package manager, or container image. The macOS warning is documented, not engineered away. | SE-6, SE-8 |
| NG-5 | No change to how a repository's state is judged. Digests still decide whether an overlay is current; the version stays information about the binary and no repository verdict derives from it. | SE-11, CTX-3 |
| NG-6 | No change to the harness itself: not the embedded canon, the overlay, the escalation loop, the diagnosis, or any runtime behaviour beyond the version string the tool reports. | SE-2, SE-5 |
| NG-7 | No new module dependency, service, dashboard, or credential. The build stays hermetic and dependency-free, and the workflow uses the platform's own credentials rather than a stored secret. | SE-9, CTX-7, CTX-11 |
| NG-8 | This snapshot authorizes no implementation, commit, push, pull request, merge, tag, release publication, or other external effect. Creating the first tag and publishing the first release each keep their own owner gate, recorded before the action. The owner's working scaffold configuration and the deferred branch `codex/dogfood-run-v0` stay untouched. | SE-7, CTX-14 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A platform the documentation claims cannot break silently. | The workflow compiles exactly the four platforms the README names, and the same commands run locally reproduce four successes and the `windows/amd64` failure, which stays outside the matrix by decision. | SE-10, CTX-5 |
| SIG-2 | The reported version follows the build, in every situation the project can be built in. | A test asserts the resolved version for each case — at a tag, after a tag, dirty tree, no version control — against the behaviour recorded in the Context Pack; and the release refuses to publish if the built binary does not report exactly the tag it was built from. | SE-12, CTX-4 |
| SIG-3 | A release is all or nothing, and verifiable by whoever downloads it. | Publishing depends on every platform's build and on the checks passing; the published set is four archives plus one checksums file, and verifying a downloaded archive against that file succeeds. | SE-2, SE-16, CTX-5 |
| SIG-4 | The canon guarantee is never green by omission. | The workflow log shows the pinned-CLI check ran; with the priming step removed the job fails rather than passing; locally without the runtime the same test still skips loudly. | SE-13, CTX-8 |
| SIG-5 | The documentation and the release agree on every name. | Each artifact name quoted in the README matches a name the workflow produces, checked against the first release's actual asset list before the download instructions are treated as true. | SE-1, SE-15, CTX-12 |
| SIG-6 | Installing without Go works, and the older route keeps working with its meaning stated. | A download-and-run walkthrough on a machine with no Go succeeds, and `go install …@latest` still installs — resolving to the tag rather than the branch tip, as the documentation now says. | SE-1, SE-15, CTX-2 |
| SIG-7 | No promoted requirement asserts something the release channel has made false. | Strict validation passes, and the updated `harness-update` requirement states the prohibition without resting on the channel's absence. | SE-14, CTX-9 |

## Ambiguities and Unknowns

| ID | Question | Evidence |
|---|---|---|
| AMB-1 | Which version the first tag carries. Proposed: `v0.1.0`, matching the constant the code already carries — nothing has been published, so no version has been claimed publicly and starting elsewhere would need a reason. Resolved by owner confirmation of this version. | SE-11, CTX-2, CTX-3 |
| AMB-2 | The artifact naming scheme, which outlives this change because the documentation, the landing page, and any install script quote it. Proposed: `gr_<version>_<os>_<arch>.tar.gz` plus one `checksums.txt`, where `<version>` is the tag without its leading `v`, so the name contains exactly the string `gr version` prints; each archive holds the `gr` binary and the licence. Resolved by owner confirmation of this version. | SE-2, SE-16, CTX-12 |
| AMB-3 | Whether continuous integration runs on pull requests as well as on the release tag. Proposed: on pull requests and pushes to `main` as well as on the tag, because pull-request coverage is what would have caught the Windows build failure before it became a release-time discovery. Resolved by owner confirmation of this version. | SE-9, CTX-1, CTX-5 |
| AMB-4 | Whether the README's download instructions may be merged before the first release exists, since they would name archives nobody can fetch yet. Proposed: they land in this change and the first tag follows the merge as its own gated act, keeping the gap to the length of one release run rather than splitting the documentation across two changes. Resolved by owner confirmation of this version. | SE-7, SE-15, CTX-12 |

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending. This version is candidate. It becomes confirmed only when the owner, having read a plain-language view of these outcomes, boundaries, and signals in the resolved owner language, actively confirms version 1 — including the four recorded ambiguities, whose proposed answers are part of what is being confirmed. A different answer to any of them is a material change and creates version 2.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
