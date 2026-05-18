---
id: goal_brownfield_reconstruction_baseline
title: "Brownfield reconstruction baseline"
status: ready
owner: TODO
module: "project"
priority: P1
authority: canonical
project_id: "goalrail"
entry_mode: brownfield
reconstruction_status: not_started
created_at: TODO
updated_at: TODO
selected_at: TODO
started_at: null
completed_at: null
blocked_by: []
scope:
  include:
    - ".punk/memory/STATUS.md"
    - ".punk/memory/reconstruction/**"
    - ".punk/memory/reports/**"
    - ".punk/instructions/**"
  exclude:
    - "work/**"
    - "knowledge/**"
    - "docs/adr/**"
    - ".punk/events/**"
    - ".punk/contracts/**"
    - ".punk/runs/**"
    - ".punk/evals/**"
    - ".punk/decisions/**"
    - ".punk/proofs/**"
    - ".punk/indexes/**"
    - ".punk/views/**"
    - ".punk/runtime/**"
    - ".punk/cache/**"
acceptance:
  - "A future source corpus manifest boundary is defined before any inventory is generated."
  - "Future claim-ledger, unknowns, contradictions, and contract-readiness artifacts remain advisory until reviewed."
  - "No project knowledge is treated as reconstructed or accepted automatically."
  - "No repo scan, AI summary, contract generation, gate decision, proof, or Writer behavior is activated."
knowledge_refs: []
contract_refs: []
report_refs: []
decision_refs: []
proof_refs: []
latest_proof_ref: null
research_gate:
  classification: R1
  required: false
  rationale: "Brownfield baseline preparation records advisory reconstruction workspace boundaries without external research or repo analysis."
  research_refs: []
  external_research_refs: []
  blocked_reason: null
doc_impact:
  classification: project-memory
  required_updates:
    - ".punk/memory/STATUS.md"
    - ".punk/memory/reconstruction/**"
    - ".punk/memory/reports/**"
  rationale: "Brownfield reconstruction preparation changes the manual Level 0 project-memory baseline."
---

## Context

The project has been initialized as a Punk brownfield project with Level 0 advisory reconstruction workspace.

Project id: `goalrail`

## Intent

Prepare a reviewed source-linked reconstruction baseline before any brownfield claims are promoted.

## Non-scope

Do not scan the repository.

Do not generate summaries, contracts, specs, claims, gate decisions, proofs, or acceptance claims.

Do not write `.punk/` runtime stores.
