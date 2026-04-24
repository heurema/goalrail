#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

changed_files="$tmpdir/changed-files.txt"
report_json="$tmpdir/goalrail-docs-check-report.json"
report_md="$tmpdir/goalrail-docs-check-report.md"

git -C "$root" diff --cached --name-only --diff-filter=ACMRT > "$changed_files"

if [[ ! -s "$changed_files" ]]; then
  echo "No staged files to check."
  exit 0
fi

set +e
python3 "$root/tools/docs-check/docs_check.py" \
  --root "$root" \
  --mode changed-files \
  --changed-files-file "$changed_files" \
  --report-json "$report_json" \
  --report-md "$report_md"
status=$?
set -e

cat "$report_md"
exit "$status"
