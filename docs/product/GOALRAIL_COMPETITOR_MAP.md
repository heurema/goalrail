# Goalrail — Competitor Map

## 1. Why this map exists

Нам нужен не обзор “AI tools вообще”, а карта тех, кто уже двигается к похожей боли:
**разрыв между задачей, инженерным контекстом, исполнением и проверкой.**

## 2. Main comparison lens

Для каждого игрока смотрим:
- где у него центр тяжести: planning, coding, orchestration, delivery control
- есть ли shared object между business/task side и engineering side
- provider-native это игрок или supplement layer
- где он уже закрывает нашу боль
- куда нам не надо лезть
- где остаётся наш wedge

## 3. Priority shortlist

### 3.1 Factory — reference #1
**Why it matters**
- очень близок по тону и уровню зрелости
- agent-native platform, которая работает across CLI, Web, Slack/Teams, Linear/Jira and Mobile
- прямо говорит про delegation without changing tools, models, or workflow
- подчёркивает vendor/interface agnostic story

**Overlap with Goalrail**
- supplement layer over existing tools
- model/provider agnostic posture
- team-level rather than single-IDE framing

**What not to copy**
- agent-centric promise as the whole product
- “delegate complete tasks to droids” as the main wedge
- making the agent the star of the story

**Goalrail wedge versus Factory**
- Goalrail should lead with **shared contract + server-side source of truth**, not with the agent itself
- Goalrail’s business-facing layer should be stronger than Factory’s

### 3.2 Atlassian Rovo Dev
**Why it matters**
- clearly moving from work item / Jira context to code generation, sandbox sessions, pull requests and acceptance-criteria checking
- very close to the “task framing -> code” seam

**What it proves**
- issue/work item to code is now a real market direction
- acceptance criteria as verification input is a real buying concern

**Where not to compete**
- don’t try to beat Atlassian on Jira-native code generation or work-item-native flow inside Jira

**Goalrail wedge**
- provider/tool neutral shared contract across systems
- business visibility outside one ecosystem

### 3.3 GitLab Duo Agent Platform
**Why it matters**
- explicit orchestration layer for collaboration between developers and AI agents
- GitLab frames itself as system of record for software development and extends that into agents

**What it proves**
- orchestration + system-of-record is a real category direction
- async multi-agent collaboration is already becoming productized

**Where not to compete**
- don’t rebuild GitLab’s full DevSecOps platform or source-control-native orchestration

**Goalrail wedge**
- lighter supplement layer above mixed stacks
- shared contract/business intent layer before execution

### 3.4 GitHub Copilot coding agent
**Why it matters**
- GitHub already extends from coding assistance toward issue-to-PR execution
- agent can be invoked from issues, agents panel, chat, CLI and MCP-compatible tools

**What it proves**
- provider-native tools will keep absorbing more coding/execution surface

**Where not to compete**
- no custom IDE-side coding assistant
- no custom issue-to-PR agent if provider-native flow is already strong enough

**Goalrail wedge**
- server-side contract / run / verify / proof layer across providers and tools

### 3.5 Sourcegraph (Cody / Amp)
**Why it matters**
- strong in code context, repo intelligence, quality and enterprise-safe AI usage

**What it proves**
- deep code context and enterprise controls are buying criteria

**Where not to compete**
- no need to become a general code-understanding platform

**Goalrail wedge**
- stronger business/task framing and contract layer

### 3.6 Devin
**Why it matters**
- shared knowledge / instructions / context bank is already a visible product capability

**What it proves**
- centralized knowledge and organization-level memory matter

**Where not to compete**
- no need to rebuild “AI employee” positioning

**Goalrail wedge**
- contract-first shared workflow, not autonomous persona

### 3.7 Harness
**Why it matters**
- adjacent delivery control player, stronger after code than before code

**What it proves**
- delivery governance and operational control remain valuable as separate product layers

**Where not to compete**
- not a CI/CD platform

**Goalrail wedge**
- earlier in the loop: intent -> contract -> execution boundary -> proof

## 4. Category grouping

### Provider-native / ecosystem-native
- GitHub Copilot coding agent
- GitLab Duo Agent Platform
- Atlassian Rovo Dev

### Agent-native / orchestration layer
- Factory
- Devin
- Sourcegraph Amp / Cody

### Adjacent delivery control
- Harness

## 5. What this means for Goalrail

### We should not build
- another coding assistant
- another chat shell
- another generic memory layer for its own sake
- another source-control-native agent platform

### We should build
- shared contract as the common object
- server-side source of truth for AI-assisted delivery
- business + engineering visibility in one layer
- run / verify / proof contour that survives provider changes
- supplement layer above heterogeneous provider tools

## 6. Working external reference

**Factory = reference #1**

Use Factory as a reference for:
- tone
- seriousness
- model-agnostic supplement-layer logic
- ecosystem breadth without SaaS cliché

Do NOT use Factory as a copy target for:
- core promise
- agent-centric narrative
- UI hero object

## 7. Sources

1. Factory site: https://factory.ai/
2. Factory GitHub org: https://github.com/Factory-AI
3. Factory main repo README: https://github.com/Factory-AI/factory
4. Factory docs welcome: https://docs.factory.ai/
5. Factory BYOK docs: https://docs.factory.ai/cli/configuration/byok
6. Atlassian Rovo Dev in Jira: https://support.atlassian.com/rovo/docs/work-with-rovo-dev-in-jira/
7. Atlassian Generate code from a work item in Jira: https://support.atlassian.com/rovo/docs/generate-code-from-a-work-item-in-jira/
8. GitHub Copilot coding agent: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/create-a-pr
9. GitLab Duo Agent Platform press release: https://about.gitlab.com/press/releases/2025-07-17-gitlab-announces-the-public-beta-of-gitlab-duo-agent-platform/
10. GitLab Duo Agent Platform docs: https://docs.gitlab.com/development/duo_agent_platform/
