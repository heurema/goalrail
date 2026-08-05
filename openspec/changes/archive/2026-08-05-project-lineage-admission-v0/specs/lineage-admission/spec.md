## ADDED Requirements

### Requirement: Committed policy deterministically classifies material changes
**Intent IDs:** OUT-5, OUT-7, OUT-8, SIG-5, SIG-9, SIG-10

Every managed project SHALL carry one versioned materiality and exception policy referenced by its project declaration. Given the same policy identity and normalized changed-path metadata, classification SHALL be deterministic and byte-stable. Material source, test, build, configuration, migration, interface, and governing documentation paths SHALL require lineage unless a narrower committed rule classifies them otherwise.

Policy rules SHALL be explicit, ordered, non-overlapping or deterministically precedence-resolved, and bounded to repository-relative paths and observable change kinds. Unknown paths, conflicting rules, invalid policy, or classification that depends on free-form interpretation SHALL fail closed as ambiguous. Diff size, authoring tool, commit message prose, generated-file claims, or file extension alone MUST NOT create an implicit exemption.

#### Scenario: Material source file changes
- **WHEN** a bounded diff changes a source path classified material by the committed policy
- **THEN** the verifier requires one matching work-unit lineage before admission

#### Scenario: Policy classifies a generated path outside material scope
- **WHEN** a narrow committed rule identifies a reproducible generated path and its required generator evidence verifies
- **THEN** the path receives that deterministic classification without a free-form exemption

#### Scenario: Two rules conflict
- **WHEN** two applicable rules have no deterministic precedence or produce different materiality
- **THEN** classification is ambiguous and shared admission is denied

#### Scenario: One-line change claims to be harmless
- **WHEN** a material or unknown path changes and the only justification is that the diff is small
- **THEN** the change remains material or ambiguous and cannot bypass lineage

### Requirement: One verifier produces categorical, machine-readable admission evidence
**Intent IDs:** OUT-7, OUT-8, SIG-5, SIG-7, SIG-8, SIG-10

Goalrail SHALL expose one deterministic verifier over an explicit repository, immutable base and head revisions or an equivalently frozen local range, committed policy identity, work-unit identity, and bounded provider-neutral evidence packet. It SHALL verify the diff, materiality classification, declaration and policy digests, every required lineage edge, exception evidence, and lifecycle phase without reaching a provider or mutating repository, user, or external state.

The verifier SHALL emit a versioned canonical result containing at least the input range and identities, material paths, work-unit reference, classification, admission decision, stable reason identifiers, missing or conflicting evidence references, and verifier version. Classifications SHALL include `VALID`, `MISSING`, `AMBIGUOUS`, `INVALID`, `EXEMPTED`, `BREAK_GLASS`, and `BOOTSTRAP`. Only `VALID` represents normal Goalrail conformance. `MISSING`, `AMBIGUOUS`, and `INVALID` SHALL always deny admission. Exception classifications MAY allow admission only when the exact committed policy says so, but MUST remain visibly non-`VALID` in every output and receipt.

Equivalent semantic inputs SHALL produce byte-identical canonical results. Wall-clock time, absolute checkout path, map order, network state, provider availability, or unstated environment variables MUST NOT affect the result; time-bounded policy evaluation SHALL use an explicit trusted evaluation time carried in the evidence packet.

#### Scenario: Normal lineage is complete
- **WHEN** one material diff has a complete, unambiguous, policy-conformant work unit and every integrity binding verifies
- **THEN** the verifier emits `VALID`, admission allowed, and no missing or conflicting relation

#### Scenario: Required receipt is missing
- **WHEN** a material diff has no terminal receipt required at its lifecycle phase
- **THEN** the verifier emits `MISSING`, admission denied, and the stable reason identifies the missing receipt relation

#### Scenario: Evidence conflicts
- **WHEN** the packet contains incompatible bindings for a required single-valued relation
- **THEN** the verifier emits `AMBIGUOUS`, admission denied, and retains references to every claimant

#### Scenario: Authorized exception permits integration
- **WHEN** committed policy permits the exact current exception evidence and scope
- **THEN** admission may be allowed while classification remains `EXEMPTED`, `BREAK_GLASS`, or `BOOTSTRAP` and never becomes `VALID`

#### Scenario: Verification is repeated elsewhere
- **WHEN** local and shared invocations receive semantically identical frozen inputs
- **THEN** their canonical result bytes and process admission outcome are identical

### Requirement: Local checks are advisory and shared admission is authoritative
**Intent IDs:** OUT-7, OUT-10, SIG-7, SIG-12

The same verifier SHALL be callable as an explicit operator command, by supported-agent pre-write or pre-finish checks, by optional Git hook shims or `prek`, and by a shared repository check adapter. Local integrations SHALL improve feedback but MUST NOT claim to be unbypassable, MUST NOT be required to discover managed-project identity, and MUST NOT create a second verifier implementation.

A shared adapter SHALL freeze provider-derived pull-request, review, check, and owner-decision evidence into the bounded provider-neutral packet before invoking the core verifier. Provider metadata, commit trailers, or a successful local hook alone MUST NOT mint missing canonical lineage. Removing or bypassing local integration SHALL not change the shared verdict for the same committed range and evidence.

Goalrail SHALL describe enforcement as protected only when activation of the required shared check at the integration boundary has been independently verified. A generated workflow file, a passing ad hoc run, or configured local hooks alone SHALL be reported as advisory or activation-unknown. Goalrail MUST NOT enable branch protection, required checks, or another external policy without separate owner authorization.

#### Scenario: Local hook is bypassed
- **WHEN** a material diff is committed after local hooks were skipped, removed, or never installed
- **THEN** the shared adapter invokes the same verifier and denies the same missing or invalid lineage

#### Scenario: Optional prek is absent
- **WHEN** a project uses Goalrail hook shims without `prek` installed
- **THEN** managed identity and verifier behavior remain available with no second mandatory dependency

#### Scenario: Workflow exists but is not required
- **WHEN** a shared-check workflow is committed but protected integration has not been verified to require it
- **THEN** diagnosis reports advisory or activation-unknown and never reports project-wide enforcement healthy

#### Scenario: Required check is verified active
- **WHEN** a registered adapter provides current read-only evidence that the integration boundary requires the Goalrail check
- **THEN** diagnosis may report shared admission active while retaining the evidence source and observation time

#### Scenario: External activation would be attempted
- **WHEN** initialization, setup, update, or verification would enable branch protection or mutate an external repository setting without a separate exact authorization
- **THEN** the action is refused and the local artifacts remain only a prepared integration

### Requirement: Admission packets are bounded, attributable, and non-authoritative by themselves
**Intent IDs:** OUT-6, OUT-7, SIG-6, SIG-10

A shared or local adapter SHALL emit a bounded admission packet with a stable schema identifier, project and policy identity, base and head, work-unit reference, trusted evaluation time when needed, and references plus integrity or provider-authenticated provenance for external evidence. The packet SHALL contain no raw prompt, transcript, comment or review body, source body, credential, token, or secret-shaped data.

The packet is evidence input, not a source of project intent or policy. The verifier SHALL resolve repository-owned identities against the checked-out committed bytes and reject substitution, path escape, unsupported schemas, untrusted timestamps, or an adapter claiming authority it cannot prove.

#### Scenario: Shared adapter binds pull-request review evidence
- **WHEN** a registered provider adapter observes the current pull request, reviews, checks, and owner decision
- **THEN** it emits bounded authenticated references and identities without copying bodies into the packet

#### Scenario: Packet attempts to replace policy
- **WHEN** an admission packet supplies materiality or exception rules different from the committed policy identity
- **THEN** the verifier rejects the packet as invalid rather than evaluating the supplied rules

#### Scenario: Packet contains a token
- **WHEN** packet construction receives a credential, token, raw review body, prompt, transcript, or source body
- **THEN** validation fails before the prohibited content is retained or printed

#### Scenario: Evaluation time is untrusted
- **WHEN** an exception depends on time and the packet cannot establish the evaluation time through the registered adapter contract
- **THEN** the exception cannot allow admission and the result names the missing trusted-time evidence
