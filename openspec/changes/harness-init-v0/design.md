## Context

`gr init` writes a two-field marker. Everything else a Goalrail repository needs —
the OpenSpec overlay carrying the `goalrail-intent` schema, the session hooks, a
way to see whether any of it is intact — is assembled by hand, and nothing
detects when a repository's copy of the schema has drifted from the canonical
one. `gr health` already reports attachment state honestly, per scaffold, with
next actions and an explicit list of what it could not verify; it is the right
foundation to grow rather than replace.

Three constraints shape this design and none of them are negotiable here. The
first supported scaffold cannot register hooks inside a repository: project-local
hooks are silently skipped there, and the session-opening event has passed before
a user could grant trust. The second can, and its project settings are an ordinary
merging layer — which is why registration moves into `gr init` for it and why the
boundary stops depending on Goalrail's own first act. And the stock OpenSpec CLI
ships only through npm and needs Node 20.19+, while `gr` is a single Go binary
that today never executes anything but `git`; the owner directed that this stay
true.

## Goals / Non-Goals

**Goals:**

- One command installs the whole harness in one repository and reports exactly
  what it did.
- The canonical schema lives in the binary; a repository holds a materialized
  copy whose drift is detectable rather than trusted.
- The session hook is registered per repository, in a file no commit can hand to
  a teammate.
- One diagnosis surface answers "is my harness intact, and if not, what do I run".
- One command brings a repository's overlay up to the installed binary's canon,
  reversibly, and stops rather than destroying a local edit.
- `gr` never needs Node; the user learns what the missing checking CLI costs
  instead of hitting it as a failure.

**Non-Goals:**

- No network release check and no binary self-update; there is no channel.
- No exporter, span emitter, or trace writer; observability is detection only.
- No fork, vendoring, or re-implementation of the stock CLI, and no provider
  prompt packs.
- No modification of user-level scaffold configuration from initialization.
- No trust record written, computed, reproduced, or simulated.

## Decisions

### 1. Registration goes to the per-user project settings file, not a committed one

Project scope is two arrangements, not one. The scaffold's committed project
settings file is shareable by design; its per-user project file is described by the
provider as the user's own and is normally ignored. Registering in the committed
file would install a hook in every teammate's session on the strength of one
user's consent — and the promoted requirement already holds that trust is a
standing consent to run a command in one's own sessions, which is not transferable.

Alternatives considered. *Committed project settings*: strictly better for
onboarding a team, and rejected because the consent it spends is not the
initializing user's to spend; a team-wide arrangement is a separate decision with
its own gate. *Staying at user scope*: keeps one code path, and rejected because
it keeps the boundary as a property of Goalrail's first act rather than of where
the hook lives, which is the whole point of the move. *Per-user project file*:
chosen — repository-scoped, unshareable, and removable by the same user.

### 2. An unignored settings path refuses the registration; a flag fixes it

If the per-user settings file is not ignored, a later `git add -A` publishes the
hook. Initialization therefore refuses to register, installs everything else, and
names the exact ignore entry plus the flag that adds it. Editing a repository's
ignore rules unasked was rejected: it is a change to content the user's team owns,
and doing it silently in the name of safety is the same class of act this design
refuses elsewhere. Registering anyway "because probably nobody commits it" was
rejected outright — that is consent by hope.

The marker is treated more leniently: a missing ignore entry is reported, not
fatal. Refusing to install the harness over a cosmetic exposure would be
disproportionate, and a committed marker misleads without granting anyone
anything.

### 3. `gr` executes no Node, and the canon is validated before it is embedded

Every `gr` path stays Go-only. The overlay is inert YAML and Markdown; producing
and comparing it needs no toolchain. What genuinely needs the stock CLI —
`validate`, `archive`, `status`, `new change` — is the agent's workflow, not
`gr`'s, so `gr doctor` reports the runtime's presence as a fact and names what
its absence costs.

The obvious hole is that a canon could ship which the stock CLI rejects. Closed on
the development side: a test in this repository materializes the embedded canon
into a temporary directory and runs the pinned CLI against it, so a bad canon fails
our build rather than a user's repository. That test is the only place the pin is
exercised, and it is skipped rather than failed where Node is absent, so
contributors without Node can still build — the canon is then verified by whoever
does have it, in CI or locally, before release.

Alternatives: *shelling out to `npx` from `gr`* (rejected by owner direction, and
it would make a first run download a package and require network);
*re-implementing `validate` and `archive` in Go* (rejected as scope — `archive`
performs the ADDED/MODIFIED/REMOVED merge into promoted specs, which is real work
and a decision about replacing the stock CLI, recorded with its revisit condition).

### 4. The canon is the schema and its templates; the config is repository-owned

`openspec/config.yaml` carries a project's own `context:` and `rules:` prose. If
the whole file were canon, every repository's own description would read as drift.
So the canon is the seven files that are genuinely ours to define — the schema and
its six templates — and the config is created when absent with only the `schema:`
key managed thereafter.

Consequence worth stating: a user who edits the schema's *rules* inside their
config is not drifting, and a user who edits `schema.yaml` is. That split is
legible because it follows ownership: what Goalrail defines, Goalrail compares.

### 5. Embedding: a copy inside the package, pinned to the repository's own overlay by a test

Go embedding cannot reach outside its package directory, so the canon lives as a
copy under the implementing package. A duplicated source of truth is exactly the
drift this change exists to detect, so a test asserts the embedded copy is
byte-identical to `openspec/schemas/goalrail-intent/**` in this repository. Editing
one without the other fails the build. The repository's overlay stays the source
the canon is taken from; the embedded copy is a build artifact that happens to be
committed.

### 6. Currency and drift are decided by digests, with a small canon history

Per-file SHA-256 against the embedded canon answers "does this file match". To
distinguish *behind* from *edited*, the binary carries an append-only table of
previous canon digests: a file matching a known older canon's version of that file
is behind; a file matching none is locally edited. A repository-recorded version
was rejected — the marker is per clone and git-ignored, so two clones of one
repository would answer differently about committed content, and the question is
about the content.

At v0 the history is empty — the embedded canon is the only one that has ever
shipped — so *behind* is unreachable in the field until a second canon ships. It is implemented and exercised with a synthetic historical
entry in tests rather than left for later, because the alternative is discovering
the shape of the state on the day it first matters.

### 7. Drift stops an update; only an explicit flag discards

A user who edited a template did it deliberately. A command whose job is
refreshing files must not be the thing that destroys the only copy of that edit,
and "it was in git" is not an answer for content that was never committed. So
update stops, names the drifted files, and changes nothing; the flag discards and
says it discarded. Where the overlay is both behind and edited, the drift wins:
a pending upgrade is not permission to overwrite.

Recoverability does not lean on Git either. Update copies the files it replaces
into a timestamped directory under the state root and names that path in its
report — the state root is already the home for everything that must live outside
the repository, and it is where a user who has committed nothing can still get
their file back.

### 8. `doctor` grows from `health`; `health` survives as an alias

The existing state machine, its honesty rules, and its per-scaffold fan-out are
promoted contract and are kept. `doctor` adds overlay presence, per-file drift,
behind-ness, the runtime fact, and the observability line, and gains a text report
plus `--json` for machines and an exit status.

`gr health` keeps its exact JSON on stdout — the promoted help requirement forbids
changing it — and prints the successor's name on stderr, so a script that parses it
does not break on a deprecation notice. Every printed remedy moves in the same
change: the connection and repair notices name `gr health` today, and a renamed
surface with stale advice is worse than either name alone. A test asserts every
remedy string resolves to a command the CLI accepts.

Exit status, for the diagnosis command specifically: `0` healthy, `1` a harness
problem the user can act on, `2` the check itself did not run — bad usage or an
internal failure. The distinction is for unattended use: a broken cron job must
not read as a healthy repository. Other commands keep the ordinary `1`. An
absent Node runtime and absent observability do not affect it — they are facts,
not faults.

### 9. Per-file drift reporting, not a single verdict

"The overlay differs" leaves the user diffing seven files. Drift is reported per
file with its classification (behind, edited), because the next action differs
between them.

### 10. Scaffold detection uses configuration presence, never session environment

Detection looks for the scaffold's own configuration home. Session environment
variables were rejected as a signal: the live-run evidence in this repository shows
them being deliberately unset to obtain a clean observation, and a detector that
consults them would answer differently depending on who launched `gr`. Where both
scaffolds are present, the repository-scope one is registered and the other's
user-scope command is named; where none is, the overlay still installs. The flag
overrides detection in every case.

### 11. Migration off the superseded user-scope registration passes through the user

A user who connected the second scaffold at user scope earlier now has a
registration that duplicates the repository-scope one — every hook would fire
twice. Initialization must not fix it: user-level configuration belongs to the
consented connection command, and reaching into it from `init` would be exactly
the unconsented write the promoted requirement forbids. So `doctor` detects it,
names it, and prescribes `gr disconnect --scaffold claude-code`, and the user's own
act completes the migration.

### 12. The trust disclosure follows a four-state evidence class

The existing discipline had two states: observed (assert the trust step) and
unobserved (say so). Two more exist now. Documentation records that the second
scaffold has no approval step for hooks configured in a settings file, that its
hook browser is read-only, and that its workspace trust dialog gates permission
rules and additional directories rather than hooks — stronger than nothing,
weaker than an observation, so it is its own state: *documented*, stated as
documented, naming what the adjacent consent surface actually gates. And the
approval probe then upgraded that scaffold past it: three interactive sessions
ran a registered hook with no approval step, which is *observed absent* — the
disclosure states the absence as observed and sends the user nowhere to approve
anything, because pointing at an approval surface the observation ruled out is
the same invention in the opposite direction. No supported scaffold sits in the
documented state today; the state exists because a third scaffold will arrive
with documentation and no observation, and quietly upgrading documentation to
an observation is the drift this discipline forbids.

The evidence class per scaffold lives in code as data, not prose, so a disclosure
cannot drift from what was established.

### 13. The registered events follow cadence, not name

The approval probe retained a question thirty-five seconds into a session,
immediately after the agent's single reply rather than at exit. The provider's
documentation gives the mechanism: on that scaffold `SessionStart` and `SessionEnd`
fire once per session while `UserPromptSubmit`, `Stop`, and `StopFailure` fire once
per turn. The existing registration names `Stop`, which on the first scaffold means
session end and here means end of turn.

Left alone, one session's single question is retained again on every subsequent
turn, and per-occurrence identity — correct when two sessions ask the same
question — multiplies one escalation instead of separating two. So the registration
names the session-ending event on that scaffold, a registration naming the per-turn
event is repaired in place, and removal walks whichever events are present rather
than only the ones the current arrangement writes.

Alternatives considered. *Deleting the reserved file after retention*: would make
`Stop` survivable, and rejected because the promoted requirement keeps the worktree
file untouched and pins that retained bytes survive its later deletion; changing
that would rewrite retention rather than fix an event mapping. *Deduplicating by
digest within a session*: rejected for the same reason — per-occurrence identity is
promoted contract, and suppressing occurrences to compensate for the wrong trigger
hides the defect instead of removing it. *Leaving it for a separate change*: the
owner chose to fold it in, since this change already rewrites that scaffold's
registration and splitting it would do the same work twice.

The event set therefore becomes per-scaffold data alongside the scope and the
disclosure evidence class, so all three answers about a scaffold live in one place.

### 14. Worktrees and the state root are left alone deliberately

The marker and the per-user project settings file live in a working tree, so each
worktree is initialized independently and `git check-ignore` answers per worktree.
That is correct rather than broken, and the diagnosis reports the directory it was
run in. Nothing here shares state between worktrees, which keeps two agents in two
worktrees from inheriting each other's attachment.

### 15. Disconnection does not delete the overlay

Removal covers what was registered. The overlay is version-controlled repository
content; deleting files a user may have committed, edited, or built on is not
disconnection's business. `doctor` reports the state and `update` repairs it.

## Correlation and Evidence

- The canon is identified by a digest over its per-file digests; that identifier
  appears in `doctor`'s structured output and in `update`'s report, so a support
  question can be answered from one string.
- `update` records the replaced files under the state root with the timestamp and
  the canon identifiers it moved between; that directory is the recovery evidence
  and is named in the report rather than left to be discovered.
- The interactive observation of the registered arrangement is recorded under this
  change's `evidence/`, in the same shape as the existing live-run record: what was
  observed, what was not, and the conditions that bound the conclusion.
- No credential, key, prompt, or session content enters any report. The
  observability line names an endpoint at most.

## Measurement and Stop Conditions

- The slice is done when a fresh directory becomes a repository the pinned CLI
  validates, in one command, on a machine without Node, and `doctor` distinguishes
  every state it claims to.
- Stop and re-ask if the interactive observation shows the second scaffold does
  gate registered hooks behind approval: the registration arrangement survives, but
  the disclosure and one `doctor` state change, and the confirmed signal that
  describes them would need a new intent version.
- Stop if making the registration unshareable turns out to be impossible in a
  common repository shape rather than an edge case; that would reopen decision 1
  rather than being worked around.
- Stop if the development-side canon test cannot be made to pass on a stock pinned
  CLI: shipping a canon the CLI rejects is worse than shipping no canon, and it
  would mean the overlay's shape, not the tooling, is wrong.

## Risks / Trade-offs

- **A user edits a template and is then blocked by every update** → drift is
  reported per file with the discard flag named, so the user always has a move;
  the flag states what it discarded.
- **The embedded canon and the repository's overlay diverge** → a test pins them
  byte-identical, so divergence fails the build rather than shipping.
- **A canon ships that the stock CLI rejects** → the development-side test
  materializes and validates it; the residual risk is a release built where Node
  was absent and the test skipped, which is why the skip is loud and the release
  path is the place to require it.
- **`behind` is unreachable at v0** → implemented against a synthetic history in
  tests; the first real second canon exercises it in the field.
- **Detection picks the wrong scaffold on a machine with both** → the override
  flag is in the same command, and the report names what it chose and why.
- **The superseded user-scope registration doubles every hook invocation until the
  user acts** → `doctor` names it explicitly with the exact command; initialization
  refuses to touch it, which is the promoted rule this accepts a cost for.
- **`gr health` scripts break on a deprecation notice** → the notice goes to
  stderr and the JSON on stdout is unchanged.
- **The report grows long enough to be ignored** → optional absences are one line
  with no next action, and only actionable states carry one.

## Rollback

Every artifact in this change is repository content on a branch; reverting the
branch removes the design without touching promoted specs. In the implementation
that follows, the surfaces are additive: `doctor` and `update` are new commands,
`health` keeps its behaviour, and the repository-scope registration is removable by
the same disconnection command that removes the user-scope one. A user who wants
the previous arrangement runs disconnection, deletes the overlay files with Git,
and is back to a marker-only repository. The embedded canon is inert unless a
command materializes it.

## Open Questions

- AMB-1 is closed by observation: the second scaffold runs a repository-scoped
  registered hook with no approval step in an interactive session, and the
  announcement reaches the agent there. What its startup screen displayed was not
  captured, so no claim is made about it. Recorded in
  `evidence/approval-probe-2026-07-30.md`.
- Whether a harnessed repository needs a committed instruction file naming the
  pinned invocation, or whether `init`'s report and `doctor` printing it suffice
  (AMB-2). This slice answers "printing suffices". Revisit condition: an agent in a
  freshly initialized repository still guesses the invocation — the cheap follow-up
  is `gr` creating the change directory itself, which also removes the
  explicit-schema trap.
