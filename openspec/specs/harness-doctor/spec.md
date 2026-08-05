# harness-doctor Specification

## Purpose

Define the single surface that answers "is this repository's harness intact, and
if not, what do I run": attachment state per scaffold, overlay drift reported per
file, currency decided by comparing digests rather than by trusting a recorded
version, the checking toolchain reported as a fact rather than a fault, and
observability reported once as optional without nagging or leaking. Every state
that is not working names an action the tool can actually perform, and the report
is readable by a machine as well as a person.
## Requirements
### Requirement: One diagnosis covers the whole harness
**Intent IDs:** OUT-2, OUT-4, OUT-10, SIG-2, SIG-3, SIG-4, SIG-12

Goalrail SHALL expose one diagnosis surface with a versioned machine contract that reports independent layers rather than one checkout-local marker verdict. It SHALL report at least: declaration presence and validity; project identity and governance contract version; explicit v0.1.8 migration need; overlay and canon-owned bootstrap presence, drift, or behind-ness per file; repository-owned policy identity and mismatch; planning toolchain readiness; supported scaffold attachment and trust state; lineage policy readiness; prepared shared-check integration; verified shared-admission activation or the absence of such verification; optional observability; review evidence; and release-channel facts.

The report SHALL distinguish `managed`, `locally_ready`, `lineage_ready`, and `shared_admission_active`. Aggregate full enforcement SHALL be healthy only when a valid declaration, required local and planning prerequisites, current governing files, valid lineage policy, and independently verified protected shared admission are all present. A managed project with missing local setup SHALL remain `managed=true` and fail closed with `GOALRAIL_SETUP_REQUIRED`; it MUST NOT be described as uninitialized or available for ordinary unmanaged work.

Every non-working required state SHALL name one exact next action the tool or documented workflow can actually perform. Optional absence SHALL remain informational. Drift SHALL be named per path and ownership class. Structured output and exit status SHALL distinguish unmanaged, declared-invalid, setup-required, locally ready but advisory, and fully enforced states. A bounded compatibility field named `initialized` MAY mirror valid declaration presence for one migration window, but it MUST be labeled deprecated and MUST NOT retain marker semantics.

#### Scenario: Repository was never managed
- **WHEN** diagnosis runs at a repository root with no declaration path and no legacy Goalrail project evidence
- **THEN** it reports unmanaged, performs no managed-project observation or write, and does not prescribe setup as though the project had opted in

#### Scenario: Managed clone lacks local setup
- **WHEN** a valid declaration is present but required binary, planning runtime, or scaffold attachment is absent
- **THEN** diagnosis reports `managed=true`, the exact local readiness failures, `GOALRAIL_SETUP_REQUIRED`, and no fully enforced verdict

#### Scenario: Declaration is invalid
- **WHEN** the reserved declaration path exists but its schema, identity, policy reference, location, or bytes are invalid
- **THEN** diagnosis reports declared-invalid and fails closed rather than reporting unmanaged

#### Scenario: Canon-owned file has drifted
- **WHEN** an overlay or managed bootstrap path differs from every canon known to the installed binary
- **THEN** diagnosis names that exact path and ownership class and reports drift rather than a general mismatch

#### Scenario: Repository-owned policy mismatches
- **WHEN** current policy bytes do not match the identity referenced by the declaration
- **THEN** diagnosis reports governance mismatch and does not prescribe an automatic overwrite

#### Scenario: Local checks exist without protected admission
- **WHEN** local hooks and a shared workflow file exist but required-check activation has not been independently verified
- **THEN** diagnosis reports local readiness or advisory integration and `shared_admission_active=false` or unknown, never full enforcement

#### Scenario: Every required layer is verified
- **WHEN** declaration, governing content, planning prerequisites, local attachment, lineage policy, and protected shared admission all verify
- **THEN** diagnosis reports each layer healthy and full enforcement working

#### Scenario: Diagnosis is consumed by a machine
- **WHEN** diagnosis is invoked for automated use
- **THEN** it emits the current schema version, categorical layer states, stable reason identifiers, and an exit status consistent with the aggregate state

#### Scenario: Next action names a nonexistent operation
- **WHEN** any required-state remedy names a command or documented user action that cannot perform the repair
- **THEN** that violates this requirement

### Requirement: Currency is judged by digests, not by a recorded version
**Intent IDs:** OUT-6, OUT-9, SIG-4

Whether a repository's overlay is current SHALL be decided by comparing its files
against the binary's canonical copies, and MUST NOT depend on a version recorded
inside the repository. The participation marker is per clone and is not shared
through the repository, so a recorded version would answer the same question
differently in two clones of one repository — and the question is about
committed content.

The `gr` version SHALL be reported as information about the binary. No verdict
about the repository may be derived from it.

#### Scenario: The version string alone changes
- **WHEN** the reported `gr` version differs while every overlay file still matches the binary's canon
- **THEN** the diagnosis reports the overlay as current

#### Scenario: The files differ while the version agrees
- **WHEN** an overlay file differs from the binary's canon while any recorded version would suggest currency
- **THEN** the diagnosis reports drift, because the files decide

### Requirement: The checking toolchain is reported as a fact, not a fault
**Intent IDs:** OUT-2, OUT-4, OUT-5, SIG-2, SIG-3, SIG-5

Diagnosis SHALL state whether every planning runtime and pinned compiler required by the managed project's declared setup profile is available and compatible. Their absence MUST NOT erase managed-project identity or make Goalrail's native diagnosis unavailable, but it SHALL make local planning readiness false, produce `GOALRAIL_SETUP_REQUIRED`, and block supported agents from material code work.

The report SHALL distinguish a missing executable, incompatible version, unavailable package, and unverified integrity identity. Its next action SHALL enter the exact read-only setup planner rather than prescribe an ad hoc global install. A project whose declared compiler profile requires no external runtime SHALL not be failed for one.

#### Scenario: Required planning runtime is absent
- **WHEN** diagnosis runs in a managed project whose setup profile requires a runtime that is not available
- **THEN** project identity remains managed, planning readiness is false, material work is blocked, and the exact setup-planning action is named

#### Scenario: Required compiler version differs
- **WHEN** an installed compiler does not satisfy the exact project setup profile
- **THEN** diagnosis reports the observed and required identities and routes through setup planning without changing either installation

#### Scenario: Planning toolchain is complete
- **WHEN** every runtime and compiler named by the setup profile verifies
- **THEN** planning readiness is true without implying that shared admission is active

#### Scenario: No external runtime is required
- **WHEN** the declared planning profile is entirely provided by verified installed Goalrail capabilities
- **THEN** diagnosis reports planning readiness without inventing a Node or other runtime requirement

### Requirement: Observability absence is reported once, without nagging or leaking
**Intent IDs:** OUT-10, SIG-10

The diagnosis SHALL report observability in one line. Absence SHALL be stated as
optional, with no next action and no effect on the overall verdict; Goalrail
works fully with no observability backend, and the local state root remains the
source of truth.

No output SHALL contain key material. Where an endpoint is configured, the report
SHALL name at most the endpoint.

#### Scenario: No observability is configured
- **WHEN** diagnosis runs with no observability endpoint configured
- **THEN** it reports the harness as working, states observability as not configured and optional, and prints no next action for it

#### Scenario: Observability is configured
- **WHEN** an endpoint and keys are present in the environment or the state root
- **THEN** the report names the endpoint at most and no key material appears in any output

### Requirement: The diagnosis reports a newer release as a fact, never as a problem
**Intent IDs:** OUT-1, OUT-2, SIG-1, SIG-2

The diagnosis SHALL report whether a newer Goalrail release exists. It SHALL do
so in the voice it uses for the toolchain and for observability: a stated fact,
with no next action, contributing nothing to whether the harness is reported as
working and nothing to the exit code.

The reason is not stylistic. The tool has no command that replaces its own
binary, so a next action here would name a command the tool does not accept,
which a promoted requirement already forbids. A repository whose harness is
intact is working whether or not its binary is a release behind, and a caller
that reads the exit code learns about the repository, not about the binary.

Where the line rests on an answer from a network source, it SHALL name that
source, so the report discloses its counterparty where the user is rather than
only in documentation. Where there is no answer and no counterparty, it names
none.

The check MUST NOT compare anything when the running binary's own version is not
a release version — a build from a checkout, from a modified tree, or one
carrying no version at all. In those cases the diagnosis SHALL say that the
question does not apply rather than guess. No wording SHALL assert that the
installation is up to date: the source can lag a published release, so the
report states what was found and when it was looked for.

Where the check produced no answer — because it was declined, or because it
failed — the diagnosis SHALL say so in one line, and a declined check SHALL be
distinguishable from a failed one. It MUST NOT surface an error, a stack trace,
or any text the source produced.

An answer read from a cache that is not yet due for refresh is an answer, not an
absence: it is reported like any other, stating when it was obtained.

#### Scenario: A newer release exists
- **WHEN** the running binary is a release version older than the newest released one
- **THEN** the diagnosis states that a newer release exists, names the service it asked, prescribes no command, and leaves the working verdict and the exit code exactly as they would have been

#### Scenario: The binary is current
- **WHEN** the running binary's version matches the newest released one
- **THEN** the diagnosis states what it found and when, names the service it asked, and does not assert that the installation is up to date

#### Scenario: The running binary is not a release build
- **WHEN** the version the binary reports is a pseudo-version, carries a modification marker, is the toolchain's development marker, or is unknown
- **THEN** no comparison is made and the diagnosis says the question does not apply

#### Scenario: The check produced no answer
- **WHEN** the check was declined or failed
- **THEN** the diagnosis says so in one line, distinguishing a declined check from a failed one, with no error, no stack trace, and no text from the source

#### Scenario: The answer comes from a cache that is not yet due
- **WHEN** a diagnosis runs within the caching interval and a cached answer is present
- **THEN** no request is made, the answer is reported compared against the running binary's version, and the report states when it was obtained rather than that the check did not run

#### Scenario: A newer release would change the verdict
- **WHEN** the existence of a newer release would make the harness report as not working, add a problem, or change the exit code
- **THEN** that violates this requirement

#### Scenario: A next action would be prescribed
- **WHEN** the line about a newer release prescribes a command
- **THEN** that violates this requirement, because the tool has no command that replaces its own binary

### Requirement: The diagnosis reports whether the branch's review still describes it
**Intent IDs:** OUT-6, SIG-4

Where a receipt exists for the current branch, the diagnosis SHALL report in one
line whether that review still describes the branch: current where the branch's
present diff digest matches the receipt's, stale where it does not. Where no
receipt exists for the branch, it SHALL report that too. Where a receipt exists
but cannot be read or its digest cannot be recomputed — truncated, malformed,
oversized, an unknown schema, a pruned base commit — the diagnosis SHALL report
it as unreadable and name the cause, because presenting "never reviewed" for a
branch whose evidence is corrupt is the one wrong answer available here. A review that was run
and then outrun by three more commits is worse than no review, because it reads
as done.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. A branch with
no review is an ordinary state, and the review is evidence rather than a gate.

The state SHALL follow from recomputing the digest alone. Committing further
work makes a review stale by itself, and running a fresh review makes it current
by itself; there SHALL be no flag, no acknowledgement command, and no stored
dismissal. A condition that must be cleared by hand is one nobody reads.

The line MUST NOT reproduce the reviewer's report. The report is stored
verbatim in the receipt for whoever wants it; reprinting it on every diagnosis
would be noise.

#### Scenario: The review still describes the branch
- **WHEN** diagnosis runs on a branch whose receipt digest matches its present diff
- **THEN** it reports the review as current, naming the reviewer and the time, with no effect on the verdict or the exit status

#### Scenario: New work has outrun the review
- **WHEN** a commit is added after a review and diagnosis runs
- **THEN** it reports the review as stale, and no flag was needed to change that

#### Scenario: A fresh review restores currency
- **WHEN** the review is run again after that commit
- **THEN** the diagnosis reports it as current again, with no dismissal or acknowledgement anywhere

#### Scenario: The branch has never been reviewed
- **WHEN** diagnosis runs on a branch with no receipt
- **THEN** it reports the absence as a condition, and the overall verdict and exit status are what they would be without the line

#### Scenario: The report is not reprinted
- **WHEN** the line is reported for a branch whose receipt holds a long report
- **THEN** no part of the report text appears in the diagnosis output

#### Scenario: The receipt is unusable
- **WHEN** diagnosis runs where a receipt exists but cannot be read or recomputed
- **THEN** it reports the review as unreadable with the cause, and the overall verdict and exit status are unchanged

### Requirement: Review evidence preserves the public non-interactive command flow
**Intent IDs:** OUT-6, SIG-4

Initialization, review, and diagnosis SHALL remain usable through their public
command surfaces with no terminal attached. Each successful command SHALL emit
one parseable JSON result and MUST NOT prompt. Adding or changing review
evidence MUST NOT alter initialization semantics or the diagnosis verdict and
exit status that would otherwise follow from repository health.

#### Scenario: The public flow runs without a terminal
- **WHEN** initialization, review, and diagnosis run in sequence with valid inputs and no terminal attached
- **THEN** every command completes without prompting and each successful result parses as JSON

#### Scenario: Review state does not become a diagnosis gate
- **WHEN** diagnosis observes a current, stale, absent, or unreadable review receipt without any other repository-health change
- **THEN** its JSON names that review state while its verdict and exit status remain what repository health independently requires

### Requirement: Shared-admission health is evidence-backed and never inferred
**Intent IDs:** OUT-7, OUT-10, SIG-7, SIG-12

Diagnosis SHALL report shared-admission activation separately for every configured integration boundary. An active verdict SHALL require current read-only evidence from a registered provider adapter or another independently verifiable activation receipt that the Goalrail check is required for the protected target. A workflow file, successful check run, local hook, commit trailer, or owner intention alone MUST NOT prove activation.

The report SHALL name the provider or boundary, required check identity, observed target, evidence source, observation time, and freshness status without exposing credentials. Missing access SHALL produce unknown, not inactive or active. Diagnosis MUST NOT enable, repair, or authenticate an external integration.

#### Scenario: Required check evidence is current
- **WHEN** the registered adapter verifies that the protected target currently requires the exact Goalrail admission check
- **THEN** diagnosis reports shared admission active with bounded provenance and observation time

#### Scenario: Provider cannot be queried
- **WHEN** current activation evidence requires unavailable access
- **THEN** diagnosis reports activation unknown and does not reuse an expired observation as current proof

#### Scenario: Only a workflow file exists
- **WHEN** the repository contains the shared-check workflow but no verified required-check setting
- **THEN** diagnosis reports the integration prepared or advisory and full enforcement remains false

### Requirement: The diagnosis reports rules that predate an adoption, and stops on its own
**Intent IDs:** OUT-5, SIG-4, SIG-5

Where the marker records an adoption of a configuration that carried at least one
rule, and that rules block still matches the digest recorded at the adoption, the
diagnosis SHALL report it in one line naming the replaced schema and when the
adoption happened. Initialization discloses the same fact once, to whoever ran
it; that is routinely an agent, and the person whose rules they are may never see
it. The diagnosis is where a standing condition belongs.

Where the configuration carried no rules at the adoption, the diagnosis SHALL
report nothing. There is nothing that predates the schema, and the advisory's
stop condition is an edit to rules that do not exist — a standing condition the
user could never clear by reviewing anything.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. Rules written
against a replaced schema are frequently still correct, and Goalrail cannot tell
which ones are not.

The report SHALL stop carrying the line once the rules block no longer matches
the recorded digest, with no flag, no acknowledgement command and no stored
dismissal. Editing the rules is the act of reviewing them, so the only honest
stop condition is the one the user's own edit produces. A condition that must be
dismissed by hand becomes a condition nobody reads.

Where the marker carries no adoption record, the diagnosis SHALL report nothing
about rules at all.

The line MUST NOT reproduce the rules themselves. Initialization discloses them
in full at the moment they can still be compared against what was replaced; a
diagnosis that reprinted them at every run would be noise.

#### Scenario: Rules are unchanged since an adoption
- **WHEN** diagnosis runs where the marker records an adoption and the rules block still matches the recorded digest
- **THEN** it reports one line naming the replaced schema and the adoption time, the overall verdict is unaffected, and the exit status is what it would have been without the line

#### Scenario: The rules have been edited since the adoption
- **WHEN** the rules block is edited after an adoption and diagnosis runs again
- **THEN** the line is absent, and no flag, command or stored acknowledgement was needed to remove it

#### Scenario: An unrelated part of the configuration is edited
- **WHEN** the configuration is edited outside the rules block after an adoption and diagnosis runs
- **THEN** the line is still reported, because the rules themselves are unchanged

#### Scenario: The repository never replaced a schema
- **WHEN** diagnosis runs where the marker carries no adoption record
- **THEN** it reports nothing about rules

#### Scenario: The adopted configuration carried no rules
- **WHEN** diagnosis runs where the marker records an adoption of a configuration that carried no rules
- **THEN** it reports nothing about rules, however many times it is run

#### Scenario: The line does not reprint the rules
- **WHEN** the line is reported for a configuration carrying several rules
- **THEN** no rule text appears in the diagnosis output
