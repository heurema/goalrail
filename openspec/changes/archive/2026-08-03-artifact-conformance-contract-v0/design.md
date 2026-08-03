## Context

The merged source of truth still has three independent readers with different
evidence strength. `LoadChange` compiles a change but has no non-test caller;
local-run verifies the referenced intent before prepared state is written; and
ambient discovery reads only `intent.md` and skips failures. The shared parser
also accepts only `<id> version <number>` although current confirmed changes use
`<id> v<number>`, while canon emits no artifact-contract metadata at all.

This design adds one bounded pair protocol inside the existing OpenSpec adapter
and wires the three consumers to it without moving OpenSpec concepts into the
canonical domain. It adds no service, dependency, runtime infrastructure,
external review authority, migration, or effect permission.

## Goals / Non-Goals

**Goals:**

- Select contract v1 or one finite legacy profile before pair semantics.
- Return the same typed, safe diagnostic from direct pair consumers.
- Make compiler, local-run, and ambient behavior independently testable against
  one stable, hand-authored corpus.
- Keep current supported syntax and preserved domain fields explicit without
  making the changing OpenSpec tree the compatibility oracle.
- Retain invalid confirmed ambient evidence while keeping the hook fail-quiet.

**Non-Goals:**

- Generalize the protocol beyond Context Pack and Intent Snapshot pairs.
- Rewrite active or archived artifacts, infer aliases, repair typos, or fall
  back from a declared contract failure.
- Replace existing domain validation, file-boundary controls, canon update
  controls, or review workflow.
- Introduce an executable old-binary oracle, dynamic active-tree scan, provider,
  database, workflow engine, parser generator, or external effect.

## Decisions

### 1. Repeat two exact contract fields in both artifacts

New canon shall emit these exact metadata rows in both `context.md` and
`intent.md`:

```markdown
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
```

The two fields make contract identity visibly distinct from artifact lineage and
from the OpenSpec schema. Repeating them lets the reader reject a mixed or
partially upgraded pair rather than assigning an implicit contract to either
file. A single declaration in Intent was considered, but rejected because a
Context file could then be paired under a contract it never declares. Inferring
the contract from artifact or schema version was rejected because those already
represent different lineages. **Evidence:** CTX-2, CTX-9.

The selector has exactly two successful modes:

- `goalrail-context-intent/v1`: both artifacts carry complete, equal exact
  declarations. This profile accepts the current canonical metadata,
  seven-column Context Items with `Verification recipe`, neutral intent group
  headings, and `<id> version <number>` Context binding.
- `goalrail-context-intent/legacy-v0`: both artifacts omit both contract fields
  and all syntax belongs to a fixed table: the supported `vN` and `version N`
  bindings, six/seven-column Context Items, and the already supported old/current
  headings.

Any exact contract field in only one artifact, an incomplete pair of fields,
different declarations, or an unknown version is a declared-contract failure.
Legacy metadata keys are also allowlisted; an unknown key therefore cannot turn
a misspelled contract field into accepted legacy input. The selector never
consults active or archived directories to grow the table. **Evidence:** CTX-2,
CTX-4, CTX-9, CTX-10.

The allowlists enumerate the current Context and Intent lifecycle metadata, not
only fields used for contract selection. In particular they include optional
`Previous version`, `Run references`, `Resolves`, and `Disposition` where the
existing reader supports them. Contract v1 changes the envelope selector; it
does not invalidate answering-intent provenance or create a fourth semantic
intent group. **Evidence:** CTX-3, CTX-4, CTX-12.

Fixtures derived from `openspec-adoption-v0` and `pre-pr-review-v0` isolate and
pin their Context binding declaration spellings inside otherwise minimal valid
pairs. They do not claim that every unrelated historical syntax in those whole
working change directories belongs to legacy-v0. The existing repaired #56
pair remains the pinned real-pair regression. This keeps each compatibility
claim single-variable and traceable. **Evidence:** CTX-2, CTX-3, CTX-10.

### 2. Centralize pair conformance in the OpenSpec adapter

Add a pair API in `internal/adapters/openspec` that accepts bounded Context and
Intent bytes plus safe logical paths and returns:

```text
ConformedPair {
  Selection ContractSelection
  Context   domain.ContextPack
  Intent    domain.IntentSnapshot
}
```

The API parses metadata once, selects a profile, parses both artifacts with that
profile, validates Context binding and evidence references, then returns domain
values. `ContractSelection` remains adapter metadata; it is not added to
`domain.ContextPack`, `domain.IntentSnapshot`, WorkSpec, or receipts. Existing
single-artifact readers remain only for flows that legitimately do not have a
pair, including the already specified archived/foreign-schema intent-only path.
**Evidence:** CTX-7, CTX-8, CTX-9.

`CompiledChange` keeps its existing Intent and Proposal fields and additionally
exposes an optional `*ConformedPair` for pair-backed changes; intent-only
compiled changes leave it empty. The concrete local-run resolver gains
`Resolve`, which returns `ResolvedIntent { Intent, Pair *ConformedPair }` after
repository, digest, status, and reference checks. Its existing
`Verify(...) error` method delegates to `Resolve` and discards the value, so the
local-run service interface and lifecycle do not widen merely to support
measurement. These two observable results let the corpus compare semantics
without reaching around the named consumer boundary. **Evidence:** CTX-3,
CTX-7.

Filesystem ownership stays with each lane. Local-run keeps its existing
repository containment, same-descriptor regular-file checks, byte bound, frozen
intent digest, and pre-persistence ordering. Ambient uses the same bounded
regular-file primitive for both sibling artifacts. The development compiler
keeps its change/proposal lifecycle but passes bounded pair bytes through the
shared API. Centralizing semantic parsing without centralizing filesystem
authority avoids weakening the stronger local-run boundary. **Evidence:**
CTX-3, CTX-7, CTX-8.

The current try-new-then-fallback parser branches are replaced at the pair entry
point by profile tables. Lower-level parsing helpers receive the selected
profile; a parser error cannot cause another profile to run. This is what makes
"no declared-contract fallback" structural instead of a convention at three
call sites. **Evidence:** CTX-2, CTX-8.

### 3. Use one typed diagnostic with a finite code registry

`ArtifactDiagnostic` lives in `internal/adapters/openspec`, implements `error`,
and preserves existing `errors.Is` behavior through an unexported cause and
`Unwrap`. It contains only:

- `code`;
- safe logical `path` and `artifact_kind`;
- `contract_mode` (`goalrail-context-intent/v1`, the named legacy mode, or
  exact `unselected` when contract selection itself failed);
- `field_or_section`;
- bounded redacted `observation`;
- `expectation`;
- `repair_hint`.

The initial code registry is finite: `ARTIFACT_INPUT_UNAVAILABLE`,
`ARTIFACT_CONTRACT_INVALID`, `ARTIFACT_CONTRACT_UNSUPPORTED`,
`ARTIFACT_CONTRACT_MISMATCH`, `ARTIFACT_FORMAT_INVALID`,
`ARTIFACT_CONTEXT_BINDING_INVALID`, and
`ARTIFACT_CONTEXT_REFERENCE_MISSING`. Call sites may retain their existing
non-conformance errors, but pair contract selection, pair format, Context
binding, and referenced-evidence failures use one of these codes and return the
diagnostic unchanged. An `unselected` diagnostic still records a bounded safe
observation of the declaration state plus the exact accepted expectation; it
never pretends that a failed selector chose legacy-v0. **Evidence:** CTX-8.

Rendering is a pure deterministic method. Logical paths are slash-normalized
and rejected if absolute or escaping. Observation values are built from the
small failing token or a fixed classification, limited to 80 Unicode code
points, control-character escaped, and replaced completely when secret-shaped.
Whole source lines, sections, claims, recipes, absolute paths, and raw parser
errors never enter the diagnostic JSON or rendered message. An alternative that
wrapped existing prose errors was rejected because callers would still need to
parse strings and unsafe raw content could survive inside the wrapper.
**Evidence:** CTX-4, CTX-8.

### 4. Preserve each consumer's lifecycle while sharing semantics

`LoadChange` uses the pair API before status, flow-intent, proposal, and coverage
validation. It remains the compiler lane and is never presented as runtime
evidence. Pair diagnostics return unchanged; proposal errors remain outside the
artifact-conformance contract. **Evidence:** CTX-3, CTX-7.

`IntentResolver.Resolve` passes the already frozen, bounded bytes into the pair
API and returns its typed diagnostic or resolved intent with optional conformed
pair semantics. `Verify` delegates to it before the local-run service writes
prepared state. The intent-only branch returns no pair and selects no pair
contract. No run ID, launch claim, or provider call is added to a failed
preparation. **Evidence:** CTX-3, CTX-7, CTX-8.

Ambient discovery changes from `(*IntentRef, reason)` to an `IntentResolution`
value containing an optional reference, an unbound reason, and an ordered list
of `BindingDiagnostic { change, diagnostic }`. It reads each active change in
sorted order. Shared envelope inspection determines whether the change requires
a pair: a sibling Context, the Goalrail intent schema, a declared Context
binding, or any artifact-contract field does; an existing foreign-schema flow
with none of those signals retains the intent-only reader. The same bounded
inspection determines only whether the Intent exactly claims
`Status: confirmed`; full binding still requires the selected pair or intent-only
path. If that confirmed claimant fails required input, contract, format, binding,
or reference validation, its diagnostic wins over every otherwise valid
candidate and the resolution remains unbound with reason
`INVALID_CONFIRMED_INTENT`. Missing, incomplete, and candidate artifacts that do
not make that exact claim remain ordinary work in progress. **Evidence:** CTX-6,
CTX-8.

`QuestionRecord` gains optional structured `binding_diagnostics` and new records
use `goalrail.ambient-question/v1`; existing v0 records remain append-only and
are not migrated. `StopSession` copies the resolution into the record but the
hook contract does not change: diagnostic discovery is never injected into the
session, awaited by it, or converted into a run outcome. A prose-only unbound
reason was considered, but rejected because it would lose the stable code,
location, safe expectation, and change correlation required for repair.
**Evidence:** CTX-6, CTX-8.

### 5. Make the corpus test-only, explicit, and lane-addressed

Create `internal/conformance/testdata/artifact-contract-v1/manifest.json` with
immutable case directories. Each manifest entry carries a stable case ID,
provenance, input mode, fixture paths and their hand-pinned SHA-256 digests,
applicable ordered lanes, and either a hand-authored semantic projection or
complete diagnostic expectation. There is no update-golden command. Duplicate
IDs, unknown modes or lanes, escaping paths, missing or mismatched digests,
missing provenance, missing expectations, and skipped declared lanes fail the
runner before consumer execution. **Evidence:** CTX-10, CTX-11.

`internal/conformance/artifact_contract_test.go` provides one command:

```sh
go test ./internal/conformance -run '^TestArtifactConformanceCorpus$'
```

The test materializes each case in a temporary repository and invokes three
independent adapters: `LoadChange`, local-run `IntentResolver.Resolve` (the path
used by `Verify`), and ambient `OpenSpecIntents.ActiveConfirmedIntent`.
Compiler-positive cases receive an explicit hand-authored valid Proposal fixture;
the runner never synthesizes it from observed output. Lane-specific setup is
explicit; one result is never reused as another lane's evidence. Ambient record
persistence and non-interference also retain focused tests in
`internal/ambient`. A dynamic scan of repository changes and differential
execution of v0.1.8 were rejected: the former makes WIP normative, and the
latter reproduces the strict parser defect. **Evidence:** CTX-7, CTX-10,
CTX-11.

Semantic projections compare the fields direct consumers retain, not merely a
nil error. Relation cases cover binding spelling equivalence, identity/version
mutation, contract mismatch/unknown version, six/seven columns, old/current
headings, recipes, Previous version, and the repaired real pair. Existing tests
for update concurrency and canon migration remain separate and unchanged.
**Evidence:** CTX-3, CTX-4, CTX-5, CTX-10, CTX-12.

### 6. Advance current canon copies without changing historical canon

Before changing current canon, freeze its repository overlay as the next
append-only previous-canon fixture and record its file digests after the existing
`canon-v1` entry. This lets repositories materialized by the current release be
classified as behind rather than locally edited after the contract-bearing
canon ships. The existing `canon-v1` fixture and digest entry remain
byte-identical and oldest-first; the migration mechanism itself is not changed.
**Evidence:** CTX-3, CTX-5.

Then add the two exact fields to Context and Intent templates in both
`openspec/schemas/goalrail-intent` and `internal/harness/canon`, and update the
corresponding mirrored schema instructions and forcing-function tests in the
same change. Historical previous-canon fixtures are recovery input, not extra
current canon copies. The embedded/current byte-identity, previous-canon
transition, and pinned CLI checks remain required. **Evidence:** CTX-3, CTX-4,
CTX-5, CTX-9.

No existing change artifact is rewritten. The change being implemented and the
current active/archive tree continue to exercise legacy-v0 until newly generated
artifacts adopt contract v1. **Evidence:** CTX-2, CTX-3, CTX-9, CTX-10.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| Context and Intent bytes | Repository-authored, therefore untrusted parser input | Non-empty bounded pair; or the existing explicitly allowed intent-only lifecycle | Missing required sibling, empty, oversized, unreadable, non-regular, or repository escape | Local-run preserves frozen bytes, digest, same-descriptor read, and pre-persistence gate; compiler reads once; ambient reads each sorted change once and remains fail-quiet if the worktree changes | Pair unit tests, local-run boundary tests, ambient mutation/error tests, corpus lanes |
| Artifact contract metadata | Exact bold metadata in both pair artifacts | Equal `goalrail-context-intent` version 1; or complete absence in both | Partial, malformed, unequal, unknown, duplicated, or unknown metadata under the selected profile | Selection happens once from the same bounded bytes later parsed; no retry under another mode and no source rewrite | Selector table tests and positive/negative corpus cases |
| Legacy syntax | Repository-owned finite profile | Pinned `vN`/`version N`, six/seven columns, old/current headings, and supported lifecycle metadata including resolution provenance | Any unlisted key, heading, column shape, alias, typo, or declared-contract failure | The table changes only through a reviewed versioned source change; active/archive directories are never consulted | Manifest cases with minimized declaration, answering-intent, and semantic fixtures |
| Active change names, schema, and status claim | Ambient repository layout, mutable and untrusted | Sorted directories; pair-required signals; exact top-level `Status: confirmed`; candidate/incomplete WIP; existing schema-authorized intent-only flow | Archive entry, non-directory entry, ambiguous valid set, or contract-invalid confirmed claimant for binding | Snapshot sorted names; bounded single reads; choose pair vs intent-only once from the same envelope bytes; a confirmed claimant that fails later forces unbound; internal failure never affects hook success | Ambient resolver and `StopSession` tests plus ambient corpus lane |
| Corpus manifest and fixtures | Reviewed repository test data; trusted as the expected-result source, not as runtime input | Known schema, unique IDs, safe relative fixture paths, pinned SHA-256 per file, explicit mode/lanes/provenance/expectation | Missing or unknown fields, path escape, digest mismatch, duplicate ID, generated expectation, skipped lane | Read-only during tests; verify every digest before lanes; no golden updater; same revision yields same ordered matrix | Manifest self-validation, tamper case, and two-run determinism test |
| Current and previous canon data | Repository and embedded current copies plus append-only recovery fixtures owned by Goalrail | Byte-identical current copies with exact contract rows; exact historical fixtures and oldest-first digest entries | Drift between current copies, changed historical bytes, missing pre-change snapshot, or reordered history | Freeze current canon before changing it; append one previous entry; never rewrite earlier fixtures | Canon byte-identity, forcing-function, pinned CLI, digest, and every previous-canon transition test |

## Correlation and Evidence

- A corpus result is keyed by `case_id + lane`; provenance explains the source
  but never decides the result.
- `ContractSelection` accompanies a parsed pair but does not alter canonical
  Context or Intent identity.
- Every diagnostic carries code, safe path, kind, mode, and location. Ambient
  wraps it with the sorted change name before persistence.
- A v1 question record joins the question occurrence, optional intent binding,
  unbound reason, and zero or more binding diagnostics. It remains a question,
  never a run or review disposition.
- Existing sentinels remain reachable through `errors.Is`; structured callers
  use `errors.As` rather than parsing rendering.

## Measurement and Stop Conditions

- The corpus command MUST report every manifest case in every declared lane,
  with 100 percent exact semantic or diagnostic agreement.
- Targeted adapter, local-run, ambient, harness, and conformance tests; race tests
  for touched packages; `go vet ./...`; `go test ./...`; strict OpenSpec
  validation; and diff audit must pass before implementation handoff.
- Stop instead of widening the slice if success requires permissive parsing,
  dynamic active-tree discovery, rewriting existing artifacts, forcing a pair
  into intent-only flows, exposing raw content, blocking an ambient session, or
  adding a runtime dependency.
- If one lane disagrees, fix or explicitly scope that lane; never copy a passing
  compiler result or weaken the expectation to make the matrix green.
- A future intentional grammar change creates contract v2 or a newly reviewed
  legacy profile version; it does not mutate contract v1 expectations in place.

## Risks / Trade-offs

- **[Repeated declarations can drift]** → Equality is intentional: fail with a
  repair hint and let canon generate both exact rows.
- **[Strict legacy metadata may expose previously hidden syntax]** → Add only
  isolated, provenance-backed variants that confirmed intent requires; never
  admit an entire active tree by observation alone.
- **[A status pre-inspection could overstate malformed WIP]** → Recognize only
  the exact top-level confirmed value using the shared bounded metadata parser;
  candidate, missing, or unreadable status is not promoted.
- **[Diagnostics could leak repository content]** → Store fixed classifications
  or bounded sanitized tokens, reject unsafe paths, redact secret-shaped values,
  and pin hostile fixtures.
- **[Ambient evidence schema advances]** → Emit v1 only for new append-only
  records, retain v0 files untouched, and keep the new field optional.
- **[Pinned fixtures duplicate repository examples]** → Record immutable
  provenance and compare fixture digests; duplication is the deliberate cost of
  keeping WIP out of the oracle.
- **[Three lanes make the test harness slower and more coupled]** → Keep the
  manifest small, test-only, deterministic, and explicit; focused package tests
  retain faster diagnosis.

## Rollback

Before any release, revert the pair API integrations, current canon content,
new previous-canon snapshot, diagnostic type, ambient v1 record field, and
test-only corpus as one bounded change; historical artifacts and the existing
`canon-v1` snapshot need no restoration because they were never modified.

After contract-v1 artifacts have been generated outside this repository, do not
remove the v1 reader or reinterpret its fixtures. A safe rollback may stop new
canon emission or disable a consumer integration while retaining read support.
Ambient v1 records are append-only evidence and are never deleted or rewritten;
older code may ignore their optional diagnostic field.

## Open Questions

None.
