## 1. Correlation Feasibility Gate

- [x] 1.1 Build a dependency-free local probe around representative Codex hook inputs and immutable `change_id`/`run_id` context; do not build the production adapter yet.
- [x] 1.2 Add deterministic cases for startup, resume, compaction where exposed, child-agent inheritance, absent context, conflicting context, and two concurrent changes.
- [x] 1.3 Run one disposable local CLI session and one disposable Codex App session, then record whether each produces a verified join, `UNLINKED`, or an unsupported result without manual identifier copying.
- [x] 1.4 Stop at an owner gate if deterministic App and CLI binding cannot be achieved without shared mutable state, transcript parsing, or a Langfuse plugin fork; do not proceed by weakening the contract.

## 2. Minimal Provider-Neutral Harness

- [x] 2.1 Initialize only the minimal Go module needed for provider-neutral Intent Snapshot, lineage, evidence-event, assessment, and canary-manifest types; add no runtime-provider dependencies.
- [x] 2.2 Implement validation for candidate/confirmed lifecycle, stable intent IDs, material amendments, required proposal coverage, and the rule that intent does not grant effect authority.
- [x] 2.3 Implement a repository-local append-only evidence record with correction links, sensitive-payload rejection, and tests proving that prior events cannot be silently replaced.
- [x] 2.4 Implement deterministic variant assignment and report calculations, including unequal-denominator rates, missing assessments, abandonment counts, pass signals, and hard stop signals.

## 3. OpenSpec and Codex Adapters

- [x] 3.1 Add the smallest OpenSpec adapter that reads the confirmed Intent Snapshot, blocks candidate intent, and validates proposal-to-intent coverage without leaking OpenSpec types into the canonical domain.
- [x] 3.2 Implement the project-local Codex correlation adapter selected by task 1, using provider-authoritative hook or launch-receipt identity and immutable per-run context with explicit `UNLINKED` behavior.
- [x] 3.3 Reference existing Langfuse observations by verified session or trace identity while remaining functional when Langfuse evidence is missing; do not fork the plugin or add credentials.
- [x] 3.4 Add integration tests that exercise intent confirmation through proposal validation, execution lineage, append-only evidence, and independent `green` versus `match` assessment.

## 4. Canary Manifest and Operator Flow

- [x] 4.1 Define and freeze eligible-change rules, the 15-position `flow → flow → baseline` assignment, check-recording rules, owner-assessment timing, missing-data treatment, and the timestamp-derived overhead formula.
- [x] 4.2 Add a minimal command or documented workflow to start, inspect, deliver, abandon, assess, and correct a canary change; keep owner assessment explicit and add no dashboard.
- [x] 4.3 Add fixtures proving `wrong-but-green`, a prevented material misunderstanding, wording-only correction, process-caused abandonment, wrong join, unresolved links, evidence rewrite, `PASS`, `STOP`, and `RESHAPE` outcomes.
- [x] 4.4 Verify that all emitted repository evidence contains only approved metadata and references, never raw prompts, transcripts, credentials, or source content.

## 5. Reversible Activation Canary

- [x] 5.1 Run a synthetic end-to-end local change with no external effects and preserve its validation, lineage, assessment, and report evidence.
- [x] 5.2 Exercise disable/rollback behavior and confirm that stopping new assignments preserves prior append-only evidence and does not affect unrelated Codex or Langfuse use.
- [x] 5.3 Produce a concise owner review packet with correlation receipts, measured synthetic overhead, known limitations, and a go/no-go request for the separate 15-real-change canary; do not start it automatically.
