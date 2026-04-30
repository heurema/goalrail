# Add PR Intake Gate v0.1

- Status: done
- Owner: Vitaly
- Related phase/checkpoint: repository governance / Level 1 tool-assisted dogfooding
- Proof target: deterministic fixture tests and GitHub draft PR check evidence
- Canon refs:
  - `docs/product/GOALRAIL_RULE_STACK.md`
  - `docs/product/GOALRAIL_RESEARCH_GATE.md`
  - `docs/ops/COMPONENTS.yaml`
  - `docs/ops/REPO_STRUCTURE.md`

## Goal

Add a deterministic GitHub PR Intake Gate so trusted maintainers keep a low-friction path while external contributors provide enough Goalrail context before ordinary code review.

## In scope

- Add `.github/workflows/pr-intake-gate.yml` using safe `pull_request_target` and trusted base checkout.
- Add `.github/pr-intake-gate.yml` policy for trivial paths, high-risk paths, trusted authors, external context sections, labels, and linked intent patterns.
- Add `scripts/pr_intake_gate.py` as a stdlib-only deterministic gate.
- Add `tools/pr-intake-gate/test_pr_intake_gate.py` fixture coverage.
- Update PR template and contributor docs for no-code/code-needed context and intake behavior.
- Update repository governance component paths.

## Out of scope

- No Goalrail product runtime behavior.
- No app, server, CLI, runner, gate/proof, or Project Spine behavior changes.
- No new dependencies.
- No reusable shared package extraction yet.
- No branch-protection mutation from the code diff itself.

## Notes

Research Gate classification: R1 quick scan. This is a bounded repository-governance change based on already-tested Punk/Signum-adjacent PR intake patterns, mapped to Goalrail-specific PR template sections and high-risk surfaces. It does not change product canon, MVP scope, or runtime trust semantics.
