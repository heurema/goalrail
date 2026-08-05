## MODIFIED Requirements

### Requirement: Setup planning is read-only, exact, and digest-bound
**Intent IDs:** OUT-7, OUT-8, SIG-8, SIG-9, SIG-10

Goalrail's supported-agent setup path SHALL begin by resolving the committed project declaration and inspecting local prerequisites without changing repository files, user configuration, executable locations, credentials, or scaffold state. Network metadata needed to resolve published artifacts MAY be read only through the documented release channel; planning MUST NOT begin an authentication flow or retain downloaded executables.

The result SHALL be one bounded human- and machine-readable setup plan with a stable schema identifier and deterministic digest. It SHALL enumerate every component and exact version, source and integrity identity, destination, repository-local or user-local configuration mutation, hook or adapter mutation, prerequisite, required interactive trust step, expected network access, rollback action, and verification command. It SHALL state that project code writes remain zero.

A plan SHALL be complete only when every represented mutation is executable and recoverable by the verified apply and rollback contracts under the exact inspected pre-state. Planning SHALL NOT request authorization for an existing executable replacement unless the plan can retain or deterministically reconstruct the prior state required for rollback. An unresolved component, unsupported platform, unavailable integrity identity, unsafe or unrecoverable destination, or newly discovered prerequisite SHALL make the plan incomplete and non-executable.

#### Scenario: Complete clean-machine plan is produced
- **WHEN** a supported clean machine and managed project resolve every required component and every mutation is executable and recoverable under the inspected pre-state
- **THEN** setup planning emits one digest-bound complete plan and performs zero filesystem or configuration writes

#### Scenario: Runtime was not anticipated
- **WHEN** the project requires a planning runtime that the machine lacks and the setup profile does not enumerate it
- **THEN** the plan is incomplete, installation does not start, and the missing prerequisite is named for a new exact permission request

#### Scenario: Release identity cannot be verified
- **WHEN** an artifact source cannot provide the version and integrity identity required by the setup profile
- **THEN** the plan is non-executable and no artifact is installed or trusted

#### Scenario: Planning would authenticate
- **WHEN** resolving a component would require login, device authorization, credential refresh, or account binding
- **THEN** planning stops and reports that enabling action separately rather than starting it

#### Scenario: Existing executable differs and cannot be recovered
- **WHEN** the selected executable destination contains a same-size regular file whose digest differs from the desired executable and apply refuses to retain the prior bytes required for rollback
- **THEN** planning reports an executable-destination conflict, marks the plan incomplete, and emits no authorizable replacement mutation

#### Scenario: Existing executable only needs mode repair
- **WHEN** the selected executable destination contains the desired bytes with a different safe file mode and apply plus rollback can preserve the represented prior mode
- **THEN** planning may emit a complete mode-repair mutation bound to the exact existing digest and mode

#### Scenario: Existing executable is already current
- **WHEN** the selected executable destination already matches the desired digest and mode
- **THEN** planning emits no executable replacement mutation for that destination
