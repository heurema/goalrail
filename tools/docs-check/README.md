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

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode report-only \
  --report-json /tmp/goalrail-docs-check-report.json \
  --report-md /tmp/goalrail-docs-check-report.md
```

### Fixture self-test

This mode validates the golden fixtures under `evals/cases/docs/` and exits with `1` when actual fixture output differs from `expected.json`.

```bash
python3 tools/docs-check/docs_check.py \
  --fixtures evals/cases/docs \
  --self-test \
  --report-json /tmp/goalrail-docs-fixtures-report.json \
  --report-md /tmp/goalrail-docs-fixtures-report.md
```

## Exit codes

- `0` = checker completed successfully
- `1` = fixture self-test mismatch, or a future explicit fail-on-hard mode
- `2` = checker config or internal error

Important:
- live report-only scans do **not** return `1` just because findings exist
- PR3 is still not a hard gate

## Current checks

- frontmatter structure for fixture docs
- repo-relative Markdown links
- forbidden local absolute paths
- lifecycle enum and review-after checks
- authority checks for fixture docs
- claims skeleton checks using fixture-local synthetic status inputs
- live `docs/ops/COMPONENTS.yaml` shape, status enum, truth-owner paths, and implementation-path existence

## Current non-goals

- no live hard enforcement
- no CI integration
- no external link checking
- no semantic interpretation of prose

## Future direction

- PR4: local/CI changed-files ratchet
