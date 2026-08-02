# Effort experiment — 2026-08-01

The measurement the full-pass defaults rest on. Recorded because the spec cited
a result ("three defects, two P1") with no way to reproduce it, which is the
`unprovable-claim` class this repository promotes a rule against.

## Design

One repository, one branch, one range, one instructions file. **Effort was the
only variable.** Three points were run; each is a `gr review --full` invocation
differing only in `--effort`.

| effort | wall time | findings | receipt |
|---|---|---|---|
| medium (loop default) | 272 s | **0** | recorded, verdict clean |
| high | 565 s | **3** (2 P1) | recorded |
| ultra | >1200 s | none — reached the 20-minute bound | no receipt, by design |

- Repository: `heurema/baseline`, branch `extract-kata-core-api`
- Range: `ab9f4325...e93a1295` (full pass, `--full`)
- Reviewer: codex, cross mode (author was a Claude Code session)
- Instructions digest at the time: `1597f9a17f4b3f9e…` (the materialized default, unedited)

## What it establishes

The moderate default produced a **false-negative verdict**: it reported the
range clean while three real defects were present. One of the three — admission
dispatch capturing a single Kata at construction — was independently reported by
the external review on the same branch, which is what makes the finding count
verifiable rather than self-asserted.

Raising the effort without raising the bound only moves the failure: at ultra
the same range returned nothing at all within twenty minutes.

## What it does not establish

The 45-minute figure first chosen for `FullDeadline` did **not** follow from
this data; it was picked by eye. A later repeat of the same command on a
different, far smaller branch ran 500 s once and exceeded 2700 s once — the same
effort and range, an order of magnitude apart. **Duration varies widely and is
not predictable from diff size or effort**, so any bound here is a safety net
rather than a plan, and a cheap failure is worth more than a generous one.

## Reproduce

The recorded run predates the defaults it justifies, so the bound and the
instructions must be pinned explicitly — otherwise a repeat inherits the very
values this experiment produced and holds nothing constant but the label.

```sh
# The 20-minute bound is the one the ultra point expired against. Today's
# full-pass default is 25 minutes, which would not reproduce that expiry.
for effort in medium high ultra; do
  gr review --repo <path> --full --effort "$effort" --deadline 20m
done
```

Instructions must be the ones measured against, not whatever the repository
happens to carry:

```sh
shasum -a 256 <path>/.goalrail-review.md   # expect 1597f9a17f4b3f9e… (the unedited default of that day)
```

A repository whose instructions digest differs is running a different
experiment. Compare the stored receipts under the per-clone state root: each
carries its mode, reviewed range, both digests, the measured duration and the
report verbatim.
