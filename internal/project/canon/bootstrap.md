# Goalrail project flow

This repository is managed by Goalrail. The committed declaration at
`.goalrail/project.json` is the project identity in every clone and worktree.
The checkout-local marker used before Goalrail v0.2 is not project identity.
Do not run `gr init` in an already declared clone or worktree.

Before changing material code or governing documentation:

1. Read the declaration, its referenced policy and setup profile, and this
   guide. Treat a present but invalid declaration as a fail-closed governance
   claim, not as an unmanaged repository.
2. Inspect local readiness without changing repository files, user
   configuration, executable locations, credentials, or scaffold state. Run
   `gr doctor --json` when a compatible `gr` is already available. Otherwise
   report `GOALRAIL_SETUP_REQUIRED` and follow the clean-machine protocol below.
3. If setup was required, wait for its terminal receipt. Then return to the
   initiating request and prepare one candidate, versioned Intent Snapshot. The
   owner must actively confirm that exact candidate before proposal or
   implementation.
4. Use one current Goalrail/OpenSpec change and keep the WorkSpec, work unit,
   run/session, commits, pull request, review, checks, terminal receipt, and
   owner decisions linked by canonical lineage.

## Clean-machine setup protocol for supported agents

Setup is a separate enabling action. A feature request, general approval, or
permission to use Goalrail is not setup authorization. Until setup has a
terminal receipt, perform zero project-code writes and keep the initiating
request paused.

### 1. Resolve a complete plan without installing

- Read `.goalrail/project.json` and the exact referenced
  `.goalrail/setup-profile.json` bytes. Resolve the selected supported scaffold
  from that committed profile; do not invent a provider-specific flow.
- Inspect operating system, architecture, user-local destinations, existing
  executable and configuration bytes, and prerequisites read-only. Do not
  start login, device authorization, credential refresh, account binding, or
  any other authentication flow.
- Through the documented anonymous HTTPS release channel, read bounded current
  release metadata and the immutable selected release's checksums and setup
  manifest. Pin one compatible release, platform archive, component tree,
  versions, sizes, and SHA-256 identities. A mutable current-release pointer is
  discovery only and is never an integrity identity.
- Freeze one canonical `goalrail.setup-plan/v1` value and its SHA-256 digest.
  The plan must be `complete` and must show every `component`, `mutation`,
  `prerequisite`, `trust_step`, `network_access`, `rollback`, and
  `verification` item, including exact sources, integrity identities,
  destinations, scopes, and expected prior bytes. `project_code_writes` must be
  `0`. An unresolved item makes the plan incomplete and non-executable.
- Present the complete canonical JSON plan and its digest. An owner-facing
  explanation may accompany it but must not replace, omit, or broaden any plan
  item.

Planning may fetch only non-executable release metadata. It must not download
an executable archive into durable storage, change configuration, create a
trust record, or retain an executable before authorization. Never use
curl-to-shell or trust mutable or unverified bytes.

### 2. Ask once for the exact digest

After displaying one complete plan, ask exactly one authorization question in
the owner's language:

> Do you authorize Goalrail setup plan `<plan_digest>` exactly as shown?

Only an explicit grant naming the current complete plan digest is authority.
Silence, ambiguity, refusal, general feature consent, or approval of an older
digest means no authorization: make zero mutations, record a bounded no-write
result when a verified Goalrail receipt path is available, and do not continue
feature implementation. Do not repeat the unchanged question.

If any plan byte or inspected fact changes, discard the old authorization,
resolve a new complete plan and digest, display it, and ask one new exact-plan
question. Never silently extend an authorization.

### 3. Apply only the authorized plan

After exact authorization:

1. Download only the immutable archive, checksum, and manifest named by the
   plan into a temporary directory. Verify names, sizes, SHA-256 identities,
   release compatibility, and archive contents before executing anything.
2. Extract into that temporary directory and run the bundled `gr setup
   verify-plan` against the canonical plan, canonical authorization, retained
   release metadata, managed repository, selected scaffold, and temporary
   bundle. Semantic inequality or changed local state invalidates authority.
3. Run bundled `gr setup apply` with the same verified artifacts and a bounded
   `--continuation-ref`. It may perform only enumerated mutations and must place
   executables and runtimes at their planned durable user-local destinations.
4. Preserve unrelated configuration. Perform every just-in-time state check,
   retain the private bounded recovery set, and use `gr setup rollback
   --recovery <path>` only for the exact failed attempt when rollback is needed.
5. Never manufacture scaffold or provider trust. If the plan includes an
   interactive trust step, direct the owner to the named provider surface and
   wait for provider-observed evidence before continuing verification.
6. Run the planned diagnosis. Keep local readiness separate from shared
   admission: a local hook or committed workflow is not proof that a protected
   shared check is active.
7. Capture the canonical `goalrail.setup-receipt/v1` output, bounded doctor
   summary, rollback reference, status, plan digest, and continuation reference.
   Remove temporary downloads only after durable placement and receipt capture.

At the first runtime, dependency, login, credential action, global setting,
privilege escalation, provider trust record, external mutation, or other
enabling action absent from the displayed plan, stop before that action. Keep
the initiating request paused, report the exact blocker, and require a newly
resolved plan and separate authorization. Do not improvise, retry an unchanged
failure, or substitute another installation route.

### 4. Resume only from the receipt

Resume the initiating request only after a terminal setup receipt exists. A
successful receipt returns the supported agent to the candidate Intent Snapshot
step; it does not confirm intent, authorize implementation, start a Goalrail
run, commit, push, open a pull request, enable branch protection or required
checks, deploy, or publish. A failure or no-write receipt leaves the request
paused and must name the exact next owner action.

Local hooks are advisory feedback. A committed workflow is only a prepared
adapter. Do not claim protected enforcement unless Goalrail has independently
verified shared-admission activation.
