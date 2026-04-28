# GoalRail Founding Pilot Offer

Date: 2026-04-28

Status: Draft

Related decision log: `docs/decision-log/2026-04-28-goalrail-ai-development-control-layer.md`

## One-liner

GoalRail helps product teams use AI coding tools without losing control over code quality, architecture, and delivery process.

## Positioning

GoalRail is an AI Development Control Layer.

It is not another AI coding assistant. It works above tools like Cursor, Claude Code, Codex, Copilot, and similar systems.

The goal is not to promise unrealistic development speed. The goal is to make AI-assisted development safer, more predictable, and easier to manage as a team process.

## Core Message

> AI coding without chaos.

Alternative message:

> We put AI-assisted development on rails.

## Why This Matters

Many teams already use AI coding tools. At first, the value is obvious: developers can produce code faster and explore ideas more easily.

The problem appears later.

Without a structured process, AI-assisted development can create hidden risks:

- architecture drift;
- duplicated logic;
- inconsistent implementation patterns;
- unclear reasoning behind changes;
- weak verification;
- loss of project context;
- growing technical debt that is hard to notice early.

GoalRail helps teams keep the benefits of AI coding while reducing the risk of uncontrolled code and process drift.

## Target Customer

Best initial fit:

- small and medium product teams;
- startups and early-stage product companies;
- teams with approximately 3–20 developers;
- teams already experimenting with AI coding tools;
- teams using Cursor, Claude Code, Codex, Copilot, v0, or similar tools;
- teams with repositories up to several hundred thousand lines of code;
- teams willing to run a pilot on a non-critical or semi-isolated product area.

## Not a Good Fit

GoalRail is currently not optimized for:

- huge legacy monoliths;
- very large enterprise codebases;
- heavily regulated environments with long procurement cycles;
- teams expecting fully autonomous development;
- teams expecting guaranteed speed improvements;
- teams that are not yet using or testing AI-assisted development;
- teams unwilling to give access to a repository or representative codebase for audit.

## Founding Pilot

The founding pilot is a fixed-scope implementation of managed AI-assisted development.

The pilot is designed to answer one practical question:

> Can this team use AI coding tools more safely and systematically on a real product area?

The pilot includes:

1. Selecting a suitable product area for the experiment.
2. Repository AI-readiness audit.
3. AI-readiness score.
4. Risk and blocker report.
5. Project knowledge base setup.
6. Contract-based task process setup.
7. Running the first real AI-assisted development workflow.
8. Final report with findings and recommended next steps.

Commercial terms are intentionally not maintained in this repository. Pricing, discounts, and client-specific terms should live in the commercial source of truth used by sales or business operations.

## What the Customer Gets

At the end of the pilot, the customer should have:

- a clear understanding of how ready their repository is for AI-assisted development;
- a list of concrete blockers and risks;
- a project knowledge base foundation;
- a structured task process for AI-assisted work;
- a tested workflow on a real product area;
- recommendations for scaling or stopping the approach;
- a clearer view of where AI coding is safe, risky, or premature.

## What We Do Not Promise

We do not promise:

- 10x development speed;
- replacement of developers;
- fully autonomous product development;
- zero bugs;
- safe operation on any codebase;
- immediate rollout across the whole engineering organization;
- enterprise-grade support for massive legacy monoliths at this stage.

## Sales Framing

Do not start with technical concepts like LLM drift, contract-based development, or agent reliability.

Start with the business problem:

> Your developers already use AI. The question is no longer whether to use it. The question is how to avoid losing control.

Then explain GoalRail:

> GoalRail adds a control layer around AI-assisted development: repository audit, project knowledge base, contract-based tasks, and verifiable execution.

## Discovery Questions

Use these questions during early sales or discovery calls:

1. Which AI coding tools does your team already use?
2. How many developers use them regularly?
3. Are they used for experiments, production code, internal tools, or core product work?
4. What makes you uncomfortable about using AI coding more aggressively?
5. Have you already seen duplicated logic, strange architecture, weak tests, or unexplained code changes?
6. Which part of the product would be safe enough for a pilot?
7. What codebase size and structure are we dealing with?
8. How strong are your tests, documentation, and code review process?
9. What would make this pilot successful for engineering?
10. What would make this pilot successful for the business?

## Qualification Criteria

A strong pilot customer usually has:

- active AI coding usage;
- a real product codebase;
- a small team that can move quickly;
- a clear owner on the engineering side;
- a non-critical but meaningful pilot area;
- willingness to share repository access or a representative codebase;
- interest in process, not only raw speed.

A weak pilot customer usually has:

- no AI coding usage yet;
- no engineering owner;
- no safe pilot area;
- unrealistic expectations about autonomous agents;
- huge legacy complexity;
- unwillingness to expose code for audit;
- a desire for custom consulting without repeatable product learning.

## Suggested Talk Track

Opening:

> We help teams that already use AI coding tools avoid the chaos that can appear after a few weeks or months of unmanaged AI-assisted development.

Problem:

> AI tools can generate code quickly, but without a control layer they can also create hidden technical debt, inconsistent implementation patterns, and unclear changes.

Solution:

> GoalRail adds structure: repository readiness audit, project knowledge base, contract-based task execution, and verifiable outcomes.

Pilot:

> We start with a focused pilot on one product area. The goal is to prove whether your team can use AI-assisted development more safely and systematically before scaling it.

## Objection Handling

### “We already use Cursor / Claude Code / Codex.”

That is exactly why GoalRail exists.

GoalRail is not a replacement for these tools. It helps teams use them with more structure, shared context, and control.

### “Will this make us faster?”

Speed may improve, but it is not the main promise.

The main promise is safer and more predictable AI-assisted development. Without that, short-term speed can create long-term rework.

### “Can agents build features fully autonomously?”

GoalRail should not be sold as full autonomy.

The current goal is controlled AI-assisted development, where humans still own decisions, review, and product judgment.

### “Can this work on our entire monolith?”

Not as the first step.

The recommended first step is a bounded pilot area where the risks are manageable and the results can be evaluated clearly.

## Pilot Success Criteria

A pilot should be considered successful if:

- the repository audit produces useful and actionable findings;
- the selected product area is suitable for AI-assisted work;
- the project knowledge base improves agent and developer context;
- at least one real workflow is completed through the structured process;
- the team can clearly see what should be scaled, changed, or avoided;
- the customer understands the value of a control layer for AI-assisted development.

## Internal Notes

This document is a GTM and sales enablement artifact.

It should stay aligned with the positioning decision log, but it should not contain exact pricing, discounts, negotiated terms, or client-specific commercial details.

Those belong in the commercial source of truth outside the development repository.
