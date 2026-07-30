## Context

Intent Snapshot `intent-observe-claude-code-live-v0` version 1 is confirmed.

The promoted requirement records that the second supported scaffold has never
been exercised live, listing announcement delivery, question retention, and
approval prompting as unverified. One live run has settled the first two and left
the third untouched.

The run used a scratch repository holding a synthetic contradiction — one
requirements file demanding both that an empty list be rejected and that it be
accepted — with a task saying to implement it exactly as specified and not to edit
the requirements. Neither file mentioned Goalrail, the reserved path, or
escalation.

## Goals / Non-Goals

**Goals:**

- Make the record match what is now known, in both directions.
- Keep the evidence in the repository rather than in a conversation.
- Withdraw a recorded blocker that turned out not to exist.

**Non-Goals:**

- No claim that the approval question is answered.
- No behaviour change and no code.
- No registration in the owner's configuration and no copying of account files.

## Decisions

### 1. The recorded blocker was wrong, and that matters more than the observation

The previous change recorded that live verification was impossible without either
writing into the owner's working configuration or copying an account file of
unrelated history. Both were correctly refused, and the conclusion drawn was that
the behaviour must stay unobserved.

Two facts undo it. Credentials are held in the operating system keychain, not in a
file tied to the home directory, so the isolation attempt failed for a reason that
was never diagnosed. And the provider's CLI takes settings for a single run, which
makes registration unnecessary for observation entirely.

The lesson is not that the earlier refusal was wrong — it was right — but that
"we cannot observe this" was recorded as a property of the scaffold when it was a
property of the two routes that had been considered. A blocker is a claim like any
other and deserves the same scepticism as the thing it blocks. The requirement
therefore withdraws it explicitly rather than quietly dropping it.

### 2. Two behaviours observed, one untouched — recorded separately

Delivery was established rather than assumed: the agent named the reserved path
and the payload schema although the task mentioned neither, so the only source was
the session-start hook. Retention was established by the artifact: a record
outside the repository carrying its own identifier, the digest, the session
reference, and an explicit unbound reason, with the retained payload byte-identical
to the file the agent wrote.

The approval question is untouched, and the temptation to read the run as settling
it is exactly what the record must resist. No prompt appeared — under two
conditions that each independently explain the absence. The run was
non-interactive, where the provider's own help states the trust dialog is skipped.
And the hooks were supplied per-run rather than registered, which is not the
arrangement a user creates.

So the record states both conditions, not just the conclusion. "No prompt appeared
but the run cannot show one" is the honest form; "no prompt appeared" alone would
read as evidence to anyone who did not reconstruct the setup.

### 3. Two general scenarios, because the shape recurs

Rather than only amending one scaffold's paragraph, the delta adds two scenarios
that generalise what went right here: a partly observed scaffold is recorded as
partly observed, and an absence produced under conditions where the behaviour
would not have appeared anyway is not evidence.

This session produced both mistakes in other places and caught them by review, so
they are not hypothetical failure modes. Writing them as scenarios makes the next
partial observation cheaper to record correctly than to record loosely.

### 4. The evidence lives in the repository

The run happened in a scratch directory that does not survive the session. Its
artifacts are copied into the change's evidence directory with the absolute
scratch paths reduced to a placeholder, so the file is readable later without
implying those paths still exist. What is kept is what a sceptical reader needs:
the invocation, the fixture's shape, the agent's reply, the question it wrote, and
the retained record.

The fixture is synthetic and contains no user content, so keeping it whole costs
no privacy.

### 5. The delta was generated from the promoted text, not retyped

A `MODIFIED` requirement replaces the promoted one wholesale, and earlier in this
session a hand-written delta silently dropped a scenario. This delta is produced
by reading the promoted requirement, substituting one paragraph and one scenario,
and asserting by set difference that no promoted scenario is missing. The guard is
part of how the file was made rather than a check performed afterwards.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-observe-claude-code-live-v0`
  version 1 and Context Pack `context-observe-claude-code-live-v0` version 1.
- **Specification evidence:** one `MODIFIED` requirement for `ambient-connect`,
  restating every promoted scenario and adding two.
- **Run evidence:** `evidence/live-run-2026-07-30.md`.
- **Withdrawn-blocker evidence:** a keychain credential entry present with no
  credential file beside the settings, and the provider CLI's own help for
  per-run settings.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation passes for the change and every promoted
  spec.
- The requirement describes announcement delivery and question retention as
  observed, and neither appears in the remaining gap.
- The remaining gap names both conditions that make the run inconclusive about
  approval.
- No promoted scenario is dropped, asserted by set difference.
- No Go diff, and the existing suite passes unmodified.
- Stop and reshape if recording the observation cannot be done without also
  claiming something the run did not establish.

## Risks / Trade-offs

- **[The observation is one run on a synthetic fixture]** → It establishes that
  the mechanism works, not how often or how well an agent uses it. The record
  claims only the former, and the fixture is kept so the reader can judge.
- **[Reading the absent prompt as evidence]** → The specific risk this change
  guards against, addressed by stating the conditions rather than the conclusion
  and by a general scenario that outlives this instance.
- **[The withdrawn blocker leaves the approval question looking easy]** → It is
  not: answering it still needs an interactive session with a registration in the
  owner's working configuration, which is the one arrangement that would fire our
  hook inside the sessions doing the work. That constraint is unchanged.

## Rollback

Delete the change directory before promotion; revert the commit after. Reverting
restores a record that understates what is known, so the evidence file is the part
worth keeping if the requirement text is ever reverted for unrelated reasons.

## Open Questions

None.
