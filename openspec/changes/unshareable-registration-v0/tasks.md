## 1. Locate the clone's rule, and gate writing it

- [x] 1.1 Add an exported way to ask whether a path is under version control at all, so `cmd/gr` can distinguish "not a repository" from "ignored"; today `IgnoreState` answers `true` for a directory that is not a repository (`internal/ambient/scope.go:343-345`) and the two cases are indistinguishable to the caller.
- [x] 1.2 Add a resolver for the clone's rule file: `git -C <root> rev-parse --git-common-dir`, joined with `info/exclude`, resolved against the repository root when the answer is relative. Do not join `.git` — it is a regular file in a linked worktree and in a submodule.
- [x] 1.3 Gate the resolver on there being somewhere to write: git resolves on the lookup path, the directory is a repository, and `git -C <root> rev-parse --is-bare-repository` reports false. Any of them failing returns "no target" rather than an error.
- [x] 1.4 Create the target's parent directory before writing; `git init` does not always create `info/`.
- [x] 1.5 Generalize `AddIgnoreEntries` over its target, keeping all three existing behaviours unchanged: skip an entry already covered, skip an entry whose check cannot run, and supply a missing final newline before appending. Make the error text name the file it failed on rather than `.gitignore`.

## 2. Write the rule as part of registering

- [x] 2.1 Write both entries — the registration path and the marker — to the clone's rule before the scaffold is selected, so registration reads the state the write produced. Keep the write off the `--fix-gitignore` path: the flag continues to write the shared rules and nothing else.
- [x] 2.2 After writing, re-read the ignore state per entry. Remove any entry still not in effect — a shared rule outranks the clone's — and let the existing refusal fire. Per entry rather than per write, because a shared rule can defeat one of the two and not the other, and undoing the whole write would discard a rule that did its job. Where the removal fails, keep the entries and report them.
- [x] 2.3 Rewrite the refusal text at `cmd/gr/ambient.go:258-263`: it currently prescribes adding the path to `.gitignore` and re-running with the flag, which is no longer the ordinary cause. It must name the cause it now has — the path is tracked, or a shared rule outranks the clone's — and prescribe the flag only for the second.
- [x] 2.4 Narrow the marker notice at `cmd/gr/ambient.go:164-177` to the case it still describes. It currently prescribes `--fix-gitignore` for a condition initialization now fixes itself.
- [x] 2.5 Keep `--fix-gitignore` and its help text, correcting the text to say it writes the rules the repository shares — the remedy for a shared rule that outranks the clone's.

## 3. Report both writes, distinctly

- [x] 3.1 Keep `ignore_entries_added` meaning entries added to shared rules, which only the flag produces.
- [x] 3.2 Add a separate field for the clone's entries, so a reader can tell committable repository content from a rule that stays in the clone. Populate it on every run that writes.

## 4. Close the two quiet edges

- [x] 4.1 Where the directory is not a repository, add a notice saying the registration becomes committable the moment it becomes one. Nothing is refused and nothing is written; initialization completes as it does today.
- [x] 4.2 Apply the work-tree gate to the flag's writer as well, so `gr init --fix-gitignore` against a bare repository writes neither ignore rules nor harness content beside `HEAD` and `objects`.

## 5. Rewrite the three pins, and leave two alone

- [x] 5.1 `cmd/gr/harness_test.go:54` `TestInitRefusesToRegisterWhereACommitCouldCarryIt` — invert it: a stock clone registers, and no tracked file differs afterwards. Keep a refusal case, driven by a tracked settings path rather than by a missing entry.
- [x] 5.2 `cmd/gr/harness_test.go:340-353` the marker-exposure subtest — the notice is now reachable only for a marker the clone's rule cannot cover.
- [x] 5.3 `internal/ambient/scope_test.go:311-338` `TestAddIgnoreEntriesIsAdditiveAndIdempotent` — it keeps its target, because the flag still writes the shared rules; add cases for the clone's rule where there is no target at all and where its directory does not exist.
- [x] 5.4 Leave `TestTheHarnessCommandsNeedNoNodeRuntime` and `TestInitCommandIsExplicitAndReportsItself` unmodified. They are the evidence that initialization needs neither git nor a repository; an edit to either is a signal the implementation is wrong.

## 6. Pin the new behaviour

- [x] 6.1 A first initialization on a repository nobody prepared registers, is active, and modifies no tracked file. Build on the existing scratch-repository helpers, which already neutralize the machine's own ignore configuration with a local `core.excludesFile`; without that the test passes on this machine whether or not the fix is present. (SIG-1)
- [x] 6.2 A tracked settings path still refuses; a shared rule that outranks the clone's still refuses; both name the reason, the entry, and — for the second — the flag. Assert the entry that could not take effect is taken back out and the one that did is kept. (SIG-2)
- [x] 6.3 Initialization completes with an empty lookup path, in a plain directory, and against a bare repository, and the bare repository gains no harness content. (SIG-3)
- [x] 6.4 Initialization from a linked worktree and from a submodule registers, in layouts where the assumed path does not exist. (SIG-4)
- [x] 6.5 A rule file whose last line has no terminator keeps every line's meaning, both entries take effect, and a second initialization changes no byte. (SIG-5)
- [x] 6.6 The report names the clone's entries. (SIG-6)
- [x] 6.7 Initialization in a directory that is not a repository carries the notice, refuses nothing, and completes. (SIG-7)

## 7. Documentation

- [x] 7.1 `README.md:108` and the flag's row in the table at `README.md:129` describe the old remedy; correct both to say initialization makes the path unshareable itself and what the flag is now for.

## 8. Verification

- [x] 8.1 `go vet ./... && go test ./...` clean.
- [x] 8.2 Walk a first run by hand in a fabricated environment: initialize, confirm the registration applied, confirm `git status --porcelain` shows no tracked modification, and read the report's disclosure of what it wrote.
- [x] 8.3 Run the same walk against a scratch clone of Goalrail's own repository, which reproduces the defect today because its tracked ignore rules cover the marker and not the settings path (CTX-3). A clone, not the working tree: the walk writes.
- [x] 8.4 `openspec validate unshareable-registration-v0 --strict`.

## 9. Defects found by review, before the commit gate

- [x] 9.1 Run every git invocation in `internal/ambient` through one helper that removes `GIT_DIR`, `GIT_WORK_TREE`, `GIT_COMMON_DIR`, `GIT_INDEX_FILE` and `GIT_OBJECT_DIRECTORY` from the environment. Without it, a shell that exported them has the rule written into another repository and read back from it, and the registration applies to a path the named repository would carry in a commit — a regression against HEAD, which refuses in that state.
- [x] 9.2 Stop returning the clone-rule write error from `runInit`. It aborted after the overlay and the marker were on disk, printed no report, and named no remedy — where HEAD merely refused the registration. Record it as a notice and let the refusal answer.
- [x] 9.3 Refuse a symlinked target rather than writing through it: the write would leave the repository, possibly into tracked content, and the take-back path would then remove a file the user owns. The same guard for the shared writer.
- [x] 9.4 Anchor entries carrying a path to the top of the work tree, so `--repo` on a subdirectory covers the paths under it. Resolve both sides with `EvalSymlinks` first, or every macOS temporary directory produces an entry that matches nothing.
- [x] 9.5 Replace the bareness guard with `git rev-parse --show-toplevel`, which also covers a repository's own directory, and apply the same helper in `runUpdate`.
- [x] 9.6 Ask `git check-ignore -v` for the rule that actually decided, and word the refusal from the answer instead of asserting a shared rule that may not exist.
- [x] 9.7 Let the ignore state, not a matching line, decide whether an entry is needed; keep the textual check only to avoid writing a duplicate line.
- [x] 9.8 Do not terminate a last line the user never terminated when the take-back keeps nothing; believe `rev-parse` only where it names a directory that exists; stop repeating the path an `os` error already carries.
- [x] 9.9 Report the resolved path of the file written inside the repository's own directory, which the user cannot derive in a linked worktree or a submodule.
- [x] 9.10 Add a delta for `harness-update`: the no-work-tree boundary belongs to the harness, not to one command.
- [x] 9.11 Correct the artifacts against the code: the no-work-tree case refuses rather than completing; restore the `unasked` qualifier the MUST NOT dropped; give the not-a-repository scenario its version-control precondition; make SIG-1's tracked-file comparison non-vacuous by committing something first.
- [x] 9.12 Correct the README's install prompt, which named one cause for a refusal that has three and prescribed the flag for all of them.
