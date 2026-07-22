## ADDED Requirements

### Requirement: Runtime-owned canary artifacts survive development lifecycle transitions
**Intent IDs:** OUT-1, SIG-1, SIG-2, SIG-3

The canary SHALL keep its current frozen manifest and default append-only evidence location under a stable provider-neutral runtime path. Runtime code MUST NOT discover or address those artifacts through an active or dated archive path owned by a replaceable development provider.

#### Scenario: Completed OpenSpec change is archived
- **WHEN** the OpenSpec change that introduced the canary is moved from the active change set into a dated archive
- **THEN** the command continues to resolve the same current frozen manifest and default evidence location without reading the active or archived change directory

#### Scenario: Repository has no OpenSpec directory
- **WHEN** the canary command uses its repository-local default store in a repository root with no `openspec` directory
- **THEN** it writes the append-only evidence chain under the stable canary runtime path and does not recreate a development-provider path

### Requirement: Current operator guidance matches executable defaults
**Intent IDs:** OUT-2, OUT-3, SIG-1, SIG-5

Current operator guidance SHALL live outside historical planning archives and SHALL name the exact default evidence path used by the canary command. Archived guidance MAY preserve historical wording but MUST NOT be presented as the current operator surface.

#### Scenario: Operator follows current guidance without a store override
- **WHEN** an operator runs the documented command without `--store`
- **THEN** the documented evidence path and the executable default resolve to `canary/intent-canary-v0/events.jsonl`

#### Scenario: Current guidance drifts from the command
- **WHEN** a code or documentation change makes the two default paths differ
- **THEN** deterministic regression validation fails before the change is accepted

### Requirement: Runtime-boundary maintenance does not activate the canary
**Intent IDs:** OUT-1, SIG-4

A development-provider archive or runtime-artifact relocation SHALL preserve the frozen canary rules and activation state. It MUST NOT create an evidence event, consume an assignment ordinal, or bypass the separate owner instruction required for real activation.

#### Scenario: Runtime artifacts are relocated before activation
- **WHEN** the frozen manifest and current guidance are moved to their stable runtime path
- **THEN** the manifest remains `frozen_not_activated`, no repository evidence chain is created, and a real assignment remains rejected
