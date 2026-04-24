---
id: goalrail_ai_sdlc_discovery_workshop
title: Goalrail AI-SDLC Discovery Workshop
kind: research_note
authority: advisory
status: current
owner: product-research
truth_surfaces:
  - local_discovery
  - global_validation
  - discussion_pack
lifecycle: incubating
review_after: 2026-05-24
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_BUILD_ROADMAP.md
  - docs/ops/STATUS.md
---

# Goalrail AI-SDLC Discovery Workshop

> Advisory discussion artifact. This document summarizes local company discovery plus global market validation for AI-assisted software delivery. It does not override `docs/product/*` canon.

## 1. Purpose

Use this document to align discussion around real user pain in AI-assisted software development and to decide which Goalrail pilots or product slices are worth validating next.

This is not an implementation plan and not a scope expansion. It is a research-backed discussion surface for product, pilot, and MVP sequencing.

## 2. Source priority

### Repo canon used

1. `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
2. `docs/product/GOALRAIL_OPERATING_MODEL.md`
3. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
4. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
5. `docs/ops/STATUS.md`
6. `AGENTS.md`

### Local discovery input

Internal meeting transcripts about:

- AI-assisted coding and mobile development without mobile specialization
- DevKit / DefKit / SpecKit-style workflows
- Claude Code, Codex, Gemini, Cursor, Playwright MCP, Jira / Confluence / MCP integrations
- agent roles, subagents, orchestrators, shared context, memory, and context-window concerns
- manual QA moving toward AI-assisted Playwright automation
- internal knowledge base / craft expertise sharing
- tool benchmarking on the same repo / same prompt / same task

### Global validation input

Public sources from 2024-2026, including developer surveys, vendor docs, open-source repositories, security guidance, and AI-SDLC product documentation. Key sources are listed in the appendix.

## 3. One-line synthesis

AI tools are already useful, but the market problem is not code generation itself. The unsolved problem is turning AI-generated work into trusted, bounded, reviewable, verified software delivery.

Goalrail's existing thesis aligns strongly with this: **from business goal to verified change in code**.

## 4. Current picture

```text
AI-assisted delivery reality

High adoption
  -> low / cautious trust
  -> fast generation of candidate changes
  -> expensive review, testing, security, and acceptance
  -> need for contract-first, proof-oriented AI-SDLC workflow
```

Important framing:

```text
AI should not be treated as directly producing accepted software.
AI produces candidate changes.
The SDLC must accept, block, or escalate those changes through explicit gates.
```

## 5. Facts, assumptions, recommendations

### Facts

- Goalrail canon already defines the product as a productized operating layer for AI-assisted delivery, not an AI IDE, generic agent framework, tracker replacement, or all-in-one DevOps suite.
- Goalrail canon already centers the flow around `Incoming task -> Clarify -> Working contract -> Tasks -> Run -> Verify -> Proof -> Feedback`.
- The MVP blueprint already includes runtime-neutral bounded execution, one primary writer runtime per writable run, advisory runtimes, risk routing, and Gate / Verify / Proof.
- Local transcripts repeatedly show friction around trust, context, validation, agents, QA automation, integrations, and adoption.
- Global research strongly confirms trust, validation, AI-SDLC workflow, ROI ambiguity, agent orchestration, integrations, and security/governance as major AI-assisted delivery themes.

### Assumptions

- Local meeting transcripts are a useful qualitative sample, but not statistically representative.
- Global public sources overrepresent visible tools, vocal developer communities, and larger English-language ecosystems.
- The strongest product opportunity is where local pain, global validation, and Goalrail canon overlap.
- Goalrail should not chase provider-native agent features directly; it should wrap them into contract, routing, verification, and proof boundaries.

### Recommendations

- Keep Goalrail positioned as a contract-first AI-assisted delivery control layer.
- Treat trust / validation as the core wedge, not broad agent autonomy.
- Keep security, governance, and accountability visible from the start, even if full enterprise governance is outside MVP.
- Validate with bounded pilots before expanding product scope.
- Track `time-to-accepted-output`, not only generation speed.

## 6. Global cluster map

| Weight | Cluster | Global confirmation | Discussion framing |
|---|---|---:|---|
| ██████████ | Trust & Validation | Strong | Can we trust the AI-generated change? |
| ██████████ | AI-SDLC Workflow | Strong | What is the repeatable path from intent to proof? |
| █████████ | ROI / Validation Cost | Strong | Is AI reducing accepted delivery time or moving work into review? |
| █████████ | Security / Governance / Accountability | Strong | Who is allowed to run what, with which data, and who owns the result? |
| ████████ | Agents / Orchestration | Strong | How do primary writer, advisory runtimes, and gate relate? |
| ████████ | Tooling / Integrations | Strong | How do Jira, GitHub, MCP, SSO, sandbox, Docker, and test envs fit? |
| ███████ | Context / Knowledge | Medium-Strong | What stable context do agents use? |
| ██████ | QA Automation | Medium-Strong | Can manual QA become maintainable AI-assisted automation? |
| ██████ | Adoption / Enablement | Medium-Strong | How does practice move from champions to teams? |
| █████ | Benchmarking / Evaluation | Medium | How do teams compare tools safely and fairly? |

## 7. Local vs global validation matrix

| Local cluster | Local evidence summary | Global validation | Global evidence summary | Product implication |
|---|---|---:|---|---|
| Trust & quality | Hallucinations, drift, fake steps, low confidence in Claude / agents | Strong | Stack Overflow 2025 reports more developers distrust AI accuracy than trust it; “almost right” output is a major frustration | Make verification / proof central, not optional |
| AI-SDLC workflow | Need developer -> reviewer -> tester loop, contract checks, one managed task cycle | Strong | Spec Kit and agentic tools move from one-shot prompting toward spec / plan / task / implementation workflows | Goalrail should own the contract-to-proof contour |
| ROI / validation cost | Sometimes faster by hand; generated diffs require heavy validation | Strong | METR found experienced OSS developers were slower with early-2025 AI in one real-world RCT; DORA notes time shifts to auditing and verification | Measure accepted-output time, not generation time |
| Agents / orchestration | Confusion about orchestrator, subagents, context windows, shared memory; multi-model review emerges | Strong | Codex, Copilot coding agent, Claude Code, OpenHands, Cline, aider, Gemini CLI, Junie compete as agentic runtimes | Stay runtime-neutral; use risk-based routing and advisory panels |
| Documentation / knowledge | Specs, “where to write,” knowledge bank repeats, reusable skills | Medium-Strong | Spec-driven development, AGENTS.md, skills/rules/memories show docs becoming agent context | Build project context / contract artifacts, not generic wiki |
| Tooling / infrastructure | Jira, Confluence, MCP, SSO, Zephyr, Docker, sandbox, account/payment friction | Strong | MCP standardizes data/tool connections but adds security and permissioning risks | Thin integration surface; permissions and audit matter early |
| QA automation | Manual QA exploring Cursor + Playwright + MCP and role-based tests | Medium-Strong | Playwright MCP gives AI agents browser automation via structured snapshots; CLI+skills may be more token-efficient for coding agents | Good bounded pilot with measurable ROI |
| Adoption / enablement | Weekly demos, craft expertise page, champions vs wider team | Medium-Strong | Surveys show high adoption but uneven trust and standards; DORA frames AI as a systems problem | Recipes and playbooks should feed operating model, not sit idle |
| Benchmarking | Same repo / same prompt / same task proposed, but validation expensive | Medium | SWE-bench-style benchmarks are useful but limited for enterprise review/security/context | Create practical internal benchmark protocol |
| Security / governance | Underrepresented locally | Strong | OWASP and MCP security guidance identify prompt injection, excessive agency, insecure output, local server compromise, token scope risk | Add lightweight governance gates without making MVP an enterprise suite |

## 8. Discussion agenda

Use this for a 60-90 minute working session.

### Part 1 — Align on problem shape

Question:

```text
Are we solving code generation, or are we solving trusted AI-assisted delivery?
```

Proposed answer:

```text
Goalrail solves trusted AI-assisted delivery from business goal to verified code change.
```

Decision needed:

- Keep Goalrail away from “better coding agent” positioning.
- Keep contract / verify / proof as core language.

### Part 2 — Validate cluster priority

Discuss the cluster order:

1. Trust & Validation
2. AI-SDLC Workflow
3. ROI / Validation Cost
4. Security / Governance / Accountability
5. Agents / Orchestration
6. Tooling / Integrations
7. Context / Knowledge
8. QA Automation
9. Adoption / Enablement
10. Benchmarking / Evaluation

Decision needed:

- Which 3 clusters become pilot-level validation targets?
- Which clusters remain advisory / deferred?

### Part 3 — Choose pilot direction

Candidate pilots:

1. AI-SDLC validation loop
2. QA automation loop
3. Tool benchmark loop

Decision needed:

- Choose one primary pilot and one secondary optional pilot.

### Part 4 — Protect MVP boundary

Question:

```text
What does this research change in MVP scope?
```

Recommended answer:

```text
It should not expand MVP into a broad agent platform.
It should sharpen the MVP around contract, bounded execution, verify, proof, and lightweight risk/policy visibility.
```

Decision needed:

- Security/governance appears as lightweight risk/policy gates in MVP language, not as full enterprise governance suite.
- Benchmarking remains an evaluation artifact, not core product surface, unless pilot evidence says otherwise.

## 9. Pilot candidates

### Pilot A — AI-SDLC Validation Loop

Goal:

```text
Prove that one real task can move through:
raw request -> clarification -> working contract -> bounded task -> AI-assisted run -> review/gate -> proof.
```

Best fit:

- PM / analyst / tech lead / developer team
- one repo
- low or medium risk feature / fix
- visible acceptance criteria

Metrics:

- time-to-accepted-output
- review time
- number of human interventions
- scope drift count
- hallucination / fake-step count
- test pass/fail
- proof completeness

Why this aligns with Goalrail:

- Directly tests the canonical Goalrail flow.
- Keeps runtime-neutral posture.
- Shows proof-oriented visibility.

Risks:

- Too much process for a small task.
- Weak initial contract can make the pilot look like prompt engineering.
- Verification criteria must be explicit enough to avoid subjective acceptance.

### Pilot B — QA Automation Loop

Goal:

```text
Turn a small set of manual QA scenarios into maintainable Playwright tests using AI assistance.
```

Best fit:

- QA-heavy team
- role-based web application flows
- repeated manual regression checks
- existing test environment or lightweight local setup

Metrics:

- manual QA time saved
- generated tests accepted
- flaky test rate
- maintenance effort
- role/scenario coverage
- defects caught vs missed

Why this aligns with Goalrail:

- Testing is a clear verification lane.
- ROI is easier to measure.
- It creates concrete proof artifacts.

Risks:

- Flaky tests can create false confidence.
- Poor selectors and environment instability can dominate outcomes.
- The pilot may prove Playwright automation value more than Goalrail value unless tied to contract/proof.

### Pilot C — Tool Benchmark Loop

Goal:

```text
Compare 3-5 AI coding workflows on the same repo, same task, same prompt, same acceptance criteria, and same review rubric.
```

Candidate tools:

- Claude Code
- Codex / Codex CLI
- Cursor
- Gemini CLI
- Spec Kit-style workflow
- DefKit / internal workflow if available

Metrics:

- task completion
- test pass rate
- diff size
- review time
- spec adherence
- hallucination / drift count
- human intervention count
- security findings
- rework / rollback
- total cost

Why this aligns with Goalrail:

- Supports runtime-neutral routing.
- Helps design risk-based task routing and advisory review.
- Produces evidence for pilot sales conversations.

Risks:

- Benchmarking itself can become expensive.
- Public benchmark results may not map to internal codebases.
- Same prompt may be unfair if tools require different optimal workflows.

## 10. Opportunity ranking

| Rank | Opportunity | Pain severity | Solution gap | Fit with Goalrail | Recommendation |
|---:|---|---:|---:|---:|---|
| 1 | Trust & Validation Layer | High | High | Very high | Core wedge |
| 2 | AI-SDLC Workflow / Contract-to-Proof | High | High | Very high | Core product path |
| 3 | Lightweight Governance / Risk Gates | High | Medium-High | High | Include as bounded policy/risk model |
| 4 | QA Automation Proof Loop | Medium-High | Medium | Medium-High | Strong pilot candidate |
| 5 | Tool Benchmark / Evaluation Suite | Medium | Medium-High | Medium | Use as research/pilot artifact first |
| 6 | Knowledge / Skills Registry | Medium | Medium | Medium | Keep as support layer, not standalone product |
| 7 | Broad Integration Platform | Medium | Low-Medium | Medium | Defer; avoid becoming generic connector product |
| 8 | Autonomous Multi-Agent Execution | High interest | High risk | Low for MVP | Avoid as core MVP positioning |

## 11. MVP implications

### Should reinforce

- Working contract is the central object.
- Execution is bounded.
- One writable run uses one primary writer runtime.
- Advisory outputs are evidence, not final authority.
- Gate / Verify / Proof is the trust center.
- Proof should be inspectable by both business and engineering roles.
- Risk and policy should affect review depth.

### Should not expand into

- generic AI IDE
- universal agent framework
- tracker replacement
- full enterprise governance suite
- unrestricted multi-agent autonomy
- opaque internal memory platform
- broad MCP marketplace

### Suggested language shift

Use:

```text
AI-generated candidate changes
```

instead of:

```text
AI-generated accepted code
```

Use:

```text
Goalrail accepts, blocks, or escalates through proof-oriented gates
```

instead of:

```text
Goalrail runs agents to complete tasks automatically
```

## 12. Proposed next decisions

### Decision 1 — Primary pilot

Recommended:

```text
Pilot A: AI-SDLC Validation Loop
```

Reason:

- strongest alignment with current product canon
- tests contract-first delivery
- tests verify/proof wedge
- avoids becoming a QA-only or benchmark-only tool

### Decision 2 — Secondary pilot

Recommended:

```text
Pilot B: QA Automation Loop
```

Reason:

- concrete measurable ROI
- local discovery has real user signal
- testing/proof naturally connects to Goalrail verification language

### Decision 3 — Benchmarking posture

Recommended:

```text
Keep Pilot C as an internal evaluation protocol, not MVP product scope.
```

Reason:

- useful for runtime-neutral routing
- expensive to execute
- can easily distract from core contract-to-proof product

### Decision 4 — Security/governance posture

Recommended:

```text
Include lightweight security/governance as risk and policy lanes in Gate / Verify / Proof, but defer full enterprise governance suite.
```

Reason:

- globally validated blocker
- already compatible with MVP blueprint
- avoids silent scope expansion

## 13. Discussion questions

1. Which user role feels the pain most acutely in our first pilot: PM/analyst, tech lead, developer, QA, or engineering manager?
2. What is the smallest real task that can produce a credible `contract -> run -> decision -> proof` demo?
3. What evidence must a proof artifact contain to be trusted by engineering leadership?
4. What should block acceptance automatically?
5. What can be advisory only?
6. How do we avoid turning Goalrail into a generic agent platform?
7. What minimum policy/security lane is needed for pilot credibility?
8. How do we measure time-to-accepted-output without creating heavy process overhead?

## 14. Proposed workshop output

By the end of discussion, produce:

```text
1. Chosen pilot direction
2. One target repo / task profile
3. Draft contract template fields
4. Draft verification lanes
5. Proof artifact outline
6. Pilot success metrics
7. Explicit non-goals
```

## 15. Appendix — Key global sources

### Surveys and productivity evidence

- Stack Overflow 2025 Developer Survey: AI usage / trust / frustration patterns. Source: `https://survey.stackoverflow.co/2025/ai`
- Stack Overflow 2025 overview: 84% use or plan to use AI tools; trust gap. Source: `https://stackoverflow.co/company/press/archive/stack-overflow-2025-developer-survey/`
- DORA 2025 AI-assisted software development: AI adoption tensions, throughput vs instability, audit/verification shift. Source: `https://dora.dev/insights/balancing-ai-tensions/`
- METR early-2025 RCT: experienced open-source developers were slower with AI in the studied setting. Source: `https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/`
- METR 2026 update: productivity effects are evolving and measurement is difficult due to selection effects and multi-agent workflows. Source: `https://metr.org/blog/2026-02-24-uplift-update/`

### Spec-driven / AI-SDLC workflow

- GitHub Spec Kit documentation: specs as executable, multi-step refinement, enterprise constraints. Source: `https://github.github.io/spec-kit/`
- GitHub blog on spec-driven development: specs as source of truth for agents, implementation, checklists, task breakdowns. Source: `https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/`

### Agents and open-source tooling

- OpenAI Codex: cloud-based software engineering agent, parallel tasks, cloud sandbox, test logs / terminal evidence. Source: `https://openai.com/index/introducing-codex/`
- OpenHands: AI-driven development with SDK, CLI, local GUI, cloud, Slack/Jira/Linear integrations, RBAC. Source: `https://github.com/OpenHands/OpenHands`
- Playwright MCP: browser automation MCP server for AI assistants; MCP vs CLI+skills tradeoff. Source: `https://github.com/microsoft/playwright-mcp`

### Context, integrations, security

- Anthropic Model Context Protocol announcement: open standard for connecting AI assistants to content repositories, business tools, and development environments. Source: `https://www.anthropic.com/news/model-context-protocol`
- MCP Security Best Practices: confused deputy, token passthrough, SSRF, session hijacking, local MCP server compromise, scope minimization. Source: `https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices`
- OWASP Top 10 for LLM Applications: prompt injection, insecure output handling, supply chain, sensitive information disclosure, excessive agency, overreliance. Source: `https://owasp.org/www-project-top-10-for-large-language-model-applications/`

## 16. Appendix — Local signal examples

These are paraphrased from internal transcript fragments.

| Signal | Cluster |
|---|---|
| AI without good context creates many revisions | Trust / Context |
| Under deadline, people often use the old predictable method | ROI / Delivery |
| Claude / agents can hallucinate even in small test instructions | Trust / Validation |
| Users want developer -> reviewer -> tester inside one managed cycle | AI-SDLC Workflow |
| Multi-model review can be more useful than same-model subagents | Agents / Orchestration |
| Manual QA wants AI-assisted Playwright tests by roles | QA Automation |
| Jira / Confluence / MCP / SSO / Docker / account friction blocks flow | Tooling / Integrations |
| Knowledge bases are created repeatedly but do not stay alive | Knowledge / Adoption |
| Same repo / same prompt benchmark sounds useful but expensive | Benchmarking |

## 17. Appendix — Draft proof artifact outline

A proof artifact for the first pilot should answer:

```text
1. What was the original business / engineering request?
2. What contract was approved?
3. What was explicitly in scope and out of scope?
4. Which runtime executed the bounded task?
5. What changed?
6. What checks ran?
7. What passed / failed?
8. What regressions were distinguished from baseline failures?
9. What advisory reviews existed, if any?
10. What was the final decision: accept, block, or escalate?
11. What risks remain?
12. What should be learned for the next task?
```

## 18. Appendix — Draft benchmark rubric

| Metric | Description | Why it matters |
|---|---|---|
| Task completion | Did the tool complete the task? | Basic utility |
| Spec adherence | Did it stay within approved contract? | Scope control |
| Test pass rate | Did existing and generated tests pass? | Functional confidence |
| Diff size | How much code changed? | Review burden |
| Review time | Human minutes to accept/reject | Real cost |
| Drift count | Number of off-scope changes | Trust risk |
| Hallucination count | Fake files, commands, APIs, steps | Trust risk |
| Security findings | Static/manual security issues | Enterprise risk |
| Human intervention count | Number of corrections/prompts | Workflow friction |
| Time-to-accepted-output | End-to-end accepted result time | Core ROI metric |
| Rework / rollback | Was output reverted or heavily rewritten? | Quality signal |

## 19. Status

Recommended status after discussion:

```text
advisory input accepted for pilot design
```

Potential promotion path:

- If workshop confirms the priorities, update `docs/product/GOALRAIL_PAIN_STATEMENT.md` or related product/market framing docs.
- If pilot metrics are selected, update `docs/product/GOALRAIL_PILOT_MODEL.md` or pilot proposal template.
- If security/governance language affects MVP semantics, update MVP blueprint or rule stack through the research gate.
