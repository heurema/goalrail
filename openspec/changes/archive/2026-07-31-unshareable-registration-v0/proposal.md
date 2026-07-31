## Why

A first `gr init` on a machine that was not prepared for it refuses the
registration. The harness installs, the report says so, and the attachment never
fires. The only remedy the tool offers is `--fix-gitignore`, which edits a file
the repository shares with everyone who clones it — so the user is asked to
change shared content in order to install something only they consented to.

The refusal is right about the danger and wrong about the remedy. A registration
a commit could carry would run in every teammate's session on one user's
consent, so it must not be written where a commit can reach it. But making the
path unshareable does not require touching shared content at all: git has a
per-clone ignore rule that no commit can carry, and the half of Goalrail that
decides whether a path is unshareable already accepts it, because it asks git
rather than reading files itself.

Now, because the first run on an unprepared machine is about to be the product's
first impression rather than an incidental path, and because the defect is
invisible from here — the maintainer's own global ignore rule hides it, and
Goalrail's own repository reproduces it.

## What Changes

Initialization makes the registration path unshareable by its own act, writing
the entry to the repository's per-clone exclude file instead of asking the user
to edit shared ignore rules. No tracked file is modified. The participation
marker is covered by the same act rather than left as a notice about a condition
initialization could have fixed itself.

The refusal survives for the cases that still deserve it: a path already tracked,
and a shared rule that outranks the per-clone one. `--fix-gitignore` survives as
the remedy for the second, because there it is the only remaining move.

The write is conditional on there being somewhere to write. Where the directory
is not a repository, where `git` is not on the lookup path, where the target is
not a file initialization may write, and where the write itself fails, it is
skipped and the command completes as it does today — initialization acquires no
dependency it did not have, and not having made a path unshareable is a condition
the registration already knows how to refuse.

Two quiet edges are closed with it. A directory that is not yet a repository now
gets a notice saying the registration becomes committable the moment it becomes
one. And a repository with no work tree — bare, or the repository's own directory
— is refused before the first write, by initialization and by update alike, so
nothing Goalrail writes lands beside `HEAD`, `objects` and `refs`.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| The entry is written to the repository's per-clone exclude rule, by the same act that registers, covering both the registration path and the marker | OUT-1, SIG-1 | NG-2, NG-4 |
| The refusal survives where the path cannot be made unshareable, naming the reason, the entry, and the flag | OUT-2, SIG-2 | NG-1, NG-3 |
| The write is skipped where there is nothing to write to: not a repository, no `git`, or a bare repository | OUT-3, SIG-3 | NG-4 |
| The file is located by asking git, its parent is created where missing, existing content survives, and re-running changes nothing | OUT-4, SIG-4, SIG-5 | NG-2 |
| The entries appear in the report among the changes initialization made | OUT-5, SIG-6 | — |
| A directory that is not yet a repository is told what becomes true when it becomes one | OUT-6, SIG-7 | NG-6 |
| Every question is asked about the directory the user named, with the caller's own repository selection disregarded | OUT-1, OUT-2, SIG-1, SIG-2 | NG-2 |
| An entry is expressed so it covers the path from wherever the rule is read, for a directory below the top of the work tree | OUT-4, SIG-4 | NG-1 |
| The write is skipped rather than fatal where the target is a link or the write fails, and the refusal answers for it | OUT-3, SIG-3 | NG-4 |
| The refusal names the rule that actually decided rather than the one usually responsible | OUT-2, SIG-2 | — |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `harness-init`: the requirement "Initialization registers the attachment where
  the scaffold allows it" changes on one point — initialization now makes the
  registration path unshareable itself, with a rule the repository cannot carry
  to anyone else, instead of refusing until the user edits shared ignore rules.
  The refusal, the explicit flag, and the consent property it protects are
  unchanged; the marker's clause narrows to the case it still describes; and a
  directory that is not yet a repository gains a statement it did not have.
- `harness-update`: the same boundary, stated where it belongs. A repository with
  no work tree is refused by update as well as by initialization, because the two
  install the same files and a rule enforced in one of them is a manner rather
  than a guarantee.

`ambient-connect` is deliberately absent. Its promoted text already requires the
path be unshareable, "made so by the same act where that is possible", and
conditions its refusal on the path not being makeable unshareable. Both stay
true under the new behaviour, so it is owed no delta — this change is the case
that clause anticipated.

## Impact

- `internal/ambient` — the ignore writer changes target and gains a location
  step that asks git, a parent-directory step, and a work-tree condition. The
  ignore reader is untouched, including how it resolves precedence.
- `cmd/gr` — initialization calls the writer as part of registering rather than
  only under a flag; the refusal text, the marker notice, and the report's list
  of changes are affected; the flag keeps its meaning for shared rules; and
  update gains the same no-work-tree refusal, through the shared helper.
- Tests — three existing tests pin the current behaviour and are rewritten to
  pin the new one. Two others must keep passing unmodified, because they are the
  evidence that initialization needs neither a repository nor `git`; an edit to
  either is a signal the implementation is wrong.
- `README.md` — the install section and its flag row describe the old remedy.
- No dependency changes. No new command, flag, or configuration.

## Non-Goals

- The explicit flag is not retired. Where a shared rule outranks the per-clone
  one, it is the only remedy, and its promoted sentence survives the delta.
- Nothing is written outside the repository. The user's global ignore file is
  the most durable location and is not touched: consent to initialize one
  repository is not consent to configure the machine.
- No change to registration scope, to which scaffold registers where, or to the
  events registered.
- No change to how the ignore state is read, including precedence between rules.
- No guarantee for a working tree shipped without its history. A tree copied by
  `tar --exclude=.git` loses the rule, as it loses every other ignored file's
  protection; the promise is about what a commit could carry.
- No self-repair of a registration that has already become shareable, and no new
  diagnosis state for it.
- This proposal authorizes no implementation, commit, push, pull request, merge,
  tag, release, or other external effect. Each keeps its own owner gate.
