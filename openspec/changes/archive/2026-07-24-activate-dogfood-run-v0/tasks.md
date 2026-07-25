## 1. Operator help

- [x] 1.1 Add a concise top-level `gr help` lifecycle summary and command routing without changing successful JSON results.
- [x] 1.2 Make `prepare`, `inspect`, `start`, and `finish` expose built-in flag help without invoking a service.
- [x] 1.3 Update `docs/local-run-v0.md` with the owner-assisted lifecycle and the distinction between preparation, start, and later review.
- [x] 1.4 Add focused CLI tests for top-level help, per-command help, and existing JSON compatibility.

## 2. Exact admission boundary

- [x] 2.1 Present the exact one-time admission representation decision to the owner; do not infer it from the confirmed intent.
- [x] 2.2 Implement the selected exact admission only if it preserves default denial, binds one digest and pinned revision, and cannot become a general activation surface.
- [x] 2.3 Add focused local-run tests for exact admission, every fail-closed mismatch, one-shot claim behavior, and preserved provider enforcement.

The owner re-authorized tasks 2.2 and 2.3 on 2026-07-24. Focused verification:
`go test ./internal/localrun -count=1` and the concurrent admission test under
`go test -race` both passed. No real provider process was launched.

## 3. Verification and review boundary

- [x] 3.1 Run `go test ./cmd/gr` and `go test ./...`; record the results and any remaining activation-decision blocker.
- [x] 3.2 Validate the OpenSpec change and present the local diff for owner review; do not start a real Codex run, perform Git actions, or continue automatically.

Verification on 2026-07-24: `go test ./cmd/gr` and `go test ./...` passed.
No admission-representation decision blocker remains. Constructing the
dogfood service with the existing Codex adapter and launching a real provider
process remain separate explicit owner gates; the default `gr` service stays
fail-closed.

Review receipt on 2026-07-24: pinned OpenSpec 1.6.0 strict validation passed
with zero issues. The read-only worktree review found five modified tracked
files and eleven untracked implementation/change-artifact files;
`git diff --check` passed. No provider process or Git write was performed.
