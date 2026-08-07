## MODIFIED Requirements

### Requirement: The checking toolchain is reported as a fact, not a fault
**Intent IDs:** OUT-1, OUT-2, OUT-3, OUT-4, SIG-1, SIG-2, SIG-3, SIG-4, SIG-5

Diagnosis SHALL state whether every planning runtime and pinned compiler required by the managed project's declared setup profile is available and compatible. Their absence MUST NOT erase managed-project identity or make Goalrail's native diagnosis unavailable, but it SHALL make local planning readiness false, produce `GOALRAIL_SETUP_REQUIRED`, and block supported agents from material code work.

The report SHALL distinguish a missing executable, incompatible version, unavailable package, and unverified integrity identity. Its next action SHALL enter the exact read-only setup planner rather than prescribe an ad hoc global install. A project whose declared compiler profile requires no external runtime SHALL not be failed for one.

That fact SHALL be established by inspection: diagnosis reads the bundle authorized setup installed and compares the digest each component's file carries in the installed manifest against the bytes on disk. It MUST NOT execute the runtime, a package runner, the stock planning CLI, or any component whose readiness it is reporting — a prohibition promoted elsewhere for initialization, update, and diagnosis alike, restated here because this is the one requirement that asks for the fact and is therefore where the wrong means is reached for.

No inspected filesystem path SHALL be derived from repository content. The declared setup profile states which components are required and at which versions; it MUST NOT decide which file is read, and a value it carries MUST NOT be resolved through the executable lookup path. A checked-out repository that names a path where a program name belongs therefore selects nothing.

The inspected location SHALL be resolved from the installation Goalrail performed rather than from a new pointer artifact, and a machine carrying no such installation SHALL report the components as not ready. An unrelated toolchain that happens to satisfy the declared version MUST NOT produce a ready verdict, because readiness is a claim about the installation the profile pins and not about whatever the machine also carries.

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

#### Scenario: The component would be executed to learn its version
- **WHEN** diagnosis resolves or runs the declared runtime, a package runner, or the planning CLI in order to report its identity
- **THEN** that violates this requirement, whichever component state the run would have produced

#### Scenario: A decoy is placed where the component would be executed
- **WHEN** executables carrying the declared runtime and adapter names are placed on the executable lookup path ahead of any other, each recording every invocation, and diagnosis runs in a managed project
- **THEN** neither records an invocation

#### Scenario: The declared runtime field carries a path
- **WHEN** the repository's declared runtime value is a path to a program inside the checked-out worktree rather than a program name
- **THEN** that program is neither resolved nor executed, and the component state follows from the installed bundle alone

#### Scenario: The machine carries no installed bundle
- **WHEN** diagnosis runs where Goalrail is present but authorized setup never installed the declared components, whatever unrelated toolchain the executable lookup path carries
- **THEN** the runtime and compiler are reported as not ready, the project is reported as setup required, and no unrelated installation is credited

#### Scenario: An installed component's bytes no longer match
- **WHEN** an installed component file differs from the digest its installed manifest records
- **THEN** diagnosis reports an unverified integrity identity rather than a ready component, and reports it without executing the file
