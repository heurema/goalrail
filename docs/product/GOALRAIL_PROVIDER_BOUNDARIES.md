# Goalrail — Provider Boundaries

## 1. Product rule

**Goalrail must supplement provider-native capabilities, not compete with them by default.**

If a major provider closes a capability well enough out of the box, Goalrail should:
- avoid building it
- wrap it instead of replacing it
- or remove its own version later

## 2. Why this rule exists

The provider layer moves too quickly for a rigid monolith.
GitHub, GitLab, Atlassian and Factory are all expanding toward issue/work-item execution, orchestration, context, reviews and background task flows.

That means a fixed “we own this forever” architecture is a trap.
Goalrail should be:
- adaptive
- replaceable by slices
- modular by boundary
- willing to delete features when the market makes them redundant

## 3. Core principle

**Build less. Wrap more. Remove aggressively.**

Goalrail should wrap execution, not own execution.

Provider-native coding agents, user skills, custom prompts, IDE workflows, and
local runtimes remain outside the product kernel. Goalrail may observe declared
runtime identity, capabilities, receipts, artifacts, and verification results,
but it should not become the place where provider-specific execution behavior is
centrally prescribed.

### Build
Build only the layers that remain under-served by provider-native tools:
- shared working contract
- server-side source of truth
- business visibility
- run / verify / proof contour
- organization-specific policy and review surfaces
- supplement layer across multiple providers and trackers

### Wrap
Wrap provider-native capabilities when they are already strong:
- coding agents
- chat surfaces
- model switching
- CLI integrations
- repo-native issue to PR flows
- provider-specific memory where it is already sufficient

### Remove
If a provider closes a gap natively:
- stop treating our version as core
- reduce it to a thin interface or adapter
- or delete it entirely

## 4. What Goalrail should not become

Goalrail should not try to be:
- another AI IDE
- another prompt shell
- another general memory system for its own sake
- another monolithic agent framework
- another all-in-one DevOps platform

## 5. Durable product wedge

The durable wedge is not “best model” or “best agent”.
It is:

**a shared server-side source of truth for AI-assisted delivery**

That means Goalrail owns:
- the shared contract
- the delivery spine
- the relationship between task, scope, execution, verification and proof
- the business/engineering visibility layer

## 6. Boundary decisions by capability

### Coding agent / code generation
Default: provider-native.
Goalrail should integrate, not compete.

### Memory
Default: do not build a giant standalone memory product.
Only build memory where it serves the shared contract / run / proof contour and where provider-native memory does not create shared truth for the team.

### Model routing
Default: adapter layer only.
Goalrail should be model/provider agnostic where possible.

### Tracker-native planning
Default: do not replace Jira / Linear.
Goalrail sits above them as intent-to-delivery layer.

### Verification and proof
This remains a strong Goalrail candidate capability because it links business intent, engineering scope and execution evidence.

## 7. Architecture implications

The system should be designed so that major slices can be replaced or removed without breaking the whole.

### Required architectural properties
- clear adapter interfaces
- feature flags by capability
- policy-driven enable/disable
- provider-specific wrappers isolated from core domain
- no provider assumptions inside shared contract model
- server-side domain truth independent of one IDE or one model vendor

### Repository access MVP boundary

Goalrail core uses `RepoBinding` as repository context in the MVP. RepoBinding
identifies which repository a Goalrail Project works with, but it is not a
credential, permission to clone, or provider connection.

MVP checkout access should be runner-owned: the API server issues bounded
checkout instructions from WorkItem / RepoBinding context, and the runner uses
local credentials or a mounted workspace. The API server must not store
repository secrets in the MVP.

Provider UI integrations, live provider metadata listing/search, GitHub App,
GitLab OAuth, and Bitbucket OAuth are not active MVP scope. If reconsidered
later, provider integrations require fresh research and a new ADR with current
requirements.

Provider-specific concepts must not become Goalrail core terminology:
- GitLab Group must not become Goalrail `Organization`
- GitLab Project must not become Goalrail `Project`
- GitHub Organization must not become Goalrail `Organization`
- Bitbucket Workspace must not become Goalrail `Organization`
- provider repository access must not become `RepoBinding`

## 8. Experimental posture

Goalrail should be sold as:
- adaptive
- experimental
- supplementing the rapidly moving provider ecosystem

Not as:
- forever-fixed platform truth
- fully deterministic universal layer
- “we replace the providers” solution

Recommended phrasing:

**Goalrail closes the gaps that providers do not yet close well for teams. When those gaps are covered natively, Goalrail should adapt or step back.**

## 9. Company customization

Goalrail must remain customizable because organizations differ in:
- approval rules
- scope boundaries
- review expectations
- proof requirements
- deployment and security posture
- model/provider policy

That means the architecture should allow:
- configurable policies
- configurable fields in contracts
- configurable review/verify surfaces
- organization-specific terminology and workflow overlays

## 10. Working doctrine

Goalrail is not the final source of truth about AI capabilities.
It is the **current best supplement layer** for turning business intent into governed AI-assisted delivery.

## 11. Source signals

1. Factory docs — model-agnostic, BYOK, local model support: https://docs.factory.ai/
2. Factory BYOK docs: https://docs.factory.ai/cli/configuration/byok
3. Factory OpenAI/Anthropic docs: https://docs.factory.ai/cli/byok/openai-anthropic
4. GitHub Copilot coding agent docs: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/create-a-pr
5. GitLab Duo Agent Platform press release: https://about.gitlab.com/press/releases/2025-07-17-gitlab-announces-the-public-beta-of-gitlab-duo-agent-platform/
6. Atlassian Rovo Dev in Jira: https://support.atlassian.com/rovo/docs/work-with-rovo-dev-in-jira/
