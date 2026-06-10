#!/usr/bin/env bash
# Apply SQL migrations that may be missing when Postgres already existed before a new
# migration file was added (docker-entrypoint-initdb.d runs only on empty data volume).
#
# Usage:
#   bash backend/scripts/ensure_migrations.sh
# Env (when not using Docker postgres):
#   PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MIGRATIONS="$ROOT/migrations"

psql_exec() {
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'local-postgres-1'; then
    docker exec -i local-postgres-1 psql -U postgres -d cyber_databrew_dev -v ON_ERROR_STOP=1 "$@"
  else
    export PGPASSWORD="${PGPASSWORD:-postgres}"
    psql -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}" \
      -d "${PGDATABASE:-cyber_databrew_dev}" -v ON_ERROR_STOP=1 "$@"
  fi
}

psql_exec_file() {
  local file="$1"
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'local-postgres-1'; then
    # Host paths are not visible inside the container; pipe SQL via stdin.
    docker exec -i local-postgres-1 psql -U postgres -d cyber_databrew_dev -v ON_ERROR_STOP=1 -f - < "$file"
  else
    export PGPASSWORD="${PGPASSWORD:-postgres}"
    psql -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}" \
      -d "${PGDATABASE:-cyber_databrew_dev}" -v ON_ERROR_STOP=1 -f "$file"
  fi
}

psql_query() {
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'local-postgres-1'; then
    docker exec local-postgres-1 psql -U postgres -d cyber_databrew_dev -tAc "$1" | tr -d '[:space:]'
  else
    export PGPASSWORD="${PGPASSWORD:-postgres}"
    psql -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}" \
      -d "${PGDATABASE:-cyber_databrew_dev}" -tAc "$1" | tr -d '[:space:]'
  fi
}

apply_if_missing() {
  local check_sql="$1"
  local file="$2"
  local name
  name="$(basename "$file")"
  local ok
  ok="$(psql_query "$check_sql" || true)"
  if [ "$ok" = "t" ] || [ "$ok" = "1" ]; then
    echo "ensure_migrations: skip $name (already applied)"
    return 0
  fi
  echo "ensure_migrations: applying $name ..."
  psql_exec_file "$file"
  echo "ensure_migrations: applied $name"
}

# eval/metrics tables (archive/014)
apply_if_missing \
  "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'asset_metrics')" \
  "$MIGRATIONS/archive/014_add_eval_metrics.sql"

# actions table (archive/016)
apply_if_missing \
  "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'actions')" \
  "$MIGRATIONS/archive/016_add_actions.sql"

# saved_queries table (archive/026)
apply_if_missing \
  "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'saved_queries')" \
  "$MIGRATIONS/archive/026_add_saved_queries.sql"

# algo_runs table (archive/031, CYB-1018)
apply_if_missing \
  "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'algo_runs')" \
  "$MIGRATIONS/archive/031_algo_runs.sql"

echo "ensure_migrations: done"
