# The first canon change crosses live — 2026-08-02

The defect the adversarial exchange predicted was exercised deliberately: both
adopting repositories held the v0.1.1–v0.1.8 overlay while a binary carrying
the new canon diagnosed and updated them.

| repository | before | update | after |
|---|---|---|---|
| baseline | `overlay: 4 file(s) differ from sha256:f9d1815c…` | 4 files updated, **no flag required**, `verified: true` | `overlay: current` |
| goalrail | already current — its mirror copy is edited by this very branch, so there was no crossing to make | `already_current: true` | `overlay: current` |

Without the recorded previous canon, every one of those files would have
classified as `edited` and `gr update` would have stopped with `ErrLocalEdits`,
demanding `--discard-local-edits` for files nobody touched.

Reproduce: `go test ./internal/harness/ -run PreviousCanon` pins the crossing
by digest; this file records the live run of the same transition.
