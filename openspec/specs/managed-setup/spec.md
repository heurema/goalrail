# managed-setup Specification

## Purpose

Define exact read-only setup planning, explicit digest-bound authorization, bounded installation, rollback, and setup receipts.
## Requirements
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

### Requirement: Setup execution requires explicit authority for the exact plan
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4

Setup execution SHALL require explicit owner authorization bound to the exact complete plan digest. General approval of a feature, an earlier plan, or Goalrail participation MUST NOT authorize a changed plan. Before the first mutation, the executor SHALL re-resolve the current declaration, plan bytes, prerequisites, destinations, and relevant existing state; any material difference SHALL invalidate the authorization and require a new plan.

No consent or an explicit refusal SHALL produce zero mutations. One authorization MAY cover every action already enumerated in the plan, but MUST NOT cover a new dependency, login, credential action, global configuration change, privilege escalation, provider trust record, or external service mutation discovered during execution.

#### Scenario: Owner declines setup
- **WHEN** the owner declines or does not provide authorization for the exact plan digest
- **THEN** setup performs zero mutations and returns a bounded no-write result

#### Scenario: Exact plan is authorized
- **WHEN** the owner explicitly authorizes the current complete plan digest
- **THEN** the executor may perform only the listed actions and no others

#### Scenario: State changes after authorization
- **WHEN** a destination, prerequisite, declaration, component identity, or plan byte changes between authorization and the first affected action
- **THEN** execution stops before that action and requires a newly resolved plan and authorization

#### Scenario: Scaffold requires its own trust action
- **WHEN** a supported scaffold requires an interactive trust decision that Goalrail is prohibited from minting
- **THEN** setup names that user action, does not write or simulate the trust record, and resumes verification only after the scaffold exposes the user's decision

### Requirement: Authorized setup is bounded, integrity-checked, and recoverable
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4

For an authorized plan, Goalrail or the supported agent adapter SHALL verify every fetched artifact against the exact integrity identity before durable placement. Executables SHALL be placed only at the enumerated durable user-local location; temporary download locations SHALL not become registered executable paths. Repository-local and user-local attachment mutations SHALL preserve unrelated content and MUST NOT escape their planned paths, follow unsafe links, or modify shared/global configuration not named in the plan.

Execution SHALL apply each mutation with a just-in-time state check and retain the bounded prior bytes or inverse action needed for rollback. Failure SHALL stop the plan, report completed and unperformed actions, and leave every completed mutation either working or recoverable. Setup MUST NOT write material project code, semantic intent, credentials, raw prompts, transcripts, or secret-shaped data.

#### Scenario: Artifact digest differs
- **WHEN** a downloaded component does not match the plan's integrity identity
- **THEN** it is not installed, later actions do not run, and the failure report names the affected component without retaining unsafe bytes as trusted state

#### Scenario: Temporary directory is removed
- **WHEN** authorized setup succeeds and its temporary download directory is later deleted
- **THEN** every registered executable still resolves to the planned durable location

#### Scenario: Existing configuration changes concurrently
- **WHEN** a planned configuration target differs at its just-in-time check
- **THEN** setup leaves the latest bytes untouched, stops, and requires a new plan rather than overwriting the concurrent change

#### Scenario: Setup fails after partial progress
- **WHEN** an action fails after earlier authorized actions completed
- **THEN** the result lists completed and pending actions and provides a verified bounded rollback for every completed mutation

### Requirement: Setup emits a bounded receipt and returns to the original intent gate
**Intent IDs:** OUT-4, SIG-3, SIG-4

Every authorized attempt SHALL emit a setup receipt with a stable schema identifier, project identity, plan digest, authorization reference, component versions and integrity identities, mutation outcomes, rollback reference, diagnosis result, terminal status, and a bounded continuation reference to the initiating task. A refusal MAY emit a no-write receipt but MUST retain no feature prompt or other raw conversation content.

A successful local setup SHALL run diagnosis and report local readiness separately from shared-admission activation. The supported agent SHALL then return to the original request at the candidate Intent Snapshot step; setup success MUST NOT confirm intent or authorize code implementation. A terminal failure or unlisted enabling action SHALL keep the original request paused and state the exact next owner action.

#### Scenario: Setup completes locally
- **WHEN** every authorized action and local verification succeeds
- **THEN** a success receipt is emitted, diagnosis reports the exact readiness layers, and the agent resumes the initiating request at candidate intent with zero setup-derived implementation authority

#### Scenario: Shared admission is not active
- **WHEN** local installation and attachment succeed but no verified protected shared check exists
- **THEN** the receipt reports local readiness and shared admission as inactive or unknown rather than calling the project fully enforced

#### Scenario: Receipt would copy the request
- **WHEN** receipt construction is given a raw feature prompt, transcript, credential, or source body
- **THEN** receipt validation rejects the prohibited payload and retains only the bounded continuation reference

#### Scenario: Setup reaches an unlisted enabling action
- **WHEN** execution discovers a dependency, authentication, credential, global setting, or external mutation absent from the authorized plan
- **THEN** the attempt stops with that exact blocker and the original request remains paused for separate permission

