## 1. Pin the current behaviour before changing it

- [x] 1.1 Add a regression to `internal/doctor` that places executables named as the declared runtime and adapter first on the lookup path, each appending to a marker file, runs the default planning observer against a canon setup profile, and asserts zero recorded invocations
- [x] 1.2 Add a regression asserting that a setup profile whose runtime value is a path to an executable inside a temporary worktree causes neither a resolution nor a read of that path
- [x] 1.3 Record that both fail against the pre-change observer, with the exact output, in the change's evidence directory

## 2. Resolve the installed bundle

- [x] 2.1 Add a resolver that builds `<home>/.local/share/goalrail/bundles/<version>/<os>_<arch>` from the running binary's version and the compiled platform, matching the construction in `internal/setup`
- [x] 2.2 Make every non-release version form — pseudo-version, modified-tree marker, development marker, unknown, empty — resolve no bundle root rather than a partial path
- [x] 2.3 Table-driven test over every version form and over an absent or unresolvable home

## 3. Read and verify the manifest

- [x] 3.1 Read the installed bundle manifest from the resolved root under the pinned schema, bounded in size and refusing a non-regular file
- [x] 3.2 Refuse a manifest that is absent, unreadable, oversized, malformed, of an unknown schema, or naming a different release version or platform, each with its own stated reason
- [x] 3.3 Locate the runtime executable through the manifest's binary identities and the declared compiler component through the manifest's components and file records, never through a path built from the setup profile
- [x] 3.4 Table-driven test with one positive fixture bundle and one case per refused manifest state

## 4. Replace the probe

- [x] 4.1 Rewrite `defaultPlanningObserver.ObservePlanning` to compare the manifest's recorded version against the profile's declared version and the recorded digest against the bytes on disk, for the runtime and the compiler entrypoint
- [x] 4.2 Hash each file from the same opened descriptor whose regular-file status was checked, refusing a symbolic link rather than following it out of the bundle root
- [x] 4.3 Delete `exec.LookPath`, `exec.CommandContext`, the bounded version-output writer, and the `--version` parser from `internal/doctor/planning.go`, leaving no unused helper behind
- [x] 4.4 Map the outcomes onto the existing states: absent file to missing, version outside the declared pin to incompatible, digest mismatch to unverified integrity, agreement to ready — adding no state and renaming none

## 5. Prove the boundary holds

- [x] 5.1 Confirm the regressions from group 1 now pass, and record the passing output alongside the failing output from 1.3
- [x] 5.2 Add an assertion that `internal/doctor` imports no process-execution facility, so a returning execution fails a test rather than a review
- [x] 5.3 Add a case asserting a complete installed bundle yields planning readiness true and a machine with no bundle yields setup required with the exact missing component named
- [x] 5.4 Run `go vet ./...` and `go test ./...` and record the result

## 6. Close the loop

- [x] 6.1 Re-run `openspec validate --changes doctor-toolchain-inspection-v0`
- [x] 6.2 Run `gr review --full` on the branch and disposition every finding before opening a pull request — five rounds including a closing full pass, ten findings, all accepted and dispositioned in `evidence/review-round-*.md`
- [x] 6.3 Report the branch state to the owner and stop; committing, pushing, and opening a pull request remain separate owner decisions
