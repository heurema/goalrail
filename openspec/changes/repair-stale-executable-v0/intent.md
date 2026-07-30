# Intent Snapshot

- **Intent ID:** intent-repair-stale-executable-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-repair-stale-executable-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (repository):** Issue heurema/goalrail#25 — a registration whose executable path is stale is reported as already present, so `gr health`'s advice to re-run `gr connect --scaffold <name> --yes` has no effect; connect must instead repair it.
- **SE-2 (repository):** `isConnected` never receives or compares the executable connection was invoked with, even though `PlanConnection` already resolves it, so presence detection cannot distinguish a current registration from a stale one.
- **SE-3 (repository):** Neither scaffold's write path replaces an existing managed registration in place — Codex appends unconditionally, Claude Code replaces only when unscoped — so correcting presence detection alone would either duplicate the Codex block or leave a stale-but-scoped Claude Code handler untouched.
- **SE-4 (repository):** A committed test comment already marks this exact defect as deliberately out of scope for the prior change and names it as the reason `gr health`'s remedy is a dead end.
- **SE-5 (repository):** The promoted `ambient-connect` requirement binds this change: connection and disconnection stay safe to repeat, disconnection never touches a handler it did not add, Claude Code session-start stays scoped to the opening occurrence, and Goalrail never writes, computes, or reproduces scaffold trust records.
- **SE-6 (repository):** Replacing a handler's command text returns that hook to an untrusted state on a scaffold that gates execution on trusting the exact current definition, so a silent repair would trade one silent failure for another.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | A registration whose command points at an executable other than the one connection was invoked with is treated as needing repair, not as already present, for both supported scaffolds. | Seed a registration with a different executable path, run `PlanConnection`, and confirm `AlreadyPresent` is false. | SE-1, SE-2, CTX-2, CTX-5 |
| OUT-2 | Repair replaces the stale registration in place rather than duplicating it: the Codex managed block is not appended twice, and the Claude Code handler for each event is swapped rather than left alongside a new one. | Seed a stale registration, connect, and confirm exactly one managed block (Codex) or one managed handler per event (Claude Code) is present afterward, now pointing at the current executable. | SE-3, CTX-3 |
| OUT-3 | A registration already pointing at the current executable is untouched — the configuration stays byte-identical on a repeated consented connection. | Connect from a given executable, connect again with the same executable, and diff the configuration before and after. | SE-1, CTX-6 |
| OUT-4 | A foreign handler for the same event, and any handler for a different event, survives the repair unchanged. | Seed a stale managed handler alongside a foreign one for the same event, connect, and confirm the foreign handler is present and unmodified afterward. | SE-5, CTX-6 |
| OUT-5 | Repaired Claude Code session-start registration stays scoped to the occurrence that opens a session; repair does not reintroduce an unscoped form. | Seed a stale-and-unscoped registration, connect, and confirm the result is both current and scoped to `startup`. | SE-5, CTX-6 |
| OUT-6 | Connection's output discloses that a repaired handler needs the scaffold's trust step again, using the same per-scaffold wording already used for a fresh connection, so the fix does not trade a silent stale path for a silent untrusted one. | Seed a stale registration, connect, and confirm the disclosure is present in the response when a repair occurred. | SE-6, CTX-7 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not change how `gr health` detects a missing executable or reports trust state; this change only makes `gr connect` able to act on what health already diagnoses. | SE-1, CTX-1 |
| NG-2 | Do not write, compute, or reproduce a scaffold trust record; disclosing that trust is needed again is not the same as granting or simulating it. | SE-6, CTX-7 |
| NG-3 | Do not change the announcement text, retention, intent binding, fail-quiet posture, or the wrapper lifecycle. | SE-5, CTX-6 |
| NG-4 | Do not change project-level registration or any other item already recorded as a deferred decision. | CTX-6 |
| NG-5 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, or provider run. Each remains a separate owner gate. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A stale registration is repaired by a repeated consented connection, without duplicating a handler, on both scaffolds. | A test seeds a stale registration for each scaffold, connects, and asserts exactly one current managed registration remains. | SE-1, CTX-5 |
| SIG-2 | A current registration is untouched on repeat. | A test connects twice from the same executable and diffs the configuration byte for byte. | SE-1, CTX-6 |
| SIG-3 | A foreign handler for the same event survives a repair. | A test seeds a stale managed handler beside a foreign one and asserts the foreign one is unchanged after repair. | SE-5, CTX-6 |
| SIG-4 | Claude Code session-start stays scoped to `startup` after a repair. | A test seeds a stale-and-unscoped registration and asserts the repaired form carries the `startup` matcher and nothing else. | SE-5, CTX-6 |
| SIG-5 | `gr health` reports `working` after a repair, having reported the missing-binary next action before it. | A test runs `Inspect` before and after repairing a registration whose old path no longer exists. | SE-1, CTX-1 |
| SIG-6 | The repair discloses that trust applies again. | A test connects onto a stale registration and asserts the response carries the trust disclosure. | SE-6, CTX-7 |

## Ambiguities and Unknowns

None blocking. Scope follows the issue precisely: repair-on-reconnect for a
stale executable path. Whether a scaffold's trust gate was observed live is
already recorded per scaffold in the promoted requirement and is not
reopened here.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-30T08:30:00Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — connection compares the registered executable against the one it was invoked with and treats a difference as needing repair, repair replaces rather than duplicates, a current registration stays byte-identical, foreign handlers survive, the repaired Claude Code session-start entry stays scoped to the opening occurrence, and the repair discloses that the scaffold's trust step applies again — together with the non-goals (health's detection unchanged, no trust record written or simulated, announcement and lifecycle untouched), and explicitly confirmed the snapshot.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
