# Decision Log: Position GoalRail as AI Development Control Layer

Date: 2026-04-28

Project: GoalRail

## Decision

We will position GoalRail as a 2-week pilot for managed and controlled AI-assisted development, not as a generic AI coding accelerator.

The core offer is:

> GoalRail Pilot — a 2-week implementation of managed AI development for product teams. We audit the repository, build a project knowledge base, introduce a contract-based task process, and launch the first safe AI pilot on a real product area.

The positioning should emphasize:

- control over AI-assisted development;
- safety and predictability;
- reduced chaos from AI-generated code;
- repository readiness for AI agents;
- project knowledge base;
- contract-based execution of tasks;
- transparent process for developers and business stakeholders.

We should avoid positioning GoalRail as:

- a replacement for developers;
- a 10x speed promise;
- a direct competitor to Cursor, Claude Code, Codex, Copilot, or similar tools;
- a generic AI coding tool;
- an enterprise-ready solution for massive legacy monoliths.

## Reason

The market is still early. Many teams already experiment with AI coding tools, but most business stakeholders do not yet clearly understand the long-term risks of unmanaged AI-assisted development.

Explaining the product through technical concepts like LLM drift, contract-based development, or agent reliability is too complex for most buyers.

A clearer business framing is:

> Your developers already use AI. The question is no longer whether to use it, but how to avoid losing control.

This makes the offer easier to understand and easier to sell.

## Target Customer

Initial target customers:

- small and medium product teams;
- startups and early-stage product companies;
- teams with approximately 3–20 developers;
- teams already using or testing Cursor, Claude Code, Codex, Copilot, v0, or similar AI coding tools;
- teams with repositories up to several hundred thousand lines of code;
- teams willing to run a pilot on a non-critical or semi-isolated product area.

We should avoid starting with:

- very large enterprises;
- huge monoliths;
- heavily regulated environments;
- teams expecting fully autonomous development;
- teams expecting guaranteed speed improvements.

## Offer

Founding Pilot:

- Duration: approximately 2 weeks
- Scope: fixed-scope pilot with a selected product area
- Commercial terms: intentionally discounted during the founding pilot stage and maintained separately from this repository

Included:

1. Repository AI-readiness audit.
2. AI-readiness score.
3. Risk and blocker report.
4. Project knowledge base setup.
5. Contract-based task process setup.
6. First real AI-assisted development workflow on a selected product area.
7. Final report with what works, what blocks scaling, and recommended next steps.

Exact pricing, discounts, and client-specific commercial terms are not maintained in this repository. They should live in the commercial source of truth used by sales or business operations.

## Messaging

Primary message:

> AI coding without chaos.

Supporting message:

> GoalRail helps teams use AI coding tools more safely by adding a control layer: repository audit, project knowledge base, contract-based tasks, and verifiable execution.

Another acceptable framing:

> We put AI-assisted development on rails.

## What This Prevents

This decision prevents:

- selling the product as another AI coding assistant;
- creating wrong expectations around 10x speed;
- competing directly with Cursor, Claude Code, Codex, Copilot, or similar tools;
- overexplaining the product through internal technical concepts;
- targeting customers whose codebases and expectations are too large for the current stage;
- premature enterprise positioning;
- scope creep into custom consulting without a repeatable productized pilot.

## Review Date

Review after one of the following happens:

- 3–5 paid pilot sales;
- first 10 serious discovery calls;
- first real implementation with an external product team;
- material change in the product architecture or ICP.

## Notes

This is a positioning and go-to-market decision, not a permanent product architecture lock-in.

The current strategic hypothesis:

> GoalRail should be sold as a control layer for AI-assisted development, not as a speed layer.
