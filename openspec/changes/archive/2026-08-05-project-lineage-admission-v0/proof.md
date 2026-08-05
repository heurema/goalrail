# Implementation Proof

Change: `project-lineage-admission-v0`
Schema: `goalrail-intent`
Verified: 2026-08-05
Result: implementation complete and ready for owner review

## Implemented behavior

- A committed project declaration and canon make managed identity portable across clones and Git worktrees without a checkout-local ambient marker.
- A clean clone reports setup as required and produces an exact, read-only setup plan before any installation or user-local mutation.
- Confirmed intent, the current change, WorkSpec, run/session evidence, commits, pull request, review, checks, terminal receipt, owner decision, exceptions, and closure use content-addressed lineage artifacts with deterministic local and shared admission verdicts.
- Admission-ready exception handling permits only the documented admission-time omissions, including post-integration closure, while retaining confirmed intent and other governance requirements.
- The release builder produces a minimal binary channel and optional exact setup bundles from integrity-pinned Node and OpenSpec sources.

## Verification receipt

- `gofmt` completed for every Go source file; build, setup, canary, and runtime outputs remained outside the repository.
- `go test ./...` passed for all packages.
- `go vet ./...` passed.
- `go test -json -run '^TestPinnedCLIAcceptsTheEmbeddedCanon$' ./internal/harness` passed with exactly one passing canon test.
- `go test ./internal/conformance` passed.
- `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate project-lineage-admission-v0 --strict --json` passed with zero issues.
- `go test ./cmd/gr -run '^TestProjectLineageAdmissionScratchCanary$' -count=1 -v` passed. The canary covered init, two clones, three worktrees, marker-free discovery, clean-machine setup planning, confirmed fixture intent/change, WorkSpec, fixture run/session and terminal receipt, late lineage, byte-identical local/shared `VALID` verdicts, visible `BREAK_GLASS` admission, legacy migration, and exact pre-commit rollback.
- Four `gr` binaries were cross-compiled for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, and `linux/arm64`; each reported `v0.2.0-canary.1` exactly.
- `goalrail-release build` and `goalrail-release verify` passed in a temporary directory with four minimal archives, four setup archives, four setup manifests, `current-release.json`, and `checksums.txt`.

## External-effect receipt

- The canary used only the in-process fixture provider and temporary local Git repositories. Its commits were immutable fixture inputs, not project history changes.
- No provider was launched, no credential was read or issued, and no user or external service setting was changed.
- No project commit, staging, push, pull request, required-check activation, release, deployment, or Deltacue mutation occurred. Project HEAD remained `03fd7da20371bb8bd6d4483dab1c0976ea8d747b` and the index remained empty.
- The release proof only downloaded public integrity-pinned source archives and wrote verified artifacts under a system temporary directory; it did not call a publication command.

## Remaining owner gates

- Archive this OpenSpec change.
- Commit the implementation.
- Push and open a pull request.
- Activate a real GitHub required check.
- Run Goalrail self-dogfood or migrate Deltacue.
- Publish a release.

Each item remains a separate explicit owner action.
