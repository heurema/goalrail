## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: the absent CI and absent tags, the version constant and where it surfaces, what Go stamps in each build situation, the four buildable platforms and the Windows failure with the promoted guarantee behind it, the toolchain-only verification baseline, the canon check that skips rather than fails, the promoted text justified by the channel's absence, the recorded revisit condition, the repository's Actions and licence facts, and the owner's decisions and boundaries from the brief.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering continuous integration, a tag-driven release, a build-derived version, the canon check in CI, install documentation without Go, and promoted requirements that stay true.
- [x] 1.3 Present a plain-language owner-facing view of version 1 in the resolved owner language, including the correction to the brief's first tripwire.
- [x] 1.4 Run the owner-directed critical review of version 1 and reproduce every finding locally rather than accepting it as reported. Five confirmed: the artifact-naming rationale contradicted the version mechanism; the macOS claim was false on the documented route; the motivating agent journey was covered by no outcome; the release's own build outputs would dirty its artifacts; and both stated verification mechanisms were unimplementable. Three refuted: tag events needing branch protection, partial asset publication, and a skip gate needing a harness switch.
- [x] 1.5 Extend the Context Pack to version 2 with the ten facts the review surfaced, each reproduced in this session, and note that several of them correct an inference the candidate drew or a claim the pack itself made.
- [x] 1.6 Amend to intent version 2 — including the reversal on the leading `v` in artifact names, the new outcome for the agent journey, and the corrected verification mechanisms — and retain version 1 as `intent-v1.md` rather than rewriting it.
- [x] 1.7 Obtain explicit owner confirmation of version 2 and record the confirmation receipt with the verification action.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves, and state that no change lies outside that table.
- [x] 2.2 Mark the version change as breaking for anything reading `harness.Version` as a compile-time constant.
- [x] 2.3 Name the capabilities: one new, one modified, with the modification justified by promoted text this change makes false rather than merely extends.
- [x] 2.4 State the impact, including that the two existing tests touching the version keep holding.

## 3. State the contract as spec deltas

- [x] 3.1 Add `release-channel`: automatic verification of every change, what a tag publishes, the version a binary reports, the publish gate that compares stamps, the canon check that cannot skip into green, the install route without Go, and the agent-facing prompt.
- [x] 3.2 Modify `harness-update`'s "does not update the binary" requirement so the no-lookup rule rests on the boundary rather than on the channel's absence, preserving both existing scenarios and adding the one that makes the new reason observable.
- [x] 3.3 Verify by set difference that the `MODIFIED` delta drops no promoted scenario: both scenarios of the promoted requirement are carried through, since a modified requirement replaces the promoted one wholesale at archival.
- [x] 3.4 Carry no normative rule that traces to no confirmed outcome.
- [x] 3.5 Run pinned, telemetry-disabled strict OpenSpec validation over this change. Valid.

## 4. Record the design

- [x] 4.1 Record why the release re-runs verification on the tagged commit rather than trusting an earlier run, and why `workflow_run` chaining was rejected.
- [x] 4.2 Record the version resolver: build information read once and reported verbatim, `unknown` where none exists, with ldflags injection and every normalization rejected and the reasons stated.
- [x] 4.3 Record that a test binary always reports the development marker, so the unit test exercises injected build information rather than real Git states.
- [x] 4.4 Record where artifacts are built and why writing outside the tree is necessary but not sufficient, with the clean-tree assertion as the guarantee and `.gitignore` rejected.
- [x] 4.5 Record the publish gate: stamps read without executing, exact equality, and the default no-tags checkout with the second-tag hazard that motivates it.
- [x] 4.6 Record the canon-check enforcement through the test's own JSON stream, why nothing inside the harness changes, and how the package to prime is derived from the binary rather than duplicated in YAML.
- [x] 4.7 Record that `gh`'s default already stages a draft and publishes last, how a leftover draft is handled, and that a published release is superseded rather than repaired.
- [x] 4.8 Record the permission declarations, the first-party actions, and the toolchain resolved from `go.mod`.
- [x] 4.9 Record the runner split and why darwin is tested rather than only compiled.
- [x] 4.10 Record the archive layout, the flat-root reason, and the naming that keeps one string across tag, report, and filename.
- [x] 4.11 Record everything the install documentation must carry, each item with the verified fact behind it.
- [x] 4.12 Record that no published binary is post-processed, with the load-bearing linker signature as the reason.
- [x] 4.13 Record the rollback: deleting two files stops the channel and changes nothing else.

## 5. Implement the version resolver

- [x] 5.1 Replace the `Version` constant with a package-level value resolved from build information, reported verbatim, falling back to `unknown` wherever the build carries no version string — whether the build information is absent or the version it carries is empty.
- [x] 5.2 Add a unit test over injected build information covering every recorded shape: exact tag, tag with local modifications, pseudo-version between tags, pseudo-version with no tags, development marker, and absent information.
- [x] 5.3 Confirm the two existing tests that reference the version still hold, and that `gr version` and `gr doctor` report the resolved value.
- [x] 5.4 Build at a throwaway tag in a scratch clone and confirm the reported version is the tag; build one commit later and confirm it names the commit instead. Observed: `v0.1.0` at the tag, `v0.1.1-0.20260730163543-9f1d30e6ec5d` one commit later, `v0.0.0-…+dirty` in this worktree, and `(devel)` with `-buildvcs=false`.

## 6. Implement the verification workflow

- [x] 6.1 Add `.github/workflows/ci.yml` triggered on pull requests and on pushes to `main`, declaring `contents: read`, resolving the toolchain from `go.mod`.
- [x] 6.2 Run vet and the full test suite on Linux and on macOS.
- [x] 6.3 Cross-compile all four supported platforms with `CGO_ENABLED=0`.
- [x] 6.4 Confirm locally that the same commands reproduce the green baseline, and that a deliberately broken platform build fails them. Observed: `gofmt` clean, `go vet` clean, `go test ./...` green, and the four cross-compiles succeed while `windows/amd64` fails at `internal/boundedio/bounded_file.go:27` — the failure the matrix surfaces before a merge.

## 7. Implement the release workflow

- [x] 7.1 Add `.github/workflows/release.yml` triggered on `v*` tags, with `contents: read` at the top and `contents: write` only on the publishing job.
- [x] 7.2 Run the full verification on the tagged commit as a prerequisite of building.
- [x] 7.3 Prime the npm cache with the pinned package derived from the binary's own report — the token after `npx --yes` in the diagnosis's `invocation` field — and fail the step when the extraction finds nothing. Run the canon check with JSON output and require an observed pass for that exact test, failing on a skip, a failure, or no verdict at all. Observed: the derivation yields `@fission-ai/openspec@1.6.0` from the diagnosis, and the gate counts exactly one pass with the cache primed.
- [x] 7.4 Assert the work tree is clean, then build the four platforms outside the checkout and assemble the archives — binary plus licence, flat — and `checksums.txt`. Observed: the four archives and `checksums.txt` were produced from a tagged scratch clone, each archive holding `gr` and `LICENSE` at its root.
- [x] 7.5 Read every artifact's stamped version without executing it and require exact equality with the tag; fail the release on any mismatch. Observed: `go version -m` read `v0.1.0` from all four artifacts on a darwin host without executing any of them.
- [x] 7.6 Publish with the preinstalled tool so the draft-then-publish sequence holds, passing the job token to the publishing step's environment, deleting a leftover draft for the tag only after reading that it is still a draft, and aborting rather than touching an already published release.
- [x] 7.7 Verify the gates by construction in a scratch run: seed a mismatched stamp and confirm nothing publishes; remove the priming step and confirm the run fails. Observed: a skipped check and an absent check each leave `go test` exiting `0` while the gate counts zero passes and fails; an untracked file makes the clean-tree step abort, and a binary built past it stamps `v0.1.0+dirty`, which the stamp gate rejects.

## 8. Rewrite the install documentation

- [x] 8.1 Rewrite the README install section around the published archives: the platform table, the download, and the extraction that takes only the binary.
- [x] 8.2 Document verification with the form that succeeds on a partial download, for both macOS and Linux.
- [x] 8.3 Document discovery without knowing the version through the one asset whose name does not change between releases.
- [x] 8.4 Document the durable location and why it matters, given that a registration records the absolute path of the executable that wrote it.
- [x] 8.5 Replace the macOS claim with what is true: no warning on the command-line route, the browser-and-Finder path that does warn, the working remedy, and "not notarized" rather than "unsigned".
- [x] 8.6 State that Windows is unsupported, with the reason and the revisit condition.
- [x] 8.7 Rewrite the agent-facing prompt around the binary route, permitting the one write outside the repository explicitly, and update the Status section so it stops saying no binaries are published and stops saying Go is required to install, while keeping the note about the stock OpenSpec CLI, which stays true.
- [x] 8.8 Keep `go install` documented as the second route, noting what `@latest` resolves to once a tag exists.
- [x] 8.9 Confirm every artifact name the documentation quotes matches what the workflow produces. Observed: the four names in the install table match the four the packaging step produced.

## 9. Verify the whole slice

- [x] 9.1 Run gofmt, build, vet, the full test suite, and `-race` over the repository, and confirm both workflow files parse before they are pushed, since the release workflow's first real execution is the act that publishes. Observed: `gofmt` clean, build clean, `go vet` clean, `go test ./...` and `go test -race ./...` green, both workflow files parse.
- [x] 9.2 Run pinned, telemetry-disabled strict OpenSpec validation across this change and every promoted spec. Observed: 14 passed, 0 failed.
- [x] 9.3 Walk the install documentation end to end on a machine with no Go on the path, using the artifacts a scratch build produced, and confirm the diagnosis is healthy after the download directory is removed. Observed with `PATH` holding no `go`: discovery from `checksums.txt`, `--ignore-missing` verification of one archive of four, extraction into `~/.local/bin`, `gr version` reporting `v0.1.0`, `gr init --scaffold claude-code`, and `gr doctor` reporting `harness: working` and exiting `0` after the download directory was removed. The documented hazard was reproduced too: initializing from a temporary copy and then deleting it makes the diagnosis report the registered executable as missing.
- [x] 9.4 Map every scenario in the delta to the test, command, or recorded observation that exercises it, and name the ones that can only be observed after the first real tag.

## 10. Stop at the owner gates

- [x] 10.1 Produce planning artifacts only this round; implementation begins only after a separate explicit owner instruction, recorded here before the first edit. Granted 2026-07-30, after the artifact review round, for sections 5 through 9 in dependency order — the version resolver, the two workflows, the install documentation, and the local verification. Recorded here before the first implementation edit. Committing, pushing, opening a pull request, merging, tagging, any provider run, and archiving each remain ungranted.
- [x] 10.2 Obtain a separate explicit owner instruction before committing; record it here before the commit. Granted 2026-07-30 after the implementation was verified, for one commit carrying the planning artifacts, the version resolver, both workflows, the rewritten install documentation, and the scenario-coverage evidence. Recorded here before the commit. Pushing, opening a pull request, merging, tagging, any provider run, and archiving each remain ungranted.
- [x] 10.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; wait for the automated review and do not merge on the strength of that instruction. Granted 2026-07-30 and recorded here before the push. Merging stays behind its own gate, after the automated review has been answered; the first tag, any provider run, and archiving each keep theirs.
- [ ] 10.4 Obtain a separate explicit owner instruction before merging, after the automated review has been answered.
- [ ] 10.5 Obtain a separate explicit owner instruction before creating the first tag. Creating it is an external act that publishes a release, and it is not granted by confirming intent, by merging, or by anything in this change.
- [ ] 10.6 Obtain a separate explicit owner authorization before any provider run is made or treated as evidence. One run closes what nothing before the first release can: an agent handed the README's prompt on a machine with no Go discovers the release through the checksums file, verifies, places the binary durably, initializes, and reports a healthy diagnosis — which stays healthy after the download directory is removed. Name the agent, the machine, and the one write permitted outside the repository, and keep the transcript and both diagnoses as the receipt.
- [ ] 10.7 Obtain a separate explicit owner instruction before archiving this change and promoting the delta, and re-read the promoted text immediately before promotion in case another change archived first.
- [ ] 10.8 Leave the owner's working scaffold configuration and the `codex/dogfood-run-v0` branch untouched throughout; every build, registration, and install walkthrough this change exercises goes to a scratch clone, a temporary directory, or a fake home.

## 11. Verify the first real release

Executed after gate 10.5 and before 10.7, because these are the checks nothing
earlier can perform: until a release exists there is nothing to download.

- [ ] 11.1 List the published assets and diff them against every artifact name the README quotes and against the set the spec requires — four archives and one checksums file, nothing else.
- [ ] 11.2 Download one archive and the checksums file on a machine with no Go on the path, and confirm the README's own verification command succeeds against the single archive fetched.
- [ ] 11.3 Extract that archive and confirm its flat root carries the binary and the licence, and that the binary reports exactly the pushed tag.
- [ ] 11.4 Confirm the checksums file resolves at the address that does not change between releases, and that it names the archives of this release.
- [ ] 11.5 If any of the above fails, record what was published, and treat a corrective release as a new tag under its own owner gate rather than editing the published one.
