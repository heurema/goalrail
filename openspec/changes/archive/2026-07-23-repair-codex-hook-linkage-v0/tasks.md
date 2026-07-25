## 1. Preserve Source Evidence

- [x] 1.1 Record read-only checksums for the spent v1 preparation, launch claims, provider observation, terminal receipt, and `.goalrail/activate-dogfood-run-v1` capsule; record the current `HEAD`, exact three-path task diff, and existing run directory set before editing.
- [x] 1.2 Confirm the source receipt remains `unlinked` with reason `INVALID_HOOK_INPUT`, both check results remain `unavailable`, and no replacement run, claim, receipt, or provider process is created.

## 2. Add the Bounded Hook Contract

- [x] 2.1 Add `internal/adapters/codex/hook_config_test.go` with a pinned-contract regression fixture that counts zero nested handlers for the spent flattened v1 shape and exactly one command handler for the repaired shape.
- [x] 2.2 Add focused assertions for the exact repaired override, fixed `hook` subcommand, whitespace and shell-metacharacter paths, embedded single quotes, non-absolute paths, and NUL/CR/LF rejection without partial output.
- [x] 2.3 Implement the narrow adapter-owned renderer in `internal/adapters/codex/hook_config.go` using only the standard library, one absolute executable input, safe POSIX argument quoting, and the fixed supported `SessionStart` structure.
- [x] 2.4 Verify the existing pinned-schema-compatible positive correlation test and unlinked negative matrix still cover provider-authoritative identity and raw-payload non-retention; extend only focused Codex adapter tests if a confirmed signal is missing.
- [x] 2.5 Run `gofmt` on only the new or modified Codex adapter Go files and review the package diff for provider-specific leakage or added authority.

## 3. Verify and Stop

- [x] 3.1 Run `go test ./internal/adapters/codex` once and stop on failure without launching Codex or adding a fallback.
- [x] 3.2 Run `go test ./...` once and stop on failure without retrying or widening scope.
- [x] 3.3 Run the pinned telemetry-disabled strict OpenSpec validation for `repair-codex-hook-linkage-v0` and resolve only change-local planning inconsistencies.
- [x] 3.4 Recompute the source-evidence checksums, `HEAD`, exact three-path task diff, and run directory set; require them to match the pre-edit baseline and require zero new provider attempt.
- [x] 3.5 Show the bounded local diff and test results, confirm implementation changes are limited to the Codex adapter repair and focused tests plus this change's artifacts, and stop without `gr finish`, provider run, commit, push, PR, merge, publish, or deploy.
