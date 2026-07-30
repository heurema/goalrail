## Context

Intent Snapshot `intent-claude-code-matcher-v0` version 1 is confirmed.

The owner asked to verify the second scaffold live, and to read its
documentation first because project-level hooks appeared to work there. Reading
came before running, and produced two findings without a single session.

**A defect in shipped code.** That scaffold's session-start registration names
which occurrence a hook applies to, one exact value per entry. The occurrences
that open a session and the ones that recur inside it are distinguished at
registration, not in the hook. Omitting the matcher makes the hook fire on every
occurrence. The connection routine writes no matcher, verified by connecting on
a stand and reading the produced file, so the registration it ships is exactly
the fire-on-everything shape. The promoted contract requires the announcement to
accompany only the opening event. The first scaffold satisfies that because its
transport inspects the occurrence and stays silent otherwise; the second has no
such guard, so the rule silently does not hold there.

Two independent sources agree on the shape: the provider's documentation, and
the owner's own working configuration, which registers four separate
session-start entries — one per occurrence — for a single unrelated hook.

**An overreach in a promoted requirement.** That requirement records that
registering inside the repository is externally blocked. The evidence was real
but came from one provider, where project-local hooks are silently skipped
behind a per-hook trust gate. For this scaffold, project-level hooks are an
ordinary settings layer that merges with user settings. The requirement stated a
property of one provider as a property of attachment.

**What could not be established.** Live verification of this scaffold was
attempted and abandoned. An isolated home does not satisfy its login state; the
remaining routes were writing into the owner's working configuration — the tool
this session runs inside — or copying an account file carrying unrelated project
history. Neither is worth a confirmation of something two agreeing sources
already describe.

## Goals / Non-Goals

**Goals:**

- Make the single-delivery rule hold on both scaffolds, not one.
- Keep removal exact under the new registration shape.
- Attribute the recorded limitation to the scaffold it came from.
- Record what remains unobserved.

**Non-Goals:**

- No move to project-level registration in this change.
- No writing into the owner's working configuration and no copying of account
  files to obtain a run.
- No change to the verified chain, health reporting, or the wrapper lifecycle.
- No claim that the second scaffold is verified end to end.

## Decisions

### 1. Register per occurrence rather than relying on the transport

The first scaffold's transport filters the occurrence, so a broad registration
there is harmless. The second scaffold's transport receives no occurrence to
filter, so the only place the rule can hold is the registration itself. Making
registration carry the constraint on both scaffolds keeps a single rule with a
single meaning, instead of one rule enforced in two unrelated ways — and it
removes the possibility that adding a third scaffold quietly reintroduces the
defect.

### 2. Removal tracks registration by marker, not by position

Only one occurrence opens a session, so the registration narrowed rather than
widened: one scoped entry replaces a form that fired on everything. Removal
still has to find whatever registration exists, including the unscoped shape an
earlier version installed, and the existing per-handler filter matches on the
managed marker rather than on position, so it does. This is stated as a rule
because the failure it prevents is silent: a partial removal leaves entries the
user did not ask for and cannot see.

### 3. The limitation is attributed, not generalised

Writing "project-local registration is blocked" without naming the provider was
an accurate observation turned into a false general claim. The correction is not
cosmetic: the requirement is what a future reader consults before deciding
whether the structural route is available, and for one of the two scaffolds it
is.

### 4. The unobserved is named rather than implied

This change fixes the second scaffold's registration without ever having run it.
Two agreeing sources justify the fix; they do not justify calling it verified.
Recording the difference in the requirement means the next person does not have
to reconstruct which parts rest on observation and which on documentation.

### 5. Project-level registration is deliberately not adopted here

The reading that corrected the record also opened a route: for this scaffold the
project layer merges ordinarily, so registering there would make the privacy
boundary structural rather than a property of our first-act check. That is a
material change to where attachment lives, and it deserves its own evidence and
its own decision rather than riding along with a matcher fix.

### 6. An adjacent defect found and deliberately not fixed here

A regression test written for this change seeded a registration from an
"earlier version" carrying an old executable path. The assertion that no stale
handler survived the repair failed, correctly: the matcher repair replaces the
session-start entry, but nothing replaces a handler whose only defect is a stale
path.

That makes `gr health`'s remedy a dead end — it detects the missing binary and
tells the user to re-run connection, while connection reports the attachment as
already present and changes nothing. Recorded as issue #25 rather than widened
into this change, whose confirmed intent covers the matcher shape and a
mis-attributed limitation, not executable-path reconciliation. The test here was
narrowed to what this change actually promises rather than left failing or
quietly relaxed.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-claude-code-matcher-v0` version 1
  and Context Pack `context-claude-code-matcher-v0` version 1.
- **Specification evidence:** one `MODIFIED` requirement for `ambient-connect`.
- **Implementation evidence:** each scenario maps to a deterministic test,
  including one asserting no registration fires on every occurrence and one
  asserting byte-exact removal with a foreign entry present.
- **Documentation evidence:** the provider's published matcher contract.
- **Configuration evidence:** an existing working configuration with one entry
  per occurrence, agreeing with the documentation.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation passes for the change and every promoted
  spec.
- The produced registration carries one entry per opening occurrence and none
  that fires unconditionally.
- Connect-then-disconnect restores the configuration byte for byte, and a
  foreign entry for the same event survives both operations.
- The first scaffold's registration and every live-verified behaviour keep their
  existing tests unmodified.
- Stop and reshape if the matcher shape cannot be expressed without changing
  where hooks are installed.

## Risks / Trade-offs

- **[The fix is unobserved]** → Two independent sources agree on the shape, and
  the requirement records that no live session has run. The alternative was
  writing into the owner's working configuration or copying account data, which
  costs more than the confirmation is worth.
- **[The list of opening occurrences may be incomplete]** → Taken from the
  provider's documented values. A value added later would not be registered,
  which fails closed: a missed occurrence means no announcement, never a
  repeated one.
- **[An earlier version's unscoped registration must still be removable]** →
  Removal matches on the managed marker rather than position or shape, so it
  finds both forms; a test pins byte-exact restoration and the repair of an
  unscoped entry.
- **[Project-level registration stays available but unused]** → Recorded as an
  open route with its own revisit condition, so it is a deferred decision rather
  than a forgotten one.

## Rollback

Delete the change directory before implementation; revert the commit after.
Reverting restores the previous registration shape, which is the defect, so the
requirement text is the part worth keeping if the code is ever reverted for
unrelated reasons.

## Open Questions

None.
