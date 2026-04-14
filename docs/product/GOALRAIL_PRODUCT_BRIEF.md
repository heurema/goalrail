# Goalrail Product Brief

## 1. Product thesis

Goalrail is an intent-to-delivery layer for software teams.

Короткая формула:

**от бизнес-цели до проверенного изменения в коде**

Goalrail does not try to replace trackers, IDEs, or developer runtimes.
It connects planning and execution in one managed delivery loop.
Goalrail is runtime-neutral by design and works with existing developer runtimes rather than replacing them.

## 2. Core user groups

### Intent side
- PM
- analyst
- product owner
- tech lead

What they need:
- turn vague requests into structured delivery input
- make scope and constraints explicit
- keep acceptance criteria visible
- hand work to engineering without ambiguity

### Delivery side
- developer
- tech lead
- QA
- CI / automation

What they need:
- receive an approved bounded contract
- execute through one primary writer runtime
- verify the result with policy-appropriate review depth
- return proof, not just a status update

## 3. Core product promise

Goalrail helps teams:
1. clarify intent
2. shape delivery contracts
3. execute bounded changes
4. verify results
5. return inspectable proof

## 4. Two product planes

### Plane A — Intent / Planning
- goal intake
- clarification
- constraints
- glossary
- handoff preparation

### Plane B — Delivery / Execution
- contract review
- task shaping
- bounded runtime
- verification
- proof

Both planes are connected through one Project Spine.

## 5. Project Spine

Canonical flow:

`Goal -> Clarify -> Contract -> Tasks -> Change -> Verify -> Proof -> Feedback`

Canonical objects:
- Project
- RepoBinding
- Goal
- Constraint
- GlossaryTerm
- Contract
- Task
- Run
- Decision
- Proof
- Learning
- Artifact
- Event

## 6. Product boundaries

Goalrail is not:
- a replacement for Jira / Linear in v1
- a generic agent framework
- an IDE
- a chat product over code
- a fully autonomous engineering platform

Goalrail is:
- a structured project memory and delivery control layer
- a runtime-neutral contract-based execution system
- a verification and proof system
- a bridge between planning intent and engineering delivery

## 7. ICP v0

Best initial teams:
- product teams with 5–30 engineers
- a PM / analyst / tech lead structure already exists
- there is pressure to adopt AI without losing project control
- there are 1–2 repos where a pilot can produce visible proof

## 8. MVP promise

The MVP should let a team:
1. create a goal
2. clarify it into a structured packet
3. review and approve a delivery contract
4. run bounded implementation work through one primary runtime
5. verify the result with explicit scope, integrity, and policy checks
6. inspect proof in a single product flow

## 9. GTM v0

Best first commercial format:
- pilot-first
- one team
- one or two repos
- one visible workflow from goal to proof

Goalrail should be sold on outcomes:
- less ambiguity between PM and dev
- bounded AI-assisted delivery
- proof-oriented visibility
