# Independent review, round 6 (pull-request reviewer)

Not a `gr review` round: the repository's GitHub pull-request reviewer commented
on `acaae114bd` after the PR was opened. Four P2 findings, inline in
`internal/doctor/planning.go`. All four verified against the code before being
acted on.

## Findings and disposition

| Finding | Verified how | Fix | Regression |
|---|---|---|---|
| duplicate binary identities for one component are not rejected | read `validateSetupManifestShape`: identities are unique by path, not by component | exactly one match required; more is ambiguous | `TestDuplicateEntrypointsForOneComponentAreInvalid` |
| a symlinked ancestor whose target stays inside home is followed | standalone probe, `evidence/symlink-inside-home.md` | every segment lstat-ed and refused if a link | `TestSymlinkedAncestorInsideHomeIsRefused` |
| the mode check used a snapshot taken before the digest window | read the ordering in `verifyInstalledFile` | mode read from the identity the digest itself verified | none; closed by construction, see design D3c |
| every bundle-load failure collapsed to `missing` | read `observeComponent` | the loader carries its classification out: absent is missing, corrupt is invalid | `TestCorruptInstallationIsInvalidNotMissing` |

## What this round says about the previous one

The second finding is a defect in the round-5 fix, not in the original change.
That fix descended from the home directory and the design then claimed no path
segment could leave the bundle — a claim about `os.Root` semantics that was
written without being measured. It is false for a relative link whose target
stays inside the root, which the retained probe demonstrates in three lines.

This is the third time in this change that an assertion shipped ahead of its
check. Recorded rather than smoothed over.

## Bounds

Findings one, two and three require write access to `~/.local/share/goalrail`.
An attacker holding that can also replace `~/.local/bin/gr` itself, so none of
them widens the reachable surface. They are corrected because the design's
guarantees are stated more strongly than the code held them, which is the gap
this repository closes rather than prices. The fourth is report quality and has
no security dimension at all.
