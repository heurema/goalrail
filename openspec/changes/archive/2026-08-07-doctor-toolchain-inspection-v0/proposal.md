## Why

Diagnosis establishes planning-toolchain readiness by executing the declared
runtime and adapter, which a promoted requirement names as a violation, and it
resolves them through a name carried in the checked-out repository and an
arbitrary lookup path rather than through the installation Goalrail itself
performed. Everything needed to answer the same question by inspection is
already written to disk by authorized setup.

## What Changes

- Replace the diagnosis's process-execution probe for the planning runtime and
  compiler with inspection of the bundle authorized setup installed: resolve the
  durable bundle root from the running binary's own version, read the installed
  bundle manifest, and compare the recorded digest of each declared component
  file against the bytes on disk.
- Stop deriving any inspected filesystem path from the repository-owned setup
  profile. The profile continues to declare which components are required and at
  which versions; it no longer decides what is read or run.
- **BREAKING** A managed project whose machine carries no Goalrail-installed
  bundle reports its runtime and compiler as not ready and the project as setup
  required, where an unrelated toolchain on the lookup path could previously
  produce a ready verdict.
- Promote the prohibition into the diagnosis capability itself, so the
  requirement that names the four component states also names inspection as the
  way they are established, and a returning execution fails a scenario rather
  than only a distant one.
- Add a regression that fails against the current behaviour: decoy executables
  named as the declared runtime and adapter record every invocation, and the
  diagnosis must record none.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Replace execution with manifest-and-digest inspection | OUT-1, OUT-3, SIG-1, SIG-2, SIG-4 | NG-1, NG-3, NG-5 |
| Resolve the bundle root from the running binary's version | OUT-2, SIG-3 | NG-2, NG-4 |
| Report an uninstalled toolchain as setup required | OUT-4 | NG-7 |
| Promote the prohibition into the diagnosis capability | OUT-1, SIG-5 | NG-3, NG-5 |
| Add the decoy-executable regression | OUT-5, SIG-6 | NG-6 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `harness-doctor`: the requirement that reports the checking toolchain as a
  fact gains the means by which that fact is established — inspection of
  Goalrail's own installed bundle against its recorded digests, never execution
  of the component — and gains scenarios for a machine with no installed bundle
  and for a repository-supplied path in the declared runtime field.

## Impact

- `internal/doctor/planning.go`: the default planning observer stops calling
  `exec.LookPath` and `exec.CommandContext` and starts reading the installed
  bundle manifest. `--version` output parsing is no longer a source of the
  observed identity.
- `internal/doctor/doctor.go`: unchanged aggregation; the component states it
  consumes keep their existing names, so `Planning.Ready`, `LocallyReady`, the
  reason codes, and the exit statuses are untouched.
- `internal/releasebundle`: read-only use of the existing manifest contract from
  the diagnosis. No change to how bundles are built or published.
- No change to the setup profile schema, to `gr setup`, to the release process,
  or to any repository-owned governance value.
- Users who installed `gr` without running authorized setup see their diagnosis
  move from a ready verdict to a setup-required one.

## Non-Goals

- Diagnosis does not install, repair, or upgrade the planning toolchain.
- No new pointer file, registry, index, or discovery contract for the installed
  bundle.
- The execution is not moved to another command, package, or opt-in flag.
- No change to the declared profile's schema, pinned versions, or any
  repository-owned value.
- No new component state, reason code, or field in the machine contract.
- The audit's other two divergences — the unreachable scaffold registration and
  the missing adoption line — are out of scope.
- A toolchain Goalrail did not install is not treated as equivalent to one it
  did in order to preserve a previously reported ready verdict.
