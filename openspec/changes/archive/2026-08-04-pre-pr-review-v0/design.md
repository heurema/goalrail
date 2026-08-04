## Context

*(recompiled against confirmed intent v6)*

Independent review is already implemented, so this is a reconciliation design,
not a greenfield design. The open task ledger understates what shipped and also
retains requirements superseded by later intent (CTX-14). More importantly, the
historical Baseline run did not prove the pre-PR success signal: a stronger
review found material issues after the PR opened, and no receipt describes that
PR's final head (CTX-15).

Three live seams remain. Codex desktop exposes `CODEX_THREAD_ID` while the
detector recognizes `CODEX_SESSION_ID` (CTX-16). An omitted base currently means
the local ref `main`, which can lag `origin/main` while still producing an
internally valid receipt over the wrong range (CTX-17). The receipt transition
has unit coverage, but the public initialization, review, and diagnosis flow has
not been proved with JSON output and no terminal (CTX-18).

The existing provider constraints still shape the repair. `codex review` and
`claude -p` must be used through their own non-interactive subscription paths;
Codex cannot combine its scope flags with custom instructions, and Claude's
safe invocation receives a rendered diff because it has no shell (CTX-3,
CTX-4). A Claude session may also carry a Codex companion variable, so provider
prefix matching is unsafe (CTX-13). The budget control remains a caller-owned
command rather than product-specific infrastructure (CTX-10).

## Goals / Non-Goals

**Goals:**

- recognize the supported current authoring surface without an override and
  preserve deterministic fresh/cross selection (OUT-1, SIG-1);
- resolve valid repository inputs before the optional budget policy and prevent
  an omitted base from selecting a stale local shadow (OUT-2, OUT-10, SIG-6);
- transport the exact reviewed diff to a refuter and keep every report verbatim
  (OUT-5, OUT-8, SIG-3);
- preserve public JSON, no-terminal, and diagnosis exit semantics (OUT-6,
  SIG-4);
- leave the historical failed proof intact and complete a bounded replacement
  canary on the reconciliation branch before its PR opens (OUT-3, OUT-4,
  SIG-5).

**Non-Goals:**

- no parsing, normalization, scoring, or disposition of review findings
  (NG-1, NG-4, NG-8);
- no mechanical commit, push, PR, or merge gate (NG-2);
- no vendor wrapper, pinned CLI, third-party reviewer, or new dependency
  (NG-3);
- no prompt, autonomous coordinator, automated fix loop, or unbounded retry
  (NG-5, NG-6, NG-8);
- no risk judgement or Goalrail-owned refute trigger (NG-7);
- no fetch or hosting-service query during base discovery (NG-9).

## Decisions

### Goalrail remains one review step; the caller owns the loop

The calling agent runs review, reads the verbatim report, decides whether the
confirmed intent permits a fix, applies any fix, and invokes review again. It
declares the ceiling before the first paid invocation. Goalrail performs one
command round, writes one receipt only after success, and never decides whether
findings remain. This keeps the existing workspace review discipline executable
without adding a coordinator or making report prose a control input (CTX-9,
CTX-14).

The fixed marker requested by the committed reviewer instructions remains
reviewer prose. The caller may use it as a reading aid; Goalrail does not
require, parse, validate, or act on it.

### Authorship uses provider marker sets, not prefixes

Detection will represent each provider as a set of registered primary markers
rather than one string. Claude keeps `CLAUDECODE`. Codex adds the observed
current desktop marker `CODEX_THREAD_ID`; the already accepted
`CODEX_SESSION_ID` remains as a compatibility marker. Multiple non-empty markers
for the same provider collapse to one provider identity. `CODEX_COMPANION_SESSION_ID`
and every other unregistered `CODEX_*` variable remain non-author evidence
(CTX-13, CTX-16).

An explicit valid override wins. With no detected provider, or primary markers
from more than one provider, the author is `unknown`; selection remains
deterministic and proceeds. Only an invalid explicit override is an input error.
Every registered and companion marker is stripped from spawned reviewer
environments so a fresh invocation cannot inherit the author's session.

Alternatives rejected:

- matching every `CODEX_*` variable — rejects CTX-13's measured companion case;
- replacing `CODEX_SESSION_ID` outright — creates an unnecessary compatibility
  break when both primary markers can safely collapse to Codex;
- refusing unknown or ambiguous authorship — contradicts the confirmed fresh
  mode and turns routing evidence into a policy gate.

### Omitted base resolves from local remote-HEAD metadata

The `--base` flag will have no semantic default. When explicitly supplied, its
exact value wins. When omitted, the resolver enumerates local symbolic refs at
`refs/remotes/*/HEAD`, keeps only targets that resolve to commits, and accepts
exactly one target. That selected remote-tracking ref wins over any same-named
local branch. Zero or multiple valid targets produce an input error naming
`--base` before the gate or any provider invocation (CTX-17).

The selected ref is resolved once to an immutable commit. The receipt retains
both the human-readable selected ref and that commit; all diff computation and
provider ranges use the immutable commit. The resolver uses local Git plumbing
only (`for-each-ref`/symbolic-ref observation and `rev-parse`) and never fetches
or contacts a hosting service.

Alternatives rejected:

- fixed local `main` or `master` — repeats the silent stale-shadow failure in
  CTX-17 and invents naming convention as repository truth;
- `git remote show` without a guaranteed no-network path, provider APIs, or a
  fetch — violates NG-9 and makes review latency and credentials part of base
  selection;
- guessing among several remote HEADs — produces a valid receipt for an
  unproved range, the exact failure this repair prevents.

### One canonical reviewed diff feeds evidence and refutation

After base and head commits are frozen, the review step renders the canonical
reviewed-range diff once with external drivers, text conversion, color, and
reader configuration disabled. Those exact bytes produce the reviewed-range
digest. Claude receives those bytes because its safe tool surface has no shell;
a refuter receives the same bytes together with the first report. Codex may
read the immutable commit range through its native review interface, preserving
the vendor contract that disallows custom instructions plus a scope flag
(CTX-3, CTX-4).

The full-branch digest remains separate from the reviewed-range digest for
incremental rounds. A previous receipt narrows the next range only when its
base commit matches, its head still resolves, and it carries a reviewed-range
digest. Otherwise the command performs a full branch-range review rather than
claiming an unproved chain.

### Valid repository inputs precede the budget gate

Repository root, attached branch, selected base ref, immutable base/head
commits, and a non-empty range resolve before any materialization, gate, or
provider invocation. This makes malformed or ambiguous repository state an
input error rather than a budget decision. After those inputs are valid, the
configured gate runs under the same process-tree deadline as the review and
remains the only policy refusal besides reviewer availability (CTX-10,
CTX-17).

No gate means proceed. Non-zero means named refusal. Timeout means the gate did
not return a decision, so it remains a timeout with bounded output rather than
being mislabeled as a budget refusal.

### Vendor invocations remain minimal, bounded, and read-only

The existing vendor commands and structural write restrictions remain. Codex
runs in a read-only sandbox; Claude receives no shell or editing tool. Model and
effort resolve per provider invocation, and a model name never crosses provider
families. One deadline bounds the gate, reviewer, optional second gate, refuter,
and every descendant process (CTX-3, CTX-4).

A vendor non-zero exit, empty or oversized report, timeout, or unusable command
produces no receipt and no altered-argument retry. Runnable-provider selection
may fall from cross to fresh before spending, with the reason recorded; a
failure after invocation is surfaced rather than silently retried elsewhere.

### Receipts remain per-clone measurements; diagnosis remains advisory

The receipt records selected base ref, immutable base/head and reviewed-base
commits, canonical range and full-branch digests, provider/mode/reason, author,
duration, effort/model settings, tree state, and verbatim report/refutation
bytes with their digests. Nothing is derived from report meaning. Receipts stay
under the per-clone state root and do not enter the repository (CTX-15).

Diagnosis recomputes the full-branch digest and reports current, stale, absent,
or unreadable. Review state never changes repository-health verdict or exit
status. Old or malformed receipts remain evidence: they are reported unreadable
or are ineligible to narrow a new review, never rewritten as absent (CTX-15,
CTX-18).

### Public command proof uses the real dispatcher without a terminal

Command-level tests will drive initialization, review, and diagnosis through
the `gr` dispatcher against a real Git fixture and stub provider executables.
Standard streams are pipes/buffers, not a terminal. Every successful result is
decoded as JSON; the test asserts no prompt text and verifies
current → stale → current plus an unreadable receipt without changing the
doctor verdict/exit state caused by repository health (CTX-18).

Unit tests remain for marker enumeration, base resolution, exact refute
transport, and receipt calculations. The command test proves their composition
rather than replacing them.

### Replacement canary is bounded before it starts

The reconciliation branch is the replacement real canary. Its hard ceiling is
three `gr review` command rounds and four total provider invocations, because a
`--refute` round adds one provider invocation. The first ordinary round reads
the branch; if findings exist and the caller is about to act, one refute round
may challenge them. After any accepted fixes are committed, the final round is
an explicit `--full` pass on the final head (CTX-15).

Success requires the final full-pass receipt to be current on the unchanged
final head, the caller to find no material issue left in the reports, and zero
owner interruptions on the clean path. If either ceiling is reached while a
material finding remains, the canary records failure and no PR opens. A real
intent divergence or intent hole still stops for the one owner question allowed
by OUT-4. The historical Baseline receipts remain failed evidence and are never
reclassified (CTX-15).

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| Repository and branch | Caller path plus local Git; authoritative for this clone only | Git worktree with attached branch and a head different from the selected base | Non-worktree, detached head, unresolvable head, or empty range | Resolve worktree root and head commit once before writes or paid work | Git fixtures cover subdirectory roots, detached HEAD, and empty/non-empty ranges |
| Base selection | Explicit CLI value or local `refs/remotes/*/HEAD`; remote refs may be stale relative to the server | Explicit resolvable ref, or exactly one resolvable local remote-HEAD target | Empty explicit value, unresolvable ref, zero or multiple omitted-base targets | Resolve selected ref once to a commit; later ref movement cannot change the current range; never refresh refs | Fixtures cover explicit precedence, divergent local/remote refs, absent and multiple remote HEADs, and no network command |
| Author identity | Process environment; routing evidence, not authorization | Non-empty registered primary markers; same-provider markers collapse; no/multi-provider becomes `unknown` | Invalid explicit override only | Snapshot once; strip all registered and companion markers from child environments | Real `CODEX_THREAD_ID` case plus table tests for same-provider duplicates, companion, absent, cross-provider, and override |
| Review flags and settings | Caller CLI/environment; trusted as requested execution settings | Valid `full`, `refute`, provider-scoped model, effort, positive deadline, state root, and optional gate | Flag parse errors, unsupported author, or non-positive explicit deadline | Parse once; per-provider model resolves immediately before each invocation | Command and invocation tests assert caller precedence and provider separation |
| Runnable providers | Local PATH and vendor executables; availability claim only | At least one supported executable resolves | None resolve; post-launch non-zero/timeout/invalid output is an execution failure | Select before spending; no post-spend fallback or altered-argument retry | Stub PATH tests cover fresh, cross, fallback reason, no reviewer, and vendor failure |
| Budget gate | Caller-provided command; trusted local policy | Absent, or command returning zero within the deadline | Non-zero is refusal; timeout/launch failure is execution failure | Capture command once and run after valid Git inputs; run again before refute | Stub gate tests cover absent, pass, refuse, timeout, output bound, and ordering |
| Reviewer instructions | Bounded regular repository file; repository-owned policy | Existing readable in-bound regular file, or absent and materialized once | Symlink, non-regular, unreadable, oversized, or ignored materialized path | Read one byte snapshot per command and hash/use that snapshot; never overwrite an existing file | Existing file-bound/materialization tests plus exact prompt assertions |
| Previous receipt | Per-clone bounded state; evidence from an earlier command | Valid supported receipt whose base/head and digests still resolve | Corrupt/unsupported receipt cannot anchor incremental range; diagnosis reports unreadable | Read one snapshot; never rewrite old receipt; later mutation affects only a later command | Receipt tests cover current/stale/unreadable, legacy fields, and incremental eligibility |
| Provider output | Vendor stdout/stderr; untrusted review prose | Non-empty report within the configured bound | Non-zero, empty, oversized, or timed-out output writes no receipt | Capture bounded bytes once; store verbatim and digest without interpretation | Stub reviewers assert byte identity, failure tails, and no receipt on invalid output |

## Correlation and Evidence

The receipt is the correlation record. The selected base ref explains the
human choice; immutable commits and canonical digests prove the actual range;
provider, mode, reason, settings, duration, and report digests prove how it was
reviewed. The refuter receives the exact canonical range bytes named by the
reviewed digest. A current diagnosis is only a recomputation against the
receipt's full-branch digest.

The failed Baseline receipts and PR metadata remain historical evidence outside
the new canary's pass/fail result (CTX-15). The replacement canary produces new
per-clone receipts for this branch; it does not rewrite or simulate the old run.

## Measurement and Stop Conditions

The repair is ready to archive only when all six confirmed signals are proved:

- a real Codex desktop invocation with `CODEX_THREAD_ID` and no author override
  selects a runnable Claude reviewer in cross mode (SIG-1);
- a failing gate refuses only after valid Git input resolution, while no gate
  proceeds (SIG-2);
- receipt commits/digests recompute and both review reports are byte-identical
  to what the providers emitted (SIG-3);
- public initialization, review, and diagnosis return parseable JSON without a
  terminal, and diagnosis moves current → stale → current/unreadable without
  changing its independent exit semantics (SIG-4);
- the bounded replacement canary finishes on its final head before the PR opens,
  or reports an honest failed ceiling with no PR (SIG-5);
- omitted base selects the divergent remote-tracking default, ambiguity invokes
  nobody and names `--base`, and explicit base wins (SIG-6).

Implementation stops at the first failing deterministic check. The real canary
stops at the first material intent divergence/hole, at either declared ceiling,
or at a successful current final full-pass receipt. It never silently extends a
round, invokes another provider, or opens a PR after failure.

## Risks / Trade-offs

- **Local remote HEAD can itself be stale.** → The receipt proves exactly the
  local commit reviewed; NG-9 intentionally leaves refresh to the caller, and
  ambiguity fails with an explicit-base remedy.
- **Repositories may not maintain `refs/remotes/*/HEAD`.** → Omitted base fails
  safely; explicit `--base` remains the escape instead of guessing `main`.
- **Session markers can drift again.** → Markers are an explicit provider set,
  the real current desktop marker has command-level coverage, and unknown still
  reviews safely rather than refusing.
- **A fresh same-provider review is weaker than cross-provider review.** → Mode
  and reason remain explicit evidence; one-provider fresh is the confirmed
  ordinary path, not a hidden downgrade.
- **Incremental review can miss a regression caused by a later fix.** → The
  limitation remains explicit and the replacement canary ends with a full pass.
- **A review can spend without producing usable evidence.** → Gate, deadline,
  bounded output, no retry, and no receipt on failure bound the loss.
- **Goalrail cannot force the caller to act on findings.** → Deliberate NG-1 and
  NG-2 boundary; the canary assesses caller discipline without adding a gate.

## Migration Plan

The repair is additive to receipt semantics: the selected base ref and commit
already have storage fields, so no receipt migration is required. Existing
receipts stay readable. `CODEX_SESSION_ID` remains accepted while
`CODEX_THREAD_ID` is added. Explicit `--base` retains its meaning; only the
unsafe omitted default changes.

Land deterministic code and command tests first. Then run the declared
replacement canary on `codex/reconcile-pre-pr-review-v0`. Open its PR only after
the final full-pass receipt is current on the final head.

## Rollback

Reverting the implementation restores the previous default and marker set;
existing receipts remain inert per-clone files and require no rewrite. During a
partial rollback, callers can always supply explicit `--base` and `--author`.
The committed instructions file remains ordinary repository content and may be
kept or removed by the repository owner.

## Open Questions

None. Automatic findings disposition, ratchet promotion, coordination, and
network-backed base discovery remain separate candidate changes, not deferred
implementation choices inside this slice.
