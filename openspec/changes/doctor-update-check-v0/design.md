## Context

`gr` cannot currently open a connection: `net/http` is absent from the shipped
binary's dependency graph, and the only network-named package it carries is
`net/url`, used to strip userinfo from a printed endpoint. The property is
structural, and confirmed intent ends it deliberately, for one line in one
command, at a measured cost of double the binary: 4.97 MB becomes 9.87 MB, and
the archive a user downloads 2.76 MB becomes 5.42 MB.

The diagnosis it lands in already has the shape this feature needs: optional
facilities are reported as facts, problems are index-paired with next actions,
and the exit contract is three-valued — healthy, ran and found problems, did not
run. The version the binary reports is frequently not a release tag, because it
comes from build information verbatim.

## Goals / Non-Goals

**Goals:**

- One line in the diagnosis, honest about what it knows and when it learned it.
- A network path that cannot spread, enforced mechanically.
- Refusals the user has already expressed, honoured before a new one is invented.
- A request indistinguishable from the toolchain's own.

**Non-Goals:**

- Self-update, a download, or any command that claims to replace the binary.
- Any check outside the diagnosis; any ambient or hook-time network access.
- A credential, a token, a shared quota, or an identifying header.
- A change to any verdict, exit code, or promoted requirement.

## Decisions

### D1 — The module proxy, not GitHub

`https://proxy.golang.org/github.com/heurema/goalrail/@latest` answers with the
newest **stable** version in one small JSON object — prereleases already filtered,
so no client-side version sorting exists to get wrong. It needs no credential, no
User-Agent, and showed no rate limit across 40 rapid requests; warm latency was
~0.27s.

*Rejected — the GitHub REST API:* 60 requests per hour per IP unauthenticated,
shared with every other tool on the address and observed already partly spent
before `gr` would ask; a conditional request answered `304` still consumes a unit,
so caching cannot buy the quota back; an empty User-Agent is refused outright; and
its `404` cannot distinguish a repository with no releases from one that does not
exist. *Rejected — the `releases/latest` redirect:* quota-free and faster, but
served from GitHub's tracked web surface, setting `_octo` and `logged_in` with a
one-year expiry. The cheaper GitHub path is the one that touches analytics.

The proxy's two costs are accepted rather than hidden: it knows module source
only and can never hand over a binary (which this slice does not need), and it can
be minutes to a quarter of an hour stale, which D7 mitigates and OUT-2's wording
refuses to paper over.

### D2 — The version comparison is a whitelist, not a parser

The binary's own version is compared only when it is a plain release tag. Every
other shape the toolchain produces — a pseudo-version between tags, any `+dirty`
suffix, `(devel)`, and `unknown` — short-circuits to "does not apply" before any
comparison happens. The proxy's answer is subjected to the same test: a tagless
module answers `200` with a pseudo-version, so an answer that is not a release
version is treated as no answer.

The eligible shape is exactly `v<major>.<minor>.<patch>`, three decimal integers
and nothing after the patch field. Anything else on **either** side — a
prerelease suffix, build metadata, `+incompatible`, a pseudo-version, `(devel)`,
`unknown` — leaves the whitelist and short-circuits. The local side needs the rule
as much as the proxy's answer does: the release workflow triggers on `v*`, so a
prerelease tag is publishable, and a binary built from one reports it verbatim
under the promoted rule that the version is never rewritten on the way out.

The ordering is a hand-written field-wise comparison of those three integers,
living in the transport package. This is stated because it is not free: the
standard library orders nothing of this shape, `golang.org/x/mod/semver` would be
a dependency that NG-5 forbids, and the two obvious improvisations are both
wrong — string comparison makes `v0.10.0` older than `v0.9.0`, and any
lexicographic ordering breaks at the first two-digit field.

This is deliberately narrower than semantic-version comparison. The question is
"is the released version different from, and newer than, mine", and the only
inputs where that is meaningful are two plain release tags.

### D3 — Confinement is a seam plus two assertions, because the import graph cannot express it

The naive mechanism does not work, and the reason is worth recording so nobody
reaches for it again: Go's import graph is package-granular, while `Diagnose` and
`Update` live in the same package — as do `runDoctor`, `runInit`, `runUpdate` and
`runHook` in `cmd/gr`. An import assertion would pass while the requirement was
being violated, and asserting that the other commands "complete with the network
unavailable" is close to tautological, since a command that never calls out
completes either way.

So confinement is structural instead. The fetcher becomes an injected seam on
`DiagnoseInput`, beside the `LookupEnvironment` and `LookPath` fields that already
exist for exactly this reason, defaulting to the real implementation when nil —
the same pattern `Diagnose` already uses. Nothing else in the package is handed
one, so nothing else can call one.

Two assertions hold it:

- `net/http` is imported by exactly the transport package, and the transport
  package is imported by exactly one other — stated at the granularity
  `go list` actually has;
- the real fetcher is constructed at exactly one call site, checkable
  syntactically.

The seam is also what the verification tasks already presuppose: driving the
diagnosis against a local stand-in source needs somewhere to inject it.

The dependency-graph assertion that exists today — `net/http` absent from
`go list -deps ./cmd/gr` — necessarily stops being true. These two replace it.

### D4 — Refusals: the toolchain's first, then ours

`GOPROXY`, `GOPRIVATE` and `GONOPROXY` are read by the `go` command alone —
verified: with `GOPROXY=off` the `go` command refuses a module lookup while a
direct request to the proxy still returns `200`. A user who set them has already
said their module lookups stay home, and a hand-rolled request would override an
instruction they believe they gave.

So the check reads them — and reads them where they actually live. `go env -w
GOPROXY=off`, which is how Go's own documentation says to set it persistently,
writes to the Go environment file and leaves the process environment empty, so a
check consulting only `os.Getenv` would see nothing, conclude the machine is at
its default, and make exactly the request OUT-4 exists to prevent. Resolution
therefore follows the `go` command's own order: the process environment first,
then the environment file at `$GOENV`, or `os.UserConfigDir()/go/env` when that is
unset, with `GOENV=off` disabling the file. The file is plain `NAME=VALUE` lines
and is parsed with the standard library; the `go` binary is never executed,
because a user who installed a release binary need not have a toolchain at all.

What counts as declining is stated rather than left to taste. `GOPROXY` is a
comma- or pipe-separated list whose default is `https://proxy.golang.org,direct`;
the check proceeds only when its first entry is the public proxy. `off`, `direct`,
or a private mirror in first position all decline, even when the public proxy
appears later as a fallback — a user who put their own mirror first has said where
their lookups go. `GOPRIVATE` and `GONOPROXY` decline when their patterns match
this module path. `GR_NO_UPDATE_CHECK` disables it on its own, for
users who want the toolchain setting untouched, following the naming every
comparable tool uses. Continuous integration is never asked.

*Rejected:* the usual sixth gate, skipping when the output is not a terminal.
Every comparable tool has it, and it does not transfer here: the agent-facing
prompt runs the diagnosis through a pipe and shows its output to the user, which
is exactly where the line belongs.

*Rejected:* honouring `GOPROXY` by routing the request through whatever proxy it
names. A private proxy may not carry this module, and a failed lookup against a
corporate mirror would be reported as a failed check; declining is honest and
cheaper.

### D5 — The request is the toolchain's, not ours

Standard library defaults, no custom agent string, no version, no platform, no
query parameter. Observed: the `go` command sets no agent string of its own and
sends no other identifying header — the standard library supplies its default for
whichever protocol is negotiated, which is `Go-http-client/2.0` against the proxy
the request actually reaches. The first observation of this was made against a
local cleartext server and therefore recorded the HTTP/1.1 form; the string is not
the requirement, and pinning either literal would be a mistake. The requirement is
that Goalrail sets nothing, so its request is whatever the standard library would
have sent anyway — indistinguishable at the proxy from ordinary toolchain traffic,
with the module path in the URL as its only signal, one a `go install` user's
machine already sends routinely.

An agent string naming the tool would be the single bit announcing "a `gr`
installation is here", which is the phone-home signal in its purest form. The
transparency it was meant to provide is delivered instead where the user is: the
report line names the service it asked, and the README states what leaves the
machine.

### D6 — The cache is a file, and its mtime is the interval

The answer is written to one small file under the existing state root, which
already resolves from `GOALRAIL_STATE_HOME`, then `XDG_STATE_HOME`, then
`~/.local/state/goalrail`. Its modification time is the freshness test — the
approach Homebrew takes — so no schema, no expiry field, and no clock written into
a file that a user may copy between machines. The interval is 24 hours, which is
what `gh`, Homebrew, Deno and npm's update-notifier all use.

The cache holds the answer, not the verdict: the comparison against the running
binary's version happens on every diagnosis, so a cached answer stays correct
after the user replaces their binary.

### D7 — The release asks for its own version after publishing

The proxy discovers a tag late when nobody asks: measured on this repository at
3m45s for `v0.1.0` and 14m46s for `v0.1.1`, with no bound documented for that
passive path. Its documentation offers one remedy — explicitly request the
version, after which it "should be available within one minute".

So the release workflow requests the new version after `gh release create`
succeeds. It runs last, and its failure does not fail the release: the artifacts
are already public, and a slow proxy is not a defective release.

It asks for two things, and the reason is a correction the `v0.1.2` release
produced rather than a plan.

Requesting `@v/<tag>.info` is what makes the proxy ingest the version. Measured
across the three tags, from the same starting events rather than mixed ones:

| Tag | tag → proxy | publication → proxy | tag → publication |
|---|---|---|---|
| `v0.1.0` | 3m45s | not published | — |
| `v0.1.1` | 16m41s | 14m46s | 1m55s |
| `v0.1.2` | 2m46s | **4s** | 2m42s |

The comparison that belongs to this step is the middle column, because
publication is the only moment the release controls: 14m46s becomes four
seconds. The left column improves far less, and most of what remains there is
the release run itself, which the step cannot affect.

CTX-10 recorded `14m46s` for `v0.1.1` while describing it as tag-to-cache; the
number was publication-to-cache, and tag-to-cache for that release is 16m41s.
Both are correct measurements of different intervals, and the pack's label was
the error. It is corrected by appending rather than by rewriting.

But the diagnosis reads `@latest`, which is a different answer with its own
sixty-second cache, and it still reported the previous release for about two
minutes after the ingest. The step therefore asks for `@latest` as well, since
that is the answer the check actually depends on, waiting past that cache
boundary rather than giving up inside it. A prerelease tag skips the wait: the
proxy's `@latest` reports the newest stable version, so a prerelease can never
equal it while a stable release exists. The remaining floor is that cache, not
discovery.

### D8 — Shape in the report and in the JSON

The human report gains one line beside `openspec cli:` and `observability:`. The
structured output gains one object of its own, carrying the state, the version
found where there is one, the source, and when it was checked — so a script can
read the answer without the exit code changing meaning, which D9 forbids.

States, all reported as facts: a newer release exists; checked and nothing newer
was published as of the time recorded; not applicable because this build is not a
release; declined; and did not run.

### D9 — No new exit code

`rustup check` documents `100` for "an update is available", and it earned that by
having the exit code mean only that from the start. `gr doctor`'s three values
already mean healthy, has a problem, and did not run, and every unattended caller
reads them that way. A fourth value would make those callers wrong about a
repository that is fine. The answer lives in the report and the structured output.

## Correlation and Evidence

The correlating identity is the release tag: it is what the proxy answers with,
what the binary reports about itself, and what the two are compared on. Nothing
else needs to agree with anything else.

The only durable artifact is the cache file, which holds a public fact — the
newest released version — and nothing about the user or the repository. No
identifier is created; nothing about a check enters the repository, the harness,
or the runtime evidence stream.

## Measurement and Stop Conditions

Pass signals: a behind binary produces the line and the same exit code as before;
a development build produces no comparison; an offline machine differs from a
healthy one by one line; a second diagnosis within the interval makes no request;
each refusal produces "declined" and no request; the transport is unreachable from
every other command.

Stop and reshape if: the check needs a second source to answer usefully (that
means the question was wrongly scoped); the wording starts needing a next action
(that means it is a problem after all, and the diagnosis is the wrong home); or
the refusal set grows past what one line can explain.

## Risks / Trade-offs

- **The binary doubles.** Measured on the implementation rather than estimated:
  4.97 MB to 9.87 MB, and 2.76 MB to 5.42 MB compressed. The planning estimate of
  +40% had measured an imported transport instead of a used one — a declared but
  uncalled client is mostly eliminated by the linker — and the corrected figure
  was put to the owner before anything was committed. Accepted, unrecoverable by
  implementation quality, and paid by every user on every platform.
- **The source can be stale.** Mitigated by D7 and by wording that never claims
  currency; residual risk is a user told nothing is newer for about a minute.
- **The proxy is a third party.** It learns an IP and a module path it already
  sees from Go toolchains. Accepted, and stated in the report and the README.
- **Reachability discipline can rot.** Mitigated by making it a test; if that
  test is ever deleted, the guarantee is gone silently, which is why it is named
  in the spec rather than left to review.
- **A private-proxy user gets "declined" rather than an answer.** Accepted as
  honest: the alternative is reporting their corporate mirror's miss as a failure.

## Rollback

Delete the check package, its call from the diagnosis, and the release workflow's
last step. The dependency-graph test returns to asserting `net/http` is absent
outright, the diagnosis loses one line and one JSON object, and nothing else in
the product changes. No state is left behind but one small cache file under the
state root, which is inert once nothing reads it.

## Open Questions

None that can materially change implementation. The five the change opened —
whether to do it at all given the price, which source, default or flag, a new exit
code, and what the request discloses — were answered in confirmed intent
version 2.
