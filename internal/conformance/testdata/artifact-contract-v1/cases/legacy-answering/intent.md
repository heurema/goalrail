# Intent Snapshot

- **Intent ID:** INT-answering
- **Version:** 2
- **Previous version:** 1
- **Status:** confirmed
- **Owner:** owner
- **Context Pack:** CTXP-answering version 2
- **Run references:** corpus fixture
- **Resolves:** question-1234567890abcdef escalation sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
- **Disposition:** answered

## Source Evidence

- **SE-1 — owner:** The owner requested a bounded conformance result.

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded result. | Inspect it. | SE-1, CTX-1 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1, CTX-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local result exists. | SE-1, CTX-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** owner
- **Confirmed at:** 2026-08-03T08:00:00Z
- **Verification action:** The owner reviewed the three semantic groups.
