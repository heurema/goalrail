## MODIFIED Requirements

### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, OUT-5, SIG-1, SIG-2, SIG-3, SIG-5

The `gr` operator surface SHALL resolve and validate the requested WorkSpec, freeze its canonical content, and display the exact snapshot and digest before launch. Preparation MUST NOT launch a provider or mutate the target repository worktree.

Before any prepared state is persisted, preparation SHALL verify the exact
referenced confirmed intent bytes and digest. When a Context Pack is present or
required, preparation SHALL additionally parse that intent against the pack,
and the pack MUST resolve inside the canonical repository root. The intent
artifact, and every Context Pack that is read, MUST resolve inside the
canonical repository root, MUST be a regular file, and MUST remain within the
same bounded artifact size limit.

Rejection of a non-regular artifact MUST NOT depend on a check that a
concurrent local process can invalidate before the artifact is read. Reading
SHALL therefore inspect the same open descriptor it reads from, and SHALL open
in a way that neither follows a substituted symbolic link nor blocks on a
substituted pipe. Preparation fails with a bounded reason instead of blocking
or reading outside the verified boundary.

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
- **WHEN** either artifact is a FIFO, device, directory, or other non-regular file
- **THEN** preparation rejects it with a bounded reason and does not block waiting for the artifact

#### Scenario: An artifact is substituted after it is resolved
- **WHEN** a concurrent local process replaces a resolved artifact with a symbolic link or a pipe between resolution and reading
- **THEN** preparation rejects it rather than following the link outside the verified boundary or blocking on the pipe, because the regular-file check inspects the same descriptor that is read
