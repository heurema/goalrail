# Dogfood Activation Specification

## Purpose

Define the exact, one-time admission boundary for one owner-selected trusted
local Codex run without creating general execution or effect authority.

## Requirements

### Requirement: Dogfood activation is exact and fail-closed
**Intent IDs:** OUT-1, OUT-3, OUT-5, SIG-1, SIG-2, SIG-5

Goalrail SHALL admit a production provider launch only when one explicit
owner-selected authorization matches the prepared WorkSpec digest and pinned
repository revision exactly. The authorization SHALL permit at most one Codex
adapter invocation. Missing, stale, malformed, mismatched, or previously
claimed authorization MUST fail before a provider invocation, run ID, or launch
claim is created. The mechanism MUST NOT expose a general activation flag,
environment override, or reusable admission path.

#### Scenario: Exact selected WorkSpec is admitted once
- **WHEN** an operator starts the exact prepared WorkSpec named by the explicit
  dogfood authorization at its pinned repository revision
- **THEN** Goalrail invokes the existing Codex adapter at most once and records
  one immutable launch claim

#### Scenario: Input does not match the authorization
- **WHEN** an operator starts a WorkSpec with an absent, different, stale,
  malformed, or already-claimed authorization
- **THEN** Goalrail rejects the start before provider invocation and creates no
  additional run ID or launch claim

### Requirement: Dogfood activation preserves provider enforcement
**Intent IDs:** OUT-3, SIG-5

The dogfood activation SHALL pass only the bounded, provider-neutral WorkSpec
and operator-supplied adapter arguments to the existing Codex adapter. Goalrail
MUST NOT auto-approve, bypass, weaken, or reinterpret Codex sandbox or approval
behavior, inject credentials, or turn an intent or authorization record into
permission for any other effect.

#### Scenario: Codex requests or denies an action
- **WHEN** Codex applies its normal sandbox or approval boundary during the
  admitted run
- **THEN** the boundary remains authoritative and Goalrail records only the
  bounded provider outcome
