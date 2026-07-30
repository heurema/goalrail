## ADDED Requirements

### Requirement: Every change is verified automatically before it can become a release
**Intent IDs:** OUT-1, SIG-1

Goalrail SHALL verify every pull request and every push to the default branch
automatically. The verification SHALL run the project's own contract — vetting
and the full test suite — on each operating system the project publishes binaries
for, and SHALL compile every platform the installation documentation claims.

The guarantee is visibility: the check SHALL be observable before a merge. It
does not block one, because no branch protection exists in this repository and
configuring it is a separate decision. A verification that reports success while
a claimed platform does not compile violates this requirement.

#### Scenario: A supported platform stops compiling
- **WHEN** a change breaks the build for a platform the installation documentation claims
- **THEN** the verification fails and names that platform, before the change is merged rather than when a release is attempted

#### Scenario: A pull request is opened
- **WHEN** a pull request is opened or updated
- **THEN** the verification runs and its result is visible on that pull request

#### Scenario: The default branch moves
- **WHEN** a commit lands on the default branch
- **THEN** the same verification runs on it

#### Scenario: A claimed platform is not compiled
- **WHEN** the documentation claims a platform that the verification does not compile
- **THEN** that violates this requirement

### Requirement: A tag publishes one complete, verifiable set of artifacts
**Intent IDs:** OUT-2, OUT-7, SIG-3

Pushing a version tag SHALL publish a release containing one archive per
supported platform and one checksums file covering every archive. Each archive
SHALL carry the `gr` binary and the project's licence, because a distributed copy
carries its licence notice.

The archive names carry the release's version, so their addresses change with
every release. The checksums file SHALL therefore keep one name that is identical
across releases, because finding the current release without already knowing its
version depends on exactly one asset whose address does not move.

The supported platforms are `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and
`linux/arm64`. Windows is deliberately absent: the escalation artifact is opened
with flags that platform lacks, and those flags are how a promoted hygiene
guarantee is implemented, so a stub that compiled would drop the guarantee
silently. The absence SHALL be stated with its reason rather than left for a user
to discover.

Publication SHALL happen only after every platform's artifact exists and every
check on the tagged commit has passed. A failure before that point SHALL leave
nothing published rather than a partial set.

#### Scenario: A tag is pushed
- **WHEN** a version tag is pushed
- **THEN** the release carries one archive per supported platform plus one checksums file, and each archive contains the binary and the licence

#### Scenario: One platform fails to build
- **WHEN** any platform's build or check fails during a release
- **THEN** nothing is published, rather than the artifacts that did succeed

#### Scenario: A downloader verifies what they fetched
- **WHEN** a user downloads one or more archives and the checksums file
- **THEN** the documented verification command confirms them, whether they fetched one archive or all of them

#### Scenario: The current release must be found without knowing its version
- **WHEN** a user or an agent knows only the repository, not the released version
- **THEN** the checksums file is reachable at an address that does not change between releases, and it names the archives of the current release

#### Scenario: The stable name would carry the version
- **WHEN** the checksums file is published under a name that changes with the release
- **THEN** that violates this requirement, because no asset is then reachable without knowing the version

### Requirement: A binary reports the version it was built from, verbatim
**Intent IDs:** OUT-3, SIG-2

The version `gr` reports SHALL be the version its build carries, reported exactly
as carried, and MUST NOT be a constant maintained by hand. A binary built at a
release tag reports that tag including its leading `v`; a binary built between
tags reports the next patch version as a pseudo-version naming the commit; a
build from a modified tree says so; a build with no version control behind it
reports the toolchain's own development marker rather than inventing a number.
Where the build carries no version string at all, the tool SHALL report the
version as unknown, rather than an empty value or an invented one.

The reported string MUST NOT be normalized, stripped, or rewritten on the way
out, and the same string SHALL appear in the tag, in the report, and inside every
archive name for that release, so nothing has to be matched loosely.

This changes what the version is, not what it decides. The rule that overlay
currency is judged by digests and that no verdict about a repository derives from
the version belongs to the diagnosis and is unaffected here.

#### Scenario: A released binary reports itself
- **WHEN** a user runs the version command on a binary from a published release
- **THEN** it reports exactly the tag that release was built from

#### Scenario: A binary is built between releases
- **WHEN** the binary is built from a checkout that is ahead of the latest tag
- **THEN** it reports a version that names the commit rather than the last release's number

#### Scenario: A second release follows a first
- **WHEN** a second release is published without anyone editing a version string
- **THEN** it reports its own tag, not the previous release's

#### Scenario: The build carries no version at all
- **WHEN** the binary is built with no version information for the toolchain to carry
- **THEN** the tool reports the version as unknown rather than an empty string or a number nothing established

### Requirement: A release refuses an artifact whose stamped version is not its tag
**Intent IDs:** OUT-2, OUT-3, SIG-2

Before publishing, the release SHALL read the version stamped into every artifact
and SHALL require exact equality with the pushed tag. The stamp SHALL be read
without executing the artifact, because most published platforms cannot run where
they are built.

The build SHALL NOT leave untracked files in the work tree it builds from. Any
untracked file — including an artifact an earlier step produced — marks the tree
as modified, and every later build then stamps a modified marker onto a binary
that claims to be a release.

#### Scenario: An artifact carries a different version
- **WHEN** any artifact's stamped version is not exactly the pushed tag
- **THEN** the release fails and publishes nothing

#### Scenario: An artifact is built for another platform
- **WHEN** the stamped version of an artifact for a platform the builder cannot execute must be checked
- **THEN** the check reads the stamp from the artifact without running it

#### Scenario: Build output lands inside the checkout
- **WHEN** a build writes an artifact into the work tree and another build follows
- **THEN** that violates this requirement, because the later artifact is stamped as built from a modified tree

### Requirement: The canon check runs before a release instead of skipping into a green result
**Intent IDs:** OUT-4, SIG-4

The development-side check that the stock CLI accepts the embedded canon skips
rather than fails when the pinned package is unavailable, so an ordinary test run
never reaches the network. A release SHALL make that package available and SHALL
fail whenever the check did not execute — whether it reported as skipped or never
ran at all — so a canon can never ship unverified.

The enforcement SHALL live in the release workflow. The harness MUST NOT gain a
switch, an environment contract, or any other new surface for it, and a
contributor running the suite without the runtime MUST still get a loud skip
rather than a failure.

#### Scenario: The check is skipped during a release
- **WHEN** the canon check reports as skipped in a release run
- **THEN** the run fails rather than reporting success

#### Scenario: The check does not run at all
- **WHEN** a release run produces no result for the canon check, because it was renamed, excluded, or never selected
- **THEN** the run fails, rather than passing on the absence of a verdict

#### Scenario: The check runs during a release
- **WHEN** the pinned package is available in a release run
- **THEN** the check executes and its result decides the run

#### Scenario: A contributor has no runtime
- **WHEN** the suite runs locally on a machine without the runtime the check needs
- **THEN** the check skips loudly and the suite passes

#### Scenario: The pin moves
- **WHEN** the pinned package version changes
- **THEN** the release makes that same version available, and a release that primes a different one fails on the check it could not run

### Requirement: The documented install route works without a Go toolchain
**Intent IDs:** OUT-5, SIG-5, SIG-6

The installation documentation SHALL describe installing a published binary
without a Go toolchain: which archive corresponds to which platform, how to
verify it against the checksums file — in a form that succeeds when only one of
the published archives was downloaded — and that Windows is unsupported together
with the reason.

Every artifact name the documentation quotes SHALL be one the release actually
produces.

The macOS note SHALL state what is true rather than a generic warning: an archive
downloaded through a browser and opened in the graphical file manager carries a
quarantine flag and meets the system's gatekeeper, while the documented
command-line download does not; the binaries are not notarized; and the remedy
given SHALL be one that still works on current macOS.

`go install` SHALL remain documented as the other route, stating that once a
release tag exists it installs that release rather than the branch tip.

#### Scenario: A user without Go installs
- **WHEN** a user with no Go toolchain follows the documentation
- **THEN** they download the archive for their platform, verify it, and run the binary

#### Scenario: Only one archive was downloaded
- **WHEN** a user verifies a single downloaded archive against the checksums file
- **THEN** the documented command succeeds rather than failing over the archives they did not download

#### Scenario: The documentation names an artifact
- **WHEN** the documentation quotes an artifact name
- **THEN** a release contains an artifact with exactly that name

#### Scenario: A user downloads through a browser
- **WHEN** a user downloads an archive with a browser and extracts it in the graphical file manager
- **THEN** the documented behaviour and the documented remedy are what they actually encounter

#### Scenario: A remedy no longer works
- **WHEN** the documented macOS remedy is one the current operating system has removed
- **THEN** that violates this requirement

### Requirement: The agent-facing prompt carries what an agent cannot guess
**Intent IDs:** OUT-7, SIG-6

The prompt the documentation offers for handing to an agent SHALL install
Goalrail on a machine with no Go toolchain. Because no install script is
provided, the prompt itself SHALL carry the steps an agent cannot derive:

- how to find the current release without knowing its version, given that only an
  asset whose name does not change between releases has a stable address;
- where to place the binary so the installation keeps working, because a
  registration records the absolute path of the executable that wrote it and the
  diagnosis fails when nothing runnable remains there — a binary left in a
  temporary download directory produces an attachment that breaks when the
  directory is cleaned;
- the verification command that succeeds on a partial download.

The prompt SHALL permit that one placement outside the repository explicitly
rather than forbidding every outside write and leaving the agent to improvise.
The documentation SHALL stop stating that no release binaries are published once
they are.

#### Scenario: An agent installs from the prompt
- **WHEN** an agent is handed the prompt on a machine with no Go toolchain
- **THEN** it discovers the current release, verifies the archive, places the binary durably, initializes the repository, and reports the diagnosis

#### Scenario: The download location is cleaned afterwards
- **WHEN** the temporary working directory an agent used is removed after installation
- **THEN** the attachment still resolves to a runnable executable and the diagnosis stays healthy

#### Scenario: The prompt forbids the placement it requires
- **WHEN** the prompt requires placing the binary outside the repository while also forbidding writes outside it
- **THEN** that violates this requirement
