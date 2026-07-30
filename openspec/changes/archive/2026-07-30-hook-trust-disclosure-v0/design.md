## Context

Intent Snapshot `intent-hook-trust-disclosure-v0` version 1 is confirmed.

This change exists because of what one live session established and one live
session failed to establish.

**Established.** With hooks actually running, the whole chain works: the
announcement reached the agent, which quoted the reserved path verbatim; given a
contradictory task the same model wrote a question instead of guessing and left
the source untouched; the stop hook retained the question with its own
identifier, digest, session reference, and the explicit reason it could not be
bound. The first run of the same task without the channel had produced a silent
guess — the exact behaviour the capability exists to replace.

**Not established, then explained.** Nothing ran at all until the hooks were
trusted. Three non-interactive runs executed nothing and printed nothing; the
first interactive session recorded trust and still ran nothing; only the session
after that executed. The scaffold's documentation states the rule: a hook runs
only after its exact definition is reviewed and trusted, recorded against the
definition's current form, managed through the scaffold's own surface.

**The stronger arrangement, attempted and blocked.** Registering inside the
repository would make "we do nothing in unconnected directories" a fact of
layout rather than a property of our code. On the stand the project carried
`trust_level = "trusted"` and a `.codex/hooks.json` the scaffold parsed
successfully — it rejected a wrong field by name, then accepted the corrected
form — and every event still listed zero installed hooks. An open external
defect describes exactly this: project-local hooks silently skipped, no trust
prompt, the session-opening event already passed by the time a user could act,
no workaround, no maintainer response.

## Goals / Non-Goals

**Goals:**

- Tell the user what remains to be done after connecting.
- Answer "is my attachment working?" on demand, with a next action.
- Refuse to forge trust, in the contract and not only in the code.
- Record the scope limitation with its cause and evidence.

**Non-Goals:**

- No change to the live-verified chain.
- No project-local hooks while they are silently skipped.
- No daemon, polling, or dialogue.
- No automation of the user's approval in any form.

## Decisions

### 1. Disclosure belongs to connection, not to documentation

The failure mode is specific: connect, work, observe nothing, conclude the
product is broken. Documentation does not reach a user in that moment; the
command output does. Connection therefore states the trust requirement at the
point where the user has just acted and is looking at the result.

### 2. Forging trust is refused in the contract, not merely omitted

The mechanism is reproducible: trust is a hash written into a configuration
file, and other integrations reportedly compute and write it directly. Leaving
this merely unimplemented would make it a matter of effort. Stating it as a
prohibition makes it a matter of principle, and gives a reviewer something to
check against.

The substance: trust is standing consent to run a command in every session the
user starts. Manufacturing that consent is not a convenience feature. It would
also depend on private implementation details that can change without notice —
so the tempting shortcut is both wrong and brittle.

Reading trust state to report it is observation, and is permitted. The line is
write, not read.

### 3. Health reporting distinguishes states rather than returning a verdict

"Not working" is useless when three different causes produce it. The report
separates unconnected, uninitialized, and untrusted because each has a different
next action, and because untrusted is the state that otherwise looks exactly
like breakage.

### 4. The limitation is recorded as a limitation

User-scope registration means the hook is invoked in every session anywhere, and
the privacy boundary is enforced by our own first act. Writing this as though it
were a design choice would be false: the structural alternative was attempted
and is externally blocked. The requirement therefore records the attempt, the
cause, and the revisit condition — so that when the external defect is fixed,
the first thing to reconsider is obvious.

### 5. The verified chain is not touched

Announcement text, retention, intent binding, and the fail-quiet posture were
proven by live session and keep their behaviour and their tests. This change
adds disclosure and diagnosis around them.

### 6. Corrections from automated review of the implementation

Nine findings, all confirmed and all fixed. Eight technical, one procedural.

The technical ones share a shape worth naming: **the health command could
report green on an attachment that cannot work.** A diagnostic that lies is
worse than no diagnostic, because it converts "something is wrong" into
"something is wrong and the tool says it isn't".

- **Trust was checked for one event, not both.** Connection registers session
  start and stop; an untrusted stop hook means questions are never retained
  while everything else looks healthy. Every registered event now needs a
  record.
- **A stale trust record read as current.** The scaffold records trust against
  the definition's current form; a key can survive a command change while its
  stored value no longer matches. The old check looked only for the key. Since
  confirming the value would mean reproducing a hash this capability forbids,
  the state is now reported as `recorded` — a record exists — with the
  unverified part named, rather than as `granted`.
- **A half-removed registration read as connected.** The opening marker alone
  was treated as proof; removing one stanza left the marker and the green
  result behind.
- **A missing binary read as working.** A registration pointing at a moved or
  deleted executable satisfies every configuration check and still cannot run —
  routine after rebuilding or relocating a local binary.
- **An unreadable configuration read as "not connected"**, recommending a
  connection that reads the same file and fails identically. It is now its own
  state with its own action.
- **The unverified scaffold was told it has a trust gate.** The disclosure
  asserted a mandatory approval step for a scaffold whose behaviour this change
  itself records as unobserved — sending the user hunting for a screen that may
  not exist. Inventing an obstacle is its own kind of misinformation, and the
  wording now differs by what was actually established.
- **A working attachment was called inert.** Rerunning the idempotent
  connection reported `active_now=false` unconditionally. It is now observed,
  and observed against trust rather than full health, because whether any
  particular repository participates is what initialization decides.
- **Health defaulted to one scaffold**, giving a user who connected the other a
  confident diagnosis about a scaffold they never chose. With no way to infer
  intent, an unqualified query now reports all of them.

The procedural finding is about this change's own process: the implementation
was committed while the tasks file still showed the pre-commit owner gate
unchecked, so the repository briefly carried a commit with no recorded evidence
that the gate had been passed. The instruction had been given; the record had
not been written. Recorded now, and the ordering corrected: tick the gate
before the action, not after.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-hook-trust-disclosure-v0`
  version 1 and Context Pack `context-hook-trust-disclosure-v0` version 1.
- **Specification evidence:** one `ADDED` and one `MODIFIED` requirement for
  `ambient-connect`; one `MODIFIED` requirement for `operator-cli-help`.
- **Implementation evidence:** each scenario maps to a deterministic test,
  including a test that no code path writes scaffold trust state.
- **Live evidence:** the verification session recorded above is the reason this
  change exists; its findings are stated in the Context Pack rather than
  summarised here.
- **Failure handling:** health reporting is read-only and reports what it
  cannot determine rather than guessing.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation passes for the change and every promoted
  spec.
- Connection output names the trust requirement, the surface, and the fact that
  nothing acts until then.
- The health command distinguishes all four states and names a next action for
  each that is not working.
- A test asserts no write path to scaffold trust records exists.
- The live-verified behaviour keeps its existing tests unmodified.
- Stop and reshape if reporting trust would require writing anything, or if
  disclosure would require the user to run a Goalrail command per task.

## Risks / Trade-offs

- **[Trust wording may drift from the scaffold's actual surface]** → The
  disclosure names the surface as documented today. If the scaffold renames it,
  the wording is wrong but the user is still told that a trust step exists,
  which is the part that matters.
- **[Reading scaffold configuration to report trust is coupling]** → Accepted
  and bounded: read-only, best-effort, and reported as unknown when it cannot be
  determined. The alternative is a health command that cannot answer the one
  question users will actually ask.
- **[User-scope registration remains the weaker boundary]** → Recorded as a
  limitation with its cause and revisit condition rather than presented as a
  choice. Our own first-act check stays in place as the enforcement.
- **[The second scaffold's trust behaviour is unobserved]** → Recorded. The live
  verification covered one scaffold; the disclosure is written to be true of any
  scaffold that gates hooks behind review.

## Rollback

Delete the change directory before implementation; revert the commits after.
Nothing in the verified chain changes, so rollback restores the previous
behaviour exactly: registration without disclosure, and no health command.

## Open Questions

None.
