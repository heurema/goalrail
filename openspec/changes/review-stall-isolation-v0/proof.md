# Implementation Proof

- **Change:** `review-stall-isolation-v0`
- **Branch:** `codex/review-stall-isolation-v0` from `6a896355958874a4989eec131c6cc20a24921995`
- **Status:** implementation complete; stopped at the owner gate before staging, commit, push, pull-request creation, activation, or release

## Failing-before

| Claim | How it was measured before the change | Observed |
|---|---|---|
| A stalled reviewer costs the whole deadline and is reported as an overrun | A temporary test ran a reviewer that emits once and then sleeps, under a 6s deadline | elapsed 6.069s against a 6s deadline; error `the codex reviewer did not finish within the review deadline`, indistinguishable from working-but-slow. The temporary test was removed once the permanent ones asserted the new behaviour |
| A reviewer cannot be run without the machine's integrations | `internal/review.reviewCommand` built a fixed argument list with no caller input, and `Input` carried no field for it | Read at `internal/review/run.go`; the workaround verified by hand could not be expressed through `gr review` at all |
| The field failure itself | Two full passes of `codex/admission-hardening-after-pr62` | 25:01 and 50:01 elapsed against 25m and 50m deadlines, no finding, no receipt; sessions recorded zero events for the remaining 23 and 49 minutes |

## Passing-after

| Outcome | Test | Result |
|---|---|---|
| A stalled reviewer is stopped well before its deadline, named distinctly, carrying its last activity, writing no receipt | `TestAStalledReviewerIsStoppedLongBeforeItsDeadline` | passed |
| A slow but speaking reviewer is not stalled and still reaches its deadline | `TestASlowButSpeakingReviewerIsNotStalled` | passed |
| A silent reviewer's report states the observation and invents no cause | `TestASilentReviewerStatesTheObservationNotACause` | passed |
| The bound is refused when it can never fire, and fitted when defaulted | `TestTheProgressBoundIsResolvedAgainstTheDeadline` | passed |
| Isolation adds exactly one removal and leaves every other argument byte-identical | `TestIsolationRendersOneRemovalAndTouchesNothingElse` | passed |
| A malformed integration name never reaches an invocation | `TestARejectedIntegrationNameNeverReachesAnInvocation` | passed |
| A provider that cannot express the removal refuses it | `TestAProviderThatCannotRemoveOneIntegrationRefuses` | passed |
| The refusal is paid before the reviewer runs, not after | `TestIsolationRefusesBeforeAnythingIsSpent` | passed |
| Goalrail ships no integration name | `TestNoIntegrationIsNamedInShippedContent`, `TestWithoutACallerNameNoIsolationArgumentIsRendered` | passed |
| The deadline still bounds a reviewer that outlives its child | `TestTheDeadlineBoundsAReviewerThatOutlivesItsChild` (stub made live so the deadline, not the bound, is what fires) | passed |

## Live exercise

One full pre-PR review of `codex/admission-hardening-after-pr62`, the branch that
had blocked twice, run with `--without` naming the integration measured blocking:

- completed in **620 seconds** against a 25-minute deadline, where the two prior
  attempts consumed 25 and 50 minutes and returned nothing;
- reviewer `codex`, mode `cross`, effort `high`, receipt written to the state
  root and bound to diff `c405f8ac…` and report `3c2ea727…`;
- returned three findings — two P1 and one P2 — against the change that could
  not be reviewed at all before.

This is the first live exercise of both the progress bound and the isolation
option, and of the review path itself since 2026-08-02.

## Commands actually run

- `go test ./internal/review -count=1` — passed.
- `go test ./...` — passed. `go vet ./...` — passed. `gofmt -l internal cmd` — empty.
- `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate review-stall-isolation-v0 --strict` — valid.
- `git diff --check` — passed.
- `gr review --full --repo <admission worktree> --author claude-code --base origin/main --without <integration>` — completed, receipt written.

## Scope notes

- Task 4.2 asked to update documentation where review outcomes are listed. No
  such document exists: the operator surface for review outcomes is
  `gr review --help`, which now states the progress bound and that a stalled
  review is not a completed one. Nothing was invented to satisfy the task.
- Three existing deadline tests used silent reviewer stubs. Silence is now a
  stall by contract, so the stubs that were testing the deadline were made to
  keep speaking, and the silent case was converted to assert the stall while
  preserving its original point: the observation is reported and no cause is
  invented.

## Untested scope and remaining risks

- The default five-minute bound has not been exercised against a real provider's
  longest legitimate quiet period; the live run never approached it. A reviewer
  that buffers everything for longer than that would be stopped while working.
  The bound is caller-nameable, and the failure is loud rather than a silent
  wrong verdict.
- Isolation is implemented for one provider because only one provider's
  interface can express it. If a second provider gains per-integration removal,
  its rendering is a separate change.
- The provider defect that motivated this work is untouched and still present.
- No commit, push, pull request, activation, or release was performed.
