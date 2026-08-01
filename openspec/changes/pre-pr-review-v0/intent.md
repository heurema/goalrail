# Intent Snapshot

- **Intent ID:** INT-pre-pr-review-v0
- **Version:** 4
- **Status:** confirmed
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-pre-pr-review-v0 v4
- **Run references:** local session, 2026-08-01

## Source Evidence

- **SE-1 (owner, 2026-08-01):** "давай подумаем как нам уменьшить кол-во правок
  в pr постоянно codex находит пробелы в наших pr" — and, on the checklist-only
  answer: "тут нужно более общее решение".
- **SE-2 (owner, 2026-08-01):** reviewers come from the scaffolds the user
  actually has: "после codex ревью делает claude или новый claude + codex для
  codex то же самое".
- **SE-3 (owner, 2026-08-01):** nitpicker rejected on authentication grounds —
  "nitpicker не работает с подписками? тогда не нужен он" (CTX-8).
- **SE-4 (owner, 2026-08-01):** "давай заводить на api уже тестировать как раз
  начнем" — build now, exercise on the running Baseline API work.
- **SE-5 (owner, 2026-08-01):** the review runs inside the working loop,
  without asking anyone: "Мы должны в нашей петле всё сделать. Никаких команд
  дополнительных, никаких вопросов … Мы написали код и свой же код проверили."
- **SE-6 (owner, 2026-08-01):** after intent confirmation the user is not
  interrupted — "Пользователь не тревожен. У нас же есть интент" — except for
  "реальные расхождения … в момент реализации".
- **SE-6a (owner, 2026-08-01):** a second trigger — a genuine hole in the
  intent found during implementation: "Дыра в интенте появилась во время
  реализации … это реальная дыра, а не какой-то там давай-ка тут допилим."
- **SE-7 (owner, 2026-08-01):** authorship is inferable — "мы запустили в
  Claude Code и не понимаем, что мы в Claude Code запустили?" (CTX-13).
- **SE-8 (owner, 2026-08-01, hard stop):** an invented fact reached the
  Context Pack — "максимально критичный баг … либо факты либо ничего". Every
  context item now carries the command that reproduces it (Context Pack v4).
- **SE-9 (owner, 2026-08-01, twice):** review latency is unacceptable — "уже
  больше 20 минут раунд идет это долго", and a hung review at fifty-three
  minutes: "уже ни куда не годится ты может завис?".
- **SE-10 (owner, 2026-08-01, reframing):** most users have **one** provider —
  "Может быть, только один Claude … В основном что-то одно у человека. И в чём
  основной инструмент тоже непонятно, кто-то в кодексе пишет, а кто-то в
  кладе." A design that refuses without a second provider does not work for the
  ordinary user, and the Claude Code plugin path serves only one side.
- **SE-11 (owner, 2026-08-01):** the mode set was approved — fresh with one
  provider, cross with two, refute kept as a flag-triggered extra pass:
  "оставь как есть … только не ясно кто будет определять риск и включать флаг".
- **SE-12 (owner, 2026-08-01):** the risk question was resolved by removing
  risk from the rule: the refute trigger is "findings exist and the caller is
  about to act on them" — decided by the calling agent that just read them,
  never by Goalrail and never by a judgement of riskiness. Approved with the
  final critique pass ("внимательно еще раз взгляни … есть ли дыры" → six
  holes, accepted).
- **SE-13 (repository, CTX-3, CTX-4):** `codex review` refuses its scope flags
  together with custom instructions; the installed `claude` has no turn limit.
  Both were discovered by running them, after reading them had produced wrong
  claims.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | One command reviews the current branch in a **fresh session that never sees the author's context** — that asymmetry is the mechanism, and a different provider is an upgrade to it, not its precondition. With one installed provider the reviewer is that provider in a clean session (**fresh**); with two, the provider that did not author the change (**cross**). Authorship is inferred from the invoking environment, an explicit override wins, and an undetectable author is recorded as unknown rather than refused. The only refusal is that no reviewer can be run at all. | On a machine with one provider, the command reviews with it and asks nothing; on a machine with two, it selects the non-author; with authorship undetectable, it still reviews and the receipt says unknown. | SE-5, SE-7, SE-10, SE-11 |
| OUT-2 | The only other machine refusal is the budget gate: a command named in configuration, run before any reviewer, whose non-zero exit stops the review and names the gate. No personal budget infrastructure ships in the product. | Configure a failing gate and confirm nothing runs; remove it and confirm the same invocation reviews. | SE-5 |
| OUT-3 | The loop closes before the pull request opens, and its cost is proportional to what changed: a re-review by default covers only the range since the previous receipt's head, while the receipt keeps the full branch digest for staleness, so the chain of receipts proves cumulative coverage. A full re-review stays available by flag. The loop's ceiling and stop condition are declared before the first round, and hitting the ceiling with findings open is reported, never absorbed. | Review, fix, re-review: the second round's reviewed range starts at the first round's head and completes in a fraction of the first round's time; the receipts chain covers every commit. | SE-5, SE-9 |
| OUT-4 | The user is not interrupted by the loop, with exactly two exceptions, each one focused question: a finding revealing a real divergence from confirmed intent, and a real hole in the intent uncovered by implementation. The test separating a hole from scope creep stands: proceedable under a stated assumption → not a hole, record and continue; an improvement idea is never a question — it is recorded as a candidate change. | Count interactions: zero clean, one on a seeded conflict, one on a seeded hole, zero on a seeded improvement idea (which appears in the record). | SE-6, SE-6a |
| OUT-5 | A completed review leaves a receipt bound to what was actually reviewed and how: base and head commits, the digest of the reviewed range, the full branch digest, the **mode** (fresh, cross, or refute) with the reason it was selected — a cross downgraded to fresh because the other CLI would not run is stated, not silent — the reviewer identity, the author or `unknown`, the measured duration, and each report as verbatim bytes with its own digest. No field is derived by interpreting any report. | Inspect a receipt after each mode, recompute the digests, and confirm the mode, reason, duration and byte-identity. | SE-8, SE-10, SE-13 |
| OUT-6 | The diagnosis reports the branch's review state in one line — current, stale, absent, or unreadable — from recomputing the digest, as a fact that never affects the verdict or exit status, with no flag and no dismissal anywhere. | Review → current; commit → stale; re-review → current; corrupt the receipt → unreadable, with exit status unchanged throughout. | SE-5 |
| OUT-7 | Reviewer instructions live in a committed repository file, materialized with a default when absent and never overwritten. Ratchet promotions land there as ordinary edits. | Edit the file, re-run, confirm the reviewer received the edits and the file is untouched afterwards. | SE-1 |
| OUT-8 | A **refute** round is available in every mode: a fresh session — the other provider where one exists, the same provider otherwise — receives the findings and the diff and tries to refute the findings rather than add new ones. It is triggered by the caller's flag, under one rule: findings exist and the caller is about to act on them; zero findings end the round. Both reports are stored verbatim in the receipt, and which findings survived is the reader's judgement, never Goalrail's. | Trigger it after a review with findings and confirm the refutation is a second verbatim report in the receipt; confirm Goalrail computed nothing from either. | SE-11, SE-12 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Goalrail never parses, scores, or interprets review text — including the refutation. Passing a report verbatim to a refuter is not parsing; deriving anything from it would be. | SE-8 |
| NG-2 | The review is evidence, never a gate. Nothing blocks a commit, push, PR, or merge mechanically. | SE-5 |
| NG-3 | No vendoring, pinning, or wrapping of vendor CLIs, and no third-party review tool. Interface drift is accepted and surfaces loudly. The Claude Code plugin is not a dependency either: it serves only one provider's side and cannot be required of a user who has the other (SE-10). | SE-3, SE-10, SE-13 |
| NG-4 | Findings disposition and the ratchet journal stay at the workspace level. | — |
| NG-5 | No interactive prompt in any Goalrail command. | SE-5 |
| NG-6 | No unbounded loops and no unbounded reviews: the loop declares its ceiling, and every review runs under a deadline that bounds the whole process tree — measured once as a twenty-minute deadline that did not stop a fifty-three-minute run, which is the failure this boundary names. | SE-9 |
| NG-7 | Goalrail never assesses risk and never decides escalation. The refute flag is the caller's statement about what they are going to do with the findings; mode selection follows from what is installed and runnable, never from a judgement. | SE-11, SE-12 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The ordinary single-provider path works and asks nothing. | One invocation, one installed provider: fresh-mode review, receipt written, zero prompts. | OUT-1 |
| SIG-2 | The gate is honored and optional. | Failing gate → nothing runs, named refusal; no gate → reviews. | OUT-2 |
| SIG-3 | The receipt proves what happened. | Digests recompute; mode, reason, duration present; reports byte-identical. | OUT-5 |
| SIG-4 | Staleness and re-review cost track the branch, not the calendar. | Second round reviews only the new range and is measurably faster; doctor line moves current→stale→current with no flag. | OUT-3, OUT-6 |
| SIG-5 | The first real use closes a real loop. | The Baseline API extraction branch: review, in-loop fixes, re-review to a current receipt before its PR opens, zero interruptions on the clean path. | SE-4 |

## Ambiguities and Unknowns

None blocking, actively checked. Left to design: the refuter's instruction text
and where it lives (the same committed file is the candidate); the deterministic
preference order when authorship is unknown and both providers run; the deadline
and effort defaults as numbers. Each is a design decision inside confirmed
boundaries, not a gap in them.

## Version history

- **v4 (2026-08-01):** Supersedes v3 on the owner's reframing (SE-10) before
  v3's implementation completed. Material changes: the mechanism is context
  asymmetry, so a single-provider **fresh** mode is a legitimate default rather
  than a refusal; **cross** is the two-provider upgrade; **refute** is available
  in every mode and triggered by the caller's findings-based rule, with risk
  removed from the vocabulary (SE-11, SE-12); re-review is incremental by
  default (SE-9); the receipt records mode, reason, duration and the reviewed
  range (OUT-5); undetectable authorship records `unknown` instead of refusing.
  The hard stop over an invented Context Pack fact (SE-8) produced Context Pack
  v4, where every item carries its reproduction command.
- **v3 (2026-08-01):** Added the intent-hole interruption trigger with its
  anti-scope test (SE-6a) and NG-7's predecessor.
- **v2 (2026-08-01):** Loop-first (SE-5), inferred authorship (SE-7), declared
  ceilings, no-interruption rule.
- **v1:** Never confirmed; superseded the same day.

## Confirmation

- **Confirmed by:** Vitaly D.
- **Confirmed at:** 2026-08-01
- **Verification action:** The owner drove four versions from plain-language views in Russian: v2 (loop-first, inferred authorship), v3 (intent-hole trigger with the anti-scope test), a hard stop over an invented Context Pack fact that produced the fact-with-check rule, and v4 (single-provider fresh mode as the ordinary case, findings-based refute trigger, incremental re-review), confirming the v4 view that covered all three modes, the single refusal, the receipt's mode and duration fields, and the latency outcome.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
