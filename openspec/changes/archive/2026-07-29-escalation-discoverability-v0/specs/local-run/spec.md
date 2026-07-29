## ADDED Requirements

### Requirement: The escalation channel is announced at launch
**Intent IDs:** OUT-1, OUT-3, OUT-4, OUT-5, SIG-1, SIG-4, SIG-5

A launched run SHALL be told that the escalation channel exists. Admitting a
channel a run cannot learn about delivers no capability: an unannounced channel
is indistinguishable from an absent one, and the measured consequence of an
absent channel is that the work is guessed rather than questioned.

The announcement SHALL state the reserved path, that it applies when the work
item cannot be completed as specified from the repository alone, that the
declared scope must be left untouched for the escalation to stand, and where the
payload shape is published. It MUST NOT describe the work item, characterise the
repository, suggest that a conflict or ambiguity exists, or instruct the run to
look for one. The wording is fixed rather than composed per run.

The announcement SHALL name no provider, model, harness, or vendor. Transport is
the adapter's responsibility and may be provider-specific; the announced content
may not be.

The announcement SHALL be delivered exactly once, at launch, as a statement. It
MUST NOT create a reply path, acknowledgement, capability handshake, retry, or
any later delivery, and MUST NOT be repeated inside a run. Where a provider
signals session events that recur within one run — resumption, clearing, or
compaction — the announcement SHALL accompany only the event that opens the run.

Delivery SHALL be verified before any irreversible step of the launch,
including before a one-time activation authorization is consumed. An
authorization spent on a launch that cannot happen is discarded with nothing to
show for it, and no retry can use it afterwards.

Goalrail MUST NOT place the announcement in the canonical WorkSpec, the terminal
receipt, or the target repository worktree. Composing the work item itself
remains outside this boundary.

#### Scenario: A launched run learns the channel exists
- **WHEN** a run is launched
- **THEN** it is told the reserved path, when the channel applies, that the declared scope must stay untouched, and where the payload shape is published

#### Scenario: The reserved path appears in no artifact the run is given
- **WHEN** the announcement is delivered
- **THEN** the reserved path still appears in no WorkSpec field, no receipt field, and no file written into the target worktree

#### Scenario: The announcement would hint at the work
- **WHEN** announcement text would describe the work item, characterise the repository, or suggest that a conflict or ambiguity is present
- **THEN** it violates this requirement, because a hint would measure the hint rather than the environment

#### Scenario: The announcement is delivered once
- **WHEN** a run is launched and continues to completion
- **THEN** exactly one announcement exists, with no acknowledgement, retry, repetition, or reply path

#### Scenario: A provider session event recurs inside a run
- **WHEN** the provider signals a session event for resumption, clearing, or compaction after the run has started
- **THEN** no further announcement accompanies it, because one launch-time statement must not become a recurring instruction

#### Scenario: An undeliverable announcement meets a one-time authorization
- **WHEN** a launch cannot deliver the announcement and the run is authorized by a one-time activation record
- **THEN** the launch fails without consuming that authorization, leaving it usable once the delivery path exists

## MODIFIED Requirements

### Requirement: Provider adapter preserves enforcement boundaries
**Intent IDs:** OUT-2, OUT-3, SIG-2, SIG-3

The local-run lifecycle SHALL obtain launch behavior and provider-authoritative root session identity through a replaceable adapter boundary. The first v0 adapter SHALL support Codex, while canonical WorkSpec and run state remain provider-neutral. Goalrail MUST NOT auto-approve, bypass, weaken, or reinterpret provider sandbox and approval decisions.

The adapter boundary SHALL additionally carry the launch-time escalation
announcement in the terms of the provider's own contract. An adapter that cannot
deliver the announcement MUST fail the launch explicitly, rather than starting a
run whose escalation channel is unreachable. A silent degradation would return
the run to the guessing behaviour the channel exists to prevent, while still
recording an outcome that looks ordinary.

An adapter MUST NOT report delivery it does not itself perform. Where delivery
depends on a component the adapter only invokes — a launcher, a capsule, or any
other collaborator that completes the path into the session — the report SHALL
come from the component that knows whether that path exists. An adapter that
assumed delivery on a collaborator's behalf would defeat this check entirely,
because the launch would proceed while the channel remained unreachable.

Carrying the announcement MUST NOT add provider fields to the canonical WorkSpec
or the terminal receipt, and MUST NOT alter the provider's sandbox or approval
decisions.

#### Scenario: Codex requests operator approval
- **WHEN** Codex gates an action through its normal approval mechanism
- **THEN** the request remains with Codex and `gr` neither approves it automatically nor represents it as Goalrail authorization

#### Scenario: Provider denies an action
- **WHEN** the provider sandbox or approval boundary denies an action
- **THEN** the action remains denied and Goalrail records the bounded provider outcome

#### Scenario: Provider configuration is unavailable
- **WHEN** the requested adapter cannot supply its required launch contract
- **THEN** Goalrail fails the run without falling back to another provider or guessing provider state

#### Scenario: The adapter carries the announcement
- **WHEN** an adapter launches a run
- **THEN** it delivers the announcement in its own contract's terms, and the canonical WorkSpec and terminal receipt gain no provider field for it

#### Scenario: The adapter cannot deliver the announcement
- **WHEN** an adapter has no path for delivering the announcement to the run
- **THEN** the launch fails explicitly and no run is started whose escalation channel would be unreachable

#### Scenario: Delivery depends on a collaborator the adapter only invokes
- **WHEN** the announcement reaches the session only if a launcher or capsule completes the path
- **THEN** the report of whether delivery happens comes from that component rather than being assumed by the adapter, so a missing collaborator fails the launch instead of passing the check
