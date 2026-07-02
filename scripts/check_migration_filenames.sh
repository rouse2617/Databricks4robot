#!/usr/bin/env bash
# Fail if backend/migrations has duplicate numeric prefixes (e.g. two 018_*.sql files).
#
# Path resolution is anchored to this script's location (not the caller's cwd),
# so it works the same whether invoked from the repo root or from backend/.
# Historically this defaulted to a cwd-relative "backend/migrations", which
# silently resolved to a non-existent "backend/backend/migrations" when CI ran
# it with `working-directory: backend` — nullglob made the empty dir look
# "clean" and the check passed without ever inspecting a single file.
set -euo pipefail
shopt -s nullglob

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)/backend/migrations"

dir="${1:-$DEFAULT_DIR}"

if [ ! -d "$dir" ]; then
  echo "check_migration_filenames: directory does not exist: $dir" >&2
  exit 1
fi

sql_files=("$dir"/*.sql)
if [ "${#sql_files[@]}" -eq 0 ]; then
  echo "check_migration_filenames: no .sql files found in $dir — refusing to pass silently" >&2
  exit 1
fi

dup=0
for f in "${sql_files[@]}"; do
  base=$(basename "$f")
  prefix="${base%%_*}"
  matches=$(find "$dir" -maxdepth 1 -name "${prefix}_*.sql" | wc -l | tr -d ' ')
  if [ "$matches" -gt 1 ]; then
    echo "duplicate migration prefix $prefix in $dir" >&2
    find "$dir" -maxdepth 1 -name "${prefix}_*.sql" >&2
    dup=1
  fi
done
exit "$dup"
