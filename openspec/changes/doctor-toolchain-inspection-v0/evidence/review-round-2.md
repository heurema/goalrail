# Independent review, round 2 (incremental)

- reviewer: codex (mode cross)
- range: 252e261e982a..47d02d13b1a8
- effort: medium, duration: 212s
- consecutive unchanged rounds: 0

## Report

The compiler readiness path trusts a binary-identity version that the manifest decoder does not require to agree with the compiler component. A structurally accepted edited manifest can therefore produce a false ready verdict.

Review comment:

- [P2] Reject manifests whose compiler versions disagree — /Users/vi/personal/heurema/goalrail/internal/doctor/planning.go:265-267
  When a user-edited manifest remains canonical but gives the compiler component and its binary identity different versions, `DecodeSetupBundleManifest` accepts it because it only binds the identity to a file (`internal/releasebundle/verify.go:258-266`). This code now reports the identity's version instead of the matched component's version, so an identity at `1.6.0` with a component at `9.9.9` and valid file digest is reported `ready` for a `1.6.0` profile, contrary to the exact-version requirement. Reject this inconsistency or retain the component version as the observed compiler version.


## Disposition

Accepted. The reader preferred the entrypoint version over the component version
without requiring them to agree, so a self-contradicting manifest reported ready.
Both are now read and required to agree, and disagreement is reported as an
invalid component. The runtime path carried the same defect without appearing in
this round diff; the check and the regression cover both.

Regression: TestDisagreeingManifestVersionsAreNotReady, failing before the fix
with `a self-contradicting manifest produced "ready", want "invalid"` for both
the runtime and the compiler subcase.
