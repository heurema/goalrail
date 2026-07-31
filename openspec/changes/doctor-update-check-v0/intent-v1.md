# Intent Snapshot

- **Intent ID:** intent-doctor-update-check-v0
- **Version:** 1
- **Status:** candidate
- **Owner:** repository owner
- **Context Pack:** context-doctor-update-check-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 (owner):** The release channel exists; the next thing to build is the diagnosis answering whether an update is available.
- **SE-2 (repository):** Two archived changes deferred exactly this and named the condition that has now been met. `harness-init-v0` NG-1: "No network release check and no self-update … Revisit when a release channel exists." `release-channel-v0` NG-1: "asking it a question is the separate change the revisit condition now unblocks" (CTX-2).
- **SE-3 (repository):** No promoted requirement forbids the network in `gr`; the only one that mentions it is scoped to the update command, which must stay inert (CTX-1).
- **SE-4 (repository, verified):** The silence is structural, not behavioural: `net/http` is absent from the shipped binary's dependency graph, so `gr` currently cannot open a connection at all (CTX-3).
- **SE-5 (repository, measured):** Ending that costs 2,998,400 bytes — the binary grows 40.2%, from 4.97 MB to 6.97 MB, on every platform every user downloads. A hand-rolled TLS path would still cost 20% and would mean writing HTTP by hand (CTX-4).
- **SE-6 (repository):** The diagnosis forbids inventing a next action for an act the tool cannot perform, and requires an absent optional facility to be stated without a next action and without contributing a failure (CTX-5, CTX-6).
- **SE-7 (repository):** `gr` has no self-update command and is not gaining one here, so "a newer release exists" has no command to prescribe.
- **SE-8 (repository, verified):** The reported version is often not a release tag — a pseudo-version between tags, a `+dirty` marker, the toolchain's development marker, or `unknown` — and the promoted contract forbids rewriting it on the way out (CTX-7).
- **SE-9 (external, verified):** `proxy.golang.org/<module>/@latest` answers the question in one small request: newest stable version, prereleases skipped, no authentication, no User-Agent requirement, no observed rate limit, ~0.27s warm (CTX-8).
- **SE-10 (external, verified):** That proxy knows module source only and can never hand over a release binary (CTX-9).
- **SE-11 (external, measured):** It also learns about a new tag late when nobody asks — 3m45s and 14m46s measured here, with no documented bound — and the documented remedy is to request the version explicitly, after which it appears within about a minute (CTX-10).
- **SE-12 (external, verified):** `GOPROXY`, `GOPRIVATE` and `GONOPROXY` are honoured by the `go` command alone. A program making its own request ignores them, so a user who set `GOPROXY=off` would be overridden silently (CTX-11).
- **SE-13 (external, verified):** The GitHub alternatives each cost something: the REST API spends a 60-per-hour per-IP quota that other tools on the same address have already partly spent, cannot be cached out of that quota, and demands a self-identifying header; the plain redirect is quota-free and faster but is served from the tracked web surface and sets year-long cookies (CTX-12, CTX-13).
- **SE-14 (external):** Six practices are settled across every CLI that checks at all: a roughly daily cache, one environment variable to disable it, the notice on standard error, silence on failure, never blocking the command, and no check in continuous integration (CTX-14).
- **SE-15 (external):** What is contested is whether to check at all. `gh` compiles its notifier out of the binaries it ships; `kubectl` never contacts an upstream feed; `hyperfine` carries no HTTP client. `rustup` offers an explicit `rustup check` with a distinct exit status for "update available". No project examined documents a privacy rationale for the check itself (CTX-15).
- **SE-16 (external, verified):** The proxy's failure shapes defeat naive handling: a tagless module answers `200` with a pseudo-version, an empty version list answers `200` with an empty body, and error bodies leak the proxy's own internals (CTX-16).

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | The diagnosis answers the question the release channel made askable: whether a newer Goalrail has been released. It answers it as a fact, in the same voice as the toolchain and observability lines — one line, no next action, no effect on whether the harness is reported as working and none on the exit code. The version still decides nothing about the repository. | Run the diagnosis on a machine that is behind a release and on one that is current, and read both reports and both exit codes. | SE-1, SE-2, SE-6, SE-7, CTX-5, CTX-6 |
| OUT-2 | The check never claims what it cannot know. A binary whose own version is not a released tag — built from a checkout, from a modified tree, or carrying no version at all — produces no comparison and says so rather than guessing. The wording never asserts that the user is up to date, because the source can be a quarter of an hour behind the truth; it states what it found and when it looked. | Run the diagnosis from a development build, from a dirty tree, and from a build with no version information, and confirm none of them claims an update or claims currency. | SE-8, SE-11, CTX-7, CTX-10 |
| OUT-3 | Asking is bounded and quiet. One request, a short timeout, no retry, and a result cached for about a day in the state root that already exists. Failure of any kind — offline, blocked, slow, malformed, an answer the tool cannot parse — produces one honest line saying the check did not run, never an error, never a stack trace, and never the source's own diagnostics echoed at the user. | Run the diagnosis with no network, with a blackholed host, and against a source returning nonsense; confirm the report degrades to one line each time and the exit code is unchanged; run it twice and confirm the second run makes no request. | SE-9, SE-14, SE-16, CTX-8, CTX-14, CTX-16 |
| OUT-4 | The user's existing refusals are honoured before any new one is invented. A user who has already told the Go toolchain not to reach the module proxy — through the environment variables that govern exactly this — is not overridden by Goalrail making the same request by hand. A dedicated switch turns the check off on its own, and continuous integration is never asked. | Set each refusal in turn and confirm no request leaves the machine, by observing the report and by a test that fails if the transport is invoked. | SE-12, SE-14, CTX-11, CTX-14 |
| OUT-5 | The network stays where it was put. Nothing else in `gr` gains the ability to reach it: not initialization, not the update command, not the session hooks, not the escalation loop. The promoted rule that the update command performs no release lookup survives exactly as written, and the reachability is enforced by a test rather than by discipline. | Read the promoted requirement unchanged; run a test asserting that the transport is reachable only from the diagnosis path, and that every other command completes with the network unavailable. | SE-3, SE-5, CTX-1, CTX-3 |
| OUT-6 | The answer is as fresh as it can cheaply be made. Because the source discovers a new release late when nobody asks, the release publishes and then asks for its own new version, which the source documents as making it available within about a minute instead of an unbounded wait. | Publish the next release and measure the interval between the tag and the source answering with the new version. | SE-11, CTX-10 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No self-update. `gr` never downloads, replaces, or repairs its own binary, and prints no command that claims to. The update command keeps its promoted meaning — the repository's harness, never the binary — and keeps performing no lookup of its own. | SE-3, SE-7, CTX-1 |
| NG-2 | No check outside the diagnosis. Nothing is checked ambiently, on initialization, inside a session hook, or as a side effect of any other command. | SE-3, SE-14 |
| NG-3 | No credential, no token, and no GitHub API. Nothing that requires a quota shared with the rest of the machine, and nothing that needs a header identifying the caller in order to be answered at all. | SE-13, CTX-12 |
| NG-4 | No telemetry and no usage signal. The request carries nothing that distinguishes one user, one repository, or one invocation from another beyond what any HTTPS request unavoidably discloses, and no query parameter is added that could. | SE-15, CTX-13 |
| NG-5 | No new module dependency. The transport is the standard library; the module stays dependency-free. | SE-5 |
| NG-6 | The check changes no verdict and no exit code. A repository whose harness is intact reports as working and exits zero whether or not a newer release exists. | SE-6, CTX-6 |
| NG-7 | No blocking and no nagging. The check never delays the diagnosis past its timeout, never repeats within its interval, and never escalates its wording when ignored. | SE-14, CTX-14 |
| NG-8 | This snapshot authorizes no implementation, commit, push, pull request, merge, tag, release, provider run, or other external effect. Each keeps its own owner gate, recorded before the action. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Offline is a normal state, not a failure. | A test runs the diagnosis with the transport unavailable and asserts the report differs from a healthy one by exactly one line, the harness is still reported as working, and the exit code is unchanged. | SE-14, CTX-6 |
| SIG-2 | A build that is not a release never claims to be behind one. | A test drives the comparison with each version shape the toolchain produces — release tag, pseudo-version, `+dirty`, development marker, `unknown` — and asserts only the release tag yields a comparison. | SE-8, CTX-7 |
| SIG-3 | Asking twice asks once. | A test primes the cache and asserts the second diagnosis makes no request; a test ages the cache past its interval and asserts the next one does. | SE-14, CTX-14 |
| SIG-4 | A refusal already expressed is honoured. | A test sets each governing environment variable in turn and asserts the transport is never invoked and the report says the check was declined rather than failed. | SE-12, CTX-11 |
| SIG-5 | The network is reachable from one place only. | A test asserts the transport is invoked from the diagnosis path and from nowhere else, and that initialization, update, and the hook entry point complete with it unavailable. | SE-3, SE-4, CTX-3 |
| SIG-6 | Nothing the source says reaches the user unfiltered. | A test feeds the parser a proxy error body, an empty version list, a pseudo-version, and malformed output, and asserts each produces the same one honest line and no source text. | SE-16, CTX-16 |
| SIG-7 | The freshness window is closed to about a minute. | The next release is measured from tag to the source answering with the new version, and the result is recorded. | SE-11, CTX-10 |

## Ambiguities and Unknowns

| ID | Question | Evidence |
|---|---|---|
| AMB-1 | Whether to do this at all, given the measured price. The check ends a structural property — `gr` cannot currently open a connection — and costs 40.2% of the binary, 2 MB on every download for every user, to answer a question the releases page also answers. Proposed: do it, because the diagnosis exists to tell the user what is true about their installation and "your Goalrail is a release behind" is part of that, and because 7 MB remains small for a downloaded tool. The alternative is real and cheap to take: record the decision not to, which answers the revisit condition with "asked and declined" rather than leaving it open. Resolved by owner confirmation of this version. | SE-4, SE-5, SE-15, CTX-3, CTX-4, CTX-15 |
| AMB-2 | Which source answers. Proposed: the Go module proxy — no quota, no credential, no cookie, no header requirement, one small answer, and it is already the service a `go install` user's toolchain talks to. Its costs are stated rather than hidden: it can be about a quarter of an hour stale, which OUT-6 mitigates, and it knows nothing about release binaries, which this slice does not need. Rejected: the GitHub API, whose 60-per-hour per-IP budget is shared with every other tool on the address; and the plain redirect, which is quota-free but served from the tracked web surface with year-long cookies. Resolved by owner confirmation of this version. | SE-9, SE-10, SE-13, CTX-8, CTX-12, CTX-13 |
| AMB-3 | Whether the diagnosis checks by default or only when asked. Proposed: by default, because running the diagnosis is already the user's deliberate act rather than an ambient hook, with the settled gates — a daily cache, a switch to turn it off, and no check in continuous integration. The usual sixth gate does not transfer: tools skip when the output is not a terminal, but the agent-facing prompt runs the diagnosis non-interactively and shows its output to the user, which is exactly where the line should appear. Resolved by owner confirmation of this version. | SE-14, SE-15, CTX-14 |
| AMB-4 | Whether "an update is available" also gets a distinct exit status, as `rustup check` does with 100. Proposed: no. The diagnosis has a three-value contract that means healthy, has a problem, or did not run, and a fourth value would make every existing unattended caller wrong about a repository that is fine. The information stays in the report and in the structured output, where a script can read it without the exit code changing meaning. Resolved by owner confirmation of this version. | SE-6, SE-15, CTX-6, CTX-15 |
| AMB-5 | What the request discloses. Any HTTPS request reveals the caller's address; the question is what else. Proposed: send a fixed identifier naming the tool and nothing else — no version, no platform, no repository, no query parameter — so the source cannot distinguish one user's installations from another's, and record in the documentation what leaves the machine. Resolved by owner confirmation of this version. | SE-13, CTX-13, CTX-8 |

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending. This version is candidate. It becomes confirmed only when the owner, having read a plain-language view of these outcomes, boundaries, and signals in the resolved owner language, actively confirms version 1 — including the five recorded ambiguities, whose proposed answers are part of what is being confirmed. A different answer to any of them is a material change and creates version 2; answering AMB-1 with "do not do this" ends the change here and is recorded as the decision rather than as an abandonment.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
