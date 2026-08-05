## MODIFIED Requirements

### Requirement: Attachment acts only in initialized directories
**Intent IDs:** OUT-1, OUT-2, OUT-10, SIG-1, SIG-2, SIG-12

The first act of every persistent hook invocation SHALL be to resolve the session's Git worktree root and inspect only the canonical Goalrail project declaration path. In a repository with no declaration claim, the hook MUST exit immediately, reading no event payload, writing nothing, and retaining no observation. A valid declaration SHALL make ambient behavior eligible in every clone and linked worktree, independent of any legacy marker; local attachment health remains a separate fact.

If the reserved declaration path exists but is invalid, the hook MUST fail closed before reading the event payload or retaining session data. It MAY emit one fixed bounded diagnostic directing the supported agent to `gr doctor`, but MUST NOT infer unmanaged status or inspect prompt-, transcript-, credential-, or authorization-bearing payload fields.

Observing sessions outside declared projects remains prohibited. A persistent user-scope hook can fire for every session, so declaration discovery MUST precede payload parsing and MUST NOT follow a symlink or path outside the resolved worktree.

#### Scenario: Session in an unmanaged directory
- **WHEN** a session starts or stops in a repository with no Goalrail declaration claim
- **THEN** the hook exits immediately with zero payload reads, writes, and retained observations

#### Scenario: Session in a managed clone without a legacy marker
- **WHEN** a session starts in a clone carrying a valid committed declaration and no checkout-local marker
- **THEN** the ambient behavior of this capability applies subject to local attachment health

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path exists but cannot be safely validated
- **THEN** the hook reads no event payload, retains nothing, and emits at most the fixed diagnosis direction

### Requirement: Attachment health is reportable and trust is never forged
**Intent IDs:** OUT-2, OUT-3, OUT-4, OUT-10, SIG-2, SIG-3, SIG-12

A registered hook does not run until the scaffold's own trust step has been completed where the scaffold has one. Goalrail SHALL report attachment health on demand, distinguishing at least: repository unmanaged; project declaration invalid; managed project not locally connected; hooks registered but trust pending or unverifiable; executable missing; configuration unreadable; and attachment locally working. Project identity SHALL come from the committed declaration and MUST NOT depend on the attachment verdict.

Health SHALL be judged in the scope where each scaffold's registration belongs. A surviving registration in a superseded scope SHALL be named with the consented removal action and MUST NOT be modified by diagnosis. Goalrail MUST NOT write, compute, reproduce, pre-approve, or simulate scaffold trust records. Reporting an observed trust state is permitted; absence of verifiable trust freshness SHALL remain explicit.

Health SHALL verify every required registered event, the exact durable executable, safe configuration parsing, and the current Goalrail-owned handler identity. Where no scaffold is named, every supported scaffold SHALL be reported. Local attachment working MUST NOT be described as full project enforcement; shared admission remains a separate diagnosis layer.

#### Scenario: Managed project is not connected locally
- **WHEN** attachment health runs in a valid declared project with no registration for the selected scaffold
- **THEN** it reports managed identity, local disconnection, and the exact consented setup action

#### Scenario: Repository is unmanaged
- **WHEN** attachment health runs where no declaration claim exists
- **THEN** it reports unmanaged and does not treat the directory as a participant

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path is present but invalid
- **THEN** attachment health reports declared-invalid and does not recommend creating an unrelated local marker

#### Scenario: Attachment is not yet trusted
- **WHEN** hooks are registered but the scaffold's required trust step is incomplete or unverified
- **THEN** the report names the exact user action or uncertainty without writing a trust record

#### Scenario: Health is judged in the scaffold scope
- **WHEN** a scaffold's registration belongs inside the repository or at user scope
- **THEN** health reads that declared scope and does not infer state from another scope

#### Scenario: Only part of registration survives
- **WHEN** one required event is missing while another remains
- **THEN** the attachment is not locally working and the missing event is named

#### Scenario: Registered executable is gone
- **WHEN** the configured Goalrail executable is missing or not runnable
- **THEN** health reports the exact failure and routes through consented setup planning

#### Scenario: Scaffold configuration cannot be read
- **WHEN** scaffold configuration is unreadable, unsafe, or malformed
- **THEN** health names that failure rather than reporting ordinary disconnection

#### Scenario: No scaffold is named
- **WHEN** a health query names no scaffold
- **THEN** every supported scaffold is reported without assuming one

#### Scenario: Attachment is locally working
- **WHEN** the project declaration is valid, every required event points to the current durable executable, configuration is readable, and trust verifies where required
- **THEN** health reports the attachment locally working and makes no claim about shared admission

#### Scenario: Trust would be written by Goalrail
- **WHEN** any Goalrail path would write, compute, reproduce, or otherwise approve scaffold trust on the user's behalf
- **THEN** that action is prohibited regardless of setup authorization
