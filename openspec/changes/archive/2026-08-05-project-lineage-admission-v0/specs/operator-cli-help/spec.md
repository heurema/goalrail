## MODIFIED Requirements

### Requirement: Gr exposes a concise lifecycle help surface
**Intent IDs:** OUT-2, OUT-4, OUT-5, OUT-7, OUT-10, SIG-2, SIG-3, SIG-5, SIG-12

The `gr help` command SHALL describe the existing `prepare → inspect → start → finish` run lifecycle, state that start remains explicit and owner-assisted, and identify built-in help for each command. It SHALL also describe the managed-project lifecycle in its dependency order: initialize or migrate project governance; diagnose project and local readiness; plan and explicitly authorize missing setup; confirm intent and select one current Goalrail change; prepare and run bounded work; inspect lineage; verify the immutable base/head range; and use the same verifier at shared admission.

Help MUST NOT state that material ordinary work needs no per-task Goalrail flow. It SHALL instead say that supported agents perform the flow through committed bootstrap guidance, while the operator can inspect each artifact and verdict. It SHALL distinguish project identity, local setup, advisory local checks, and protected shared admission, and MUST NOT call local hooks or `prek` unbypassable.

Initialization, migration, setup planning, setup execution, diagnosis, update, connection, disconnection, lineage inspection, and lineage verification SHALL be discoverable as separate operations with their authority boundaries. Help SHALL state that setup permission covers only an exact plan, that Goalrail does not automate provider trust, and that external required-check activation is separate owner-authorized work.

A superseded command name SHALL keep working during its declared migration window and name its successor. Every printed remedy SHALL name an accepted command or an exact documented user/provider action. Internal hook entry points, fixture-only controls, and broader activation controls MUST NOT appear as operator commands.

#### Scenario: Operator requests top-level help
- **WHEN** an operator invokes `gr help`
- **THEN** help presents both the bounded run lifecycle and the managed-project setup, intent/change, lineage, and admission lifecycle in dependency order

#### Scenario: Operator looks for ordinary feature flow
- **WHEN** an operator reads top-level help after cloning a managed project
- **THEN** help directs supported agents through diagnosis and exact setup when needed, then confirmed intent and one current change before material code work

#### Scenario: Operator compares local and shared checks
- **WHEN** an operator reads help for lineage verification
- **THEN** help states that local integrations are advisory and protected shared admission is authoritative only after verified activation

#### Scenario: Operator encounters an untrusted scaffold
- **WHEN** help describes setup or connection for a scaffold with its own trust step
- **THEN** it names the user's provider action and does not imply Goalrail can pre-approve trust

#### Scenario: Superseded command name is used
- **WHEN** an operator invokes a command retained for migration compatibility
- **THEN** it still runs within the declared window and its output names the current command and changed semantics

#### Scenario: Help would advertise a hook entry point
- **WHEN** generated help would list a scaffold- or Git-invoked internal hook as an operator command
- **THEN** that violates this requirement
