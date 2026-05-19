#!/usr/bin/env bash
# Fail if backend/migrations has duplicate numeric prefixes (e.g. two 018_*.sql files).
set -euo pipefail
shopt -s nullglob

dir="${1:-backend/migrations}"
dup=0
for f in "$dir"/*.sql; do
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
