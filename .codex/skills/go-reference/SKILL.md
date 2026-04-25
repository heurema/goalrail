---
name: go-reference
description: Reference-guided Go decision navigator for Goalrail. Use when Codex writes, reviews, or changes Go code in this repository, especially for architecture, API shape, package layout, concurrency, dependency, or test strategy decisions; consult repository context and official Go or Google Go references when useful, then choose the smallest suitable implementation without treating style guidance as hard policy.
---

# Go Reference

Use this skill as a reference compass, not as a restrictive style policy.
It should improve Go decisions without blocking straightforward work.

## Intent

Do not blindly apply a fixed local style checklist.

Before meaningful Go design or architecture decisions, check repository context and,
when useful, consult the most relevant official or high-quality Go reference.

If the implementation path is clear from the task and repository context, proceed
directly. Do not ask for confirmation just to satisfy this skill.

If there is uncertainty about the right Go approach, package boundary, API shape,
concurrency model, dependency choice, test strategy, or public/private layout,
consult the relevant references first, then proceed with the smallest suitable
reversible decision.

## Decision behavior

When working on Go code:

1. Read the relevant Goalrail product, architecture, and ops context first.
2. Prefer the smallest implementation that satisfies the current task.
3. Avoid broad frameworks, speculative abstractions, or extra layers unless repo
   context or references make them clearly useful.
4. If a decision is obvious, implement it.
5. If a decision is not obvious, check the relevant Go references below.
6. After checking references, decide and proceed.
7. Mention a reference only when it materially influenced the decision.
8. Ask the user only when repo context and references still leave a risky or
   irreversible product/architecture choice unresolved.

## Reference map

Use these references as navigation points, not rigid rules.

### General Go idioms and baseline

- Effective Go: https://go.dev/doc/effective_go

Use for naming, formatting expectations, package naming, interfaces, error
handling, concurrency idioms, and general idiomatic Go direction.

### Readability and style principles

- Google Go Style Overview: https://google.github.io/styleguide/go/
- Google Go Style Guide: https://google.github.io/styleguide/go/guide

Use for readability tradeoffs, clarity vs cleverness, simplicity,
maintainability, and whether a style concern is worth changing.
Do not use these as a reason for churn-only refactors.

### Specific style and design questions

- Google Go Style Decisions: https://google.github.io/styleguide/go/decisions
- Google Go Style Best Practices: https://google.github.io/styleguide/go/best-practices

Use when the core guide is not enough and a common Go pattern needs tradeoff
context. Treat these as references, not hard constraints.

### Code review questions

- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments

Use for common review issues around error strings, context usage, goroutine
lifetimes, interfaces, package names, and useful test failures.
Treat this as a supplement to Effective Go.

### Module and repository layout

- Organizing a Go module: https://go.dev/doc/modules/layout

Use for root vs subpackage placement, `internal/`, `cmd/`, public/private package
boundaries, server project layout, and multi-command repositories.

### Public documentation and exported APIs

- Go Doc Comments: https://go.dev/doc/comment

Use for exported names, package comments, command comments, caller-visible
behavior, and edge cases.

### Concurrency and synchronization

- Go Memory Model: https://go.dev/ref/mem

Use for goroutine synchronization, shared state, channel vs mutex reasoning,
race-sensitive code, and designs that may be too clever.

### Dependencies and security

- Managing dependencies: https://go.dev/doc/modules/managing-dependencies
- govulncheck tutorial: https://go.dev/doc/tutorial/govulncheck

Use for dependency changes, vulnerability checks, module maintenance, and deciding
whether a dependency is necessary.

## Lightweight review prompt

Before finalizing a non-trivial Go change, silently check:

- Does this fit the product and repository direction?
- Is this the smallest useful design?
- Is the package boundary clear?
- Is anything being made public too early?
- Are errors, context, and goroutine lifetimes understandable?
- Are tests close to the behavior being changed?
- Did any uncertainty require checking a Go reference?

Only mention these checks in the final response when they affect the decision.

## Output behavior

When implementation is straightforward:

- write the code or patch directly
- do not over-explain
- do not ask unnecessary clarification questions

When there was a meaningful design choice:

- briefly state the decision
- briefly state why it fits the repository
- cite or name the reference only if it was actually used

When uncertainty remains after checking references:

- choose the smallest reversible option when safe
- document the assumption
- avoid committing the repository to broad architecture prematurely
- ask the user only if the remaining choice is risky, irreversible, or outside
  the task bounds
