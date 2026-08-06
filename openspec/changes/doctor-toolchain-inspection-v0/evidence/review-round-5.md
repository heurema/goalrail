# Independent review, round 5 (closing full pass)

The promoted rule is that the last round before a pull request is a full pass:
incremental rounds price the loop, but a fresh whole-branch read sees what
deltas cannot. This round read `23e30a5..b9ed98b` at high effort.

- reviewer: codex (mode cross)
- effort: high, duration: 467s
- integrations removed: codebase-memory-mcp, node_repl

## Report

The observer can report readiness for a non-executable component or for files reached through a symlinked bundle root, and it emits a noncanonical digest in the machine report. The committed review-count claim is also inconsistent with its evidence.

Full review comments:

- [P2] Reject symlinks before opening the bundle root — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:159-159
  When the bundle directory or any ancestor below `home` is replaced with a symlink after setup, `os.OpenRoot(path)` follows it, so the confined root is established around the symlink target and an external manifest and matching files can be reported ready. This contradicts the no-escape guarantee in `openspec/changes/doctor-toolchain-inspection-v0/design.md:83-89`; open the path relative to a previously confined home root or explicitly refuse symlinked ancestors.

- [P2] Verify executable mode before reporting ready — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:269-269
  When `runtime/node/bin/node` is changed from mode `0755` to `0644`, its digest remains unchanged and this read succeeds, so the runtime is reported ready even though executing it fails with permission denied. The manifest records the required mode and the release builder installs this entrypoint as `0755` (`internal/releasebundle/sources.go:270`), so readiness must compare the observed mode as well as the digest.

- [P2] Preserve the SHA-256 prefix in the report — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:245-245
  For every verified runtime and compiler this strips `sha256:` before assigning a `domain.SHA256Digest`, so JSON emits a bare 64-character value in `required_integrity`. That value fails the domain's own `IsSHA256Digest` contract and is inconsistent with every other digest field; retain the canonical prefixed manifest value while normalizing only for the byte comparison.

- [P2] Correct the unsupported review finding count — /Users/vi/personal/heurema/goalrail/openspec/changes/doctor-toolchain-inspection-v0/tasks.md:37-37
  The committed round reports contain six findings—three in round 1 and one in each of rounds 2, 3, and 4—but this completed task claims seven, and the referenced `evidence/review-round-*.md` set contains no seventh disposition. Correct the count or add the missing review record so the completion claim is verifiable.


## Disposition

All four accepted. Three are defects the incremental rounds could not have seen
— each lives in code they had already read and approved, which is what a full
pass is for.

| Finding | Fix | Regression |
|---|---|---|
| the confined root was established around a symlinked bundle directory | the root is descended from the home directory instead of resolved in one step | `TestSymlinkedBundleRootIsNotFollowed` |
| a cleared executable bit left the runtime ready | the manifest's recorded mode is compared with the bytes | `TestClearedExecutableBitIsNotReady` |
| `required_integrity` emitted a bare sum, failing the domain digest contract | the manifest's canonical spelling is kept; normalization happens only in the comparison | `TestReportedIntegrityKeepsItsCanonicalForm` |
| the task list claimed seven findings where the records hold six | corrected, and restated with this round included | the count is the `- [P` lines across `review-round-*.md` |

The fourth is the same class as round 4's: a count asserted without its check.
Twice in two rounds, from the same author, in the same artifact set. Recorded
here rather than smoothed over — the checkable form is the one the records
themselves produce.

Three regressions failing before the fix are in `review-round-5-failing-before.txt`.
