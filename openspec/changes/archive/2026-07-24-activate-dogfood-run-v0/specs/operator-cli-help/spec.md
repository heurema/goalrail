## ADDED Requirements

### Requirement: Gr exposes a concise lifecycle help surface
**Intent IDs:** OUT-2, SIG-3

The `gr help` command SHALL describe the `prepare → inspect → start → finish`
lifecycle, state that start remains explicit and owner-assisted, and identify
how to obtain help for each command. The help text MUST NOT expose fixture-only
or broader activation controls.

#### Scenario: Operator requests top-level help
- **WHEN** an operator invokes `gr help`
- **THEN** the command prints the concise lifecycle and points to help for
  `prepare`, `inspect`, `start`, and `finish`

### Requirement: Each Gr command exposes built-in flag guidance
**Intent IDs:** OUT-2, SIG-3

Each supported `gr` command SHALL expose built-in help for its flags and
required arguments. Adding help MUST NOT change the JSON output or execution
semantics of successful `prepare`, `inspect`, `start`, or `finish` commands.

#### Scenario: Operator requests command help
- **WHEN** an operator invokes a supported command with `--help`
- **THEN** the command prints its flag guidance without invoking the lifecycle

#### Scenario: Existing machine-readable command succeeds
- **WHEN** an operator invokes a supported command with valid non-help input
- **THEN** its existing JSON result and execution semantics remain unchanged
