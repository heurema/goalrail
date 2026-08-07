# Intent Snapshot

- **Intent ID:** doctor-toolchain-inspection-v0
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** confirmed
- **Owner:** role:repository-owner
- **Context Pack:** doctor-toolchain-inspection-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 — Owner statement:** The owner asked for the reported defect to be
  corrected, and chose the full planning cycle over a direct code change, so the
  decision and its grounds are recorded in the repository rather than only in a
  session.
- **SE-2 — Owner statement:** The owner asked, before authorizing work, for the
  defect to be explained in the simplest terms available. The correction is
  therefore expected to be describable without reference to internal structure.
- **SE-3 — Repository fact:** Diagnosis executes the declared planning runtime
  and adapter on every run, which a promoted requirement names as a violation
  (CTX-1, CTX-2, CTX-3).
- **SE-4 — Repository fact:** The name of the executed program is repository
  content bounded only by length, NUL, and secret-shaped content, so a
  checked-out repository selects the program that runs (CTX-14).
- **SE-5 — Repository fact:** Diagnosis resolves that program through the
  executable lookup path, which answers about an arbitrary installation rather
  than the one the declared profile pins (CTX-13).
- **SE-6 — Repository fact:** Everything needed to answer the availability
  question without executing anything is already written to disk by authorized
  setup: a durable location, a pinned version, and a per-file digest (CTX-5,
  CTX-6, CTX-7), and the installed location is derivable from the running
  binary's own version because a release refuses to publish a mismatched stamp
  (CTX-8, CTX-9).
- **SE-7 — Repository fact:** Diagnosis must still report availability and
  compatibility, and the states it must distinguish already include an
  unverified integrity identity (CTX-4, CTX-11).
- **SE-8 — Repository fact:** No existing test fails on the current behaviour
  (CTX-12).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Diagnosis establishes planning runtime and compiler readiness by inspection alone, executing neither the runtime, nor a package runner, nor the stock planning CLI. | Run diagnosis in a managed project with executables of those names placed earlier on the lookup path that record every invocation; observe that nothing was recorded. | SE-3, CTX-1, CTX-3 |
| OUT-2 | Diagnosis derives the inspected location from Goalrail's own installation record rather than from a name carried in the checked-out repository, so repository content can no longer select what is inspected. | Place a path in the repository's declared runtime field and run diagnosis; observe that the declared value does not decide what is read or run. | SE-4, CTX-14 |
| OUT-3 | Diagnosis continues to report the four component states the promoted requirement names, and a verified installation still reaches a ready verdict. | Run diagnosis against a complete installed bundle and observe planning readiness true; remove or alter one inspected file and observe the state change without an execution. | SE-7, CTX-4, CTX-10, CTX-11 |
| OUT-4 | An installation that never ran Goalrail setup is reported as requiring setup rather than as ready, whatever unrelated toolchain the machine happens to carry. | Run diagnosis on a machine with a mismatched system runtime and no Goalrail-installed bundle; observe a setup-required verdict naming the exact missing component. | SE-5, CTX-13, CTX-15 |
| OUT-5 | The correction carries a regression that fails against the current behaviour, so the executed form cannot return unnoticed. | Run the new test against the pre-change code and observe it fail; run it after the change and observe it pass. | SE-8, CTX-12 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not make Goalrail able to install, repair, or upgrade the planning toolchain from diagnosis. Diagnosis reports; authorized setup installs. | CTX-4, CTX-6 |
| NG-2 | Do not introduce a new pointer file, registry, index, or discovery contract for the installed bundle. The running binary's own version already identifies it. | CTX-8, CTX-9 |
| NG-3 | Do not relax the prohibition by moving the execution to another command, another package, or an opt-in flag. | CTX-3 |
| NG-4 | Do not change the declared setup profile's schema, its pinned versions, or any repository-owned governance value. | CTX-2, CTX-14 |
| NG-5 | Do not add a component state, reason code, or field that consumers of the machine contract must learn in order to keep working. | CTX-11 |
| NG-6 | Do not correct the two other divergences found in the same audit — the unreachable scaffold registration and the missing adoption line. They are separate work items. | SE-1 |
| NG-7 | Do not treat a toolchain that Goalrail did not install as equivalent to one it did, in order to preserve a previously reported ready verdict. | CTX-13, CTX-15 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Diagnosis performs zero process executions of the declared runtime and adapter. | Recorded invocations from decoy executables on the lookup path: exactly zero, against two today. | CTX-1 |
| SIG-2 | Diagnosis completes normally on a machine with no runtime present at all. | The command exits with its ordinary diagnosis status and reports the absence as a component state, with no error from a missing executable. | CTX-3 |
| SIG-3 | A repository-supplied path in the declared runtime field results in no execution of that path. | The named program records no invocation and the diagnosis states a component state instead. | CTX-14 |
| SIG-4 | Planning readiness remains reachable. | A complete verified bundle yields planning readiness true. | CTX-10 |
| SIG-5 | The machine contract's component states are unchanged in name and number. | The four states in the report are the ones the requirement already names. | CTX-4, CTX-11 |
| SIG-6 | The regression fails before the change and passes after it. | One recorded run of each. | CTX-12 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** role:repository-owner
- **Confirmed at:** 2026-08-06
- **Verification action:** The owner was shown a plain-language view of this exact
  version in the conversation language, covering every outcome, boundary, and
  success signal — including the one visible behaviour change, that an
  installation which never ran authorized setup stops being reported as ready —
  and explicitly confirmed version 1 by name.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
