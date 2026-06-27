# 02 Source Map

| id | source | type | mode | relevance |
|---|---|---|---|---|
| S01 | `/Users/vi/personal/heurema/goalrail/scripts/install_oss.sh` | code | local | Installer flags, fatal/warning helpers, Goalrail install sequence |
| S02 | `/Users/vi/personal/heurema/goalrail/tests/scripts/test_install_oss.py` | test | local | Existing shell-library test pattern for installer changes |
| S03 | `/Users/vi/personal/heurema/goalrail/goalrail/cli.py` | code | local | `goalrail setup` command and setup side-effect location |
| S04 | `/Users/vi/personal/heurema/goalrail/goalrail/onboarding/sandboxes/base.py` | code | local | Remote host image and `pip --no-deps` wheel overlay behavior |
| S05 | `/Users/vi/personal/heurema/codebase-memory-mcp/pkg/pypi/pyproject.toml` | packaging | local | PyPI package name and console script |
| S06 | `/Users/vi/personal/heurema/codebase-memory-mcp/pkg/pypi/src/codebase_memory_mcp/_cli.py` | code | local | Wrapper downloads GitHub release binary on first run |
| S07 | `/Users/vi/personal/heurema/codebase-memory-mcp/src/cli/cli.c` | code | local | `install`, `--plan`, `-y`, config/index mutation behavior |
| S08 | `/Users/vi/personal/heurema/codebase-memory-mcp/README.md` | docs | local | Documented install and auto-index behavior |
| S09 | `provider_router.py doctor --probe` | tool result | local | Available critic routes and agy limitation |
| S10 | `claude-sonnet` proposer pass | provider opinion | live | Positive implementation path |
| S11 | `vibe-default` skeptic pass | provider opinion | live | Failure-mode critique |

## Excluded sources

- External web docs: not needed for this local architecture decision.
- Goalrail code implementation edits: out of scope for this run.

## Project config sources

| Source ID | Source | Mode | Trust | Watch | Status |
|---|---|---|---|---|---|

## Primary sources

| Priority | Source | Mode | Why it matters | Status |
|---|---|---|---|---|

## Secondary sources

| Priority | Source | Mode | Why it matters | Status |
|---|---|---|---|---|

## External source pass

List the external sources checked before synthesis. Include official docs, public repos, product docs, customer-facing pages, market examples, or other primary sources when relevant.

If no external sources are used, write the reason here and mark the run as `internal-only` in the synthesis and critic review.

| Priority | Source | Mode | What it can support | Status |
|---|---|---|---|---|

## Social / recency signals

| Source | Mode | What it can support | Limitations |
|---|---|---|---|

## Sources to avoid or treat carefully

| Source | Reason |
|---|---|
