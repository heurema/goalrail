# 07 Ideas

## Implementation options

### A. Minimal installer companion

Install the wrapper and force first-run binary download with `--version`; set
`auto_index=true`; skip full `install -y`.

Pros:
- small diff;
- low risk;
- Goalrail install remains reliable;
- no hidden index deletion.

Cons:
- MCP agent config may not be repaired automatically on machines where CBM was
  not previously installed.

### B. Installer companion plus interactive repair

Do A, then in interactive mode run `install --plan`, summarize detected agents,
and ask before `install -y`.

Pros:
- closer to "works immediately";
- preserves explicit consent.

Cons:
- more installer complexity;
- tests need stubs for plan/prompt branches;
- non-interactive path still needs a skip/manual instruction.

### C. Setup-managed repair

Keep installer to binary + auto_index only. Add `goalrail setup` check/repair
because setup is already a user-visible configuration flow.

Pros:
- clearer mental model;
- better place for prompt text and plan display;
- avoids bloating shell installer.

Cons:
- users must run setup or first-run flow before full repair.

### D. Upstream CBM repair mode

Add or wait for a CBM command that refreshes agent configs without index
deletion prompts, then call that from Goalrail.

Pros:
- safest long-term command surface;
- avoids Goalrail parsing CBM internals.

Cons:
- requires upstream change before full automatic repair.

## Preferred combination

First PR: A + C.

Second PR or upstream prerequisite: D.

Optional follow-up: B only if UX needs full local agent repair during the
installer run.

## Candidate ideas

| ID | Idea | Evidence rows | Confidence | Next step |
|---|---|---|---|---|

## Rejected ideas

| Idea | Reason |
|---|---|

## Open idea questions
