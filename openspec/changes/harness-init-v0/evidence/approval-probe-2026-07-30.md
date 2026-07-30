# Approval probe on the shipping arrangement — 2026-07-30

Three interactive Claude Code sessions (`claude` 2.1.215) in one scratch
repository whose hooks were registered the way this change will register them: in
the repository's own per-user project settings file, read from the ordinary
settings layer. Nothing was supplied per run. The owner's working configuration
was neither read as a source nor modified — a check afterwards found zero managed
handlers in `~/.claude/settings.json` and zero in `~/.codex/config.toml`, so
nothing observed here can be attributed to a user-scope registration.

## Arrangement under test

- Scratch repository outside the Goalrail tree, with its own Git history.
- `.gitignore` covering `.goalrail/` and `.claude/settings.local.json`;
  `git check-ignore` confirmed both before the first session, and `git status`
  stayed clean throughout.
- `.claude/settings.local.json` holding the managed handlers — `SessionStart` with
  the `startup` matcher, and `Stop` — each naming the built `gr` with the managed
  marker, in the exact shape connection writes today.
- `GOALRAIL_STATE_HOME` pointed at a scratch state root.
- A stale question seeded at `.goalrail/blocked.md` before each session, so a fired
  session-start hook is provable from the filesystem alone rather than from what an
  agent reports.

## Observed

**A hook registered in the repository's per-user project settings file runs, with
no approval step for the hook.** Each of the three sessions archived the seeded
question into the state root and left the reserved path clear:
`ambient/archive/` holds entries at `12:10:55Z`, `12:15:13Z`, and `12:18:58Z`,
against a stand built at `12:06:00Z`. Each archived payload was byte-identical to
the seed, and no error record was written. No session presented a prompt asking the
user to review, trust, or approve the registered commands.

This is the question the promoted requirement recorded as open for this scaffold.
It is now observed, three times, in interactive sessions with a registration in the
settings layer rather than supplied per run.

**The announcement reaches an interactive agent.** In the third session — the one
that authenticated — the agent was asked what instructions it had received at
session start, and it reported the escalation announcement, naming
`.goalrail/blocked.md` and the `goalrail.escalation/v0` payload format together
with the change-nothing-else condition. Neither file in the stand mentions the
reserved path or the schema; only the session-start hook carries them.

**The stop handler fires and retains the question.** A fresh question written to
the reserved path after session start was retained at
`ambient/questions/f98cb149abfe5c48/question.md`, byte-identical to the file, with
a record naming its own question id, the repository, the session reference, the
digest, and `unbound_reason: NO_ACTIVE_CHANGE` — correct, since the scratch
repository has no OpenSpec change to bind to.

## What the run additionally established, unasked

**On this scaffold, `Stop` is a per-turn event, not a session-end event.** The
retention record is stamped `12:19:33Z`, roughly thirty-five seconds after that
session's start and immediately after the agent finished its single reply — not at
session exit. The provider's documentation confirms the mechanism rather than
leaving it to inference: it fires `SessionStart` and `SessionEnd` once per session,
and `UserPromptSubmit`, `Stop`, and `StopFailure` once per turn, describing `Stop`
as "when Claude finishes responding" and `SessionEnd` as "when a session
terminates".

The first supported scaffold's `Stop` is a session-scoped event, and the existing
registration was built against that meaning. The consequence on this scaffold is
concrete: a question that stays at the reserved path is retained again on every
subsequent turn, minting a new record each time. Per-occurrence identity is
correct by the promoted requirement — two sessions asking a byte-identical question
must keep separate records — but the occurrences here are turns within one session,
which is not what "at session stop" was written to mean.

The registration shape is this change's territory, so the choice of event is a
decision it can take: `SessionEnd` is the event that carries the promoted meaning
on this scaffold. It is recorded here rather than acted on, because it changes
behaviour on a path that already ships.

## Not established

**What the startup screen displayed.** Whether a workspace trust dialog appeared,
and what it listed, was not captured. The provider documents that dialog as gating
permission rules and additional directories rather than hooks, and the hooks ran
regardless, which is what the open question asked.

**Authentication in a spawned session, and why the first two sessions could not
authenticate.** The launcher initially inherited the host session's
`ANTHROPIC_BASE_URL` and host-auth markers; scrubbing every injected variable did
not resolve it either, and the macOS credential store then reported that no
keychain could be found to store the login. That is a property of the owner's
machine and of how a child CLI is launched, not of the scaffold or of Goalrail, and
it bears on nothing this probe was run to answer. It is recorded so a later session
does not mistake it for a Goalrail finding. No credential was read, written, or
handled by Goalrail, and the keychain dialog was cancelled rather than reset.

## Bearing on the requirement

The disclosure for this scaffold may now state, as observed rather than documented,
that a registered repository-scope hook runs without an approval step, and that the
announcement reaches the agent in an interactive session. The documented facts
about the trust dialog's scope remain documentation and stay labelled as such.
Nothing here licenses a claim about the first scaffold, whose trust gate was
observed to exist.
