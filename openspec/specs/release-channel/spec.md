# release-channel Specification

## Purpose

Define how Goalrail is verified, versioned, and distributed: what every change
must pass before it can become a release, what a tag publishes and what makes
that set complete and verifiable, how a binary's version is established from the
build rather than maintained by hand and checked against the tag it claims, and
what the documented install route must carry for a user or an agent with no Go
toolchain.
## Requirements
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
**Intent IDs:** OUT-2, OUT-3, OUT-4, SIG-2, SIG-3, SIG-4

The repository bootstrap prompt and release documentation offered to a supported agent SHALL work on a machine with no Go toolchain and no existing Goalrail binary. Before any installation, the prompt SHALL tell the agent to resolve the managed project's declared setup profile, inspect the machine read-only, resolve exact compatible Goalrail and other mandatory component versions and integrity identities, and present one digest-bound setup plan with destinations, configuration or hook changes, network access, prerequisites, rollback, and zero project code writes.

The prompt SHALL require explicit owner authorization for that exact plan before downloading an executable into durable storage or changing configuration. It SHALL explain how to discover a compatible release without assuming an unpinned version, verify a partial platform download, place the binary at the enumerated durable user-local location, preserve temporary-download cleanup, attach the selected scaffold without forging trust, run diagnosis, emit a setup receipt, and return to the initiating request at the intent gate. An unlisted runtime, dependency, login, credential action, global setting, privilege escalation, or external mutation SHALL require a new plan and permission.

The prompt MUST NOT contain an executable curl-to-shell installer, require the agent to trust mutable unverified bytes, forbid the durable outside-repository placement it needs, initialize an already declared clone again, enable branch protection, or convert general feature consent into setup authority. Platform limitations and macOS quarantine behavior SHALL remain truthful and consistent with the documented command-line and browser routes.

#### Scenario: Agent plans setup on a clean managed clone
- **WHEN** a supported agent receives a feature request with no Goalrail or Go toolchain installed
- **THEN** it resolves and displays the exact integrity-verifiable setup plan and performs zero installation and project code writes before owner authorization

#### Scenario: Owner authorizes the exact plan
- **WHEN** the owner explicitly authorizes the current plan digest
- **THEN** the agent installs only its enumerated components, places executables durably, performs only listed attachment changes, verifies diagnosis, emits the setup receipt, and returns to candidate intent

#### Scenario: Owner declines the plan
- **WHEN** the owner refuses or does not authorize setup
- **THEN** the agent leaves machine and repository state unchanged and does not continue the feature implementation

#### Scenario: Download location is cleaned afterwards
- **WHEN** temporary download content is removed after successful setup
- **THEN** the registered executable and required runtime still resolve from their planned durable locations

#### Scenario: New dependency is discovered
- **WHEN** execution finds a mandatory component or enabling action absent from the authorized plan
- **THEN** it stops before that action and requests separate exact authorization

#### Scenario: Provider trust is required
- **WHEN** the scaffold requires its own interactive trust action
- **THEN** the prompt directs the user through the provider surface and forbids the agent or Goalrail from manufacturing the trust record

### Requirement: The channel may be asked one bounded question, from one place
**Intent IDs:** OUT-3, OUT-5, SIG-3, SIG-5, SIG-6

Goalrail MAY ask whether a newer release exists. Asking SHALL be one request with
a short timeout and no retry, and its answer SHALL be cached so that repeated
diagnoses within the caching interval ask nothing.

The transport SHALL be reachable from the diagnosis alone. Initialization, the
update command, the session hooks, and the escalation loop MUST NOT acquire the
ability to reach the network, and every one of them MUST complete with the
network unavailable. This SHALL be enforced by a check that fails when the
transport becomes reachable from anywhere else, because a rule about where code
may be called from does not survive on discipline.

Every failure mode SHALL be indistinguishable to the user in kind: unreachable,
rejected by the source, slow, malformed, empty, or an answer naming a version
that is not a release. Each produces the same statement that the check did not run. The
source's own error text MUST NOT reach the user, and a transport failure MUST NOT
fail the diagnosis.

No credential, token, or account SHALL be required to ask, and asking MUST NOT
consume a quota that the machine shares with unrelated tools.

#### Scenario: The same diagnosis runs twice
- **WHEN** a second diagnosis runs within the caching interval
- **THEN** no request is made and the cached answer is used

#### Scenario: The cache is stale
- **WHEN** a diagnosis runs after the caching interval has passed
- **THEN** one request is made and the answer replaces the cached one

#### Scenario: The network is unavailable
- **WHEN** the transport cannot reach the source, or is slow past the timeout
- **THEN** the diagnosis reports that the check did not run, completes normally, and its exit code is unchanged

#### Scenario: The source answers something unusable
- **WHEN** the answer is malformed, empty, or names a version that is not a release
- **THEN** the diagnosis reports that the check did not run and surfaces no text from the source

#### Scenario: Another command would reach the network
- **WHEN** initialization, the update command, a session hook, or the escalation loop could reach the transport
- **THEN** that violates this requirement

#### Scenario: Asking would need an account
- **WHEN** asking requires a credential, a token, or a quota shared with unrelated tools on the machine
- **THEN** that violates this requirement

### Requirement: Asking honours refusals the user has already expressed
**Intent IDs:** OUT-4, NG-4, SIG-4

Before Goalrail asks, it SHALL respect a refusal the user has already given to
the toolchain: where their environment directs module lookups away from the
public proxy, or excludes this module from it, Goalrail SHALL NOT make the
request. Those settings govern exactly this traffic, and a program making the
same request by hand would otherwise override an instruction the user believes
they have given.

A dedicated switch SHALL disable the check on its own, for users who want the
toolchain setting untouched. The check SHALL NOT run in continuous integration.

The request SHALL disclose nothing that identifies the tool, its version, its
platform, the repository, or the invocation. It SHALL carry no query parameter,
and its shape SHALL be the standard one the toolchain itself produces, so it is
indistinguishable from traffic the machine already sends.

Where the check is declined, the diagnosis SHALL say it was declined rather than
that it failed.

#### Scenario: The user has directed module lookups away from the proxy
- **WHEN** the environment directs module lookups elsewhere or excludes this module
- **THEN** no request is made and the diagnosis reports the check as declined

#### Scenario: The dedicated switch is set
- **WHEN** the switch that disables the check is set
- **THEN** no request is made and the diagnosis reports the check as declined

#### Scenario: The diagnosis runs in continuous integration
- **WHEN** the diagnosis runs in a continuous integration environment
- **THEN** no request is made and the diagnosis reports the check as declined, in the same form as any other decline — distinguishable from a failure, not from other declines, each of which names its reason

#### Scenario: The request would identify the installation
- **WHEN** the request carries an agent string naming the tool, its version or platform, or any query parameter
- **THEN** that violates this requirement

### Requirement: A published release makes itself discoverable promptly
**Intent IDs:** OUT-6, SIG-7

Because the source discovers a new release late when nobody asks it — measured on
this repository at minutes to a quarter of an hour, with no bound documented for
that path — the release SHALL request its own new version from the source after
publishing. The documented effect is that the version becomes available within
about a minute, which bounds the window in which the check is confidently wrong.

The request SHALL be made only after the release is published, and its failure
MUST NOT fail the release: the artifacts are already public, and a slow source is
not a defective release.

#### Scenario: A release is published
- **WHEN** the release finishes publishing its artifacts
- **THEN** it requests its own new version from the source, so the check stops reporting the previous release as newest within about a minute

#### Scenario: The source does not answer
- **WHEN** that request fails or times out
- **THEN** the release still succeeds, because the artifacts are already published

#### Scenario: The request would precede publication
- **WHEN** the new version is requested from the source before the release is published
- **THEN** that violates this requirement

### Requirement: Published metadata supports exact pre-install planning
**Intent IDs:** OUT-3, OUT-4, SIG-3, SIG-4

Every published Goalrail release SHALL provide bounded machine-readable metadata sufficient to resolve an exact pre-install plan before placing the binary. The metadata SHALL identify its schema, release version, supported platforms, archive names, sizes, cryptographic digests, binary version identity, minimum compatibility facts, and the checksum artifact that independently covers the same archives. It SHALL be immutable for one release and discoverable through the documented bounded release channel.

The release process SHALL verify that metadata, checksums, archives, and embedded binary versions agree before publication. A mutable discovery response MAY point to an immutable release, but MUST NOT itself become the integrity identity. No planning metadata SHALL require an account, credential, or device authorization to read.

#### Scenario: Agent resolves one platform before installation
- **WHEN** an agent reads current discovery metadata and selects a supported platform archive
- **THEN** it can pin the immutable release, archive name, size, and digest in the setup plan without downloading or installing the executable

#### Scenario: Release artifacts disagree
- **WHEN** metadata, checksum file, archive bytes, or embedded binary version identify different releases or digests
- **THEN** release verification fails and the inconsistent set cannot be published as a usable setup bundle

#### Scenario: Discovery response changes later
- **WHEN** the mutable current-release pointer advances after a plan was authorized
- **THEN** the authorized plan remains bound to its immutable release and digest, and execution does not silently upgrade it

#### Scenario: Metadata requires authentication
- **WHEN** the documented planning metadata cannot be read without an account or credential
- **THEN** that violates the clean-machine setup contract

### Requirement: A release refuses a dependency closure nobody recorded
**Intent IDs:** OUT-1, OUT-2, OUT-4, SIG-1, SIG-2, SIG-4

The pinned runtime and compiler SHALL be accompanied by a record of the
dependency closure they resolve to: how many packages it holds, how many of
those carry an install script, and its digest. Loading the release inputs SHALL
compare the computed closure against that record and SHALL refuse when they
disagree, naming which of the three facts did.

The comparison SHALL sit where the inputs are loaded rather than in one command.
Building a release, verifying one, and inspecting the pins all read the same
inputs, so a check held by one of them leaves the other two able to produce a
release against a set nobody looked at.

Recording the counts beside the digest is not redundancy. A digest tells a
reviewer that something changed; the counts tell them what, and a package
acquiring an install script is the single fact most worth seeing move — a
compromise arrives through the maintainer's own publishing pipeline with valid
provenance, so what distinguishes it is the shape of the set, not its
attestations.

The expanded document SHALL carry its own schema identifier and SHALL be
registered as such, because its fields are mandatory and its decoding is strict:
one identifier cannot describe both shapes when each rejects the other's
documents.

#### Scenario: The closure matches its record
- **WHEN** the computed closure agrees with the record beside the pins
- **THEN** the release inputs load and the pins are reported

#### Scenario: A package appears or disappears
- **WHEN** the computed closure holds a different number of packages than the record
- **THEN** loading refuses and names the package count as what disagreed

#### Scenario: A package acquires an install script
- **WHEN** the computed closure holds a different number of packages carrying an install script than the record
- **THEN** loading refuses and names the install-script count as what disagreed

#### Scenario: The closure digest moves
- **WHEN** the computed closure digest differs from the record
- **THEN** loading refuses and names the digest as what disagreed

#### Scenario: A release is built without inspecting the pins first
- **WHEN** a release is built or verified after the dependency lock changed and the record was not updated
- **THEN** it refuses on the same comparison, because the check is not held by the inspecting command alone

#### Scenario: The expanded document claims the previous identifier
- **WHEN** a source lock carrying the closure record and adoption dates claims the identifier of the shape that carried neither
- **THEN** it is refused

### Requirement: A pin discloses when it was published and when it was adopted
**Intent IDs:** OUT-3, SIG-3, SIG-5

Each pinned component SHALL record when its version was published upstream and
when this repository adopted it, so the age at adoption is visible to whoever
reviews a pin move. A version adopted days after publication is the exposure a
quarantine practice exists to avoid, and this is the evidence that makes it
visible.

These dates SHALL be disclosure rather than a gate, and SHALL NOT be described
as one. Both are supplied by whoever moves the pin, so no waiting period can be
enforced on itself; the single inconsistency the record establishes alone is a
pin claiming to have been adopted before it was published. How long a version
should have been public before adoption is not settled by this requirement.

#### Scenario: Both dates are present
- **WHEN** a pinned component records its upstream publication and its adoption here
- **THEN** the record is accepted and the age at adoption is readable from it

#### Scenario: A date is absent
- **WHEN** either date is missing for a pinned component
- **THEN** the record is refused

#### Scenario: Adoption precedes publication
- **WHEN** a pin records an adoption earlier than its own publication
- **THEN** the record is refused, because that is the one inconsistency it can establish without an outside source

#### Scenario: The dates would be read as a waiting period
- **WHEN** the record is described as enforcing a minimum age before adoption
- **THEN** that violates this requirement, because the dates come from the same act they would gate

