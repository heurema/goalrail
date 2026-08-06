## 1. Failing-before regressions

- [x] 1.1 Add a reviewer fake that emits output and then falls silent, and assert against current behavior that the review runs to its deadline and reports a deadline overrun; record the observed elapsed time.
- [x] 1.2 Add a reviewer fake that emits output continuously until it exceeds its deadline, and assert it is reported as a deadline overrun; this is the case the progress bound must not touch.
- [x] 1.3 Assert against current behavior that no reviewer invocation can be run without the machine's provider integrations, so the isolation gap is recorded before it is closed.

## 2. Progress bound

- [x] 2.1 Track the instant of the last captured reviewer output across both streams, updating it on arrival and never rewinding it, without interpreting content.
- [x] 2.2 Resolve a progress bound alongside the deadline: caller-named where given, defaulted otherwise, and refused when zero, negative, non-parsing, or not shorter than the effective deadline.
- [x] 2.3 Stop the review when the reviewer produces nothing for longer than the bound, reusing the existing whole-process-tree teardown so no reviewer process survives.
- [x] 2.4 Report a stalled reviewer as an outcome distinct from a deadline overrun, carrying the elapsed time, the exceeded bound, and the tail of captured reviewer output.
- [x] 2.5 Write no receipt for a stalled review, and add a test proving a stalled run is not readable as completed, finding-free, or passing.
- [x] 2.6 Prove the two fakes from 1.1 and 1.2 now classify differently, and that a slow-but-live reviewer still reaches its deadline.

## 3. Reviewer integration isolation

- [x] 3.1 Add one repeatable command option naming an integration to remove from the reviewer's session, off by default, documented as the caller's knowledge of their own machine.
- [x] 3.2 Validate each name against a bounded identifier character set, refusing empty, oversized, whitespace-bearing, and structure-altering values before any reviewer starts.
- [x] 3.3 Render each accepted name into the provider's own documented disable syntax per provider, adding nothing else, and refuse the option for a provider that has no such syntax rather than ignoring it.
- [x] 3.4 Add per-provider tests over the rendered argument list asserting the sandbox, model, effort, and reviewer identity arguments are byte-identical with and without isolation, under every accepted and refused input.
- [x] 3.5 Add a check that no provider integration, server, or tool name appears in shipped repository content outside this change's own evidence.

## 4. Surfaces and documentation

- [x] 4.1 Extend `gr review` help with the two new options in the existing style, stating that isolation is off by default and that a stalled review is not a completed one.
- [x] 4.2 Update the reviewed-and-recorded documentation where review outcomes are listed, so a stalled outcome is described alongside the deadline overrun.
- [x] 4.3 Keep the enforcement-claims check passing, and add no claim that isolation makes a review authoritative.

## 5. Integrated verification and owner gate

- [x] 5.1 Run `gofmt` on changed Go files and keep generated, build, and temporary artifacts out of the repository.
- [x] 5.2 Run `go test ./internal/review ./cmd/gr -count=1`, then `go test ./...` and `go vet ./...`, stopping at the first persistent deterministic failure rather than widening the change.
- [x] 5.3 Run `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate review-stall-isolation-v0 --strict`.
- [x] 5.4 Run `git diff --check` and inspect the final diff for unrelated, generated, or dependency changes.
- [x] 5.5 Run one live full pre-PR review of `codex/admission-hardening-after-pr62` with isolation, and record whether it completes and writes a receipt; this is the first live exercise of both changes.
- [x] 5.6 Complete a proof receipt with failing-before and passing-after evidence, the exact commands run, untested scope, and remaining risks, then stop for owner review before staging, commit, push, pull-request creation, activation, or release.
