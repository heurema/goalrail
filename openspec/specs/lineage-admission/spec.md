# lineage-admission Specification

## Purpose

Define deterministic material-change classification, categorical verification, and the boundary between advisory local checks and authoritative shared admission.
## Requirements
### Requirement: Committed policy deterministically classifies material changes
**Intent IDs:** OUT-5, OUT-8, SIG-6, SIG-9, SIG-10

Every managed project SHALL carry one versioned materiality and exception policy referenced by its project declaration. Given the same policy identity and normalized changed-path metadata, classification SHALL be deterministic and byte-stable. Material source, test, build, configuration, migration, interface, and governing documentation paths SHALL require lineage unless a narrower committed rule classifies them otherwise.

Policy rules SHALL be explicit, ordered, non-overlapping or deterministically precedence-resolved, and bounded to repository-relative paths and observable change kinds. For a prefix matcher, the normalized path `.` SHALL represent the repository root and SHALL match every valid repository-relative changed path. A matching rule SHALL retain its configured classification, rule identity, priority, and required evidence kinds; classification MUST NOT silently fall through to a hard-coded default when the declared root rule applies. Unknown paths, conflicting rules, invalid policy, or classification that depends on free-form interpretation SHALL fail closed as ambiguous. Diff size, authoring tool, commit message prose, generated-file claims, or file extension alone MUST NOT create an implicit exemption.

#### Scenario: Material source file changes
- **WHEN** a bounded diff changes a source path classified material by the committed policy
- **THEN** the verifier requires one matching work-unit lineage before admission

#### Scenario: Policy classifies a generated path outside material scope
- **WHEN** a narrow committed rule identifies a reproducible generated path and its required generator evidence verifies
- **THEN** the path receives that deterministic classification without a free-form exemption

#### Scenario: Repository-root prefix rule applies
- **WHEN** a prefix rule with normalized matcher path `.` covers a root or nested add, modify, rename, delete, mode, or submodule change
- **THEN** that rule matches the changed path and contributes its rule identity, classification, priority, and required evidence kinds

#### Scenario: Two rules conflict
- **WHEN** two applicable rules have no deterministic precedence or produce different materiality
- **THEN** classification is ambiguous and shared admission is denied

#### Scenario: One-line change claims to be harmless
- **WHEN** a material or unknown path changes and the only justification is that the diff is small
- **THEN** the change remains material or ambiguous and cannot bypass lineage

### Requirement: One verifier produces categorical, machine-readable admission evidence
**Intent IDs:** OUT-2, OUT-3, OUT-6, OUT-8, SIG-2, SIG-3, SIG-7, SIG-9, SIG-10, OUT-3, OUT-6, SIG-1, SIG-2, SIG-3, SIG-6

Goalrail SHALL expose one deterministic verifier over an explicit repository, immutable base and head revisions or an equivalently frozen local range, committed policy identity, work-unit identity, and bounded provider-neutral evidence packet. It SHALL verify the diff, materiality classification, declaration and policy digests, every required lineage edge, semantic validity of evidence whose state affects admission, exception evidence, and lifecycle phase without reaching a provider or mutating repository, user, or external state.

Where exception evidence declares the `restoration` class, the verifier SHALL decide three further questions from the frozen range: whether the reference and digest the claim names appear together on one target the lineage records, whether the range adds or amends a policy-declared normative path inside the claim's effect scope, and whether the claim's own artifact entered the history before every commit touching a material path in that scope. The verifier MUST derive that last answer from the changed paths and commit ancestry of the range, and MUST NOT read it from a field carried by the claim. Ancestry SHALL be decided by reachability over commit parents rather than by position in an ordering.

All three are decidable from references, digests, changed paths and commit ancestry; none requires reading what the change means. A claim failing any SHALL deny admission with a stable reason identifier, and a claim recorded too late SHALL carry a different identifier from one that binds nothing valid, because the two are different defects with different remedies: the second can be corrected, and the first cannot.

The verifier MUST NOT attempt to decide whether a diff restores the bound requirement or moves its boundary. That question is semantic, is reserved to review, and an affirmative verifier result MUST NOT be reported as having settled it.

The verifier SHALL emit a versioned canonical result containing at least the input range and identities, material paths, work-unit reference, classification, admission decision, stable reason identifiers, missing or conflicting evidence references, and verifier version. Classifications SHALL include `VALID`, `MISSING`, `AMBIGUOUS`, `INVALID`, `EXEMPTED`, `BREAK_GLASS`, and `BOOTSTRAP`. Only `VALID` represents normal Goalrail conformance. `MISSING`, `AMBIGUOUS`, and `INVALID` SHALL always deny admission. Exception classifications MAY allow admission only when the exact committed policy says so, but MUST remain visibly non-`VALID` in every output and receipt.

Confirmed-intent evidence SHALL count only when the exact retained snapshot is structurally valid, digest-bound, and actively confirmed. Terminal-receipt evidence SHALL count only when the exact retained receipt is structurally valid, digest-bound, and has terminal status `passed`. Candidate or malformed intent and failed, blocked, unlinked, launch-failed, launch-attempted-unknown, or verification-incomplete receipts MUST NOT satisfy their required relations.

Equivalent semantic inputs SHALL produce byte-identical canonical results. Wall-clock time, absolute checkout path, map order, network state, provider availability, or unstated environment variables MUST NOT affect the result; time-bounded policy evaluation SHALL use an explicit trusted evaluation time carried in the evidence packet. Every packet accepted by domain validation SHALL produce a canonical result or a bounded validation error; missing time, incompatible provenance, or another negative input MUST NOT panic the verifier.

#### Scenario: Normal lineage is complete
- **WHEN** one material diff has a complete, unambiguous, policy-conformant work unit and every integrity and semantic binding verifies
- **THEN** the verifier emits `VALID`, admission allowed, and no missing or conflicting relation

#### Scenario: Required receipt is missing
- **WHEN** a material diff has no terminal receipt required at its lifecycle phase
- **THEN** the verifier emits `MISSING`, admission denied, and the stable reason identifies the missing receipt relation

#### Scenario: Terminal receipt did not pass
- **WHEN** the required terminal-receipt target decodes to any valid status other than `passed`
- **THEN** the receipt relation remains non-satisfying and normal admission is denied with a stable semantic-evidence reason

#### Scenario: Intent snapshot is not confirmed
- **WHEN** the required confirmed-intent target is malformed, candidate, or not bound to the exact retained snapshot digest
- **THEN** the intent relation remains non-satisfying and normal admission is denied with a stable intent reason

#### Scenario: Restoration claim precedes the work
- **WHEN** the commit that introduced a `restoration` claim's artifact is an ancestor of every commit touching a material path in its effect scope, and the reference and digest it names appear together on a recorded target
- **THEN** the verifier accepts the precedence and the binding, and the result is classified by the exception rather than as `VALID`

#### Scenario: Restoration claim was committed before the range
- **WHEN** the claim's artifact is not among the changed paths of the range under admission
- **THEN** it was committed before the range began, precedence holds for every commit in it, and no field of the claim is consulted to establish that

#### Scenario: Restoration claim follows the work it excuses
- **WHEN** the commit that introduced the claim's artifact is not an ancestor of some commit touching a material path in its effect scope
- **THEN** admission is denied with the stable ordering reason, and no timestamp or self-declared position recorded on the event changes the outcome

#### Scenario: Restoration claim binds nothing the lineage records
- **WHEN** the reference and digest the claim names do not appear together on any target the work unit's lineage records
- **THEN** admission is denied with the stable binding reason, which is distinct from the ordering reason

#### Scenario: The work amends a normative path in the claim's scope
- **WHEN** the range adds or amends a path the committed policy declares normative, inside the claim's effect scope
- **THEN** admission is denied, and the outcome does not depend on whether that path is the artifact the claim names

#### Scenario: Evidence conflicts
- **WHEN** the packet contains incompatible bindings for a required single-valued relation
- **THEN** the verifier emits `AMBIGUOUS`, admission denied, and retains references to every claimant

#### Scenario: Provider evidence completes late relations
- **WHEN** the committed work unit and one authenticated packet together provide every required repository and provider relation for the frozen range
- **THEN** the verifier evaluates the effective complete evidence set and may emit `VALID` without requiring provider observations to have been precommitted

#### Scenario: Provenance has no trusted evaluation time
- **WHEN** a domain-valid packet supplies non-repository provenance but no evaluation time usable under the adapter contract
- **THEN** the verifier returns a canonical non-valid result identifying missing trusted-time evidence and does not panic

#### Scenario: Authorized exception permits integration
- **WHEN** committed policy permits the exact current exception evidence and scope
- **THEN** admission may be allowed while classification remains `EXEMPTED`, `BREAK_GLASS`, or `BOOTSTRAP` and never becomes `VALID`

#### Scenario: Verification is repeated elsewhere
- **WHEN** local and shared invocations receive semantically identical frozen inputs
- **THEN** their canonical result bytes and process admission outcome are identical

### Requirement: Local checks are advisory and shared admission is authoritative
**Intent IDs:** OUT-1, OUT-2, OUT-8, SIG-1, SIG-2, SIG-9, SIG-10

The same verifier SHALL be callable as an explicit operator command, by supported-agent pre-write or pre-finish checks, by optional Git hook shims or `prek`, and by a shared repository check adapter. Local integrations SHALL improve feedback but MUST NOT claim to be unbypassable, MUST NOT be required to discover managed-project identity, and MUST NOT create a second verifier implementation.

A shared adapter SHALL freeze provider-derived pull-request, review, check, and owner-decision evidence into the bounded provider-neutral packet before invoking the core verifier. The verifier SHALL combine exact authenticated packet references with committed work-unit evidence into one deterministic effective evidence view for the current evaluation only; it MUST NOT persist, fabricate, or rewrite repository lineage while doing so. Provider metadata, commit trailers, branch-controlled actor labels, repository-root decision files, or a successful local hook alone MUST NOT mint authenticated owner authority or missing canonical lineage. An owner decision SHALL satisfy shared admission only when its actor, outcome, range, and evidence identity are authenticated by the registered provider contract or an equivalently unforgeable authority contract and match committed policy. Removing or bypassing local integration SHALL not change the shared verdict for the same committed range and evidence.

Goalrail SHALL describe enforcement as protected only when activation of the required shared check at the integration boundary has been independently verified. A generated workflow file, a passing ad hoc run, or configured local hooks alone SHALL be reported as advisory or activation-unknown. Goalrail MUST NOT enable branch protection, required checks, or another external policy without separate owner authorization.

#### Scenario: Local hook is bypassed
- **WHEN** a material diff is committed after local hooks were skipped, removed, or never installed
- **THEN** the shared adapter invokes the same verifier and denies the same missing or invalid lineage

#### Scenario: Authenticated provider observations complete the work unit
- **WHEN** the registered adapter supplies current authenticated pull-request, review, check, and owner-decision references for the exact frozen range
- **THEN** the shared verifier evaluates those references as the corresponding late relations without requiring branch-authored copies

#### Scenario: Branch claims repository-owner authority
- **WHEN** a branch-authored event or repository file claims an allowed owner actor but has no authenticated authority evidence
- **THEN** shared admission denies the owner-decision relation regardless of actor label or target cardinality

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
**Intent IDs:** OUT-1, OUT-2, OUT-6, OUT-8, SIG-1, SIG-2, SIG-7, SIG-9

A shared or local adapter SHALL emit a bounded admission packet with a stable schema identifier, project and policy identity, base and head, work-unit reference, trusted evaluation time whenever provider provenance or time-bounded evidence is evaluated, and references plus integrity or provider-authenticated provenance for external evidence. Provider provenance SHALL bind the registered adapter, provider source, evidence digest, observation time, current range, and any actor or outcome semantics used for authority. The packet SHALL contain no raw prompt, transcript, comment or review body, source body, credential, token, or secret-shaped data.

The packet is evidence input, not a source of project intent or policy. The verifier SHALL resolve repository-owned identities against the checked-out committed bytes and reject substitution, path escape, unsupported schemas, untrusted timestamps, or an adapter claiming authority it cannot prove. A repository-owned evidence reference MAY establish integrity for committed semantic artifacts, but MUST NOT by itself establish external owner identity or provider-observed approval.

#### Scenario: Shared adapter binds pull-request review evidence
- **WHEN** a registered provider adapter observes the current pull request, reviews, checks, and owner decision
- **THEN** it emits bounded authenticated references and identities without copying bodies into the packet

#### Scenario: Authenticated owner decision is supplied
- **WHEN** a registered adapter supplies an owner decision for the exact range with authenticated actor, outcome, evidence digest, and observation time matching committed policy
- **THEN** the verifier may use that decision to satisfy the owner boundary

#### Scenario: Repository file claims an owner decision
- **WHEN** a repository-root target or branch-authored event claims an owner actor without authenticated authority provenance
- **THEN** the packet cannot elevate that reference into an authorized owner decision

#### Scenario: Packet attempts to replace policy
- **WHEN** an admission packet supplies materiality or exception rules different from the committed policy identity
- **THEN** the verifier rejects the packet as invalid rather than evaluating the supplied rules

#### Scenario: Packet contains a token
- **WHEN** packet construction receives a credential, token, raw review body, prompt, transcript, or source body
- **THEN** validation fails before the prohibited content is retained or printed

#### Scenario: Provenance omits evaluation time
- **WHEN** a packet carries non-repository provenance but no trusted evaluation time
- **THEN** domain validation or verification returns a stable non-valid trusted-time result without dereferencing absent state

#### Scenario: Evaluation time is untrusted
- **WHEN** an exception or provider observation depends on time and the packet cannot establish the evaluation time through the registered adapter contract
- **THEN** the evidence cannot allow admission and the result names the missing trusted-time evidence

