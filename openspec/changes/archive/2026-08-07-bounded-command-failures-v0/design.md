## Context

Three commands of eight relay Git's message outside a repository (CTX-1). The
one that does not was repaired without recording anything, so nothing carries
the behaviour anywhere (CTX-2). The repository already demands bounded failures
of two particular surfaces and of no command in general (CTX-3), and the
classification needed to name the condition without losing a real failure is
already in place (CTX-5).

## Goals / Non-Goals

**Goals:** state the condition in Goalrail's words (OUT-1); record it for the
command surface rather than per command (OUT-2); keep a broken discovery
distinguishable (OUT-3).

**Non-Goals:** silence (NG-1); exit statuses outside this condition (NG-2);
rework of commands that already behave (NG-3); error wording in general (NG-4);
a Git-specific rule (NG-5).

## Decisions

**D1 — The requirement lives in `operator-cli-help`.** It is the only capability
whose subject is the `gr` operator surface as a whole; every other one owns a
behaviour rather than the command line. The alternative — repeating the rule in
`harness-init` and `harness-update` — would place a general statement in two
capabilities that each see part of it, which is how the rule came to be missing
in the first place. The capability's purpose widens from what a command says
when asked, to what a command says.

**D2 — Translate at the command boundary, not in discovery.** `ResolveWorktreeRoot`
returns a classified error and its callers include the ambient hook, which must
stay silent, and the diagnosis, which already renders its own report. Changing
the error's text there would reach paths that do not print it. Each command that
prints translates the classification it already receives.

**D3 — One shared translation, not three.** The three commands would otherwise
drift, which is the failure this change exists to stop. A single helper turns the
classified condition into Goalrail's sentence, and a command that acquires the
discovery path later reaches for the same one.

**D4 — A failure that is not the condition keeps its own outcome.** `#67` narrowed
the classification so only Git's own verdict means "not a repository"; this
change relies on that rather than re-deriving it, so an absent Git surfaces as
the discovery failing and keeps the status that says the check did not run.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| The classified discovery error | `internal/project`, already narrowed so only Git's own verdict means the condition | The not-a-repository classification, or any other failure | Nothing new is refused; this slice changes what is printed, not what is accepted | Read once per command invocation | A survey across every command outside a repository, and a run with no Git on the lookup path |

## Correlation and Evidence

No new artefact. The evidence is the command output itself, which is what the
survey reads.

## Measurement and Stop Conditions

- The survey across every command contains no `fatal:` and no `exit status`,
  where three of eight carry one today (SIG-1).
- The requirement is one statement about commands generally (SIG-2).
- A run with no Git differs from a run outside a repository in wording and in
  exit status (SIG-3).
- Three regressions fail before the change (SIG-4).

Stop and reshape if naming the condition requires each command to re-derive the
classification, since three copies of a rule is the shape this replaces.

## Risks / Trade-offs

- **The user loses Git's own words.** That is the point, but it costs a reader
  who would have recognised them. The condition Goalrail names is the same one,
  and a genuine failure of the tool still surfaces as a failure.
- **The requirement widens a capability's purpose.** Stated in D1 rather than
  left for a reader to notice.

## Rollback

Reverting restores the previous messages; nothing durable is written and no
consumer stores the text.

## Open Questions

None.
