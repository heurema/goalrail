# Intent Canary v0 Owner Review Packet

- **Prepared:** 2026-07-21
- **Decision owner:** Goalrail owner
- **Manifest:** `intent-canary-v0` version 1, `frozen_not_activated`
- **Real assignments:** 0 of 15
- **Recommendation:** `NO-GO_NOW`

OpenSpec reports 19 of 19 change tasks complete and strict validation passes.
This means the bounded implementation-and-review change is complete; it does
not mean the real canary is ready, approved, archived, or started.

## Decision requested

Decide whether the separate 15-real-change canary may proceed. The evidence
supports the intent semantics, automatic CLI lineage, append-only record,
deterministic report, privacy boundary, and rollback mechanism. It does not yet
support a safe live start because the activation operator has material gaps
listed below.

Recommended owner response:

- `NO-GO_NOW` — keep manifest v1 frozen and authorize no real assignments; or
- `HARDEN_THEN_REVIEW` — authorize a separate OpenSpec activation-hardening
  change, then return to a second explicit `GO_LIVE` owner gate.

Neither response authorizes commit, push, deploy, credential changes, hook
installation, or the 15-change canary itself.

## Evidence matrix

| Claim | Evidence | Result |
|---|---|---|
| Intent is owner-confirmed and proposal-gated | `intent.md`; intent and OpenSpec adapter tests | Verified locally |
| CLI `change_id -> run_id -> root session_id` is automatic | `evidence/correlation-live-probe.md` | Live disposable CLI join verified |
| Goalrail App receipt never accepts manual identity | launch-receipt adapter tests and owner-approved design | Deterministic adapter verified; live App-to-Langfuse join not verified |
| Synthetic lifecycle preserves lineage, checks, assessment, and report | `canary/synthetic-e2e-v1.md`; preserved `internal/integration/testdata/synthetic-e2e-v1/*` | Byte-for-byte fixture verified |
| Repository evidence excludes raw prompts, transcripts, and credentials | `canary/evidence-privacy-audit.md`; serialized-output tests | Pass for internal non-sensitive v1 scope |
| Rollback stops assignments without rewriting evidence or changing Langfuse config | `canary/rollback-v1.md` | Verified in isolated store |
| Current implementation passes repository checks | `go vet ./...`, `go test -race ./...`, correlation probes, strict OpenSpec validation | Pass on 2026-07-21 |

## Measured synthetic overhead

- One synthetic flow-only root-turn interval: 10 seconds.
- One synthetic owner-review interval: 5 seconds.
- Formula result: `(10 + 5) / 60 = 0.25` minutes.
- Report: `PENDING`, one delivered flow assignment, verified lineage, owner
  `match`, `repeat_optin=yes`, no hard-stop signal.

This measurement validates the frozen formula only. It is not live Langfuse
timing and predicts neither real owner effort nor the 15-change median.

## Activation blockers

1. **Real activation is intentionally unavailable.** `Service.Start` rejects
   every non-synthetic assignment, and no persistent hook or App launcher is
   installed. Removing that guard requires a separate owner-approved change.
2. **Eligibility and exclusion are not enforceable operator events.** Manifest
   rules exist, but the CLI has no pre-assignment eligibility receipt or
   rejected-candidate record that proves an exclusion consumed no ordinal.
3. **Check timing is not evidenced.** Check references are written with the
   terminal delivery/abandonment event; the current record cannot prove that
   the check set was frozen before the terminal verification run.
4. **Live overhead provenance is not collected.** The domain formula validates
   trace and owner-review intervals, but the operator currently accepts a
   caller-supplied numeric `--overhead-minutes`. It does not fetch Langfuse turn
   envelopes or record `review-start`/`review-end` itself.
5. **Hard-stop evaluation does not itself stop assignments.** `report` returns
   `STOP`, and `disable` safely appends the stop marker, but a hard-stop report
   does not automatically append that marker before the next `start`.
6. **The App path lacks a live end-to-end receipt.** The tested App project-hook
   path was `UNSUPPORTED`; `create_thread.threadId` binding is deterministic in
   the adapter, but equality with the root identity observed by Langfuse has not
   been proven in one disposable Goalrail-launched App task.
7. **Reviewed default-branch source remains an activation precondition.** This
   packet was prepared before branch and PR review. Publishing the foundation
   for review does not itself approve or start the real canary.

These are activation-readiness gaps, not reasons to weaken automatic identity
or substitute manual ID copying, transcript parsing, shared mutable
`current-change` state, or a Langfuse plugin fork.

## Preconditions for a future `GO_LIVE`

1. Add append-only eligibility/exclusion and pre-verification check-selection
   evidence without exposing the next variant before eligibility.
2. Derive flow overhead from verified Langfuse trace intervals plus explicit
   owner-review timestamps; do not ask the operator to transcribe the total.
3. Make every report-level hard stop atomically bar the next assignment while
   preserving the existing chain.
4. Run one disposable Goalrail-launched App receipt-to-Langfuse join probe, or
   create a separately approved manifest version whose eligibility excludes
   App tasks.
5. Review and land the implementation through an owner-approved branch/PR, then
   install only the project-local activation configuration.
6. Return with an activation receipt and request the owner's separate exact
   `GO_LIVE` instruction before ordinal 1.

## Retained limits after activation

- Fifteen changes provide directional evidence, not causal proof.
- Shared Langfuse retention remains eligible only for internal, non-sensitive
  work.
- Missing trace evidence stays missing and blocks completion readiness; it is
  never imputed.
- OpenSpec remains a pinned, replaceable development adapter rather than the
  canonical Goalrail runtime model.
