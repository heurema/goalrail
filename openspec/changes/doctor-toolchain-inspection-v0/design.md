## Context

The default planning observer answers "is the declared toolchain available and
compatible" by resolving a name from the repository's setup profile through the
executable lookup path and running it with `--version` (CTX-1, CTX-2). A
promoted requirement forbids initialization, update, and diagnosis from
executing Node, a package runner, or the stock planning CLI, and states that
prohibition as a violation test (CTX-3). The same run also answers about
whichever installation the lookup path happens to reach (CTX-13) and lets
repository content choose the program (CTX-14).

Authorized setup already writes everything the question needs: a durable bundle
root, the bundle manifest inside it, and a recorded digest and version per file
and component (CTX-5, CTX-6, CTX-7). The root is derivable without a new
artifact because a release refuses to publish a binary whose stamped version is
not its tag, so a `gr` that came from a bundle reports that bundle's release
version (CTX-8); the stable executable is a byte copy and carries no link back
(CTX-9).

## Goals / Non-Goals

**Goals:**

- Establish runtime and compiler readiness by reading and hashing, never by
  running (OUT-1).
- Take every inspected path from Goalrail's own installation record rather than
  from repository content (OUT-2).
- Keep the four component states, the readiness aggregation, the reason codes,
  and the exit statuses exactly as they are (OUT-3).
- Report an installation that never ran authorized setup as setup required
  (OUT-4).

**Non-Goals:**

- Installing, repairing, or upgrading the toolchain from diagnosis (NG-1).
- A new pointer file, registry, or discovery contract (NG-2).
- Moving the execution elsewhere or behind a flag (NG-3).
- Changing the setup profile schema, its pins, or any repository-owned value
  (NG-4).
- New component states or machine-contract fields (NG-5).
- The audit's other two divergences (NG-6).
- Crediting a toolchain Goalrail did not install (NG-7).

## Decisions

**D1 — The bundle root is derived from the running binary's version.**
`<home>/.local/share/goalrail/bundles/<gr version>/<os>_<arch>/`, the exact
construction authorized setup uses (CTX-6). It needs no new artifact because the
release process guarantees the equality it rests on (CTX-8), and the alternative
routes are closed: the stable executable is a copy, not a link (CTX-9), and
enumerating sibling bundle directories would have to guess which one installed
the running binary. A binary whose version is not a release version — a
development build or a modified tree — resolves no bundle and therefore reports
the components as not ready, which is the state such a build already reaches
through the bundle-compatibility component today (CTX-11).

**D2 — Verification is by digest, against the installed manifest.**
The manifest records a version and a SHA-256 for the runtime executable and a
SHA-256 for every installed file (CTX-7). Reading it and hashing the declared
component entrypoints answers availability, version, and integrity at once, and
maps onto the states the requirement already names (CTX-4, CTX-11): absent file
is missing, manifest version outside the declared pin is incompatible, digest
mismatch is unverified integrity, agreement is ready.

**D3 — Only the two declared component entrypoints are hashed, not the tree,
and each is the one the manifest names.** The accepted limit is stated rather
than hidden: authorized setup verifies every installed file against the manifest
at install time, and diagnosis does not repeat that walk, so a sibling file
inside the bundle altered after installation is outside what this verdict
claims. Hashing a bundle that carries a full private Node on every diagnosis
would pay a cost proportional to the whole runtime for a question about two
files.

Which file is the entrypoint comes from the manifest's binary identities, for
both components. Choosing one of a component's files by any rule of the
reader's own — the first by path, say — verifies whatever sorts earliest, and
the compiler is an npm package whose licence sorts before its executable: an
independent review found exactly that, and a regression now pins it. The
release builder already emits a binary identity per component, and the bundle
contract requires each identity to bind an exact manifest file, so the reader
never has to choose.

**D3a — Every bundle file is opened through a root descended from the home
directory.** `O_NOFOLLOW` refuses a substituted symbolic link at the final
pathname component only, so a parent directory replaced by a link to an external
tree is followed before any per-file protection applies, and a component would
be verified against bytes outside the bundle. The root is opened once and every
read goes through it, so no path segment can leave it. Establishing that root by opening the bundle
path outright would not confine its ancestors — the path resolves the ordinary
way first, so a bundle directory replaced by a link becomes the root, and the
confinement then holds around the wrong tree. The root is therefore descended
from the home directory — and every segment of that descent is inspected with
lstat and refused if it is a symbolic link. Confinement alone is not enough: a
root permits a link whose target stays inside it, so a version directory
repointed at another tree under the same home is followed and a planted bundle
presented as the installed one. That was measured rather than reasoned about,
and the probe is retained in `evidence/symlink-inside-home.md`. All three parts
were independent review findings, each with a regression or a retained probe.

**D3b — A manifest that contradicts itself is invalid, not stale.** The manifest
states a component's version twice: on the component and on the entrypoint that
belongs to it. The bundle contract binds an entrypoint to a file but does not
require those two fields to agree, so an edited manifest could name a component
at one version and its entrypoint at the version a profile demands. Both are
read and required to agree; disagreement is reported as an invalid component
rather than resolved by preferring either field. A second independent review
round found this on the compiler, and the runtime path carried the same shape
without appearing in that round's diff, so the check and its regression cover
both.

**D3c — The recorded mode is verified with the bytes.** The manifest records a
mode per file, and only that says whether an entrypoint can still be run. A
runtime whose bytes are intact and whose executable bit was cleared fails to
execute while its digest still matches, and nothing here runs it to find out,
so the mode is compared too. Reported as an unverified integrity identity: the
installation no longer matches its own record.

The mode is read from the identity the digest itself verified rather than from a
separate earlier stat, because a snapshot taken before the read is already stale
when the digest exists: a change inside that window would pass the mode check
against the old metadata while the digest described the new bytes. This is
closed by construction rather than by a regression — a timing window is not
something a test can reliably reproduce, and stating that is better than
implying coverage that does not exist.

**D3d — One entrypoint per component, or none.** The bundle contract requires
binary identities to be unique by path, not by component, so a canonically valid
manifest can carry two entrypoints for one component. Taking the first match
would bind an intact alternate file while the component's real entrypoint is
damaged, and report ready. Exactly one match is required; more is the manifest
being ambiguous rather than the reader's choice to make.

**D3e — Absent and corrupt are different facts.** A bundle that was never
installed and one that is installed but damaged are not the same state, and
collapsing both into a missing component would tell a machine consumer that
something was never installed when it was installed and then broken. The loader
carries its failure classification out with its reason: absent resolves to
missing, unreadable or invalid resolves to invalid.

**D4 — The setup profile keeps declaring, and stops locating.**
It continues to name the required components and their exact versions, which is
what the comparison is against. It contributes no filesystem path, and no value
it carries reaches `exec.LookPath` or a file open. A profile value that is a
path is therefore inert rather than dangerous (CTX-14), without adding a
character-level rule that would only narrow one exploit of one input.

**D5 — Component identity is matched through the manifest, not through a path
built from the profile.** The declared compiler name is looked up among the
manifest's components, and the file paths come from the manifest's own records.
Identity lookup by declared name is not path derivation: the repository says
what is required, the installation says where it is.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| Running binary's version string | Build stamp of the executing process; trusted as its own identity | A release version equal to a bundle directory name | A pseudo-version, a modified-tree marker, the toolchain development marker, unknown, or empty — each resolves no bundle | Fixed for the process lifetime; cannot change during a run | Unit test over the resolver: each non-release form yields no bundle root and a not-ready component, never a lookup elsewhere |
| Home directory | Process environment, as the diagnosis already resolves it for attachment | An absolute existing directory | Absent or unresolvable — components report not ready rather than falling back to another root | Read once per diagnosis | Existing diagnosis input plumbing; test with a temporary home |
| Installed bundle manifest JSON | Written by authorized setup into the bundle root and verified there at install time (CTX-6); trusted only after the bundle contract's own decoder accepts it | Whatever `releasebundle.DecodeSetupBundleManifest` accepts: canonical encoding, pinned schema, safe relative paths, digest shapes, sorted unique records, and every binary identity binding an exact file | Absent, unreadable, oversized, not a regular file, non-canonical, malformed, unknown schema, a path that leaves the bundle, or naming a different release or platform — all yield not ready with a stated reason, never a partial read | Re-read on every diagnosis; a manifest replaced between reads only affects the next run, and no decision spans two reads | Table-driven test per refused state, including an edited path that escapes the bundle; one positive fixture bundle built to the full contract |
| Runtime executable bytes | Inside the bundle root, path taken from the manifest's binary identity for the declared component | A regular file whose SHA-256 and permission bits match the manifest record | Absent, non-regular, digest mismatch, mode mismatch, reachable only outside the confined root, or exceeding the existing bundle-file bound | Hashed from the same opened descriptor whose type was checked, so a substitution between check and read cannot redirect the read; opened through the confined root, so no path segment — final or parent — can leave the bundle | Test that a modified byte and a cleared executable bit each yield unverified integrity, and that a symlinked parent directory and a symlinked bundle root are refused rather than followed |
| Compiler entrypoint bytes | Same bundle root, path taken from the manifest's binary identity for the declared component | As above | As above | As above | As above, plus a test that replacing the entrypoint is caught while an earlier-sorting file of the same package stays intact |
| Declared setup profile | Repository content already read by the diagnosis; trusted for declaration, never for location | Component identifiers and exact versions | A value used as a path, a lookup name, or any filesystem input — refused by construction, since no code path passes it to a resolver or an open | Repository content can change between runs; each run reads it afresh | Test that a profile whose runtime value is a path to an executable inside the worktree causes no execution and no read of that path |
| Platform of the running process | Compile-time `GOOS`/`GOARCH` of the executing binary | The `<os>_<arch>` key authorized setup used | No refusal case: a bundle installed for another platform simply does not exist at the derived path | Fixed at build time | Covered by the resolver test's path construction assertion |

## Correlation and Evidence

No new evidence is produced. The diagnosis keeps emitting the same
`ComponentReadiness` values under the same kinds and identifiers, so the machine
contract, the reason codes, `Planning.Ready`, `LocallyReady`, and the exit
statuses are unchanged (OUT-3, SIG-5). The observed identity a component reports
becomes the version the installed manifest records rather than a string parsed
from a program's output; the field and its name are unchanged.

## Measurement and Stop Conditions

- Decoy executables named as the declared runtime and adapter, placed first on
  the lookup path, record zero invocations across a diagnosis; the same
  measurement records two today (SIG-1).
- A diagnosis completes normally on a machine with no runtime present at all
  (SIG-2).
- A profile whose runtime value is a path inside the worktree causes no
  execution of that path (SIG-3).
- A complete installed bundle yields planning readiness true (SIG-4).
- The regression fails against the pre-change observer and passes after (SIG-6).

Stop and reshape if reaching a ready verdict requires reading anything the
authorized setup does not already write, or if preserving the existing component
states requires adding a state.

## Risks / Trade-offs

- **A previously green installation turns amber.** An installation that never
  ran authorized setup but carries a satisfying system toolchain moves from
  ready to setup required. This is the confirmed intent (OUT-4, NG-7) and it is
  the honest reading of a requirement about the pinned profile, but it is the
  one visible regression and it lands on users who installed through the
  documented `go install` route (CTX-15).
- **Hashing cost per diagnosis.** Two files, one of them a private Node binary.
  Accepted in preference to a size-and-timestamp fast path, which would answer a
  question about integrity with evidence that is not integrity.
- **Narrower verification surface than install time.** D3 states the limit
  rather than implying whole-bundle assurance.
- **The derivation in D1 rests on a release-process guarantee.** If a future
  release published a binary whose stamp differs from its tag, the bundle would
  not resolve. That guarantee is itself a promoted requirement with its own
  scenario, so the dependency is on a checked property rather than on a habit.

## Rollback

The change is confined to the default planning observer behind the existing
`PlanningObserver` interface, which the diagnosis already accepts as an
injectable field. Reverting the observer's body restores the previous behaviour
without touching the report shape, the aggregation, or any caller.

## Open Questions

None.
