## MODIFIED Requirements

### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-1, OUT-2, OUT-3, SIG-1, SIG-2, SIG-4

The `gr` operator surface SHALL resolve and validate the requested WorkSpec, freeze its canonical content, and display the exact snapshot and digest before launch. Preparation MUST NOT launch a provider or mutate the target repository worktree.

Before freezing the WorkSpec, preparation SHALL verify the exact referenced
confirmed intent bytes and digest, then resolve the sibling Context Pack that
the intent declares and parse the confirmed intent against it. The intent
artifact and its Context Pack MUST both resolve inside the canonical repository
root, and each MUST remain within the same bounded artifact size limit.

A declared Context Pack that is missing, unreadable, malformed, mismatched,
oversized, or resolving outside the canonical repository root, and any context
reference that the pack does not contain, MUST fail preparation before any
prepared state is written and before any run ID, launch claim, or provider
invocation exists. An intent that declares no Context Pack SHALL remain valid
and MUST NOT be rejected for the absence of a sibling artifact.

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
- **WHEN** preparation reads the exact confirmed intent and its declared sibling Context Pack, both resolving inside the canonical repository root
- **THEN** the intent is parsed against that pack, every context reference it uses is present, and the WorkSpec is frozen

#### Scenario: Declared Context Pack is absent
- **WHEN** the confirmed intent declares a Context Pack but no sibling context artifact resolves next to it
- **THEN** preparation fails before writing prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Context Pack is invalid or escapes the repository
- **WHEN** the sibling Context Pack is unreadable, malformed, oversized, identifies a different pack than the intent declares, omits a referenced context item, or resolves outside the canonical repository root
- **THEN** preparation fails before writing prepared state, and no run ID, launch claim, or provider invocation is created

#### Scenario: Intent declares no Context Pack
- **WHEN** the confirmed intent declares no Context Pack and no sibling context artifact exists
- **THEN** preparation verifies the intent alone and freezes the WorkSpec without requiring a pack
