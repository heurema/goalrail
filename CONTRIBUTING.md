# Contributing to Goalrail

Thanks for your interest in contributing.

Goalrail is currently a **docs-first** repository with an evolving implementation
baseline. That means contribution quality depends on keeping the product canon,
operating model, and implementation reality aligned.

**Language:** English is preferred for public artifacts because this repository
is open to a broad contributor base. Russian is also acceptable when that makes
a contribution clearer or faster.

## Read this first

Before opening a non-trivial issue or pull request, read these files in order:

1. `docs/INDEX.md`
2. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
3. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
4. `docs/ops/STATUS.md`
5. `docs/ops/NEXT.md`
6. `docs/ops/DECISIONS.md`
7. `docs/ops/COMPONENTS.yaml`
8. `AGENTS.md`
9. `README.md`

## What is especially useful right now

At the current stage, the most helpful contributions are:

- corrections and clarifications to product / architecture canon;
- improvements to repo hygiene, contributor experience, and docs tooling;
- bounded implementation slices that clearly map to documented MVP scope;
- contradiction fixes across product docs, ops docs, and repository structure;
- better validation, proof, and guardrail mechanisms for future PRs.

## Ground rules

### 1) Respect the source-of-truth order

When documents disagree, prefer:

1. `docs/product/*`
2. `docs/ops/*`
3. chat or ad-hoc discussion

### 2) Do not silently expand MVP scope

Do not turn Goalrail into a broad generic AI platform.
Keep changes aligned with the documented MVP and current operating frame.

### 3) Do not present unimplemented capabilities as existing

If something is still conceptual, draft, prototype, or not wired end-to-end, say
so clearly in docs, code comments, commit messages, and PR descriptions.

### 4) Keep docs and implementation synchronized

If a change affects any of the following, update the relevant docs in the same PR
unless maintainers explicitly ask for a different sequence:

- user-visible behavior;
- product boundaries;
- architecture or layer model;
- component ownership / status;
- validation or proof expectations.

### 5) Keep PRs small and reviewable

Prefer narrow PRs with one clear purpose.
Large cross-cutting refactors are much harder to review in a docs-first repo.

### 6) Map implementation to documented components

If you add or change implementation, identify the affected area in
`docs/ops/COMPONENTS.yaml` and reflect status changes where needed.

### 7) Bring proof

Every meaningful PR should include some form of evidence:

- a doc rationale;
- command output;
- screenshots;
- tests;
- schema validation output;
- before/after examples;
- or a clear explanation of why proof is not yet applicable.

## Contribution flow

1. Open or find an issue for non-trivial work.
2. Make sure the idea fits current MVP boundaries.
3. Fork the repository and create a focused branch.
4. Make the smallest useful change that solves one problem well.
5. Run the relevant checks.
6. Sign off every commit using the DCO.
7. Open a PR and complete the project PR template honestly.

## Commit sign-off (DCO)

This repository uses the **Developer Certificate of Origin (DCO)** instead of a
Contributor License Agreement.

Add a sign-off line to every commit:

```bash
git commit -s -m "Your message"
```

That adds a line like:

```text
Signed-off-by: Your Name <you@example.com>
```

By doing so, you certify the terms in `DCO.md`.

## Pull request expectations

Please use the PR template and make sure it includes:

- the goal / intent of the change;
- explicit scope boundaries;
- component impact;
- documentation impact;
- validation / proof;
- anything reviewers should be careful about.

A PR may be asked to change shape, split into smaller pieces, or add docs before
merge if it moves faster than the documented product and architecture canon.

## Reporting bugs

Use the issue templates when possible and include:

- what happened;
- what you expected;
- how to reproduce;
- what area is affected;
- screenshots / logs / commands if relevant.

## Proposing features

Feature requests are welcome, but they must stay grounded in Goalrail's current
positioning and MVP boundaries. Requests that imply a large product expansion may
be deferred even if they seem technically possible.

## Security

Please do **not** report vulnerabilities in public issues.
See `SECURITY.md` for the private reporting process.

## Code of conduct

By participating in this project, you agree to follow `CODE_OF_CONDUCT.md`.
