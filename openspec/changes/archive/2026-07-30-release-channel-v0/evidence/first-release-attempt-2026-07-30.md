# First release run — what it established, 2026-07-30

Tag `v0.1.0` was pushed at `b469386` and ran
[release workflow 30565879390](https://github.com/heurema/goalrail/actions/runs/30565879390).
It failed at its last step. Nothing was published: the releases list is empty and
no draft was created, because the failure happened before any API call that
creates one.

## What the run established

Everything except publication worked on a clean runner, and several of these
could not be observed anywhere else:

| Job or step | Result |
|---|---|
| `vet and test (ubuntu-latest)` | success |
| `vet and test (macos-latest)` | success |
| `the pinned CLI accepts the embedded canon` | success — the pin was derived from the binary's own diagnosis, the npm cache was primed with it, and the gate observed a real pass rather than a skip |
| Refuse to build from a modified tree | success |
| Build every published platform outside the work tree | success |
| Require every artifact to carry this tag | success — all four artifacts stamped `v0.1.0`, read without executing any of them |
| Assemble the archives and the checksums file | success |
| Publish the release | **failure** |

The canon gate is the one that had never run outside this machine. It works.

## What failed, and why

```
failed to run git: fatal: not a git repository (or any of the parent directories): .git
```

`gh` resolves which repository it is acting on from the git remote of the working
directory. The publish step did `cd "$RUNNER_TEMP/dist"` — the artifact
directory, which is deliberately outside the checkout — so `gh release create`
had a valid token and no repository to use it against.

`GH_TOKEN` was set and correct. The missing piece was `GH_REPO`. The design
recorded the credential and missed the addressing.

## The defect the failure exposed

The step's own state detection read:

```sh
state="$(gh release view "$TAG" --json isDraft --jq .isDraft 2>/dev/null || echo none)"
```

That turns *every* failure into "no release exists" — including this one. The
read that exists to protect an already published release from being deleted
would have reported "nothing is there" for a reason that has nothing to do with
releases. It is the same shape as the skip-into-green failure this change closed
in the canon check, in the code written to close it.

The repair distinguishes the cases: a successful read is used, a "release not
found" error means there is nothing to clean up, and any other failure refuses to
publish blind.

## Verified locally after the repair

Against the real repository, with `gh release create` and `gh release delete`
stubbed so nothing was published:

- `GH_REPO` set, no release for the tag: the step proceeds and hands four
  archives and `checksums.txt` to the create call.
- `GH_REPO` unset, run outside a checkout — the exact failing condition: the step
  now fails with "could not read the release state … refusing to publish blind"
  and prints the underlying git error, instead of reporting "none".
- One artifact missing: the step refuses before touching the release, listing
  what it found.

## Why the tag is not moved

`proxy.golang.org` already serves `v0.1.0` for this module, resolved to
`b469386`. Deleting the tag and re-creating it on a different commit would leave
the proxy and the checksum database pinned to content that no longer exists at
that version — a failure for anyone running `go install …@v0.1.0`, and exactly
the kind of quiet breakage this product exists to remove.

So `v0.1.0` stands as a valid module version with no binary assets, and the
repaired workflow publishes `v0.1.1`. That is the same recovery the design
already prescribes for a defective release: supersede it, never repair it in
place.
