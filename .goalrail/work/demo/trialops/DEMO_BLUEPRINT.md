# TrialOps Demo Sandbox — Demo Blueprint

## Purpose

Define a reliable live-demo shape for Goalrail that demonstrates the operating model on one believable business case without pretending that Goalrail is already a finished platform.

This Phase 0 blueprint is planning-only.
The current local demo surface lives in this repository under `apps/web/demo-change-packet`.

## Audience

Primary audience:
- CTO / Head of Engineering
- product-minded engineering leaders
- founders or operators evaluating a managed pilot

Secondary audience:
- internal presenter running a live demo
- future implementation agent extending the local demo surface in `apps/web/demo-change-packet`

## Positioning guardrails

The demo must stay aligned with Goalrail canon:
- Goalrail is a **productized operating layer** for AI-assisted delivery
- the first sellable object is a **managed pilot**
- the message is **one team, one repo, one case, one visible flow to proof**
- the demo shows a **prototype / pilot operating model**, not production maturity

The demo must not imply:
- full automation
- public self-serve SaaS readiness
- autonomous live coding for arbitrary user prompts
- production-ready platform scope

## Repo layout

Planning artifacts live here:
- `.goalrail/work/demo/trialops`

Current local demo surface lives here:
- `apps/web/demo-change-packet`

Rule:
- this repo stores both the planning pack and the current bounded demo surface
- new runtime/backend expansion still needs a separate bounded slice and component mapping before code is added

## Demo domain

Working name:
- `TrialOps Demo Sandbox`

Business context:
- a small B2B SaaS team handles inbound trial and onboarding requests through an internal dashboard

Minimal domain objects for the future sandbox:
- trial request
- request status
- assignee / owner
- comments or notes
- audit event
- dashboard counters

## Primary scenario

Scenario ID:
- `workflow-change`

Business request:

> Before a trial request can be approved, we need a manual review step. The reviewer must assign an owner and provide a decision reason. The dashboard should reflect the new status, and the audit log should show who made the decision.

Why this scenario:
- easy for non-technical buyers to understand
- spans backend, frontend, validation, docs, and proof
- shows a workflow change rather than a cosmetic field tweak
- creates clear acceptance evidence in UI and audit history

## Demo flow

Target live flow:

1. Introduce the startup-like TrialOps domain and show fake seeded data.
2. Show the incoming business request in business language.
3. Show clarification and the working contract.
4. Show the bounded task plan.
5. Show execution or dry-run checkpoints in a controlled way.
6. Show verification and proof artifacts.
7. Show the resulting product behavior in the sandbox.
8. Close with the pilot CTA: one team, one repo, one case, one visible flow to proof.

Canonical narrative:

`business request -> clarification -> working contract -> bounded task plan -> execution/dry-run -> verify/proof -> pilot CTA`

## What to show live

Show live in the current local demo surface:
- trial request list
- trial request detail
- dashboard counters
- basic status flow
- audit log visibility
- the main workflow-change behavior after implementation

Show live in presenter workflow:
- business request card
- clarification notes
- working contract draft
- task plan summary
- proof readout summary

## What to show as prepared proof

Prepared artifacts should be readable, stable, and reusable later for guided replay:
- `business-request.md`
- `contract-draft.md`
- `task-plan.md`
- `proof.md` or `readout.md`
- optional replay events file for later self-serve mode

Prepared proof is preferred for:
- unstable or long-running steps
- narration of intent-to-proof transitions
- deterministic presenter fallback when live execution is risky

## What not to show

Do not show:
- arbitrary prompt box that runs a live public AI workflow
- real customer data
- real external service integrations
- auth, payments, multi-tenant security, or platform admin surfaces
- cloud deployment theatre
- fake claims that Goalrail already has a full product runtime
- broad platform architecture beyond the bounded demo case

## Reliability requirements

Future demo implementation must preserve:
- one command to start
- one command to reset
- one command to run smoke checks
- deterministic fake seed data
- deterministic proof artifact paths
- no real external dependencies
- no secrets
- readable presenter runbook
- fast recovery when a live step fails

Presenter fallback rule:
- if any live step becomes unstable, switch to the prepared proof artifact and continue the narrative without changing the core message

## Phase 0 exit

Phase 0 is complete when:
- another agent can extend `apps/web/demo-change-packet` without inventing demo posture or repo split
- the primary scenario is fixed
- future scenario cards are defined but not implemented
- replay-readiness requirements are explicit
