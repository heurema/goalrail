## Why

Three of eight commands relay Git's own message and exit status when run outside
version control, and the one that does not behaves correctly because it was
repaired rather than because anything requires it (OUT-1, OUT-2).

## What Changes

- Record a requirement covering the `gr` command surface generally: a command
  reports a condition it meets in Goalrail's own words and relays no message,
  exit status or output produced by a program it invoked.
- Correct `init`, `migrate` and `update`, which relay Git's message today.
- Keep a broken discovery distinguishable from an absent repository, in wording
  and in exit status, so a bounded message does not become a way of losing a
  genuine failure.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Record the requirement for the command surface generally | OUT-2, SIG-2 | NG-4, NG-5 |
| Correct the three commands that relay today | OUT-1, SIG-1, SIG-4 | NG-3 |
| Keep a failed discovery distinguishable | OUT-3, SIG-3 | NG-1, NG-2 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-cli-help`: the capability that owns the `gr` operator surface gains a
  requirement about what a command says when it cannot proceed, alongside what it
  says when asked for help.

## Impact

- `cmd/gr`: `init`, `migrate` and `update` translate the not-a-repository
  condition instead of returning the discovery error unchanged.
- No change to `doctor`, which already does this, or to the commands that never
  reached the discovery path.
- No change to exit statuses outside this condition.

## Non-Goals

- Not silence: the condition is still named, in Goalrail's terms.
- Not a rewording of every error a command can produce; the subject is a foreign
  program's output reaching the user.
- Not specific to Git, which is only the program that appears today.
