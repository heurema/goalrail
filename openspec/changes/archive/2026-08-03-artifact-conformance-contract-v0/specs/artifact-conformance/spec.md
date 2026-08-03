## ADDED Requirements

### Requirement: Every Context and Intent pair selects one artifact contract
**Intent IDs:** OUT-1, SIG-1, SIG-2, SIG-3

Goalrail SHALL select an artifact contract before it interprets the semantics of
a Context Pack and Intent Snapshot pair. The artifact contract identifier and
version SHALL be metadata distinct from the Context Pack ID and version, Intent
ID and version, Context binding, and OpenSpec schema name or version.

Current canon SHALL emit these identical exact contract-v1 rows in both
artifacts:

```markdown
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
```

Contract v1 SHALL accept only that repository-owned identifier and supported
integer version. If either artifact declares any contract field, both artifacts
MUST carry a complete equal declaration; a partial, malformed, mismatched, or
unknown declaration SHALL fail closed and MUST NOT be retried as legacy input.

Only a pair in which both contract declarations are absent SHALL enter legacy
mode. Legacy mode SHALL be a finite repository-owned allowlist that includes the
pinned `vN` and `version N` Context binding spellings and the specifically named
historical table and heading variants. It MUST NOT infer aliases, correct typos,
or expand when another artifact appears in the active or archive tree. Contract
selection and parsing SHALL NOT rewrite either source artifact.

The profile metadata allowlists SHALL include all existing supported lifecycle
metadata, including optional `Previous version`, `Run references`, `Resolves`,
and `Disposition` fields where applicable. A contract selector MUST NOT reject a
valid answering intent merely because it carries its existing escalation
resolution provenance.

#### Scenario: Canon emits contract v1
- **WHEN** Goalrail materializes a new Context Pack and Intent Snapshot pair from canon
- **THEN** both artifacts carry the exact equal `goalrail-context-intent` version 1 declaration alongside their artifact-specific identity and version metadata

#### Scenario: Contract v1 pair is consumed
- **WHEN** both artifacts carry the supported equal contract v1 declaration
- **THEN** Goalrail selects contract v1 before parsing pair semantics and does not enter legacy mode

#### Scenario: Pair has no contract declaration
- **WHEN** neither artifact declares any artifact contract field and every syntax variant belongs to the pinned legacy allowlist
- **THEN** Goalrail selects the named legacy mode and parses only those allowed variants

#### Scenario: A declaration is partial or differs across the pair
- **WHEN** one declaration is incomplete, only one artifact declares a contract, or the two declarations differ
- **THEN** Goalrail rejects the pair as a declared-contract failure without attempting legacy parsing

#### Scenario: A declared contract version is unknown
- **WHEN** either artifact declares an unsupported artifact contract version
- **THEN** Goalrail rejects the pair without fallback, typo correction, alias discovery, or source rewriting

#### Scenario: An unrelated change appears
- **WHEN** a new active or archived change uses a syntax not present in the pinned legacy allowlist
- **THEN** that syntax remains unsupported until an explicit versioned contract or allowlist change adds it

#### Scenario: An answering intent carries lifecycle provenance
- **WHEN** a supported pair's Intent carries valid `Resolves` and `Disposition` metadata
- **THEN** its selected profile preserves and validates that metadata rather than rejecting it as an unknown contract field

### Requirement: Contract and format failures carry safe actionable diagnostics
**Intent IDs:** OUT-2, SIG-3, SIG-4

Every artifact contract selection, pair format, Context binding, or referenced
Context evidence failure SHALL
return one structured diagnostic carrying a stable repository-owned code, the
artifact kind and a safe logical path relative to the caller's bounded root,
the selected contract version or named legacy mode — or the exact `unselected`
state when selection itself failed — the failing field or section, a bounded
redacted observation, the accepted expectation, and a concrete repair hint.

Diagnostic codes SHALL come from a finite registry and SHALL distinguish at
least invalid declaration, unsupported contract, pair declaration mismatch,
invalid artifact format, invalid Context binding, and missing Context evidence.
Rendering SHALL be deterministic. The observation and rendered form MUST escape
control characters, bound attacker-controlled content, and redact secret-shaped
values; they MUST NOT reproduce an entire line, section, artifact, absolute user
path, or credential-like value merely to explain the failure.

The pair compiler and local-run verifier SHALL return the structured diagnostic
as an error value that callers can inspect without parsing its rendered text.
Ambient SHALL retain the same diagnostic semantics in its binding evidence when
an active artifact claims confirmed status but cannot conform.

#### Scenario: A malformed field is rejected
- **WHEN** a pair contains a malformed required field under its selected contract or legacy mode
- **THEN** the consumer returns the pinned diagnostic code, location, bounded redacted observation, expectation, and repair hint for that field

#### Scenario: Untrusted content resembles a secret
- **WHEN** rejected artifact content contains a credential-like value or control characters
- **THEN** the structured observation and rendered message redact or escape that content and remain within the fixed diagnostic bound

#### Scenario: A caller inspects the failure
- **WHEN** the compiler or local-run verifier receives a conformance failure
- **THEN** it can branch on structured diagnostic fields without matching human-readable error text

#### Scenario: A diagnostic is rendered twice
- **WHEN** the same diagnostic value is rendered more than once
- **THEN** the output is byte-identical and contains no absolute user path or unbounded artifact content

### Requirement: A pinned corpus is the conformance oracle
**Intent IDs:** OUT-3, OUT-4, SIG-5

A deterministic repository-owned corpus SHALL define supported pair behavior.
Every case SHALL have a stable ID, local provenance, immutable fixture inputs
with an expected SHA-256 digest for every file, an explicit contract-v1 or named
legacy input mode, independently authored expected domain semantics or an exact
structured diagnostic, and an explicit set of applicable consumer lanes.
Expectations and fixture digests MUST NOT be generated or rewritten from the
implementation under test.

The corpus SHALL include contract-v1 positives, pinned legacy positives, parser
negatives, cross-file binding negatives, and malformed-confirmed ambient cases.
If repository examples are copied into fixtures, their provenance SHALL be
recorded, and the runner MUST NOT discover cases by scanning the changing active
or archive trees. A review source used as provenance MUST NOT own, veto, or
dynamically decide an expected result.

One deterministic repository command SHALL execute every declared case in every
applicable lane and fail if a case, fixture, expectation, provenance record, or
declared lane is missing, unexpectedly rewritten, skipped, or produces a
different result.

#### Scenario: The corpus runs unchanged
- **WHEN** the conformance command runs twice at the same repository revision
- **THEN** it executes the same case and lane matrix and reports the same ordered results

#### Scenario: An active change is added
- **WHEN** an unrelated change directory is added outside the corpus fixtures
- **THEN** corpus membership and expected results remain byte-identical

#### Scenario: An expectation is absent
- **WHEN** a manifest case has no independently pinned semantic or diagnostic expectation
- **THEN** the corpus command fails rather than deriving an expectation from current consumer output

#### Scenario: A fixture differs from its manifest digest
- **WHEN** any pair or lane-setup fixture does not match its pinned SHA-256 digest
- **THEN** the corpus command fails before invoking a consumer lane

#### Scenario: A declared lane does not execute
- **WHEN** a case names an applicable consumer lane that the runner skips or cannot invoke
- **THEN** the corpus command fails and identifies the missing case-lane result

### Requirement: Consumer lanes prove their own behavior
**Intent IDs:** OUT-4, SIG-1, SIG-4, SIG-5

The conformance matrix SHALL expose three separate lanes: the pair compiler
through `LoadChange`, local-run verification through `IntentResolver`, and
ambient active-intent discovery. Each lane MUST invoke its named production
boundary and report its own result. Success in one lane MUST NOT be copied,
adapted, or reported as execution evidence for another lane.

The compiler and local-run lanes SHALL compare normalized Context Pack and
Intent Snapshot semantics or the exact structured diagnostic. The ambient lane
SHALL compare binding identity, unbound reason, and retained diagnostic evidence
while preserving background non-interference.

#### Scenario: One case applies to all lanes
- **WHEN** a manifest case declares compiler, local-run, and ambient applicability
- **THEN** the runner invokes all three named boundaries and records three independent results

#### Scenario: Compiler behavior succeeds alone
- **WHEN** the compiler lane passes but another applicable lane returns a different result
- **THEN** the corpus command fails and does not describe compiler success as runtime or ambient proof

#### Scenario: A lane does not apply
- **WHEN** a case explicitly excludes a lane because that consumer cannot receive the input mode
- **THEN** the runner records the declared non-applicability instead of treating it as a pass or silently skipping it

### Requirement: Conformance relations compare semantics and preserved data
**Intent IDs:** OUT-5, SIG-1, SIG-2, SIG-3

Corpus relations SHALL assert domain behavior rather than parse success alone.
The supported legacy `vN` and `version N` Context binding spellings SHALL yield
equal Context Pack and Intent Snapshot semantics. Mutating the bound Context
Pack identity or version, making the two contract declarations unequal, or
declaring an unknown contract version SHALL yield the pinned failure rather
than normalized success.

The corpus SHALL pin six-column and seven-column Context Items, old and current
supported semantic-group headings, Verification recipe preservation, and the
real repository pair regressions as explicit fixture cases. Preserved domain
fields SHALL survive every supported parse and render or copy path exercised by
their direct consumers without truncation, invention, or loss. These baselines
MUST use pinned copies and MUST NOT reopen or replace the existing update
concurrency, previous-canon migration, recipe-retention, or Previous version
requirements.

#### Scenario: Supported binding spellings are equivalent
- **WHEN** otherwise identical pinned legacy pairs differ only between the supported `vN` and `version N` Context binding spellings
- **THEN** their parsed Context Pack and Intent Snapshot domain semantics are equal

#### Scenario: Context identity or version is mutated
- **WHEN** a valid pair's Intent binding names a different Context Pack ID or version
- **THEN** the pair is rejected with the pinned binding diagnostic rather than accepted on parse success

#### Scenario: A preserved field traverses a supported path
- **WHEN** a fixture carrying a Verification recipe, Previous version, or another retained domain field traverses its applicable direct-consumer path
- **THEN** the resulting domain value preserves the fixture expectation without truncation, invention, or loss

#### Scenario: Historical syntax remains pinned
- **WHEN** the corpus executes its six/seven-column, old/current-heading, recipe-preservation, and real-pair fixtures
- **THEN** each fixture produces its independently pinned semantics and no dynamic repository scan supplies or changes the expectation
