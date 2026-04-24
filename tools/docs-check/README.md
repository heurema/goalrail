# Goalrail Docs Check

This checker is the report-only docs-governance scaffold through PR3.

## Current guarantees

- report-only for live repository scans
- no CI hard gate yet
- no network calls
- no semantic or LLM judge
- no live metadata migration requirement
- live scans can read and validate `docs/ops/COMPONENTS.yaml`
- generated reports should not be committed

## Supported modes

### Live report-only scan

This mode scans the repository and always exits with `0` when the checker completes successfully, even if findings are present.
It uses the real current date by default.

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode report-only \
  --report-json /tmp/goalrail-docs-check-report.json \
  --report-md /tmp/goalrail-docs-check-report.md
```

### Changed-files ratchet

This mode checks only the supported changed files passed in a newline-separated repo-relative file list.

- missing or deleted paths are ignored safely
- unsupported paths such as tool or schema files are ignored by this mode
- hard findings in changed supported files return `1`
- warnings in changed files do not fail the run
- it uses the real current date by default

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode changed-files \
  --changed-files-file /tmp/changed-doc-files.txt \
  --report-json /tmp/goalrail-docs-check-report.json \
  --report-md /tmp/goalrail-docs-check-report.md
```

### Fixture self-test

This mode validates the golden fixtures under `tools/docs-check/fixtures/` and exits with `1` when actual fixture output differs from `expected.json`.
When `--today` is omitted, fixture self-test uses the frozen evaluation date `2026-04-20` so lifecycle fixtures stay deterministic as calendar time moves forward.

```bash
python3 tools/docs-check/docs_check.py \
  --fixtures tools/docs-check/fixtures \
  --self-test \
  --report-json /tmp/goalrail-docs-fixtures-report.json \
  --report-md /tmp/goalrail-docs-fixtures-report.md
```

Optional override:

```bash
python3 tools/docs-check/docs_check.py \
  --fixtures tools/docs-check/fixtures \
  --self-test \
  --today 2026-04-21
```

## Exit codes

- `0` = checker completed successfully
- `1` = fixture self-test mismatch, or hard findings in changed-files mode
- `2` = checker config or internal error

Important:
- live report-only scans do **not** return `1` just because findings exist
- repo-wide report-only scans are still not a hard gate

## Current checks

- frontmatter structure for fixture docs
- repo-relative Markdown links
- forbidden local absolute paths
- lifecycle enum and review-after checks
- authority checks for fixture docs
- claims skeleton checks using fixture-local synthetic status inputs
- live `docs/ops/COMPONENTS.yaml` shape, status enum, truth-owner paths, and implementation-path existence
- changed-files ratchet for supported changed docs only

## Current non-goals

- no live hard enforcement
- no repo-wide hard gate in CI
- no external link checking
- no semantic interpretation of prose

## CI posture

- pull requests may run fixture self-test plus changed-files mode only
- legacy repo-wide violations remain report-only outside the changed file list
- generated reports should stay in logs or temporary files, not committed into the repo
