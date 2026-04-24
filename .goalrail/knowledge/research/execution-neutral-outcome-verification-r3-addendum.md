# Execution-Neutral Outcome Verification — R3 Addendum

- Date: 2026-04-24
- Status: classified
- Question: Should Goalrail explicitly adopt a "User-Owned Execution, Goalrail-Owned Verification" boundary as part of the same core law as "Outcome Rails, Not Behavior Cages"?
- Recommendation: `adapt`
- Related canon:
  - `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
  - `docs/product/GOALRAIL_OPERATING_MODEL.md`
  - `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
  - `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
  - `docs/PROJECT_SPINE_SCHEMA.md`
  - `docs/product/GOALRAIL_RULE_STACK.md`
  - `docs/adr/ADR-0001-runtime-neutral-cli-first.md`

## Summary

Additional candidate boundary:

Goalrail should be execution-neutral.

It should not prescribe or optimize the user's internal execution setup:
prompts, skills, subagents, IDE settings, provider-specific workflows,
model selection, or private agent configuration.

Goalrail should provide a bounded task packet and then evaluate the submitted
outcome against the approved contract through verification lanes, gate decision,
proof artifacts, and feedback.

Short law candidate:

```text
Do not own the executor.
Own the contract, boundary, verification, decision, and proof.
```

Alternate phrasing:

```text
User-Owned Execution, Goalrail-Owned Verification
Execution-Neutral Outcome Verification
```

## R3 classification

Verdict: `adapt`.

Goalrail should adopt the boundary, but with two wording constraints:

1. Do not imply that Goalrail can never restrict execution. Policy may still
   restrict dangerous actions, sensitive exposure, runtime choice, or required
   evidence.
2. Do not imply that Goalrail should rank or optimize private execution setups.
   Goalrail should expose evidence and failure patterns; users and runtimes own
   setup improvements.

Adapted law:

```text
Goalrail is execution-neutral and outcome-owned.

Goalrail does not prescribe how humans or agents internally execute bounded
work. It standardizes the contract, boundary, receipt requirements,
verification, gate decision, proof, and feedback around the submitted outcome.
```

Promotion path:

- add a small canon patch reinforcing the existing boundary
- do not add implementation scope
- do not expand runtime metadata beyond receipt/evidence needs
- do not encode provider-specific prompt, skill, model, or IDE doctrine

## Evidence to evaluate

The current Goalrail canon already appears compatible with this boundary:

- Product Concept positions Goalrail as a layer from business goal to verified
  code change, not as an IDE, coding agent, or provider-native runtime
  replacement.
- Operating Model defines the central object as the working contract, not a
  prompt, ticket, PR, or agent; execution and final verification are separate.
- MVP Blueprint states: "Runtime may execute; gate decides; proof preserves."
  It also defines runtime-neutral adapters and verification lanes.
- Provider Boundaries says Goalrail should supplement provider-native
  capabilities rather than compete with them by default.
- ADR-0001 fixes a runtime-neutral, CLI-first kernel where runtime-specific
  behavior stays behind adapter boundaries.
- Project Spine separates ownership: runtime writes `Run`, receipts, and
  execution artifacts; Gate writes `Decision` and `Proof`; final verdict belongs
  only to Gate.

External source directions to consider during R3:

- Provider docs increasingly recommend keeping persistent model instructions
  concise and moving deterministic control to hooks, tools, checks, and
  guardrails rather than relying on giant prompt cages.
- Agent guardrail guidance generally treats tool safeguards, access control,
  output validation, risk ratings, and escalation as layered controls around
  execution rather than as attempts to control hidden model cognition.

## Source set

Tier A, local Goalrail canon:

- `docs/product/GOALRAIL_PRODUCT_CONCEPT.md` — Goalrail does not replace IDEs,
  code agents, or provider-native runtimes; fixed core includes bounded
  execution, final evaluation separated from execution, and inspectable proof.
- `docs/product/GOALRAIL_OPERATING_MODEL.md` — the central object is the
  working contract, not a prompt, ticket, PR, or agent; runtime operator may be a
  human or controlled runtime layer.
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md` — "Runtime may execute; gate decides;
  proof preserves"; runtime-specific logic must stay outside the kernel.
- `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md` — coding agents are
  provider-native by default; Goalrail should not become another AI IDE or
  prompt shell.
- `docs/PROJECT_SPINE_SCHEMA.md` — runtime writes `Run` and execution
  artifacts; Gate writes `Decision` and `Proof`; final verdict is written only
  by Gate.
- `docs/adr/ADR-0001-runtime-neutral-cli-first.md` — runtime-neutral CLI-first
  kernel; runtime-specific behavior lives behind adapters.

Tier A, external vendor guidance:

- Anthropic Claude Code best practices:
  `https://code.claude.com/docs/en/best-practices`
  - `CLAUDE.md` guidance favors short, human-readable persistent instructions.
  - Claude Code performs better when it can verify work through tests, outputs,
    and clear success criteria.
  - Hooks are described as deterministic compared with advisory instructions.
- Anthropic Claude Code hooks guide:
  `https://docs.claude.com/en/docs/claude-code/hooks-guide`
  - hooks provide deterministic control at lifecycle points
  - hooks can deny or escalate tool calls
- OpenAI Agents SDK, Guardrails and human review:
  `https://developers.openai.com/api/docs/guides/agents/guardrails-approvals`
  - guardrails validate input, output, or tool behavior
  - human review pauses runs for sensitive approval decisions
  - tool guardrails belong next to side-effecting tools
- OpenAI Agents SDK, Integrations and observability:
  `https://developers.openai.com/api/docs/guides/agents/integrations-observability`
  - traces expose run records of model calls, tool calls, handoffs, and
    guardrails
  - eval loops score behavior systematically after observability exists

## Research question

Should Goalrail explicitly adopt a "User-Owned Execution, Goalrail-Owned
Verification" boundary as part of the same core law as "Outcome Rails, Not
Behavior Cages"?

Evaluate:

1. Whether execution-neutral outcome verification is consistent with current
   Goalrail canon: contract-first flow, bounded execution, runtime-neutral
   adapters, gate authority, and inspectable proof.
2. Whether Goalrail should avoid becoming a provider-specific prompt, skill,
   model, IDE, or setup optimizer.
3. What minimal runtime metadata Goalrail should capture:
   runtime identity, operator type, capabilities, receipt, artifacts, baseline,
   declared checks, and submitted claims.
4. What Goalrail should not capture by default:
   private prompts, skills, chain-of-thought, full provider settings, or private
   user configuration, unless explicitly submitted as an artifact for a specific
   proof or audit need.
5. How Goalrail can expose execution reliability without prescribing setup:
   accept/block/escalate rates, lane failure taxonomy, scope drift, integrity
   regressions, proof completeness, retry count, and time-to-accepted-proof.
6. Failure modes:
   - execution neutrality becoming too hands-off
   - insufficient evidence from opaque runtimes
   - runtime adapters secretly encoding provider doctrine
   - product drifting into Claude Code, Codex, Cursor, or Gemini configuration
     consulting
   - users misreading metrics as universal provider rankings
   - privacy risks from collecting too much runtime configuration
7. Recommended wording for:
   - Product Concept
   - Operating Model
   - MVP Blueprint
   - Provider Boundaries
   - Project Spine Schema
   - Rule Stack
   - `AGENTS.md`

## Implications for Goalrail

If R3 returns `adopt` or `adapt`, the smallest canon patch should reinforce the
existing boundary rather than add new runtime scope.

Candidate patch targets:

```text
docs/product/GOALRAIL_PRODUCT_CONCEPT.md
- add Execution ownership boundary

docs/product/GOALRAIL_OPERATING_MODEL.md
- clarify Stage 5: Goalrail standardizes handoff and evidence, not private
  execution setup

docs/product/GOALRAIL_MVP_BLUEPRINT.md
- add principle: execution setup is user-owned; verification and proof are
  Goalrail-owned
- clarify runtime adapter neutrality

docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md
- strengthen: wrap execution, do not own execution

docs/PROJECT_SPINE_SCHEMA.md
- note: Run receipts capture evidence, not private prompts or settings by
  default

docs/product/GOALRAIL_RULE_STACK.md
- add root law: Execution-Neutral Outcome Verification

AGENTS.md
- add agent-facing rule: no provider-specific execution doctrine in core
```

Non-goals for the patch:

- no implementation scope changes before C1
- no provider-specific Claude Code, Codex, Cursor, Gemini, or local-runtime setup
  logic in the kernel
- no schema expansion unless a later proof or audit requirement makes it
  necessary
- no default capture of private prompts, skills, model settings, chain-of-thought,
  or user runtime configuration

Open question:

Should Goalrail ever recommend improvements to execution setup, or should it
only expose evidence, failure taxonomy, and reliability patterns so users can
adjust their own setup?
