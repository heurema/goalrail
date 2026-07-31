## Context

`gr init` refuses to register the session hooks unless the scaffold's settings
path is already ignored. On a machine that was never prepared, it is not, so the
refusal is the default outcome of a first run: the harness installs, the report
says the registration was refused, and the attachment never fires. The remedy the
tool offers — `--fix-gitignore` — writes into `.gitignore`, which the repository
carries to everyone who clones it.

The refusal's reasoning is sound. What is wrong is treating "ignored" as a
condition the user must arrange in shared content. Git keeps a per-clone ignore
rule that no commit can carry, and the half of Goalrail that decides whether a
path is unshareable already accepts it, because `IgnoreState`
(`internal/ambient/scope.go:338-356`) asks `git check-ignore` rather than reading
ignore files itself. Seeding that rule by hand flips the registration from
refused to applied with no code changed. This slice makes initialization write
it.

## Goals / Non-Goals

**Goals:**

- A first initialization on an unprepared machine registers and attaches, with no
  tracked file modified.
- The consent property is bit-for-bit what it was: nothing a commit could carry.
- Initialization acquires no dependency on git or on being run inside a
  repository.
- The clone's rule is located by asking git, and writing it damages nothing the
  user put there.
- The report discloses what was written inside `.git`.

**Non-Goals:**

- Retiring `--fix-gitignore`. It is the only remedy where a shared rule outranks
  the clone's.
- Writing anything outside the repository, including the user's global ignore
  file.
- Changing how the ignore state is read, including precedence.
- Changing registration scope, scaffold selection, or the events registered.
- Protecting a working tree that was copied without its history.

## Decisions

**Location comes from git, not from a path.** The target is
`$(git rev-parse --git-common-dir)/info/exclude`. Not `--git-dir`: in a linked
worktree that resolves to `.git/worktrees/<name>`, which has no `info/`, while
the rule git actually consults lives in the common directory. Not
`filepath.Join(root, ".git", ...)`: in a linked worktree and in a submodule
`.git` is a regular file, and the join names nothing. The parent directory is
created before writing, because `git init` does not always create `info/` — it
does not under this machine's own `init.templateDir`.

**The write fails open, including when the write itself fails.** Anything that
means "there is nowhere to write" — not a repository, git absent, git answering
unintelligibly, the target a symlink, `MkdirAll` or the write failing on
permissions — skips the write and lets the existing paths answer. This is not
defensive coding, it is the requirement: two existing tests initialize in a plain
directory and with an empty lookup path, and a prototype that wrote
unconditionally failed both. Returning the write error instead would be worse
than the defect being fixed: it aborts after the overlay and the marker are on
disk, prints no report at all, and leaves no remedy named — where the old code
merely refused the registration and said why.

**The boundary is the work tree, not bareness.** `--is-bare-repository` answers
false for a repository's own directory, which has no work tree either and where
content lands beside `HEAD`, `objects` and `refs` just the same.
`git rev-parse --show-toplevel` answers both with one question, and it is asked
before the first write so a refusal leaves the directory as it was. The same
helper gates `gr update`, which materializes the same overlay: a rule enforced in
one command is a manner rather than a guarantee.

**Every question is asked about the directory the user named.** `GIT_DIR`,
`GIT_WORK_TREE` and their relatives name a repository directly and override
`-C`, so a shell that exported them — the bare dotfiles-repository arrangement is
the common one — would have the rule written into that repository and then read
back from it, and the answer would come out "ignored" about a path the named
repository would carry in a commit. Every git invocation in the package therefore
runs with those variables removed from the environment.

**Entries are anchored where the rule is read.** The clone's rule is matched from
the top of the work tree, and `--repo` may name a directory below it. An entry
containing a slash is anchored, so it has to carry the path back down or it names
something that does not exist — and the refusal that followed would then blame a
shared rule the repository does not have. Entries without an interior slash match
at any depth and are left alone. The two paths are made comparable with
`EvalSymlinks` first, because version control answers with links resolved and the
caller's path generally is not; on macOS, where the temporary and system
directories are links, skipping that produces an entry that matches nothing.

**The write is speculative, and each entry that achieves nothing is taken back
out.** Whether the clone's rule can cover a path cannot be decided without
writing it, because deciding it in advance would mean implementing git's
precedence rules, which this change refuses to do. So the entries are written,
the state is read again, and any entry still not in effect is removed. The check
is per entry rather than per write, because a shared rule can defeat one of the
two and not the other — `!.claude/**` leaves the marker's entry working — and
undoing the whole write would discard a rule that did its job. An entry that
provably has no effect is noise the user would later have to diagnose. Where the
removal itself fails, the entries stay and the report names them, because a
silent partial state is worse than a visible one.

**The existing writer is extended, not duplicated.** `AddIgnoreEntries` already
skips entries that are already covered, already skips a tracked path whose check
cannot run, and already supplies a missing final newline before appending — the
guard that stops an append from concatenating itself onto a user's last
unterminated line and leaving neither path ignored. All three behaviours are
needed here unchanged; only the target and the error text are target-specific.

**The refusal names the cause it has, not the one usually responsible.** Saying
"a rule your repository shares overrides this" is an assertion about the user's
repository, and it is false whenever something else was in the way — a negation
later in the clone's own rule, a subdirectory anchoring mistake, an unwritable
target. `git check-ignore -v` reports which file decided, so it is asked, and the
wording follows the answer.

**The two writes are reported separately.** `ignore_entries_added` keeps its
meaning: entries added to rules the repository shares, which only the explicit
flag produces. The clone's entries get their own field, because one of these
writes is committable repository content and the other is not, and a single list
would tell the user that entries were added without telling them which kind.

**The file written inside the repository's own directory is named.** Its path is
not one the user can derive: it is elsewhere in a linked worktree and in a
submodule. The report carries the resolved path alongside the entries.

**Registration and the marker are covered by one act.** Both entries are written
before the scaffold is selected, so registration sees the state the write
produced. The marker's notice narrows accordingly: it now describes only the case
where the clone's rule could not cover it, which is the case it was always about.

**Tests must neutralize the machine's own ignore configuration or they prove
nothing.** A test asserting that registration applies would pass without the fix
on any machine whose global ignore file already carries the entry — which this
one's does, and which is why the defect was invisible here. Both test packages
already solve it, and correctly: their scratch-repository helpers set
`core.excludesFile` to `/dev/null` in the repository's own config, which outranks
the global file. That is better than fabricating `HOME` and `XDG_CONFIG_HOME`,
because the excludes file has a built-in default path that no environment
variable removes — `GIT_CONFIG_GLOBAL=/dev/null` alone leaves it in play, and the
existing helper says so in a comment. Every new test builds on those helpers
rather than inventing a third mechanism.

## Correlation and Evidence

Every fact this design rests on was reproduced rather than reasoned, and the
Context Pack records the command for each: the refusal under a fabricated
environment (CTX-1), the read path accepting the clone's rule with no code
changed (CTX-4), `.git` as a regular file in worktrees and submodules (CTX-8),
the missing `info/` directory (CTX-9), the four negation shapes that outrank the
clone's rule (CTX-10), the two commands a naive writer breaks (CTX-12), the bare
repository (CTX-13), and the append corruption (CTX-14).

The evidence that the change works is the report of a first initialization in a
fabricated environment together with `git status --porcelain`. The evidence that
it changed nothing else is that the two tests pinning initialization without a
repository and without git pass unmodified; an edit to either is a signal that
the implementation is wrong, not that the test is stale.

## Measurement and Stop Conditions

Pass signals are the seven in the intent, each carried by a test. The change is
complete when a first initialization in a fabricated environment registers with
no tracked file modified, the refusal still fires for a tracked path and for an
outranking shared rule, and initialization still completes in a plain directory,
with no git, and against a bare repository.

Stop and reshape if the clone's rule turns out to need git's precedence rules
reimplemented to be used correctly — that would put the change in conflict with
its own non-goal, and the honest answer would be to keep the refusal and improve
its wording instead.

## Risks / Trade-offs

- **The rule does not travel.** A working tree shipped without `.git` lands in a
  fresh clone with the registration stageable. Accepted and stated as a non-goal:
  the promise is about what a commit could carry, and that copy carries every
  other ignored file the same way. A plain re-clone is unaffected, because the
  registration is absent too and initialization runs again.
- **One write covers sibling worktrees.** Linked worktrees share the common
  directory's rule, so initializing one covers all of them. Harmless, but the
  report must not imply the write is scoped to the worktree it ran in.
- **A negation added later.** A user who adds an outranking rule after
  initializing keeps a registration that has quietly become shareable.
  Detecting it is declined as a non-goal; the diagnosis reports the attachment,
  not the ignore rule's continued effect.
- **A write inside `.git` is unusual.** Users do not expect a tool to write
  there. Mitigated by disclosure in the report rather than by not doing it: the
  alternative locations are shared content or the user's machine-wide
  configuration, both worse.

## Rollback

Revert the commit. Nothing migrates and nothing is left broken: the entries
already written are inert lines in a per-clone file, `--fix-gitignore` still does
what it always did, and a repository initialized under the new behaviour keeps
working under the old one — its path is ignored, which is exactly the condition
the old refusal checks for.

## Open Questions

None.
