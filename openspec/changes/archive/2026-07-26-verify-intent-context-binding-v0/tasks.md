## 1. Amend the snapshot to cover the implementation

- [x] 1.1 Retain version 1 unmodified as `intent-v1.md`; do not repair history by rewriting prior evidence.
- [x] 1.2 Raise the Context Pack to version 2, record each confirmed review finding as a context item, and carry exactly one valid evidence reference per row.
- [x] 1.3 Write candidate intent version 2 whose outcomes cover the shipped implementation instead of listing it as a non-goal.
- [x] 1.4 Obtain explicit owner confirmation of candidate version 2; continuation, silence, and absence of objection are not confirmation.

## 2. State the requirement at the boundary that holds

- [x] 2.1 Replace "before freezing" with "before any prepared state is persisted", matching the order `Service.Prepare` uses.
- [x] 2.2 Replace the unconditional no-pack allowance with the change loader's schema-aware rule.
- [x] 2.3 State that both bounded artifacts must be regular files and are rejected before being opened.
- [x] 2.4 Preserve both pre-existing scenarios verbatim and add scenarios for the requirement rule and the non-regular artifact.

## 3. Close the bypass and the hang

- [x] 3.1 Call `changeRequiresContext` from the missing-context path so preparation and the change loader cannot drift.
- [x] 3.2 Keep the intent's own Context Pack declaration as an additional trigger for changes the loader does not cover.
- [x] 3.3 Route both bounded reads through `readBoundedRegularFile`, which opens with `O_NOFOLLOW` and `O_NONBLOCK` and inspects the same descriptor it reads, so the guarantee cannot be raced by a substituted link or pipe.
- [x] 3.5 Qualify the requirement so the artifact clauses apply to the Context Pack only when one is present or required, keeping it consistent with the supported no-pack path.
- [x] 3.4 Add regression tests: a current project-schema change without a pack is rejected; an archived change without one and a change under another schema still succeed; a FIFO supplied as either artifact fails with a bounded reason under a deadline; and the read primitive rejects a substituted symbolic link, does not block on a substituted pipe, and still reads a regular file.

## 4. Verify

- [x] 4.1 Run `gofmt -l`, `go build ./...`, and `go vet ./...`.
- [x] 4.2 Run `go test ./...` and `go test -race ./...`.
- [x] 4.3 Run pinned telemetry-disabled strict OpenSpec validation.
- [x] 4.4 Confirm every Context Pack source cell matches the accepted evidence-reference pattern.
- [x] 4.5 Confirm no file under `openspec/changes/archive/` is modified, no existing intent ID or version is reused, and canonical schemas and command behaviour are unchanged.

## 5. Stop at the owner gates

- [x] 5.1 Show the owner the amended artifacts, the fix for each finding, and the verification results.
- [ ] 5.2 Obtain an explicit owner instruction before resolving the review threads and approving the pull request.
- [ ] 5.3 Obtain a separate explicit owner instruction before merging.
- [ ] 5.4 Obtain a separate explicit owner instruction before archiving this change and promoting the delta into `openspec/specs/local-run/spec.md`; re-read the promoted text immediately before promotion in case another change archived first.
