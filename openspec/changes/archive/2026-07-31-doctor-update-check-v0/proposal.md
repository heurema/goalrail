## Why

The release channel made a question askable that nothing can answer: whether the
`gr` a user is running is a release behind. For anyone who installed by
downloading a binary, the diagnosis is their only surface where that could
appear, and today it says nothing.

Two archived changes deferred exactly this and named the condition: `harness-init-v0`
NG-1 — "Revisit when a release channel exists" — and `release-channel-v0` NG-1,
which recorded that asking the channel a question is the separate change the
condition now unblocks. This is that change.

It also ends something deliberate, which is why the price is stated rather than
discovered: `net/http` is absent from the shipped binary's dependency graph, so
`gr` currently cannot open a connection at all. Adding it doubles the binary —
4.97 MB becomes 9.87 MB, and the archive a user downloads goes from 2.76 MB to
5.42 MB, both measured on the implemented change. The planning estimate said
+40%, having measured an imported transport rather than a used one; the corrected
figure was put to the owner before anything was committed and the decision was
taken again on it. Confirmed intent accepts that price for one line in one
command, and spends it under gates that keep the network from spreading anywhere
else.

## What Changes

- The diagnosis reports whether a newer Goalrail has been released, in the voice
  it already uses for the toolchain and observability: a fact, with no next
  action, no effect on whether the harness is working, and no effect on the exit
  code. `gr` has no self-update command, so a line prescribing one would violate
  the promoted rule against next actions naming commands the tool does not have.
- The line names the service it asked. Transparency lives where the user is,
  not only in documentation.
- The check never claims what it cannot know. A binary whose own version is not a
  release tag — a checkout build, a modified tree, a build carrying no version —
  produces no comparison and says so. No wording asserts that the user is up to
  date, because the source is measurably able to be a quarter of an hour behind.
- Asking is one request with a short timeout, no retry, and a result cached for
  about a day in the state root that already exists. Every failure — offline,
  blocked, slow, malformed, unparseable — degrades to one honest line, never an
  error and never the source's own diagnostics echoed at the user.
- **The source is the Go module proxy.** It answers with the newest stable
  version in one small response, needs no credential, no header and no quota, and
  is already the service a `go install` user's toolchain talks to. The GitHub API
  was rejected for spending a 60-per-hour per-IP budget shared with everything
  else on the address; the plain redirect was rejected for being served from the
  tracked web surface with year-long cookies.
- **A refusal the user already expressed is honoured before a new one is
  invented.** `GOPROXY`, `GOPRIVATE` and `GONOPROXY` are read by the `go` command
  alone, so a hand-rolled request would silently override someone who has already
  said their module lookups stay home. Goalrail reads them. A dedicated switch
  turns the check off on its own, and continuous integration is never asked.
- **The request is shaped exactly as the toolchain's own** — standard library
  defaults, no agent string naming the tool, no version, no platform, no query
  parameter. At the proxy it is indistinguishable from traffic a Go developer's
  machine already produces, and the module path is its only signal.
- **The network stays where it was put.** Initialization, update, the session
  hooks and the escalation loop gain nothing, the promoted rule that the update
  command performs no release lookup survives verbatim, and reachability is
  enforced by a test rather than by discipline.
- The release publishes and then requests its own new version, collapsing the
  window in which the check is confidently wrong from an unbounded wait —
  measured at 3m45s and 14m46s on this repository's own releases — to about a
  minute.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| The diagnosis reports a newer release as a fact, with no next action and no effect on the verdict or the exit code. | OUT-1, SIG-1 | NG-1, NG-6 |
| The report line names the service it asked. | OUT-1 | NG-4 |
| A version that is not a release tag produces no comparison, and no wording claims currency. | OUT-2, SIG-2 | NG-7 |
| One bounded request, cached for about a day, with every failure degrading to one line. | OUT-3, SIG-1, SIG-3, SIG-6 | NG-7 |
| The source is the module proxy; the GitHub API and the web redirect are rejected with their costs stated. | OUT-3 | NG-3 |
| The toolchain's own refusal variables are honoured, plus a dedicated switch, and no check in continuous integration. | OUT-4, SIG-4 | NG-4 |
| The request carries no identifier distinguishing the tool, the user, the platform or the invocation. | OUT-4 | NG-4 |
| Reachability of the transport is confined to the diagnosis and enforced by a test. | OUT-5, SIG-5 | NG-2, NG-5 |
| The promoted rule that the update command performs no lookup survives unchanged, while its reason stops asserting a decision this change makes. | OUT-5 | NG-1, NG-2 |
| The release asks for its own new version after publishing. | OUT-6, SIG-7 | NG-2 |

No proposed change lies outside this table.

## Capabilities

### Modified Capabilities

- `harness-doctor`: the diagnosis gains one more thing it can report about the
  installation — whether the binary is a release behind — under the rules it
  already carries for optional facilities and for next actions.
- `release-channel`: the channel becomes something that may be asked a question,
  with the terms of asking fixed — which source, what the request discloses, what
  refusals are honoured, where the transport may be reached from, and the
  publisher's duty to make a new release promptly discoverable.
- `harness-update`: the rule that the command performs no lookup is unchanged,
  but its stated reason must stop saying the decision is open. It was open when
  the channel had no one asking it; this change answers it, in the diagnosis and
  nowhere else, and the requirement now says so.

## Impact

- `internal/harness`: a bounded update check and its cache, and one more field
  and line in the diagnosis.
- `cmd/gr`: no new command and no new flag; the check is turned off by an
  environment variable, not by the command line.
- `.github/workflows/release.yml`: one step after publication that requests the
  new version from the proxy.
- `README.md`: what the check does, what leaves the machine, and how to turn it
  off.
- The binary doubles on every published platform: 4.97 MB to 9.87 MB, and 2.76 MB
  to 5.42 MB compressed, which is what a user actually downloads. This is the
  change's main cost, it is not recoverable by implementation quality, and nothing
  cheaper exists short of writing HTTP by hand over a TLS socket.
- No new module dependency; the transport is the standard library. No change to
  the canon, the overlay, the escalation loop, or how currency is judged.

## Non-Goals

- No self-update: `gr` never downloads, replaces or repairs its own binary, and
  prints no command claiming to.
- No check outside the diagnosis — not ambient, not on initialization, not in a
  session hook.
- No credential, no token, no GitHub API, no quota shared with the machine.
- No telemetry and no usage signal; nothing in the request distinguishes one
  user, repository or invocation from another.
- No new module dependency, no change to any verdict or exit code, no blocking
  and no nagging.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, tag, release, provider run, or other external effect.
