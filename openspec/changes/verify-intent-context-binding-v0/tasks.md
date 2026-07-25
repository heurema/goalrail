## 1. State the requirement

- [x] 1.1 Read the promoted `local-run` requirement `Run preparation is inspectable and does not launch` from merged `main` and preserve its existing text and both existing scenarios verbatim in the delta.
- [x] 1.2 Add the preparation-time Context Pack paragraphs: intent bytes and digest first, then the declared sibling pack, both confined to the canonical repository root under the same bounded artifact size limit.
- [x] 1.3 State that a missing, unreadable, malformed, mismatched, oversized, or root-escaping pack, and any absent context reference, fails before prepared state, run ID, launch claim, or provider invocation.
- [x] 1.4 State that an intent declaring no Context Pack remains valid, so the check cannot later be tightened into a regression.
- [x] 1.5 State that the check depends on no proposal, specification, design, task, provider, or activation artifact and adds no Context Pack fields to the canonical WorkSpec.

## 2. Bind every scenario to existing evidence

- [x] 2.1 Map the accepted path to the context-bound resolver test and confirm no new code is required.
- [x] 2.2 Map each rejection class to the invalid-binding test table: missing context, mismatched context identity, unknown context reference, and context symlink escaping the repository.
- [x] 2.3 Map the no-pack path to the confirmed-identity and digest test.
- [x] 2.4 Record the mapping in the design so a later reader can re-verify it without re-deriving the behaviour.

## 3. Verify the change is specification-only

- [x] 3.1 Confirm the working tree contains no production code change beyond the already-committed implementation.
- [x] 3.2 Run `go build ./...`, `go vet ./...`, and `go test ./...` once and require results unchanged from the committed baseline.
- [x] 3.3 Run pinned telemetry-disabled strict OpenSpec validation for this change and every promoted spec.
- [x] 3.4 Confirm no file under `openspec/changes/archive/` is modified and no existing intent ID or version is reused.
- [x] 3.5 Confirm the canonical WorkSpec and receipt schemas, `gr` command behaviour, and every other capability are byte-unchanged.

## 4. Stop at the owner gate

- [x] 4.1 Show the owner the change artifacts, the scenario-to-test mapping, and the verification results.
- [ ] 4.2 Obtain an explicit owner instruction before committing this change; planning completion is not commit authority.
- [ ] 4.3 Obtain a separate explicit owner instruction before pushing the branch or opening a pull request.
- [ ] 4.4 Obtain a separate explicit owner instruction before archiving this change and promoting the delta into `openspec/specs/local-run/spec.md`; re-read the promoted text immediately before promotion in case another change archived first.
