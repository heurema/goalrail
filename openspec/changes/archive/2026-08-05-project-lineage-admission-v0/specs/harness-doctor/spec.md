## MODIFIED Requirements

### Requirement: One diagnosis covers the whole harness
**Intent IDs:** OUT-2, OUT-4, OUT-10, SIG-2, SIG-3, SIG-4, SIG-12

Goalrail SHALL expose one diagnosis surface with a versioned machine contract that reports independent layers rather than one checkout-local marker verdict. It SHALL report at least: declaration presence and validity; project identity and governance contract version; explicit v0.1.8 migration need; overlay and canon-owned bootstrap presence, drift, or behind-ness per file; repository-owned policy identity and mismatch; planning toolchain readiness; supported scaffold attachment and trust state; lineage policy readiness; prepared shared-check integration; verified shared-admission activation or the absence of such verification; optional observability; review evidence; and release-channel facts.

The report SHALL distinguish `managed`, `locally_ready`, `lineage_ready`, and `shared_admission_active`. Aggregate full enforcement SHALL be healthy only when a valid declaration, required local and planning prerequisites, current governing files, valid lineage policy, and independently verified protected shared admission are all present. A managed project with missing local setup SHALL remain `managed=true` and fail closed with `GOALRAIL_SETUP_REQUIRED`; it MUST NOT be described as uninitialized or available for ordinary unmanaged work.

Every non-working required state SHALL name one exact next action the tool or documented workflow can actually perform. Optional absence SHALL remain informational. Drift SHALL be named per path and ownership class. Structured output and exit status SHALL distinguish unmanaged, declared-invalid, setup-required, locally ready but advisory, and fully enforced states. A bounded compatibility field named `initialized` MAY mirror valid declaration presence for one migration window, but it MUST be labeled deprecated and MUST NOT retain marker semantics.

#### Scenario: Repository was never managed
- **WHEN** diagnosis runs at a repository root with no declaration path and no legacy Goalrail project evidence
- **THEN** it reports unmanaged, performs no managed-project observation or write, and does not prescribe setup as though the project had opted in

#### Scenario: Managed clone lacks local setup
- **WHEN** a valid declaration is present but required binary, planning runtime, or scaffold attachment is absent
- **THEN** diagnosis reports `managed=true`, the exact local readiness failures, `GOALRAIL_SETUP_REQUIRED`, and no fully enforced verdict

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path exists but its schema, identity, policy reference, location, or bytes are invalid
- **THEN** diagnosis reports declared-invalid and fails closed rather than reporting unmanaged

#### Scenario: Canon-owned file has drifted
- **WHEN** an overlay or managed bootstrap path differs from every canon known to the installed binary
- **THEN** diagnosis names that exact path and ownership class and reports drift rather than a general mismatch

#### Scenario: Repository-owned policy mismatches
- **WHEN** current policy bytes do not match the identity referenced by the declaration
- **THEN** diagnosis reports governance mismatch and does not prescribe an automatic overwrite

#### Scenario: Local checks exist without protected admission
- **WHEN** local hooks and a shared workflow file exist but required-check activation has not been independently verified
- **THEN** diagnosis reports local readiness or advisory integration and `shared_admission_active=false` or unknown, never full enforcement

#### Scenario: Every required layer is verified
- **WHEN** declaration, governing content, planning prerequisites, local attachment, lineage policy, and protected shared admission all verify
- **THEN** diagnosis reports each layer healthy and full enforcement working

#### Scenario: Diagnosis is consumed by a machine
- **WHEN** diagnosis is invoked for automated use
- **THEN** it emits the current schema version, categorical layer states, stable reason identifiers, and an exit status consistent with the aggregate state

#### Scenario: Next action names a nonexistent operation
- **WHEN** any required-state remedy names a command or documented user action that cannot perform the repair
- **THEN** that violates this requirement

### Requirement: The checking toolchain is reported as a fact, not a fault
**Intent IDs:** OUT-2, OUT-4, OUT-5, SIG-2, SIG-3, SIG-5

Diagnosis SHALL state whether every planning runtime and pinned compiler required by the managed project's declared setup profile is available and compatible. Their absence MUST NOT erase managed-project identity or make Goalrail's native diagnosis unavailable, but it SHALL make local planning readiness false, produce `GOALRAIL_SETUP_REQUIRED`, and block supported agents from material code work.

The report SHALL distinguish a missing executable, incompatible version, unavailable package, and unverified integrity identity. Its next action SHALL enter the exact read-only setup planner rather than prescribe an ad hoc global install. A project whose declared compiler profile requires no external runtime SHALL not be failed for one.

#### Scenario: Required planning runtime is absent
- **WHEN** diagnosis runs in a managed project whose setup profile requires a runtime that is not available
- **THEN** project identity remains managed, planning readiness is false, material work is blocked, and the exact setup-planning action is named

#### Scenario: Required compiler version differs
- **WHEN** an installed compiler does not satisfy the exact project setup profile
- **THEN** diagnosis reports the observed and required identities and routes through setup planning without changing either installation

#### Scenario: Planning toolchain is complete
- **WHEN** every runtime and compiler named by the setup profile verifies
- **THEN** planning readiness is true without implying that shared admission is active

#### Scenario: No external runtime is required
- **WHEN** the declared planning profile is entirely provided by verified installed Goalrail capabilities
- **THEN** diagnosis reports planning readiness without inventing a Node or other runtime requirement

## ADDED Requirements

### Requirement: Shared-admission health is evidence-backed and never inferred
**Intent IDs:** OUT-7, OUT-10, SIG-7, SIG-12

Diagnosis SHALL report shared-admission activation separately for every configured integration boundary. An active verdict SHALL require current read-only evidence from a registered provider adapter or another independently verifiable activation receipt that the Goalrail check is required for the protected target. A workflow file, successful check run, local hook, commit trailer, or owner intention alone MUST NOT prove activation.

The report SHALL name the provider or boundary, required check identity, observed target, evidence source, observation time, and freshness status without exposing credentials. Missing access SHALL produce unknown, not inactive or active. Diagnosis MUST NOT enable, repair, or authenticate an external integration.

#### Scenario: Required check evidence is current
- **WHEN** the registered adapter verifies that the protected target currently requires the exact Goalrail admission check
- **THEN** diagnosis reports shared admission active with bounded provenance and observation time

#### Scenario: Provider cannot be queried
- **WHEN** current activation evidence requires unavailable access
- **THEN** diagnosis reports activation unknown and does not reuse an expired observation as current proof

#### Scenario: Only a workflow file exists
- **WHEN** the repository contains the shared-check workflow but no verified required-check setting
- **THEN** diagnosis reports the integration prepared or advisory and full enforcement remains false
