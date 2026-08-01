# Intent Snapshot

- **Intent ID:** INT-pre-pr-review-v0
- **Version:** 3
- **Status:** confirmed
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-pre-pr-review-v0 v2
- **Run references:** local session, 2026-08-01

## Source Evidence

- **SE-1 (owner, 2026-08-01):** "давай подумаем как нам уменьшить кол-во правок
  в pr постоянно codex находит пробелы в наших pr" — and, on the checklist-only
  answer: "тут нужно более общее решение".
- **SE-2 (owner, 2026-08-01):** the reviewer must be chosen from the scaffolds
  the user actually has installed, cross-provider: "после codex ревью делает
  claude или новый claude + codex для codex то же самое".
- **SE-3 (owner, 2026-08-01):** nitpicker was considered and rejected on
  authentication grounds — "nitpicker не работает с подписками? тогда не нужен
  он" (CTX-8 records the measured basis).
- **SE-4 (owner, 2026-08-01):** "давай заводить на api уже тестировать как раз
  начнем" — build it now and exercise it on the running Baseline API work.
- **SE-5 (owner, 2026-08-01, superseding v1's consent framing):** the review
  belongs inside the working loop, run without asking anyone: "Мы должны в
  нашей петле всё сделать. Никаких команд дополнительных, никаких вопросов …
  Мы написали код и свой же код проверили." Findings are fixed in the same
  loop before the pull request opens; reviewers argue among themselves and hand
  the author what must be fixed.
- **SE-6 (owner, 2026-08-01):** after intent is confirmed, the user is not
  interrupted — "Пользователь не тревожен. У нас же есть интент … Зачем нам
  лишний вопрос" — with exactly one exception: "только если реальные
  расхождения не появились в момент реализации".
- **SE-6a (owner, 2026-08-01, extending SE-6):** a second legitimate trigger —
  a genuine hole in the intent discovered during implementation: "Дыра в
  интенте появилась во время реализации … но важно не уходить в бесконечный
  скоуп: это реальная дыра, а не какой-то там давай-ка тут допилим."
  Improvements and scope growth are explicitly not holes.
- **SE-7 (owner, 2026-08-01):** inferring authorship is expected, not
  impossible: "мы запустили в Claude Code и не понимаем, что мы в Claude Code
  запустили?" — confirmed by measurement (CTX-13).
- **SE-8 (repository, CTX-1, CTX-2):** fourteen findings in one day, all
  post-PR; six classes; two crossed the promotion threshold immediately. The
  defect is timing and independence, not reviewer quality.
- **SE-9 (repository, CTX-3, CTX-4, CTX-8):** `codex review --base` and
  `claude -p` are installed, non-interactive, and subscription-legal;
  `codex review` emits no machine-readable output; neither CLI is pinnable.
- **SE-10 (repository, CTX-10):** every review spends a metered subscription;
  the global rules gate substantial spend through a machine-readable guard.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | One command runs an independent review of the current branch as a step of the author's own working loop: it infers the author's provider from the invoking environment, selects the reviewer as the other installed provider, assembles the instructions, executes the review, and leaves the receipt — with no question to anyone on the ordinary path. An explicit override exists for authorship; refusal is only for the case where nothing is detectable and nothing is stated. | Invoke it from inside a Claude Code session with no flags and confirm it selects Codex, runs, and receipts without prompting. | SE-5, SE-7, CTX-13 |
| OUT-2 | The only machine refusal is the budget gate: a configurable gate command, when set, runs first, and a non-zero exit stops the review and names the gate as the reason. No personal budget infrastructure is built into the product, and no human consent step exists inside the loop. | Configure a failing gate and confirm nothing runs; remove it and confirm the same invocation reviews. | SE-5, SE-10 |
| OUT-3 | The loop closes before the pull request opens: findings return to the author in the same session, the author fixes what is real, and the review is re-run until it reports nothing material or a hard round ceiling is reached — the ceiling and the stop condition are declared before the first round, and reaching the ceiling with findings still open is reported as the loop's honest result, never silently absorbed. | Drive a branch with a known defect through the loop and confirm the fix lands and the re-review receipt is current before any pull request exists. | SE-5, SE-8 |
| OUT-4 | The user is not interrupted by the loop. After the work's intent is confirmed, review findings, fixes, and re-reviews produce no questions and no messages to the user — with exactly two exceptions, both stopping the loop for one focused question instead of guessing: a finding that reveals a real divergence from the confirmed intent, and a real hole in the intent uncovered by implementation — a question the intent does not answer and that must be answered to build correctly. The test separating a hole from scope creep: if the work can proceed correctly under a stated assumption, it is not a hole — the assumption is recorded and the loop continues; an improvement idea is never a hole and never a question — it is recorded as a candidate for a future change. | Run a full loop and count user interactions: zero on the clean path; exactly one on a seeded intent conflict; exactly one on a seeded intent hole; zero on a seeded improvement idea, which appears in the record instead. | SE-6, SE-6a |
| OUT-5 | A completed review leaves a receipt bound to what was actually reviewed: base and head commits, a digest of the diff, the reviewer identity, the time, and the reviewer's report stored as verbatim bytes with their own digest. No field of the receipt is derived by interpreting the report's content. | Inspect a receipt, recompute the digests by hand, and confirm byte-identity of the stored report. | SE-9 |
| OUT-6 | The diagnosis reports the review state of the current branch in one line — current, stale, or absent — judged by comparing the receipt's diff digest against the branch's present diff, as a fact that never affects the verdict or exit status. Committing new work makes it stale by itself; a fresh review makes it current by itself. | Review, confirm current; commit an edit, confirm stale; re-review, confirm current — with no flag or dismissal anywhere. | SE-8 |
| OUT-7 | The reviewer instructions live in a committed repository file that the review reads, with a default materialized by Goalrail where none exists. Promotions from the findings ratchet land there as ordinary edits, shared and versioned like any other repository content. | Edit the file, re-run the review, and confirm the reviewer received the edited instructions. | SE-1, SE-8, CTX-2 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Goalrail never parses, scores, or interprets the review text. The loop's author reads the findings and decides what is real; Goalrail stores bytes and digests. Building a prose parser into the feature that exists to catch confident misreads of foreign formats would be the defect class it is meant to catch. | SE-9 |
| NG-2 | The review is evidence, never a gate. Nothing here blocks a commit, a push, a PR, or a merge mechanically; the discipline lives in the loop's policy, not in a refusal. | SE-5, CTX-9 |
| NG-3 | No vendoring, pinning, or wrapping of the vendor CLIs, and no third-party review tool dependency. The drift risk of unpinned CLIs is accepted and stated rather than papered over. | SE-3, SE-9 |
| NG-4 | Findings disposition and the ratchet journal stay at the workspace level. The receipt records that a review of a specific diff happened and what it said; who accepts or rejects each finding is not Goalrail's record. | CTX-11 |
| NG-5 | No interactive prompt anywhere in Goalrail itself. The one permitted user question (OUT-4) belongs to the authoring agent's loop, not to any Goalrail command. | SE-5, SE-6 |
| NG-6 | Unbounded re-review loops. Every loop declares its round ceiling and stop condition before the first round, per the global operating rules; Goalrail provides the review step, not the orchestration. | SE-5 |
| NG-7 | Scope growth through the question channel. A mid-loop idea to extend, improve, or generalize is recorded as a candidate for a future change and neither asked about nor implemented in the current loop. The question channel exists for divergences and holes alone, and widening it is how a bounded loop becomes an unbounded one. | SE-6a |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The ordinary path asks nothing. | From a Claude Code session, one invocation with no flags selects Codex, runs the review, and leaves a receipt; zero prompts, zero questions, unchanged exit-status semantics. | OUT-1 |
| SIG-2 | The gate is honored and optional. | With a failing gate command configured, the invocation executes no review and the refusal names the gate; with no gate configured, the same invocation reviews. | OUT-2 |
| SIG-3 | The receipt cannot describe a different diff. | Recomputing the diff digest from the receipt's own base and head reproduces the recorded digest; the stored report is byte-identical to what the reviewer emitted. | OUT-5 |
| SIG-4 | Staleness needs nobody. | The diagnosis line moves current → stale on a new commit and stale → current on a fresh review, with no flag, no dismissal, and no change to exit status at any point. | OUT-6 |
| SIG-5 | The first real use closes a real loop. | The Baseline API extraction branch goes through review, in-loop fixes, and a current re-review receipt before its PR opens, with zero user interruptions on the clean path. | SE-4, OUT-3, OUT-4 |

## Ambiguities and Unknowns

None blocking, actively checked. Design-level decisions deliberately left to
design: where the receipt lives (the per-clone state directory is the natural
candidate), the instruction file's path and name, the exact round ceiling
default (the global rules require one to be declared; its number is not intent),
and how the review step reports "nothing material" without Goalrail interpreting
prose (candidate: the reviewer is instructed to end its report with a fixed
verdict marker, which remains the reviewer's statement rather than Goalrail's
parse — the receipt stores it either way).

## Version history

- **v3 (2026-08-01):** Supersedes v2 on the owner's refinement, before any
  confirmation of v2. Material change: the interruption exception in OUT-4 now
  has two triggers — a divergence from confirmed intent, and a real hole in the
  intent uncovered by implementation — with an explicit test separating a hole
  from scope creep, and NG-7 closing the question channel to improvements.
- **v2 (2026-08-01):** Supersedes v1 on the owner's correction, before any
  confirmation of v1. Material changes: the review runs inside the working loop
  by default instead of printing a command and waiting for consent (SE-5);
  authorship is inferred from the invoking environment instead of demanded from
  the caller (SE-7, CTX-13); the loop closes findings before the pull request
  opens with a declared ceiling (OUT-3); and the no-interruption rule with its
  single exception is now an outcome (OUT-4). v1 was never confirmed; nothing
  downstream consumed it.

## Confirmation

- **Confirmed by:** Vitaly D.
- **Confirmed at:** 2026-08-01
- **Verification action:** The owner corrected v1 (loop-first, inferred authorship) and v2 (the intent-hole trigger with its anti-scope test) from plain-language views in Russian, then confirmed the v3 view, which covered both exceptions, the scope guard, the budget gate, the receipt binding, and the measured interaction counts.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
