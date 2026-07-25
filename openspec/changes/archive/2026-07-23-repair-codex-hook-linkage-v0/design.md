## Context

Intent Snapshot `intent-repair-codex-hook-linkage-v0` version 1 is confirmed.
This design repairs only the locally rendered Codex `SessionStart` hook
contract and grants no provider-run, receipt-mutation, Git, or external-effect
authority.

The consumed run
`run-cad86018a6a3e1fe9241249ad3d48cef` is terminal and correctly fail-closed.
Its capsule emitted:

```text
hooks.SessionStart=[{command=[<capsule>,"hook"]}]
```

Pinned Codex `rust-v0.144.4` instead models `SessionStart` as a list of matcher
groups. Each matcher group contains a nested `hooks` list, and a command
handler contains `type="command"` plus one string `command`. Because the
matcher-group decoder tolerates unknown fields, the spent top-level `command`
was ignored and the missing nested list defaulted to empty. No handler
connected to Goalrail's private IPC, so the adapter returned
`INVALID_HOOK_INPUT`.

The provider payload schema already matches the bounded fields consumed by
`BindLocalRunLifecycleHook`; correlation parsing is not the root cause.
Codex executes a command handler through a shell and writes the lifecycle JSON
to its stdin, so the executable path must be shell-safe inside the provider's
single command string.

## Goals / Non-Goals

**Goals:**

- Place one exact, testable `SessionStart` override renderer at the existing
  Codex adapter boundary.
- Emit exactly one matcher group containing exactly one nested synchronous
  command handler.
- Safely quote one absolute capsule executable followed by the fixed `hook`
  subcommand.
- Prove the spent shape has zero handlers under a pinned-contract fixture and
  prove the repaired shape has exactly one.
- Reuse the existing private one-shot IPC and fail-closed correlation path
  without retaining raw provider payloads.
- Preserve the failed v1 run, state evidence, capsule, task diff, and unchanged
  base/terminal `HEAD`.

**Non-Goals:**

- Editing, rebuilding, rerunning, resuming, or cleaning the spent v1 capsule.
- Creating or activating a replacement WorkSpec, run ID, claim, receipt, or
  provider process.
- Parsing arbitrary Codex configuration or supporting multiple Codex versions,
  lifecycle events, handler types, commands, or providers.
- Persistent provider configuration, credentials, alternate identity,
  sandbox or approval changes, retry, background work, extra agents,
  automatic Git, or external effects.
- New dependencies, services, workflow engines, authorization engines, effect
  gateways, sandbox platforms, or removal of retained target architecture.

## Decisions

### 1. Put the repair in the existing internal Codex adapter

Add a small adapter-owned renderer in
`internal/adapters/codex/hook_config.go`. Its narrow public surface within the
repository accepts only an absolute capsule executable and returns the exact
invocation-local override for the supported `SessionStart` contract. The
subcommand is fixed to `hook`; callers cannot supply arbitrary shell fragments,
hook events, or handler lists.

The renderer rejects non-absolute paths and NUL, CR, or LF input. It cleans the
path, applies POSIX single-argument shell quoting (including embedded single
quotes), appends the fixed subcommand, and encodes that command as one TOML
basic string in this exact structural shape:

```text
hooks.SessionStart=[{hooks=[{type="command",command=<quoted-string>}]}]
```

This package is already provider-specific and already owns Codex lifecycle
correlation. The WorkSpec and canonical local-run domain therefore remain
provider-neutral.

**Alternative rejected:** Patch
`.goalrail/activate-dogfood-run-v1/contract.go`. That capsule belongs to a
consumed one-shot boundary; changing it would weaken the historical evidence
and make accidental reuse easier.

**Alternative rejected:** Add the renderer to canonical WorkSpec or
`internal/localrun`. Provider hook syntax is not canonical run intent.

**Alternative rejected:** Create a generic activation/configuration service.
This repair needs one exact provider contract, not standing activation
authority.

### 2. Generate one fixed contract instead of accepting or parsing raw TOML

Production code constructs one typed, fixed shape and serializes only the
bounded executable-derived command. It neither accepts raw override text nor
implements a general TOML parser. This makes invalid cardinality and missing
handler fields unrepresentable through the renderer.

`hook_config_test.go` carries a narrow test-only model of the pinned upstream
`MatcherGroup` and command-handler fields. The regression fixture represents
the spent top-level `command` as an unknown matcher-group field and therefore
counts zero nested handlers. The repaired fixture counts exactly one handler
and is paired with an exact rendered-string assertion. Quoting cases cover
spaces, shell metacharacters, and embedded single quotes; unsafe paths fail
without a partial result.

**Alternative rejected:** Add a TOML dependency solely to round-trip one fixed
string. It would expand the dependency and accepted-input surface without
strengthening the runtime boundary.

**Alternative rejected:** Invoke Codex as a test parser. This milestone
explicitly stops before another Codex process; actual integration remains a
separate owner-gated canary.

### 3. Reuse the existing correlation path unchanged

The repair does not introduce another identity source. Focused tests pass one
pinned-schema-compatible `SessionStart` payload through
`BindLocalRunLifecycleHook` with the immutable local-run context and verify the
existing provider-authoritative lineage. The existing negative matrix remains
authoritative for empty, malformed, conflicting, and unsupported payloads and
must continue returning explicit unlinked reasons.

The hook command's stdin remains bounded private one-shot IPC owned by a future
exact activation capsule. Raw hook JSON, transcripts, terminal output,
credentials, and provider-only fields are not persisted or copied into the
WorkSpec or receipt.

### 4. Make future activation consume the repair explicitly

This change creates and tests the adapter boundary but does not modify a spent
capsule or create a new one. Any future owner-confirmed activation must use a
new WorkSpec and one-shot capsule, import this renderer instead of duplicating
hook syntax, and separately prove the exact installed Codex contract before
launch.

Planning or implementing this repair does not activate that future path.

## Correlation and Evidence

- Stable source evidence is the existing v1 run ID, launch claim, provider
  observation, terminal receipt, frozen WorkSpec digest, unchanged
  base/terminal `HEAD`, and exact three-path task diff.
- Repair evidence is deterministic: exact renderer output, handler cardinality
  under the pinned fixture, safe quoting cases, one positive correlation case,
  and the existing unlinked negative matrix.
- The implementation cycle records only Go test outcomes and local Git/state
  comparisons. It creates no run ID, provider session, claim, observation, or
  receipt.
- A real provider callback is deliberately not claimed as evidence by this
  change. That proof belongs to a separately authorized activation.

## Measurement and Stop Conditions

The implementation slice passes only when:

1. the legacy fixture resolves to zero nested handlers;
2. the renderer produces exactly one supported handler and rejects unsafe
   input;
3. focused correlation coverage verifies one supported payload and preserves
   all unlinked negative cases;
4. `go test ./internal/adapters/codex` exits 0;
5. `go test ./...` exits 0;
6. no new provider attempt or Goalrail run state exists;
7. the source v1 claim, observation, receipt, capsule, `HEAD`, and three-path
   task diff are unchanged; and
8. the final implementation diff is limited to the Codex adapter repair,
   focused tests, and this change's planning artifacts.

Stop immediately on an unexpected source-evidence mutation, a need to launch
Codex, a new dependency requirement, scope escape, or a failing repository
test. Do not retry, fall back, repair the source receipt, or continue into a
provider run.

## Risks / Trade-offs

- **[Pinned Codex may change its hook contract]** → Bind the fixture and design
  evidence to `rust-v0.144.4`; a later version requires a new compatibility
  decision and tests.
- **[Shell execution remains a provider behavior]** → Accept only one absolute
  executable, use a fixed subcommand, quote one shell argument explicitly, and
  reject control characters.
- **[A fixed-string test could drift from the fixture]** → Assert both the
  exact rendered string and the one-handler pinned-contract model in the same
  focused test.
- **[Local proof is not live integration proof]** → State that limitation
  explicitly and preserve a separate owner gate for the next exact provider
  canary.
- **[Dormant adapter code could be bypassed by future capsule duplication]** →
  Require the next activation design and tests to consume this renderer
  directly.

## Rollback

Before any future activation consumes the helper, rollback is deletion of the
new adapter renderer and its tests. Existing canonical schemas, WorkSpec,
local-run state, provider observation, spent v1 capsule, and retained receipt
need no migration or rewrite.

If implementation evidence fails, leave the spent v1 evidence untouched,
report the bounded failure, and stop. Do not restore functionality by editing
the consumed capsule or weakening fail-closed correlation.

## Open Questions

None.
