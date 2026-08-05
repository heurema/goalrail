## MODIFIED Requirements

### Requirement: The agent-facing prompt carries what an agent cannot guess
**Intent IDs:** OUT-2, OUT-3, OUT-4, SIG-2, SIG-3, SIG-4

The repository bootstrap prompt and release documentation offered to a supported agent SHALL work on a machine with no Go toolchain and no existing Goalrail binary. Before any installation, the prompt SHALL tell the agent to resolve the managed project's declared setup profile, inspect the machine read-only, resolve exact compatible Goalrail and other mandatory component versions and integrity identities, and present one digest-bound setup plan with destinations, configuration or hook changes, network access, prerequisites, rollback, and zero project code writes.

The prompt SHALL require explicit owner authorization for that exact plan before downloading an executable into durable storage or changing configuration. It SHALL explain how to discover a compatible release without assuming an unpinned version, verify a partial platform download, place the binary at the enumerated durable user-local location, preserve temporary-download cleanup, attach the selected scaffold without forging trust, run diagnosis, emit a setup receipt, and return to the initiating request at the intent gate. An unlisted runtime, dependency, login, credential action, global setting, privilege escalation, or external mutation SHALL require a new plan and permission.

The prompt MUST NOT contain an executable curl-to-shell installer, require the agent to trust mutable unverified bytes, forbid the durable outside-repository placement it needs, initialize an already declared clone again, enable branch protection, or convert general feature consent into setup authority. Platform limitations and macOS quarantine behavior SHALL remain truthful and consistent with the documented command-line and browser routes.

#### Scenario: Agent plans setup on a clean managed clone
- **WHEN** a supported agent receives a feature request with no Goalrail or Go toolchain installed
- **THEN** it resolves and displays the exact integrity-verifiable setup plan and performs zero installation and project code writes before owner authorization

#### Scenario: Owner authorizes the exact plan
- **WHEN** the owner explicitly authorizes the current plan digest
- **THEN** the agent installs only its enumerated components, places executables durably, performs only listed attachment changes, verifies diagnosis, emits the setup receipt, and returns to candidate intent

#### Scenario: Owner declines the plan
- **WHEN** the owner refuses or does not authorize setup
- **THEN** the agent leaves machine and repository state unchanged and does not continue the feature implementation

#### Scenario: Download location is cleaned afterwards
- **WHEN** temporary download content is removed after successful setup
- **THEN** the registered executable and required runtime still resolve from their planned durable locations

#### Scenario: New dependency is discovered
- **WHEN** execution finds a mandatory component or enabling action absent from the authorized plan
- **THEN** it stops before that action and requests separate exact authorization

#### Scenario: Provider trust is required
- **WHEN** the scaffold requires its own interactive trust action
- **THEN** the prompt directs the user through the provider surface and forbids the agent or Goalrail from manufacturing the trust record

## ADDED Requirements

### Requirement: Published metadata supports exact pre-install planning
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4

Every published Goalrail release SHALL provide bounded machine-readable metadata sufficient to resolve an exact pre-install plan before placing the binary. The metadata SHALL identify its schema, release version, supported platforms, archive names, sizes, cryptographic digests, binary version identity, minimum compatibility facts, and the checksum artifact that independently covers the same archives. It SHALL be immutable for one release and discoverable through the documented bounded release channel.

The release process SHALL verify that metadata, checksums, archives, and embedded binary versions agree before publication. A mutable discovery response MAY point to an immutable release, but MUST NOT itself become the integrity identity. No planning metadata SHALL require an account, credential, or device authorization to read.

#### Scenario: Agent resolves one platform before installation
- **WHEN** an agent reads current discovery metadata and selects a supported platform archive
- **THEN** it can pin the immutable release, archive name, size, and digest in the setup plan without downloading or installing the executable

#### Scenario: Release artifacts disagree
- **WHEN** metadata, checksum file, archive bytes, or embedded binary version identify different releases or digests
- **THEN** release verification fails and the inconsistent set cannot be published as a usable setup bundle

#### Scenario: Discovery response changes later
- **WHEN** the mutable current-release pointer advances after a plan was authorized
- **THEN** the authorized plan remains bound to its immutable release and digest, and execution does not silently upgrade it

#### Scenario: Metadata requires authentication
- **WHEN** the documented planning metadata cannot be read without an account or credential
- **THEN** that violates the clean-machine setup contract
