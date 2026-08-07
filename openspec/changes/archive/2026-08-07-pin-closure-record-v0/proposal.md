## Why

The pinned dependency closure was computed on every release path and compared
against nothing, so a pin could move — or a transitive package appear, disappear,
or gain an install script — inside eighty packages of lock-file churn with no
verdict attached (OUT-1, OUT-2).

## What Changes

- Record the reviewed closure beside the pins in the release source lock:
  package count, install-script count, and closure digest.
- Refuse to load release inputs when the computed closure disagrees with that
  record, naming which of the three facts disagreed. The comparison sits where
  inputs are loaded, so building a release, verifying one, and checking the lock
  all pass through it rather than one command holding the guarantee alone.
- Record, for the runtime and for the compiler, when the pinned version was
  published upstream and when this repository adopted it. This is disclosure, not
  a gate: both dates are supplied by whoever moves the pin. The only
  inconsistency the record catches by itself is a pin claiming to predate its own
  publication.
- **BREAKING** The source lock carries a new schema identifier,
  `goalrail.setup-source-lock/v2`, because the new fields are mandatory and
  decoding is strict, so a v1 reader and the v2 validator reject each other's
  documents. The file is a build input and is published in no release asset, and
  every reader of it moves in the same commit.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Record the reviewed closure beside the pins | OUT-1, SIG-1 | NG-3, NG-6 |
| Refuse a disagreeing closure on the shared input path | OUT-2, SIG-2 | NG-3, NG-7 |
| Record publication and adoption dates as disclosure | OUT-3, SIG-3 | NG-1, NG-2 |
| Version the expanded document and register it | OUT-4, SIG-4 | NG-4, NG-5 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-channel`: a release refuses to build against a dependency closure that
  disagrees with the one recorded beside the pins, and the recorded pins disclose
  when each was published and adopted.

## Impact

- `release/setup/source-lock.json` gains a closure record and two adoption dates
  per pinned component, and its schema identifier becomes v2.
- `internal/releasebundle`: the comparison runs in `loadReleaseInputs`, so
  `Build`, `Verify` and `CheckSourceLock` inherit it.
- `docs/contracts-v0.2.md` registers the new identifier.
- `AGENTS.md` states the mechanism and says plainly that the freshness rule is
  not decided.
- The workflows that read `.platforms[]` from the same file are untouched.

## Non-Goals

- No decision about how current a pin should be, or how long a version should
  have been public before adoption. That is deferred with the deferral stated in
  the contract.
- The adoption dates are not presented as a gate, because nothing can enforce a
  waiting period on a date its own author supplies.
- No claim that a human read what changed; the mechanism makes a difference stop
  the build and require an explicit update.
- No external compatibility or migration promise for a file that is not
  published.
- No change to the supported-platform contract.
- No pinned version moves in this change.
