#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  scripts/local-refresh-plan.sh [--base <git-ref>] [--] [path ...]

Print non-destructive local component refresh guidance for Goalrail dogfood
changes. When paths are supplied, they are classified directly. With --base,
changed paths are read from git diff --name-only <base>...HEAD plus local
working-tree and staged changes. With no paths and no --base, local staged and
unstaged changes are used.

This helper does not apply migrations, restart servers, run workers, mutate
Goalrail state, or inspect secret environment values.
USAGE
}

base=""
paths=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --base)
      if [ "$#" -lt 2 ]; then
        echo "error: --base requires a git ref" >&2
        exit 2
      fi
      base="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    --*)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      break
      ;;
  esac
done

if [ "$#" -gt 0 ]; then
  paths=$(printf '%s\n' "$@")
elif [ -n "$base" ]; then
  paths=$(
    {
      git diff --name-only "$base"...HEAD
      git diff --name-only
      git diff --cached --name-only
    } | sort -u
  )
else
  paths=$(
    {
      git diff --name-only
      git diff --cached --name-only
    } | sort -u
  )
fi

has_cli=0
has_server=0
has_migration=0
has_worker=0
has_docs=0
has_scripts=0
has_other=0

if [ -n "$paths" ]; then
  while IFS= read -r path; do
    [ -n "$path" ] || continue
    case "$path" in
      apps/cli/*)
        has_cli=1
        ;;
      apps/server/internal/postgres/migrations/*|apps/server/migrations/*)
        has_migration=1
        ;;
      apps/server/*)
        has_server=1
        ;;
      apps/worker/*)
        has_worker=1
        ;;
      docs/*|README.md|AGENTS.md|*.md)
        has_docs=1
        ;;
      scripts/*)
        has_scripts=1
        ;;
      *)
        has_other=1
        ;;
    esac
  done <<EOF
$paths
EOF
fi

echo "Goalrail local refresh plan"
echo

if [ -z "$paths" ]; then
  echo "Changed paths: none detected"
  echo
  echo "Refresh recommendation: no runtime refresh detected from git diff."
  echo "Validation: run the checks relevant to the task before committing."
  exit 0
fi

echo "Changed paths:"
printf '%s\n' "$paths" | sed 's/^/- /'
echo

echo "Affected components:"
[ "$has_cli" -eq 1 ] && echo "- cli"
[ "$has_server" -eq 1 ] && echo "- server"
[ "$has_migration" -eq 1 ] && echo "- server_migrations"
[ "$has_worker" -eq 1 ] && echo "- worker"
[ "$has_docs" -eq 1 ] && echo "- docs"
[ "$has_scripts" -eq 1 ] && echo "- scripts"
[ "$has_other" -eq 1 ] && echo "- other"
echo

echo "Recommended refresh actions:"
if [ "$has_migration" -eq 1 ]; then
  echo "- Apply server migrations before validating server behavior that depends on schema changes."
fi
if [ "$has_server" -eq 1 ]; then
  echo "- Rebuild and restart the local server from current source before live validation."
fi
if [ "$has_cli" -eq 1 ]; then
  echo "- Use current-source CLI via go run from apps/cli, or rebuild the CLI before using an installed binary."
fi
if [ "$has_worker" -eq 1 ]; then
  echo "- Rerun the planning worker from current source for one-shot worker validation, or rebuild the worker binary before use."
fi
if [ "$has_docs" -eq 1 ]; then
  echo "- No runtime refresh is normally needed for docs-only changes; run docs/repo checks as appropriate."
fi
if [ "$has_scripts" -eq 1 ]; then
  echo "- Rerun the changed script or its documented checks with representative inputs."
fi
if [ "$has_other" -eq 1 ]; then
  echo "- Review uncategorized paths and choose component-specific validation before live dogfood claims."
fi
echo

echo "Suggested validation order:"
if [ "$has_migration" -eq 1 ]; then
  echo "1. Apply migrations with runtime-owned DB credentials redacted from output."
else
  echo "1. Confirm whether migrations are unchanged."
fi
if [ "$has_server" -eq 1 ]; then
  echo "2. Restart server from current source, then check /livez, /readyz, and /version."
else
  echo "2. Reuse existing server only if server code and migrations are unchanged."
fi
if [ "$has_cli" -eq 1 ]; then
  echo "3. Run current-source CLI commands from apps/cli with go run."
else
  echo "3. Avoid stale installed binaries when CLI behavior matters."
fi
if [ "$has_worker" -eq 1 ]; then
  echo "4. Run the worker from current source only when the current flow allows it."
else
  echo "4. Do not rerun workers unless worker behavior is in scope."
fi
if [ "$has_docs" -eq 1 ] || [ "$has_scripts" -eq 1 ]; then
  echo "5. Run docs/script checks for changed operational guidance or helper behavior."
else
  echo "5. Run task-specific checks and git diff hygiene checks."
fi
echo

echo "Safety notes:"
echo "- This helper is dry-run guidance only."
echo "- It does not restart servers, apply migrations, run workers, or mutate Goalrail state."
echo "- Do not print or commit .goalrail/project.yml, auth files, token material, DB passwords, JWT secrets, wrapper contents, private host details, or personal machine paths."
