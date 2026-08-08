## MODIFIED Requirements

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
