## Context

Intent Snapshot `intent-repair-stale-executable-v0` version 1 is confirmed.

Issue #25 records a defect where two correct-looking components combine into a
dead end. `gr health` verifies that the executable a registration names still
exists, and when it does not, refuses to report the attachment working and tells
the user to re-run `gr connect --scaffold <name> --yes`. Connection then decides
presence by looking for the managed marker inside a registered handler command,
which the stale registration still carries, so it reports the attachment as
already present and writes nothing.

Neither component is wrong on its own. Health checks the right thing.
Connection's marker test is what makes disconnection exact and prevents removing
a lookalike handler belonging to another tool. What is missing is that presence
was defined as *recognisability* when the question being asked is *runnability*.

The failure is silent by construction. Background mode is fail-quiet toward the
scaffold, so a hook pointing at a missing binary produces no error the user ever
sees; health is the only surface that reports it, and its remedy does nothing.

The defect was found by a regression test written for the previous change, which
seeded a registration from an "earlier version" carrying an old executable path.
That test's comment records the boundary decision explicitly rather than leaving
the failure to be rediscovered.

## Goals / Non-Goals

**Goals:**

- Make connection able to perform the repair health already prescribes.
- Replace a stale registration rather than accompany it.
- Keep a current registration untouched, and keep foreign handlers untouched.
- Disclose that a repaired hook needs the scaffold's review step again.

**Non-Goals:**

- No change to health's detection, its trust reporting, or its state machine.
- No trust record written, computed, reproduced, or simulated.
- No change to the announcement, retention, intent binding, fail-quiet posture,
  or the wrapper lifecycle.
- No move on project-level registration or any other deferred decision.

## Decisions

### 1. Presence and currency are separate questions, and only one of them is health's

The obvious fix — pass the intended executable into the shared presence check —
would break health. Health inspects a registration it did not create, from a
binary that may legitimately differ from the registered one; asking "does this
name *my* path" there would report a perfectly good attachment as disconnected
whenever health ran from a different copy of `gr`.

So the marker test keeps its existing meaning and its existing caller, and
staleness is a second, separate question that only connection asks. Connection
needs both answers — present, and naming the current executable — and it is the
only component that has the intended path to compare against.

This also keeps a promoted health scenario intact: a half-removed registration
must read as *not connected*, which is a property of the marker test, not of the
path comparison.

### 2. Compare the executable path, not the whole command

Comparing the entire command string against the one connection would write today
is simpler and would catch more drift. It was rejected because it converts any
future change in how the command is spelled — an added argument, a quoting
change — into a silent forced rewrite of every user's registration, and each
rewrite costs the user a trust review.

The confirmed test is path inequality, which is deliberately broader than the
observed defect. The defect was a path that no longer resolves; inequality also
fires when the registered binary is perfectly runnable and merely different — an
installed `gr` alongside a locally built one, which is the ordinary state while
working on Goalrail itself. Those repairs are real writes and cost a real review
step.

That breadth is accepted rather than overlooked, because the narrower rule buys
less than it appears to. Repairing only an unresolvable path would leave a
registration pinned to an older binary that still exists, and would leave the
connection command unable to do the one thing its name promises: make the
attachment use the binary it was invoked from. The cost is bounded — a
registration that already names the current executable is never rewritten, so
repeating the command in a stable installation is free — and it is disclosed at
the moment it is incurred.

One consequence follows and is not mitigated: connecting from a binary that will
not outlive the connection installs a path that will break later. That hazard
predates this change, since it is how the original defect arises on a first
connection, and this change neither widens nor closes it. It is recorded here
rather than left for rediscovery.

### 3. Repair replaces, and for one scaffold that means remove-then-write

The two scaffolds fail differently if repair is naive.

The first scaffold's registration is an appended marker-bracketed block.
Appending a second one would leave two handlers per event — every hook firing
twice — and a removal routine that finds only the first block, so disconnection
would leave residue that looks like an active attachment. The write path
therefore removes any existing block before writing, unconditionally rather than
only on the repair path: it is reached exactly when something must be written, so
removing first is always correct and needs no second decision about which case it
is in. This also closes a latent duplication path that has nothing to do with a
stale executable — a managed block missing one of its stanzas already reads as
not connected, and today that would append a second block beside the first.

Two consequences of reusing disconnection's removal are accepted rather than
hidden. It splices the block out at its original offset while the new block is
appended at the end, so a repair relocates the block past any configuration the
user keeps after it; content is preserved, position is not. And it refuses to act
on a block whose end marker is missing, because the extent of such a block is
unknown and removing a guessed extent could delete user content — so a repair in
that state fails with that reason instead of writing. That turns a silent no-op
into a reported error, which is the better of the two for a command the user
invoked explicitly; the fail-quiet posture governs the hook, not the CLI.

The second scaffold's registration is a JSON handler list. Repair strips our
handlers for the event and appends the current one, per event, reusing the strip
helper the existing session-start repair already uses — so a group containing a
foreign handler survives with that handler intact, and a group that was only ours
is dropped rather than left empty. That helper is not the same code as
disconnection's removal, which inlines an equivalent handler-level filter of its
own; they agree in behaviour and are pinned separately.

Repair introduces no third notion of "our registration": it reuses the marker
test for identity and, on each scaffold, the removal or strip that already
existed.

### 4. Per-event reconciliation, not whole-configuration rewrite

Each event is judged and repaired on its own. A configuration where session
start is current and stop is stale repairs only stop. This falls out of the
existing per-event structure and keeps the byte-identical guarantee meaningful:
the parts that are correct are not rewritten, so they do not lose their trust
records.

### 5. The repair disclosure exists because this is the only moment we know

The scaffold records trust against the hook definition's current form. After a
repair, the stale trust record is still sitting in the configuration, keyed by
event, and it now refers to a command that no longer exists there. Reading it
tells us a record is present, which is exactly why the promoted requirement
forbids calling that "trusted" — confirming freshness would mean reproducing a
private hash, which is prohibited.

But at the moment of repair we do not need to read anything: we changed the
definition ourselves, so we know the record no longer matches. That makes repair
the one point in the system where this can be stated with certainty, and if it
is not stated there it cannot be stated anywhere. Skipping it would trade the
silent stale-path failure for a silent untrusted-hook failure — the same
symptom, one layer down, and the issue that prompted this change named that trap
in advance.

Consequently a repair never reports the attachment as active on the strength of
the existing record.

### 6. The disclosure stays per-scaffold, for the same reason as the original

The existing connection notice is assertive for the scaffold whose trust gate
was observed live and marks the behaviour unverified for the one where it was
not, because asserting a mandatory approval step that may not exist sends the
user hunting for a screen that is not there. The repair disclosure inherits that
discipline exactly. Writing one confident sentence for both scaffolds would
reintroduce, in new text, the overreach the previous change was partly about
removing.

### 7. The test helper's executable becomes explicit, which exposes a gap

The shared test helper created a new temporary file on every call, so the two
idempotency tests connected with one path and then planned with a different one.
They passed only because staleness was invisible; with staleness detected they
fail, correctly, and reveal that same-binary idempotency was never actually
pinned.

The helper therefore takes an explicit executable and the tests hold one path
across both calls. This is not incidental cleanup: the byte-identical guarantee
is one of this change's outcomes, and it had no real test before.

### 8. An adjacent gap in health, left alone deliberately

Health extracts the first managed executable it finds and checks that one. A
hand-edited configuration where session start names a live binary and stop names
a dead one would be reported as working. After this change connection repairs
every event to the same path, so the mixed state can only be produced by hand.

It is not fixed here: the confirmed intent excludes changes to health's
detection, and the refactor this change makes to command parsing is written so
health's behaviour stays byte-for-byte identical rather than quietly improved.
Recorded so the next reader does not have to rediscover it.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-repair-stale-executable-v0`
  version 1 and Context Pack `context-repair-stale-executable-v0` version 1.
- **Specification evidence:** one `MODIFIED` requirement for `ambient-connect`,
  restating every promoted scenario and adding eight. A `MODIFIED` requirement
  replaces the promoted one wholesale at archival, so scenario parity was checked
  by set difference against the promoted text rather than by reading.
- **Implementation evidence:** each new scenario maps to a deterministic test,
  including a repair for each scaffold, a byte-identical repeat for each, a
  foreign handler surviving a repair, and a health report that says working after
  a repair having named the missing binary before it.
- **Issue evidence:** heurema/goalrail#25, which also records the trust trap this
  design answers in decision 5.
- **Prior-change evidence:** the committed test comment that scoped this defect
  out of the matcher change.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation passes for the change and every promoted
  spec.
- A stale registration is repaired by one repeated consented connection on both
  scaffolds, leaving exactly one registration per event.
- A current registration leaves the configuration byte-identical on both
  scaffolds.
- A foreign handler for a repaired event survives with its command and occurrence
  unchanged.
- Health reports the missing binary before the repair and reports working after.
- No test asserting trust is never written is weakened, and none is added that
  writes a trust record from production code.
- Stop and reshape if repair cannot be expressed without either changing health's
  detection or rewriting parts of a configuration that are already correct.

## Risks / Trade-offs

- **[A repair costs the user a trust review]** → The command must change for the
  attachment to name a different binary at all, so the review cannot be avoided
  once a repair is warranted. It is disclosed rather than hidden, and a
  registration that already names the current executable is never rewritten, so
  the review is demanded only when the registration does not name the binary
  connection was invoked from — including when the registered one still runs.
  Decision 2 records why that breadth is accepted.
- **[Path comparison misses other kinds of drift]** → Accepted deliberately in
  decision 2. A stale path is the defect that was observed and reported; broader
  command comparison would force rewrites, and each rewrite has a real cost.
- **[Connecting from a short-lived binary installs a path that will break]** →
  Not mitigated, and not new: this is how the original defect arises on a first
  connection. Recorded in decision 2 so it is a known hazard rather than a
  surprise, and left to its own decision if it is ever worth a guard.
- **[Remove-then-write on the first scaffold widens the write]** → It reuses the
  exact removal disconnection uses, including the refusal on an unterminated
  block, and a byte-identical test pins that a current registration is not
  rewritten at all. Decision 3 records the two visible consequences: the block
  relocates to the end of the file, and an unterminated block makes the repair
  fail loudly rather than silently.
- **[The repair path is exercised only in tests]** → True for the second
  scaffold, whose end-to-end behaviour remains unobserved and is already recorded
  as such. For the first scaffold, the removal routine the repair reuses is the
  one disconnection uses, and the registration it writes is the shape a live
  session exercised; the repair sequence itself has not been run live.
- **[Extraction of the path from a command is textual]** → Parsing is done from
  the decoded command string for the JSON scaffold and from file content for the
  TOML one, matching how each was written. A path containing a quote character
  would defeat it. That weakness predates this change in health's own extraction,
  but this change does give it a second consumer whose verdict causes a write
  rather than a report, so a malformed extraction there would produce a needless
  repair rather than a wrong diagnosis. Bounded by the same test fixtures.

## Rollback

Delete the change directory before implementation; revert the commit after.
Reverting restores the dead-end behaviour, so the requirement text and issue #25
remain the record of why the repair exists.

## Open Questions

None.
