# Implementation Proof

- **Change:** `review-stall-isolation-v0`
- **Branch:** `codex/review-stall-isolation-v0` from `6a896355958874a4989eec131c6cc20a24921995`
- **Status:** implementation complete and reviewed; committed and pushed for a pull request under the owner's explicit instruction. Activation, release, installation, and merge remain separate gates

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
| Each provider renders its own removal, and an unknown one still refuses | `TestEachProviderRendersItsOwnRemoval` | passed |
| The refusal is paid before the reviewer runs, not after | `TestIsolationRefusesBeforeAnythingIsSpent` | passed |
| The provider configuration surfaces appear only where they are rendered | `TestTheRemovalSyntaxAppearsOnlyWhereItIsRendered`, `TestWithoutACallerNameNoIsolationArgumentIsRendered` | passed |
| An isolated review is recorded as one and keeps its own receipt | `TestAnIsolatedReviewIsRecordedAsOne` | passed |
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

## Pre-PR review findings and their disposition

One full pre-PR review ran on 2026-08-06 against `246443f`, reviewer `codex`,
mode `cross`, effort `high`, 774 seconds — the first review of this change, run
through the mechanism the change itself repairs. All four findings are accepted.

| Finding | Claim | Disposition | Fix and check |
|---|---|---|---|
| P1 | Claude Code can deny one configured MCP server per invocation via `--settings`, so refusing every request on the claim that its interface removes all or none was false and blocked a supported provider | Accepted, and the claim was mine: I read `--help`, saw no per-server flag, and wrote an absence into both the code and the spec. `--help` lists flags, not settings keys. Measured afterwards on 2.1.221: with the setting the named server disappears from the session's tools while the others remain | Both providers now render their own removal; `TestEachProviderRendersItsOwnRemoval` asserts each rendering and that an unknown provider still refuses. The spec now requires capability to be established by running an interface, never by reading its flag listing |
| P2 | A dotted name renders `mcp_servers.vendor.tool.enabled=false`, which one provider reads as nested configuration and fails on, while the test explicitly blessed dotted names | Accepted — the name reached the provider and removed nothing, which is the silent-failure shape isolation exists to avoid | Dots are refused with a reason rather than escaped; escaping would be a claim about a vendor parser this repository would have to keep true. Covered by the rejected-name table and by `TestIsolationRefusesBeforeAnythingIsSpent` |
| P2 | Isolation reached `invoke` only: no receipt recorded it, and two otherwise identical runs addressed one receipt path | Accepted — the design refers to an isolation record and there was none, so a consumer could not tell an isolated review from an ordinary one | The receipt records `without_integrations` and the stored path is keyed by it; `TestAnIsolatedReviewIsRecordedAsOne` asserts both the record and the distinct path |
| P2 | The absence check searched one literal in a subset of file extensions while claiming shipped content names no integration | Accepted — the claim was broader than the check, which is the failure this repository's own rules name | The claim is narrowed to what a check can verify — the provider configuration surfaces are confined to the code that renders them — the scan covers every text file, and both surfaces are searched |

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
- Isolation is implemented for both installed providers. Each rendering was
  established by running that provider's interface; a future provider needs its
  own verified rendering.
- The provider defect that motivated this work is untouched and still present.
- No workflow activation, release, installation, or merge was performed. The
  fixes made in response to the review's own findings have not themselves been
  reviewed.
