#!/usr/bin/env bash
set -euo pipefail

modules=(
  "apps/server"
  "apps/cli"
)

for module in "${modules[@]}"; do
  echo "==> go test: ${module}"
  (cd "${module}" && go test ./...)
done

if command -v staticcheck >/dev/null 2>&1; then
  for module in "${modules[@]}"; do
    echo "==> staticcheck: ${module}"
    (cd "${module}" && staticcheck ./...)
  done
else
  echo "staticcheck not found; skipping"
fi

if command -v golangci-lint >/dev/null 2>&1; then
  for module in "${modules[@]}"; do
    echo "==> golangci-lint: ${module}"
    (cd "${module}" && golangci-lint run ./...)
  done
else
  echo "golangci-lint not found; skipping"
fi
