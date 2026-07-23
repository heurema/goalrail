## ADDED Requirements

### Requirement: Run preparation is inspectable and does not launch
**Intent IDs:** OUT-2, SIG-2

The `gr` operator surface SHALL resolve and validate the requested WorkSpec, freeze its canonical content, and display the exact snapshot and digest before launch. Preparation MUST NOT launch a provider or mutate the target repository worktree.

#### Scenario: Operator prepares a valid run
- **WHEN** the operator prepares a valid WorkSpec
- **THEN** `gr` displays the frozen snapshot and digest and records no provider launch

#### Scenario: Preparation fails validation
- **WHEN** the WorkSpec, confirmed intent reference, repository revision, scope, checks, stop conditions, or posture is invalid
- **THEN** preparation fails with a bounded reason and no run ID or provider launch is created

### Requirement: Start is explicit and one-shot
**Intent IDs:** OUT-2, SIG-2, SIG-3

Goalrail SHALL launch a prepared run only after a separate explicit operator start action. A successful start SHALL generate one run ID and permit at most one provider launch for the frozen WorkSpec instance. Goalrail MUST reject duplicate starts and MUST NOT retry, start another agent, or continue the run in the background.

#### Scenario: Explicit start launches once
- **WHEN** the operator explicitly starts one valid prepared WorkSpec
- **THEN** Goalrail generates one run ID and invokes the selected provider adapter exactly once

#### Scenario: Start is repeated
- **WHEN** start is requested again for a WorkSpec instance whose launch was already attempted
- **THEN** Goalrail rejects the request and does not invoke the provider again

#### Scenario: Provider launch fails
- **WHEN** the single provider launch returns an error or terminates before completing
- **THEN** Goalrail records a terminal failure without retrying or starting a replacement run

### Requirement: Provider adapter preserves enforcement boundaries
**Intent IDs:** OUT-2, OUT-3, SIG-4

The local-run lifecycle SHALL obtain launch behavior and provider-authoritative root session identity through a replaceable adapter boundary. The first v0 adapter SHALL support Codex, while canonical WorkSpec and run state remain provider-neutral. Goalrail MUST NOT auto-approve, bypass, weaken, or reinterpret provider sandbox and approval decisions.

#### Scenario: Codex requests operator approval
- **WHEN** Codex gates an action through its normal approval mechanism
- **THEN** the request remains with Codex and `gr` neither approves it automatically nor represents it as Goalrail authorization

#### Scenario: Provider denies an action
- **WHEN** the provider sandbox or approval boundary denies an action
- **THEN** the action remains denied and Goalrail records the bounded provider outcome

#### Scenario: Provider configuration is unavailable
- **WHEN** the requested adapter cannot supply its required launch contract
- **THEN** Goalrail fails the run without falling back to another provider or guessing provider state

### Requirement: Run lineage uses provider-authoritative identity
**Intent IDs:** OUT-4, SIG-3, SIG-5

Each launched run SHALL bind its generated run ID and frozen WorkSpec digest to the exact root session identity returned by the provider-authoritative launch receipt. Missing, malformed, or conflicting identity MUST remain explicit and MUST NOT be repaired through manual entry, prompt parsing, transcript parsing, or heuristic matching.

#### Scenario: Provider returns verified root identity
- **WHEN** the adapter returns one valid authoritative root session identity for the launched run
- **THEN** Goalrail records the verified WorkSpec-to-run-to-session lineage

#### Scenario: Root identity is missing or conflicting
- **WHEN** the adapter returns no root identity, malformed identity, or identity that conflicts with the immutable run context
- **THEN** Goalrail records an unlinked terminal outcome and does not guess or accept a manual replacement

### Requirement: Terminal receipt is bounded and reviewable
**Intent IDs:** OUT-4, OUT-5, SIG-5

Goalrail SHALL produce a terminal receipt containing the WorkSpec ID, version, and digest; run ID; provider-authoritative root session reference when verified; pinned repository revision; effective path scope; exact selected check references and results; terminal status; and a bounded worktree-delta reference. The receipt MUST NOT contain raw prompts, transcripts, credentials, secrets, raw source bodies, or copied worktree content.

#### Scenario: Run completes with all selected checks
- **WHEN** the launched run reaches a terminal state and every selected check has an explicit result
- **THEN** Goalrail emits a complete review receipt with the exact check set and bounded worktree-delta reference

#### Scenario: Required check result is missing
- **WHEN** a terminal receipt lacks a result for any selected check
- **THEN** Goalrail marks verification incomplete and does not report the run as fully passing

#### Scenario: Receipt contains prohibited payload
- **WHEN** receipt data contains raw prompts, transcripts, credentials, secrets, raw source bodies, or copied worktree content
- **THEN** receipt validation fails before the prohibited payload is retained

### Requirement: Real activation remains separately gated
**Intent IDs:** OUT-5, SIG-6, SIG-7

The v0 implementation SHALL support deterministic local fixtures without activating real Goalrail work. Until a later owner-authorized activation change exists, every non-fixture start MUST fail before provider launch. The presence or completion of context, intent, proposal, specs, design, tasks, tests, or implementation MUST NOT activate a real run.

#### Scenario: Planning and implementation are complete
- **WHEN** all planning artifacts and local implementation checks exist but no later activation change has been authorized
- **THEN** Goalrail records zero real provider launches and continues to reject non-fixture start

#### Scenario: Fixture exercises the lifecycle
- **WHEN** a deterministic fixture starts the v0 lifecycle
- **THEN** Goalrail exercises prepare, one-shot launch simulation, lineage, checks, and receipt behavior without creating a real provider session

### Requirement: Gr performs no hidden follow-on actions
**Intent IDs:** OUT-5, SIG-6

The `gr` flow MUST stop after its terminal receipt. It MUST NOT automatically create or switch branches, stage files, commit, push, create or merge a pull request, rebase, deploy, publish, release, inject credentials, perform external effects, retry, launch another agent, or schedule later continuation.

#### Scenario: Terminal receipt is emitted
- **WHEN** a fixture or later authorized real run emits its terminal receipt
- **THEN** `gr` returns control to the operator without any automatic follow-on action
