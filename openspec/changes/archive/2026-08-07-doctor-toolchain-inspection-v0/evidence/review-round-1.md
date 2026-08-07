# Independent review, round 1 (full pass)

- reviewer: codex (mode cross — codex did not author this change)
- author: claude-code
- range: 23e30a5c8f6d..252e261e982a
- effort: high, duration: 456s
- diff sha256: bf63cdbc061df756e63f7968e816407369a336c1c59801a6b397f9729ee33a64

## Report

Planning readiness can remain green while the actual compiler entrypoint is damaged, and malformed manifests or parent symlinks can escape the intended bundle boundary. These defects invalidate the patch's core integrity claim.

Full review comments:

- [P1] Verify the compiler identity recorded by the manifest — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:213-220
  For a valid bundle where the compiler package owns a path sorting before `bin/openspec.js`, such as `LICENSE` or `README.md`, `entrypointFor` hashes that unrelated file. If the actual CLI entrypoint is deleted or modified while the earlier file remains intact, the compiler remains `ready`. The release builder already emits the exact compiler identity (`internal/releasebundle/build.go:397-400`), and the declared contract requires hashing the compiler entrypoint (`openspec/changes/doctor-toolchain-inspection-v0/design.md:65-70`), so use that binary identity instead.

- [P1] Validate the installed manifest before trusting its paths — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:147-149
  When the manifest is edited into syntactically valid but structurally invalid JSON—for example with a runtime identity path such as `../../../../tmp/node` and a matching digest—this accepts it because only schema, release, and platform are checked; `verifyInstalledFile` can then hash outside the bundle and report `ready`. Use the existing `releasebundle.DecodeSetupBundleManifest` boundary (`internal/releasebundle/contracts.go:185-195`), which validates safe paths, digests, ordering, and cross-references.

- [P2] Reject symlinks in every component path segment — /Users/vi/personal/heurema/goalrail/internal/boundedio/bounded_file.go:81-81
  When a parent directory such as `runtime/node` is replaced by a symlink to an external tree containing bytes with the expected digest, this open succeeds because `O_NOFOLLOW` protects only the final pathname component. Diagnosis therefore follows a symlink outside the bundle and can report the component `ready`, contrary to the explicit no-follow requirement in `openspec/changes/doctor-toolchain-inspection-v0/tasks.md:22-24`; every path component must be resolved without following symlinks.


## Disposition

All three findings accepted. Each was verified against the code before the fix,
then pinned by a regression that fails against the reviewed commit and passes
after.

| Finding | Disposition | Fix | Regression |
|---|---|---|---|
| P1 compiler entrypoint chosen by sort order | accepted | the manifest's binary identity names it, for both components | `TestReplacedCompilerEntrypointIsCaught` |
| P1 manifest trusted before validation | accepted | `releasebundle.DecodeSetupBundleManifest` replaces the raw decode | `TestManifestCannotPointOutsideTheBundle`, `TestManifestRefusals/not_canonical`, `TestManifestRefusals/component_path_escapes_the_bundle` |
| P2 symlinked parent directory followed | accepted | every bundle file is opened through a confined `os.Root` | `TestSymlinkedParentDirectoryIsNotFollowed` |

Failing-before output for all three is in `review-round-1-failing-before.txt`.
No refute round was run: refute exists to challenge findings before acting on
them, and each of these was instead confirmed by reproduction — the pre-fix run
reports the compiler `ready` while verifying `.../openspec/LICENSE`, which is
the finding's own claim reproduced rather than argued.
