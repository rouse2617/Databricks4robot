#!/usr/bin/env bash
# Apply incremental Postgres deltas to an *existing* database.
#
# Old incremental migrations (001–038) have been squashed into 000_initial.sql
# and archived in migrations/archive/. This script applies only new deltas
# committed after the squash.
#
# Usage:
#   DATABASE_URL=postgresql://user:pass@host:5432/db bash scripts/apply_pg_deltas.sh
#   APPLY_CF_LEGACY_DROP=1 ...   # also run 013_drop_cf_legacy_columns.sql (from archive)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS="$(cd "${SCRIPT_DIR}/../migrations" && pwd)"
ARCHIVE="${MIGRATIONS}/archive"
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"
PGDATABASE="${PGDATABASE:-cyber_databrew_dev}"
DB_URL="${DATABASE_URL:-postgresql://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${PGDATABASE}}"

ensure_migrations_table() {
  psql "$DB_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  migration_name TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL
}

migration_applied() {
  local name="$1"
  [[ "$(psql "$DB_URL" -v ON_ERROR_STOP=1 -v migration="$name" -Atqc "SELECT 1 FROM schema_migrations WHERE migration_name = :'migration'")" == "1" ]]
}

record_migration() {
  local name="$1"
  psql "$DB_URL" -v ON_ERROR_STOP=1 -v migration="$name" -c "INSERT INTO schema_migrations (migration_name) VALUES (:'migration') ON CONFLICT (migration_name) DO NOTHING"
}

apply_one() {
  local f="$1"
  local name
  name="$(basename "$f")"
  if migration_applied "$name"; then
    echo "==> $name (already applied, skip)"
    return
  fi
  echo "==> $name"
  psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$f"
  record_migration "$name"
}

ensure_migrations_table

shopt -s nullglob
deltas=()
for f in "$MIGRATIONS"/*.sql; do
  b="$(basename "$f")"
  case "$b" in
    000_initial.sql) continue ;;
    *) deltas+=("$f") ;;
  esac
done

IFS=$'\n' sorted="$(printf '%s\n' "${deltas[@]}" | sort)"
unset IFS

while IFS= read -r f; do
  [[ -n "$f" ]] || continue
  apply_one "$f"
done <<< "$sorted"

if [[ "${APPLY_CF_LEGACY_DROP:-}" == "1" ]]; then
  apply_one "$ARCHIVE/013_drop_cf_legacy_columns.sql"
else
  echo "==> (skip 013_drop_cf_legacy_columns.sql — set APPLY_CF_LEGACY_DROP=1 to apply)"
fi

echo "Done."
