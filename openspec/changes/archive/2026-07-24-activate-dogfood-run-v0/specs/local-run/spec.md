## MODIFIED Requirements

### Requirement: Real activation remains separately gated
**Intent IDs:** OUT-1, OUT-5, SIG-1, SIG-2, SIG-7

The v0 implementation SHALL continue to reject every non-fixture production
start before provider launch unless an explicit, owner-selected dogfood
authorization matches one prepared WorkSpec digest and its pinned repository
revision exactly. The matching authorization SHALL permit at most one provider
invocation and SHALL remain distinct from intent confirmation, planning
completion, or general effect authority. Context, intent, proposal, specs,
design, tasks, tests, or implementation MUST NOT otherwise activate a real
run.

#### Scenario: Planning and implementation are complete without authorization
- **WHEN** all planning artifacts and local implementation checks exist but no
  exact dogfood authorization exists
- **THEN** Goalrail records zero real provider launches and rejects every
  non-fixture start before provider invocation

#### Scenario: Exact dogfood authorization exists
- **WHEN** an operator starts the one prepared WorkSpec named by a matching
  owner-selected authorization at its pinned repository revision
- **THEN** Goalrail invokes the selected provider adapter at most once and
  continues to reject every other non-fixture start

#### Scenario: Fixture exercises the lifecycle
- **WHEN** a deterministic fixture starts the v0 lifecycle
- **THEN** Goalrail exercises prepare, one-shot launch simulation, lineage,
  checks, and receipt behavior without creating a real provider session
