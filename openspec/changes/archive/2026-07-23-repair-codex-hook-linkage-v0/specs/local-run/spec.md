## MODIFIED Requirements

### Requirement: Run lineage uses provider-authoritative identity
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, SIG-1, SIG-2, SIG-4

Each launched run SHALL bind its generated run ID and frozen WorkSpec digest to
the exact root session identity returned by the provider-authoritative launch
receipt. Missing, malformed, or conflicting identity MUST remain explicit and
MUST NOT be repaired through manual entry, prompt parsing, transcript parsing,
provider terminal text, or heuristic matching.

For the exact supported Codex contract, an invocation-local `SessionStart`
definition used for lineage SHALL contain exactly one matcher group with
exactly one nested synchronous command handler. The handler SHALL declare
`type="command"` and one safely quoted command string that invokes the absolute
local capsule executable with the fixed `hook` subcommand. Goalrail MUST reject
unsafe handler input and the spent flattened v1 shape instead of emitting or
accepting a definition that registers no handler.

The bounded provider lifecycle payload SHALL travel through private one-shot
IPC to the existing correlation path. Goalrail MUST NOT retain the raw payload
or add provider-specific hook fields to the WorkSpec. Deterministic contract
and correlation tests MUST prove this boundary without launching Codex or
creating another provider attempt.

#### Scenario: Exact Codex hook definition is rendered
- **WHEN** Goalrail receives a valid absolute local capsule executable for the exact supported Codex contract
- **THEN** it renders one `SessionStart` matcher group containing exactly one nested synchronous command handler with a safely quoted capsule command

#### Scenario: Spent flattened hook definition is rejected
- **WHEN** the old v1 shape places a command array directly on the matcher group and omits the nested handler list
- **THEN** deterministic contract validation identifies zero registered handlers and Goalrail does not treat the shape as usable lineage configuration

#### Scenario: Unsafe hook command input is supplied
- **WHEN** the capsule executable is non-absolute or contains input that cannot be safely represented by the bounded handler command
- **THEN** rendering fails without returning a partial hook definition or starting a provider

#### Scenario: Provider returns verified root identity
- **WHEN** one pinned-schema-compatible `SessionStart` payload reaches the private one-shot IPC for the immutable run context
- **THEN** Goalrail records the verified WorkSpec-to-run-to-session lineage and retains no raw hook payload

#### Scenario: Root identity is missing or conflicting
- **WHEN** the adapter returns no root identity, malformed identity, or identity that conflicts with the immutable run context
- **THEN** Goalrail records an unlinked terminal outcome and does not guess or accept a manual replacement
