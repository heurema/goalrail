## MODIFIED Requirements

### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, OUT-5, SIG-1, SIG-2, SIG-3, SIG-5

The `gr` operator surface SHALL resolve and validate the requested WorkSpec, freeze its canonical content, and display the exact snapshot and digest before launch. Preparation MUST NOT launch a provider or mutate the target repository worktree.

Before any prepared state is persisted, preparation SHALL verify the exact
referenced confirmed intent bytes and digest, and SHALL parse that intent
against the Context Pack of its change. The intent artifact and its Context
Pack MUST both resolve inside the canonical repository root, MUST each be a
regular file, and MUST each remain within the same bounded artifact size limit.
A non-regular artifact MUST be rejected before it is opened, so preparation
fails with a bounded reason instead of blocking.

A Context Pack SHALL be required whenever the change loader would require one:
a current change under the project intent schema always requires it, an
archived change requires it only when its intent declares one, and a change
under another schema that declares none does not. When a Context Pack is
required and absent, unreadable, malformed, mismatched, oversized, non-regular,
or resolving outside the canonical repository root, and when any context
reference the intent uses is absent from the pack, preparation MUST fail before
any prepared state is persisted and before any run ID, launch claim, or
provider invocation exists.

This verification MUST NOT depend on proposal, specification, design, task,
provider, or activation artifacts, and MUST NOT add Context Pack fields to the
canonical WorkSpec.

#### Scenario: Operator prepares a valid run
- **WHEN** the operator prepares a valid WorkSpec
- **THEN** `gr` displays the frozen snapshot and digest and records no provider launch

#### Scenario: Preparation fails validation
- **WHEN** the WorkSpec, confirmed intent reference, repository revision, scope, checks, stop conditions, or posture is invalid
- **THEN** preparation fails with a bounded reason and no run ID or provider launch is created

#### Scenario: Context-backed confirmed intent is verified
- **WHEN** preparation reads the exact confirmed intent and the Context Pack of its change, both resolving inside the canonical repository root as regular files
- **THEN** the intent is parsed against that pack, every context reference it uses is present, and no prepared state is persisted until both succeed

#### Scenario: Required Context Pack is absent
- **WHEN** a current change under the project intent schema supplies no Context Pack, or an intent declares a pack that does not resolve next to it
- **THEN** preparation fails before persisting prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Context Pack is invalid or escapes the repository
- **WHEN** the Context Pack is unreadable, malformed, oversized, identifies a different pack than the intent declares, omits a referenced context item, or resolves outside the canonical repository root
- **THEN** preparation fails before persisting prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Context Pack is not required
- **WHEN** an archived change declares no Context Pack, or a change under another schema declares none, and no sibling context artifact exists
- **THEN** preparation verifies the intent alone and continues without requiring a pack

#### Scenario: Intent artifact or Context Pack is not a regular file
- **WHEN** either artifact resolves to a FIFO, device, directory, or other non-regular file
- **THEN** preparation rejects it with a bounded reason before opening it, and does not block waiting for the artifact
