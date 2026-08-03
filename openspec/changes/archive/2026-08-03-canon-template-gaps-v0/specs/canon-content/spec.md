## ADDED Requirements

### Requirement: The design template asks for the consumed inputs
**Intent IDs:** OUT-2, SIG-2, SIG-3

The design template SHALL carry a Consumed Inputs section covering every
behavior-affecting external input the slice introduces or changes: the input,
its source and trust, the states it accepts, the states it refuses, its policy
under mutation, and how each claim is verified. The largest recorded finding
class is precisely the input nobody enumerated, and its reviewer-prose
promotion did not stop it; the template moves the question to the author,
before the work exists, and leaves the reviewer a concrete claim to contradict
rather than a judgement to make.

The section's legitimate empty form SHALL be exact and falsifiable — naming the
scope checked and the bounded search or trace that checked it — and a bare
"N/A" or "None" SHALL count as an unfilled section. An empty form that costs
nothing to write reflexively would rot the section into noise.

This is an author-side forcing function, and it MUST NOT be described as
schema-enforced anywhere: the stock validator was measured accepting a gutted
design outright.

#### Scenario: A change that reads external input
- **WHEN** a design is written for a slice that reads a file, format, environment, or another program's output
- **THEN** the Consumed Inputs table names each such input with its accepted and refused states and their verification

#### Scenario: A change that reads nothing
- **WHEN** a slice introduces or changes no external input handling
- **THEN** the section states the scope checked and the bounded search that checked it, and a bare N/A reads as unfilled

### Requirement: Material decisions cite the context they rest on
**Intent IDs:** OUT-5

The design instruction SHALL require every material decision to cite the
Context Pack item IDs it relies on, and SHALL require a new Context Pack
version before settling any decision that rests on a fact absent from the
current pack or stale since its observation time. Refreshing context SHALL stop
design, return intent to candidate as a new version linked to its predecessor
and the refreshed pack, and require exact owner confirmation before design
resumes. The pack is verified before intent and can be outdated by the time
design settles; citation is the minimal binding that exposes that gap without
inventing a per-decision artifact.

#### Scenario: A decision on fresh context
- **WHEN** a material decision cites context items present and current in the pack
- **THEN** it settles with those citations recorded

#### Scenario: A decision on a missing or stale fact
- **WHEN** a material decision depends on a fact the current pack lacks or observed too long ago
- **THEN** design stops, the pack gains a version carrying that fact, intent returns as a linked candidate version, and exact owner confirmation precedes both resumed design and the settled decision

### Requirement: The intent template does not assert confirmation outside the status
**Intent IDs:** OUT-3

The intent template's semantic-group headings SHALL be neutral — an outcome is
an outcome and a boundary is a boundary, whatever the snapshot's status — and
the schema SHALL describe the artifact as candidate-or-confirmed. Confirmation
SHALL be established only by the Status field and the Confirmation block. A
heading that says "Confirmed" over candidate content presents interpretation as
agreement to every downstream reader.

#### Scenario: A candidate snapshot
- **WHEN** an intent snapshot is in candidate status
- **THEN** nothing in its headings or the schema's description claims its content is confirmed

### Requirement: Context items carry a verification recipe
**Intent IDs:** OUT-4

The context template SHALL carry a Verification recipe column: bounded,
read-only, stating its prerequisites and the observation expected. For
historical or non-repeatable evidence the recipe SHALL be the honest form —
not independently reproducible, with the reason and a stable reference to the
retained evidence — and a refresh command MUST NOT be described as reproducing
an earlier observation, because it measures the present instead. An invented
fact reached a confirmed intent through a context item nobody could check;
the column exists so every item arrives with the means to challenge it.
The Goalrail reader SHALL retain the recipe as domain data and apply the same
bounded, single-line, control-character, and secret-shaped-content protections
used for other retained context text. It SHALL continue to accept the legacy
six-column form without inventing a recipe for already-written artifacts.

#### Scenario: A reproducible claim
- **WHEN** a context item states something the repository or a tool can show now
- **THEN** its recipe names the read-only steps, the prerequisites, and the expected observation

#### Scenario: A historical claim
- **WHEN** a context item records an observation that cannot be independently repeated
- **THEN** its recipe says so, names the reason, and points at the retained evidence

#### Scenario: A recipe is consumed
- **WHEN** Goalrail reads the seven-column Context Items form
- **THEN** it retains the exact recipe and rejects unbounded or secret-shaped recipe text

#### Scenario: A legacy item has no recipe column
- **WHEN** Goalrail reads the supported six-column Context Items form
- **THEN** it accepts the item with no invented recipe

### Requirement: The canon's copies move together, and its test tells the truth
**Intent IDs:** OUT-6

Every canon content change SHALL land simultaneously in the embedded canon
templates, the canon schema's instructions, and the mirrored repository schema
copies, keeping the existing byte-identity test green — a template that asks
for a section the instruction never mentions produces artifacts that ignore
it. The canon CLI test MUST NOT claim a validation it does not run: it was
found promising strict validation while never invoking the validator.

#### Scenario: A content edit
- **WHEN** a canon template or instruction changes
- **THEN** all mirrored copies change in the same commit and the byte-identity test passes

#### Scenario: The test's promise
- **WHEN** the canon CLI test describes what it verifies
- **THEN** the description matches what it executes
