# Start Assistant Static Answers

These answers are for `/start` v0.1, before the real source-grounded assistant is connected.

They must be short, public-safe, and aligned with Goalrail canon.

## what-is-goalrail

Goalrail is a control layer for AI-assisted software delivery.

It helps teams move from a business goal to a verified code change through goal intake, clarification, contract, bounded execution, checks, proof, and human approval.

It is not an AI IDE, not a Jira replacement, and not a generic agent platform.

## repo-readiness

A repo is not ready for coding agents just because it builds.

Repo readiness means the repository exposes enough working signals for an agent to operate safely: how to run it, which checks matter, what not to touch, where ownership lives, how to prove behavior stayed intact, and how to recover when a change goes wrong.

## contract-first-execution

Contract-first execution means the agent does not start from a free prompt.

The work is first bounded by a contract: goal, scope, non-goals, affected areas, required checks, expected artifacts, and proof criteria.

The goal is not more process. The goal is fewer hidden decisions during execution.

## proof-before-approval

Output is not proof.

Proof before approval means reviewers should not accept AI-generated work only because the diff looks clean or the agent summary sounds confident.

They should compare contract, diff, checks, artifacts, and remaining risk before approving the change.

## different-from-ai-ide

Goalrail is not an AI IDE.

AI IDEs help generate or edit code. Goalrail focuses on the control layer around AI-assisted delivery: intent, scope, contract, checks, proof, and human approval.

It works around the tools a team already uses.

## pilot-fit-check

A pilot fit check is a lightweight conversation to see whether Goalrail fits one real team, repo, or workflow.

The best fit is a small or mid-sized team already using AI coding tools and starting to feel review, context, scope, or proof problems.

The first useful pilot shape is one visible task-to-proof loop.

## ai-delivery-drift

AI delivery drift is what happens when AI-assisted work starts moving away from the original intent, scope, architecture, or proof expectations.

The code may still look clean, but the team has to reconstruct why the change was made, whether it stayed in bounds, and what was actually verified.

## review-ai-generated-changes

AI review should not stop at the diff.

A reviewer should ask: what contract was executed, which checks passed, what artifacts were produced, what remains unverified, and which risk is being accepted.

That keeps human approval in the loop, but makes the decision less vague.

## why-cto-care

A CTO should care because AI coding increases output before it necessarily increases control.

If the process is weak, AI can accelerate unclear goals, hidden assumptions, review bottlenecks, and rework.

Goalrail focuses on making AI-assisted delivery governable before it scales.

## hidden-risk

AI coding creates hidden risk when the agent makes reasonable assumptions that the team never approved.

Examples: choosing the wrong file, widening scope, skipping an edge case, changing behavior that should have stayed stable, or producing a confident summary without enough proof.
