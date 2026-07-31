## ADDED Requirements

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
