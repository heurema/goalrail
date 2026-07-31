# Scenario coverage — doctor-update-check-v0

Every scenario in the three delta specs, and what exercises it.

The end-to-end runs used binaries built at real tags in a scratch clone, so the
check ran for real against `proxy.golang.org` rather than against a description
of it. One reported `v0.0.1` — a release behind — and one reported `v0.1.1`, the
current release. Nothing outside the session's temporary directories was touched.

## harness-doctor — a newer release is a fact, never a problem

| Scenario | Evidence |
|---|---|
| A newer release exists | end to end: a binary reporting `v0.0.1` asked the live proxy and printed `update: v0.1.1 is released, this is v0.0.1 (asked proxy.golang.org at 2026-07-31T07:44:04Z)` — no next action, and the exit code unchanged. Test: `TestAvailabilityReportsEachStateAsAFact`. |
| The binary is current | end to end: a binary reporting `v0.1.1` printed `update: nothing newer than v0.1.1 found as of … (asked proxy.golang.org)`. The wording states what was found and when; `TestTheCheckNeverClaimsCurrency` pins that it never says the user is up to date. |
| The running binary is not a release build | end to end: this worktree's own build reports a pseudo-version and printed `update: not checked (this build reports …, which is not a release)`. Test: `TestNoCheckIsMadeWhenThisBuildIsNotARelease`, which also asserts no request was made. |
| The check produced no answer | test: `TestAvailabilityReportsEachStateAsAFact` for the failed case, `TestADeclinedCheckReadsDifferentlyFromAFailedOne` for the distinction the requirement asks for, and `TestAFailingCheckSaysNothingTheSourceSaid` for the rule that the source's own text never reaches a user. |
| The answer comes from a cache that is not yet due | end to end: a second diagnosis left the cache file's modification time untouched, so nothing was asked, while the answer was still reported. Test: `TestAnAnswerFromTheCacheIsStillAnAnswer` asserts the reported answer carries the time it was obtained. |
| A newer release would change the verdict | test: `TestTheUpdateLineChangesNoVerdict` drives every state through the summary and asserts the problem set never moves. End to end: a healthy repository reported `harness: working` beside the update line and exited `0`. |
| A next action would be prescribed | test: `TestAvailabilityReportsEachStateAsAFact` asserts no state's line contains a prescribed command, and `TestTheReportCarriesOneUpdateLine` asserts the line never appears among the problems. |

## release-channel — one bounded question, from one place

| Scenario | Evidence |
|---|---|
| The same diagnosis runs twice | test: `TestTheComposedCheckAsksOnlyWhenTheCacheIsDue` drives the composed check against a stand-in source counting requests — a fresh cache answers with zero further requests. End to end: the cache file's modification time was unchanged by a second real run. |
| The cache is stale | test: the same composed-check test ages the cache past the interval and observes exactly one further request and the file rewritten with the source's answer. End to end: ageing the real file made the next run rewrite it. |
| The network is unavailable | test: `TestAvailabilityReportsEachStateAsAFact` with a failing check; the diagnosis completes and the exit code is unchanged. |
| The source answers something unusable | test: `TestLatestAcceptsOnlyAnAnswerToTheQuestionAsked` over a pseudo-version, a `404` whose body carries the proxy's own paths, an empty body, and non-JSON — each rejected. |
| Another command would reach the network | test: `TestOnlyTheTransportPackageImportsAnHTTPClient`, `TestOnlyTwoPackagesImportTheTransport`, and `TestTheRealCheckIsConstructedOnce`. The last is the one that carries the weight: the import graph cannot separate two functions in one package, so what confines the network is that the transport is handed in as a value from a single construction site. |
| Asking would need an account | the request carries no credential and the proxy requires none — `TestTheRequestSaysNothingAboutTheInstallation` and its HTTP/2 twin assert the outgoing request carries no header beyond what the standard library sends, and `TestARedirectIsAFailureNotAHop` pins that the one request cannot be sent onward to another host. |

## release-channel — refusals already expressed

| Scenario | Evidence |
|---|---|
| The user has directed module lookups away from the proxy | end to end: `GOPROXY=off` printed `update: not checked (GOPROXY does not start with the public proxy)`, as did a private mirror in first position; `GOPRIVATE=github.com/heurema/*` printed `update: not checked (GOPRIVATE covers this module)`. Test: `TestDeclinedHonoursTheToolchainSettingWhereItActuallyLives` covers the case that matters most — a setting written by `go env -w`, which leaves the process environment empty — and `TestDeclinedReadsEachRefusal` covers the rest, including the first-entry rule for a list. |
| The dedicated switch is set | end to end: `GR_NO_UPDATE_CHECK=1` printed `update: not checked (GR_NO_UPDATE_CHECK is set)`. Test: `TestADeclinedCheckNeverReachesTheTransport` asserts no request is made even with the source reachable. |
| The diagnosis runs in continuous integration | end to end: `CI=true` printed `update: not checked (this is continuous integration)`. |
| The request would identify the installation | test: `TestTheRequestSaysNothingAboutTheInstallation` observes the real outgoing request and asserts no agent string of ours, no query parameter, and no header the toolchain would not have sent. |

## release-channel — a published release makes itself discoverable

| Scenario | Evidence |
|---|---|
| A release is published | **observed on `v0.1.2`**: tag at 09:31:52Z, release published 09:34:34Z, the warm step's first attempt answered `200` at 09:34:38Z, and the proxy carried the version immediately afterwards — four seconds, against 3m45s and 16m41s of passive discovery on the two previous tags. |
| The source does not answer | by construction: the step carries `continue-on-error` and bounded retries before giving up with a message. Observable in full only if the proxy is slow during a real release. |
| The request would precede publication | by construction: the step is ordered after `gh release create` in the same job. |

## harness-update — the rule survives, its reason changes

| Scenario | Evidence |
|---|---|
| The user reads the command's help | unchanged behaviour; the promoted scenario is carried through the `MODIFIED` delta untouched. |
| A release lookup would be attempted | test: the confinement tests above. The update command is in the same package as the diagnosis and is handed no transport, which is exactly why the assertion is about construction sites rather than imports. |
| A release channel exists | the channel exists and is asked — by the diagnosis. The update command asks nothing, and the requirement now says so instead of calling the decision open. |

## What the first release carrying this check established, and corrected

`v0.1.2` was the first release to ship the check and the first to run the warm
step. Both worked, and one claim they were supposed to support did not survive
contact:

- The check works end to end against reality. The published `v0.1.2` binary,
  downloaded through the documented route and verified against `checksums.txt`,
  reports `update: nothing newer than v0.1.2 found as of … (asked
  proxy.golang.org)`. A binary carrying the check but stamped `v0.1.1` reports
  `update: v0.1.2 is released, this is v0.1.1`.
- The warm step collapses ingest to seconds: four, measured, against 3m45s and
  16m41s passively.
- **But it warmed the wrong answer.** The step requested `@v/<tag>.info`, while
  the diagnosis reads `@latest` — a different endpoint with its own sixty-second
  cache. Between 09:34:38Z and about 09:37:1xZ the check honestly reported
  "nothing newer" while a newer release existed, observed live. The design's
  "about a minute" was written about ingest and read as though it were about the
  check. The step now asks for `@latest` as well; the remaining floor is that
  cache rather than discovery, and it is measured at the next release.

That miss is the same shape this project keeps producing — a true observation
generalised past what was observed — and is recorded in issue #43 with the
others.
