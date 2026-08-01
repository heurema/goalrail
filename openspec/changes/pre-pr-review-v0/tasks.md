## 1. Decide the author, or refuse

- [ ] 1.1 Enumerate the legal invoking environments before writing any detection: exactly one provider's primary session marker, two, none, and an explicit override. Every one gets a decided outcome; there is no default branch. This is the promoted `input-space` discipline applied to the first thing the command does.
- [ ] 1.2 Detect a Claude Code session by `CLAUDECODE` and a Codex session by its own `CODEX_*` session marker. Match the **primary session marker only** — measured on this machine, a Claude Code session also carried `CODEX_COMPANION_SESSION_ID`, so treating any `CODEX_*` variable as evidence would make the first real invocation choose the author as its own reviewer.
- [ ] 1.3 Refuse on two primary markers or none: state what was seen and name the override. Refusing is the requirement, not a fallback — a silent wrong choice here fails invisibly.
- [ ] 1.4 Add the authorship override and give it precedence over detection unconditionally.
- [ ] 1.5 Test all four environments, including the two-marker case built from the real variable names, and assert no code path picks a provider without evidence.

## 2. Select the reviewer from what is installed

- [ ] 2.1 Read the installed and active scaffolds from `internal/ambient` rather than introducing a second notion of what is available; initialization and the diagnosis already report exactly this (CTX-5).
- [ ] 2.2 Select as reviewer an installed provider that is not the author's. Where none exists, refuse and name the missing side.
- [ ] 2.3 Test a repository where only the author's provider is installed and assert it refuses rather than reviewing with the author's own provider.

## 3. Run the budget gate first

- [ ] 3.1 Add a configuration key naming a gate command. It is a command, never a path, provider, or budget service compiled in — the workspace's personal guard belongs in one user's configuration, not in a released binary.
- [ ] 3.2 Run the gate before invoking any reviewer; a non-zero exit refuses the review and the refusal names the gate as the reason.
- [ ] 3.3 No gate configured means the review proceeds. Do not report a missing gate as a condition; turning absence into a refusal would reintroduce the consent step the intent removes.
- [ ] 3.4 Test all three: gate fails, gate passes, no gate.

## 4. Assemble the instructions as repository content

- [ ] 4.1 Read the instructions from a committed dot-prefixed file at the repository root. It must not sit under `.goalrail/`, which Goalrail's own ignore rule excludes from version control — instructions that cannot be committed cannot be shared.
- [ ] 4.2 Materialize a default when the file is absent, and never overwrite an existing one.
- [ ] 4.3 Write the default to carry what the ratchet has already promoted: check the diff against the repository's own `AGENTS.md`, and challenge every claim of absence, count, or consistency that ships without its check.
- [ ] 4.4 Have the default ask the reviewer to end its report with a fixed marker stating whether anything material remains. It is the reviewer's statement, stored verbatim; Goalrail must not require, validate, or act on it, and a report without it must still produce a valid receipt.
- [ ] 4.5 Test that an edited file is used verbatim and is unmodified afterwards, and that materialization happens exactly once.

## 5. Invoke the reviewer through its own interface

- [ ] 5.1 Invoke `codex review --base <ref>` with the instructions on stdin, and `claude -p` with the instructions and a bounded tool surface. Minimum documented flags only: every flag is a surface that can drift, and neither CLI is pinnable.
- [ ] 5.2 Surface a non-zero exit or unusable invocation to the caller as the reviewer's own failure. Do not retry with altered arguments and do not translate the vendor's message — a wrapper turns a loud break into a quiet one.
- [ ] 5.3 Write no receipt when the reviewer did not complete. A receipt for a review that did not happen is worse than none.
- [ ] 5.4 Test both providers against a stub that exits non-zero and assert no receipt appears.

## 6. Write a receipt that measures rather than reads

- [ ] 6.1 Render the reviewed range as a canonical diff — no external diff drivers, no colour, no textconv, full index — so the same range digests identically on any machine. The staleness comparison in section 7 is only meaningful if this is stable.
- [ ] 6.2 Record base commit, head commit, diff digest, reviewer identity, time, the report as verbatim bytes, and a digest of those bytes.
- [ ] 6.3 Derive no field from the report's content. Not a summary, not a count, not a severity, not the marker from 4.4. Goalrail stores what was said and what was reviewed; deciding what is real belongs to the loop's author.
- [ ] 6.4 Write the receipt to the per-clone state root (`internal/localrun` already resolves it) and never into the repository. A receipt someone else receives by pulling would assert a review of a diff they may not have.
- [ ] 6.5 Test that recomputing the digest from the receipt's own base and head reproduces the recorded value, that the stored report is byte-identical, and that the repository is unchanged apart from a first-use instructions file.

## 7. Report the review state in the diagnosis

- [ ] 7.1 Add one diagnosis line for the current branch: current where the recomputed diff digest matches the receipt's, stale where it does not, absent where there is no receipt (`internal/harness/doctor.go`).
- [ ] 7.2 Report it as a fact, not a fault: no effect on the overall verdict, no effect on exit status, no next action. Follow the observability line already in that report, which exists for this shape.
- [ ] 7.3 Derive the state from the digest comparison alone — no flag, no acknowledgement command, no stored dismissal. A commit makes it stale by itself and a fresh review makes it current by itself.
- [ ] 7.4 Never reprint the report; the receipt holds it verbatim for whoever wants it.
- [ ] 7.5 Test the full sequence on one branch: review → current, commit → stale, re-review → current, with exit status unchanged throughout.

## 8. Prove the loop on the work that motivated it

- [ ] 8.1 Run the whole path on the Baseline API extraction branch before its pull request opens: review, fix what is real in the same session, re-review to a current receipt. Record the receipts as evidence.
- [ ] 8.2 Declare the round ceiling and the stop condition before the first round, and report reaching the ceiling with findings still open as the loop's honest result rather than absorbing it.
- [ ] 8.3 Count user interactions across the run and assert zero on the clean path.
- [ ] 8.4 Assert that initialization, review, and diagnosis produce unchanged machine-readable shape and exit status with no terminal attached, so nothing here became interactive.

## 9. v4: modes, incremental rounds, refute (supersedes 1.3/2.2-2.3 refusal semantics)

- [ ] 9.1 Fresh mode: with only the author's provider runnable, review with it in a clean session instead of refusing. The single refusal left is "no reviewer runs at all". Record mode (fresh/cross) and the selection reason in the receipt; a cross downgraded to fresh because the other CLI would not run must say so.
- [ ] 9.2 Unknown author: undetectable authorship proceeds, records `author: unknown`, and picks the reviewer deterministically. The override still wins.
- [ ] 9.3 Receipt v4 fields: mode, mode_reason, duration_seconds, reviewed base/head (this round) alongside the full branch digest used for staleness.
- [ ] 9.4 Incremental by default: where a receipt exists, the new round's base is the previous receipt's head; `--full` reviews the whole branch. The chain of receipts proves cumulative coverage.
- [ ] 9.5 Refute round (`--refute`): a fresh session — the other runnable provider, else the same — receives the previous report and the diff with refuter instructions from the same committed file; both reports stored verbatim, each with its own digest. The gate runs again for it.
- [ ] 9.6 Tests for each mode, the fallback reason, unknown author, incremental base selection, and the refuted receipt carrying two verbatim reports.
