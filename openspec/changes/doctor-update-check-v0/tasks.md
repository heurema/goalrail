## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: the promoted rules that do and do not constrain the network, the structural absence of a transport in the shipped binary, the measured price of adding one, the diagnosis's rules for optional facilities and next actions, the version shapes that are not release tags, the three candidate sources with their costs, the toolchain variables that govern this traffic, the settled and the contested practice, and the failure shapes the source produces.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering the reported fact, the refusal to claim what cannot be known, bounded and quiet asking, honoured refusals, confinement of the network, and prompt discoverability.
- [x] 1.3 Present a plain-language owner-facing view of version 1, leading with the measured price rather than burying it.
- [x] 1.4 Answer the owner's request for a critical comparison against practice: establish from primary sources what is settled, what is contested, and that no project examined documents a privacy rationale for the check itself.
- [x] 1.5 Observe what the toolchain's own proxy request looks like, which version 1 had only assumed, and record it as CTX-17 in Context Pack version 2.
- [x] 1.6 Amend to intent version 2 on that observation — the request blends into toolchain traffic rather than naming the tool, and the report line takes on naming its counterparty — retaining version 1 as `intent-v1.md`.
- [x] 1.7 Answer the owner's question about rewriting the project in Rust from the repository's own record rather than by preference, and identify the real signal in it: doubt about checking by default.
- [x] 1.8 Put the strict alternative to the owner explicitly — the network only behind a flag — with its honest advantage stated, and record their decision and its reason as SE-18.
- [x] 1.9 Obtain explicit owner confirmation of version 2 and record the confirmation receipt with the verification action.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves, and state that no change lies outside that table.
- [x] 2.2 State the price in the proposal itself, since it is the change's main cost and is not recoverable by implementation quality.
- [x] 2.3 Name the capabilities: two modified, none new — the diagnosis gains something it can report, and the channel gains terms on which it may be asked.

## 3. State the contract as spec deltas

- [x] 3.1 Add to `harness-doctor`: the fact-not-problem rule, the naming of the counterparty, the whitelist of version shapes that permit a comparison, the refusal to claim currency, and the violating scenarios for a prescribed command and a changed verdict.
- [x] 3.2 Add to `release-channel`: one bounded question with a cache, confinement of the transport enforced by a check, uniform failure, no credential and no shared quota.
- [x] 3.3 Add to `release-channel`: refusals honoured before new ones are invented, the dedicated switch, no check in continuous integration, and a request that identifies nothing.
- [x] 3.4 Add to `release-channel`: the publisher's duty to request its own new version after publishing, and that its failure does not fail the release.
- [x] 3.5 Modify `harness-update`: the no-lookup rule and all three of its scenarios survive, while the reason stops asserting that whether `gr` consults the channel is still open — this change closes that. The initial judgement that no delta was needed was half right and was corrected: the rule survives, its rationale does not.
- [x] 3.6 Carry no normative rule that traces to no confirmed outcome — in particular, keep the source's identity in design rather than promoting a vendor into the spec, and cite the disclosure rule to the non-goal that actually carries it rather than to the refusal outcome beside it.
- [x] 3.7 Run pinned, telemetry-disabled strict OpenSpec validation over this change. Valid.
- [x] 3.8 Resolve the contradiction the review found in three independent lenses: a cache hit inside the interval is an answer, reported like any other with the time it was obtained, not an instance of "the check did not run". The two deltas had specified opposite behaviour for what is, at a daily interval, the ordinary path.
- [x] 3.9 Separate the vocabulary the two deltas had colliding: "refused" belongs to the user, a transport rejection is the source's, and a declined check is distinguishable from a failed one everywhere it appears — including the continuous-integration case, which previously said only that no request is made.
- [x] 3.10 Scope the duty to name the source to lines that rest on a network answer, matching confirmed OUT-1, so the states with no answer and no counterparty name none.

## 4. Record the design

- [x] 4.1 Record why the module proxy, with both rejected alternatives and the measured reason each was rejected.
- [x] 4.2 Record that the comparison is a whitelist of two release tags rather than a version parser, and why.
- [x] 4.3 Record that confinement is a test, and what replaces today's dependency-graph assertion once it necessarily stops holding.
- [x] 4.4 Record the refusal order — the toolchain's variables first, then the dedicated switch — and the two rejected alternatives, including why the terminal gate does not transfer.
- [x] 4.5 Record that the request is shaped as the toolchain's own, with the observation behind it and where the transparency went instead.
- [x] 4.6 Record the cache as a file whose modification time is the interval, holding the answer rather than the verdict.
- [x] 4.7 Record the post-publication request, why it runs last, and why its failure cannot fail the release.
- [x] 4.8 Record the report and structured-output shape, and the states, all of them facts.
- [x] 4.9 Record why no new exit code, against the one precedent that has one.
- [x] 4.10 Record the risks, the stop conditions, and a rollback that removes the transport entirely.

## 5. Implement the check

- [x] 5.1 Add one package owning the transport: one request to the module proxy for the newest stable version, a short timeout, no retry, and no custom agent string or query parameter. Implemented as `internal/updatecheck`, the only package that links an HTTP client.
- [x] 5.1a Make the fetcher an injected seam on the diagnosis input, beside the environment and path lookups that already exist, defaulting to the real implementation when nil — the mechanism that actually confines it, since the import graph cannot separate two functions in one package. The check is a field on the diagnosis input beside the environment and path lookups, defaulting to nothing rather than to a real transport, so a caller that supplies none reaches no network.
- [x] 5.2 Treat every failure identically — unreachable, timed out, non-success status, malformed body, empty body, or an answer that is not a release version — as "did not run", surfacing no text from the source. Every failure returns the same state; the source's own body is never read past the version field.
- [x] 5.3 Read the toolchain's refusal variables and the dedicated switch before making any request, and report a declined check distinctly from a failed one. Observed end to end: each refusal prints its own reason, and a declined check reads differently from a failed one.
- [x] 5.3a Resolve those variables the way the `go` command does — process environment first, then its environment file — because the documented way to set them writes to that file and leaves the process environment empty, which a check reading only the environment would miss entirely. Never execute the `go` binary: a user who downloaded a release need not have one. Pinned by `TestDeclinedHonoursTheToolchainSettingWhereItActuallyLives`, over a setting written to the environment file with an empty process environment.
- [x] 5.3b Implement the decline rule for a list-valued setting explicitly: proceed only when the public proxy is the first entry, so a private mirror in front declines even when the public one follows as a fallback. Observed: a private mirror in first position declines even with the public proxy following.
- [x] 5.4 Skip the check in continuous integration. Observed: `CI=true` declines.
- [x] 5.5 Cache the answer in one file under the existing state root, using its modification time as the freshness test, and compare against the running binary's version on every diagnosis rather than caching the verdict. Observed: the file is written on the first run, left untouched by the next, and rewritten once aged past the interval.
- [x] 5.6 Compare only when both the local version and the answer are plain release tags — three decimal fields and nothing after — short-circuiting every other shape, on both sides, to "does not apply". Pinned on both sides by `TestOnlyPlainReleasesAreComparable`.
- [x] 5.6a Write the ordering by hand as a field-wise numeric comparison, with a test that pins the case every improvisation gets wrong: a two-digit field must not order below a one-digit one. Pinned by `TestNewerOrdersNumericallyNotAsText`, including `v0.10.0` against `v0.9.0`.

## 6. Report it in the diagnosis

- [x] 6.1 Add one line to the human report beside the toolchain and observability lines, naming the service asked, prescribing no command, and wording currency as what was found and when rather than as an assertion. Observed in every state, beside the toolchain and observability lines.
- [x] 6.2 Add one object to the structured output carrying the state, the version found, the source, and the time of the check. The diagnosis carries an `update` object with the state, the version, the source, and when it was checked.
- [x] 6.3 Confirm the working verdict, the problem list, and the exit code are untouched in every state. Pinned by `TestTheUpdateLineChangesNoVerdict` and observed: a healthy repository reports working and exits `0` with the line present.

## 7. Confine the network

- [x] 7.1 Add a test asserting the transport package is the only place in `cmd/gr`'s reachable tree that imports an HTTP client, and that exactly one package imports the transport. Pinned by `TestOnlyTheTransportPackageImportsAnHTTPClient` and `TestOnlyTwoPackagesImportTheTransport`. The test caught the design's claim of one importer: there are two, both named and justified.
- [x] 7.1a Add the assertion that carries the real weight, since the import graph cannot distinguish functions in one package: the real fetcher is constructed at exactly one call site. Pinned by `TestTheRealCheckIsConstructedOnce`, which also caught itself counting its own file.
- [x] 7.2 Add a test asserting initialization, update, and the hook entry point complete with the transport unavailable — recorded as the weaker of the two assertions, because a command that never calls out completes either way. Covered by the confinement trio plus the failed-check path of the state tests; recorded honestly as the weaker half — a command that never calls out completes either way — with the construction-site scan carrying the real weight.
- [x] 7.3 Confirm the promoted `harness-update` requirement still holds by reading it against the shipped behaviour. Read against the shipped behaviour: the update command is handed no transport and asks nothing.

## 8. Close the freshness window

- [x] 8.1 Add a step to the release workflow that requests the new version from the proxy after publication succeeds, last, and cannot fail the release. Added as the last step of the publishing job, with `continue-on-error` and three bounded attempts.
- [x] 8.2 Verify by construction that the step runs only after publication and that a failure leaves the release published. By construction: ordered after publication, and its failure leaves the release published.

## 9. Document what leaves the machine

- [x] 9.1 State in the README what the check does, what the request contains, which service answers it, and that no identifier is sent. A section stating what the check does, the one request, what it carries, and that it identifies nothing.
- [x] 9.2 Document both ways to turn it off — the toolchain setting a user may already have, and the dedicated switch. Both are documented: the dedicated switch, and the toolchain setting a user may already have.

## 10. Verify the whole slice

- [x] 10.1 Run gofmt, build, vet, the full test suite, and `-race` over the repository. gofmt clean, build clean, vet clean, the full suite and `-race` green.
- [x] 10.2 Run pinned, telemetry-disabled strict OpenSpec validation across this change and every promoted spec. 15 passed, 0 failed.
- [x] 10.3 Exercise the states end to end against a local stand-in source injected through the seam: newer, current, not applicable, declined, failed, and answered from a fresh cache — six distinct reports, with the cached one carrying an answer rather than an absence. Walked end to end against the live proxy with binaries built at real tags: a release behind, the current release, the switch, the toolchain setting, a private mirror, a private module, continuous integration, and the cache both reused and aged.
- [x] 10.4 Measure the binary before and after, and record the actual growth against the 40.2% the decision was made on. Measured: 4,969,826 to 9,869,954 bytes, +98.6%, and 2,760,261 to 5,419,593 compressed, +96% — two and a half times the estimate, because the estimate measured an imported transport rather than a used one. Recorded as CTX-19, put to the owner with the corrected figure before anything was committed, and the decision taken again on it (SE-19).
- [x] 10.4a Assert the request Goalrail sends carries no header the standard library would not have sent anyway, comparing against what the `go` command sends for the same module rather than against a hard-coded agent string. Pinned by `TestTheRequestSaysNothingAboutTheInstallation` and its HTTP/2 twin, which observe Goalrail's real outgoing request and hold it to the standard library's own defaults — verified once against a captured go-command request, then pinned as the two protocol constants rather than compared live.
- [x] 10.5 Map every scenario in the deltas to the test or observation that exercises it, and name those observable only at the next real release. Recorded in `evidence/scenario-coverage.md`, naming what only the next real release can establish.

## 12. Answer the pre-push review

Sixteen findings survived adversarial verification; one was refuted. Three
lenses independently found the worst one, and the session's own toolchain probe
had already confirmed it before the review returned.

- [x] 12.1 `matchesModule` missed the toolchain's prefix semantics: `GOPRIVATE=github.com/heurema` — the canonical corporate value — covered this module for the `go` command and not for `gr`, which is the silent override OUT-4 exists to prevent. Replaced with a stdlib replica of the matching the toolchain itself uses, `path.Match` over an element-truncated prefix with no trimming and no case folding, and pinned with the divergent shapes proven against `module.MatchPrefixPatterns`: the bare organization, the bare host, a trailing slash, a glob prefix, a list, and four negatives.
- [x] 12.2 `startsWithPublicProxy` declined shapes the toolchain resolves to the public proxy — a schemeless `proxy.golang.org` and a leading empty entry — with a printed reason that was false under Go's semantics. It now resolves the first entry the way the toolchain's own proxy list does.
- [x] 12.3 A redirect is now a failure, not a hop: the client refuses to follow one, so the one request cannot be sent onward to a host the report would not name. Pinned by a test asserting the redirected-to host receives nothing.
- [x] 12.4 The cache is read and written with the discipline the escalation artifact taught this repository — `O_NOFOLLOW`, `O_NONBLOCK`, a regular-file check on the open descriptor, and a size bound — because a planted FIFO hung the diagnosis before any timeout applied, and a planted symlink was followed on both the read and the write. Pinned by a test planting both.
- [x] 12.5 The release whitelist rejects zero-padded fields, closing the route by which a hostile answer could carry tens of kilobytes of chosen bytes into the report as a "version".
- [x] 12.6 `Latest` is unexported: `New` is the only route to the transport from outside, so the construction-site scan holds the whole surface.
- [x] 12.7 The construction-site scan resolves imports instead of grepping a literal, catches aliases, rejects dot imports, and names the one site rather than counting to one. Both defeats that beat the old scan — an aliased call from an allowed importer, `net/http` in an unrelated package — were replayed against the new tests and fail.
- [x] 12.8 The composed check's cache wiring is tested against a stand-in source counting requests: a fresh cache asks nothing, an aged one asks once and rewrites. The evidence rows that previously cited a primitives-only test now cite these.
- [x] 12.9 The header test holds the request to the standard library's defaults for both protocols instead of pinning the HTTP/1.1 literal the design itself said not to pin, and an HTTP/2 twin repeats the check on the protocol the real proxy negotiates.
- [x] 12.10 The CI-decline scenario's wording is aligned with the shipped design — one form, each decline naming its reason — and the README's Status section and doctor row stopped contradicting the new section.
- [x] 12.11 Two receipts that overstated their artifacts were corrected rather than defended: 10.4a described a live go-command comparison the test does not make, and 7.2 called itself covered without naming what actually covers it.

## 13. Answer the automated review on the pull request

Seven findings, all seven verified against the toolchain or the code before
being accepted, and all seven answered.

- [x] 13.1 `GONOPROXY` replaces `GOPRIVATE` rather than adding to it — verified live: with `GONOPROXY=none` set, the toolchain does not consult `GOPRIVATE` for the proxy decision at all. The refusal now resolves the effective setting the way the toolchain does, so a user who granted proxying back with `none` gets the check back too, and the printed reason names the variable that actually decided.
- [x] 13.2 The Go environment file honours the last assignment, as the toolchain's own reader does — verified live with a duplicated `GOPROXY` line. The parser previously returned the first match, which would have overridden a final `off` in a hand-merged file.
- [x] 13.3 The environment file is now read with the same discipline as the cache — `O_NOFOLLOW`, `O_NONBLOCK`, a regular-file check — because a FIFO or device at `GOENV` could hang the diagnosis before any timeout applied.
- [x] 13.4 The cache write could hang on a planted FIFO even after the read refused it: a write-only open blocks until a reader appears. `O_NONBLOCK` makes it fail immediately instead, and the write also verifies it holds a regular file. Pinned, with the environment-file and write-side FIFO plants, by three new tests.
- [x] 13.5 The structured state `current` claimed what the wording rules forbid claiming: a machine consumer would read it as authoritative while the source measurably lags a fresh release. Renamed to `nothing_newer_found`, which says exactly what one lookup can say.
- [x] 13.6 The README's privacy paragraph said the request carries "not the tool's name" while its URL names the module — which names the tool. Reworded: the module path is the request's one disclosure, the same one a `go install` of this tool already makes, and everything beyond it is what the toolchain would send anyway.
- [x] 13.7 The agent paste-prompt forbade writes outside the repository that its own last step now performs: `gr doctor` writes the update-check cache into the state root. The prompt names that directory as the second permitted write instead of contradicting itself.

## 11. Stop at the owner gates

- [x] 11.1 Produce planning artifacts only this round; implementation begins only after a separate explicit owner instruction, recorded here before the first edit. Granted 2026-07-30 after the review round and after the owner chose the cheaper repair for the disclosure rule — cited to the non-goal that carries it, with the test kept as task 10.4a — over a further intent version. The grant covers sections 5 through 10 in dependency order. Committing, pushing, opening a pull request, merging, any release that would exercise the new workflow step, and archiving each remain ungranted.
- [x] 11.2 Obtain a separate explicit owner instruction before committing; record it here before the commit. Granted 2026-07-31, after the corrected cost was put to the owner and the decision retaken on it, for one commit carrying the planning artifacts, the transport package, the diagnosis line, the confinement tests, the release workflow step, and the documentation. Recorded here before the commit. Pushing, opening a pull request, merging, any release exercising the new workflow step, and archiving each remain ungranted.
- [x] 11.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; wait for the automated review and do not merge on the strength of that instruction. Granted 2026-07-31, after the owner-directed pre-push review round of section 12 was answered in full. Recorded here before the push; merging, any release, and archiving each keep their own gates.
- [x] 11.4 Obtain a separate explicit owner instruction before merging, after the automated review has been answered. Granted 2026-07-31, after all seven review findings were fixed, verified and answered in-thread, and every check passed on the answering commit. Recorded here before the merge. Any release exercising the new workflow step, and the archival, keep their own gates.
- [x] 11.5a Obtain a separate explicit owner instruction before any release that would exercise the new workflow step. Granted 2026-07-31 for `v0.1.2`, on `main` at `08df584`. Recorded here before the tag. This release is the first to carry the update check at all, and the first to run the post-publication proxy request, so it is what closes the two scenarios nothing earlier could observe.
- [ ] 11.5b Obtain a separate explicit owner instruction before archiving this change and promoting the deltas, and re-read the promoted text immediately before promotion in case another change archived first.
- [ ] 11.6 Leave the owner's working scaffold configuration and the `codex/dogfood-run-v0` branch untouched throughout; every check, cache, and stand-in source this change exercises lives in a temporary directory or a fake state root.
