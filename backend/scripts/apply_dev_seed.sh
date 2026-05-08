#!/usr/bin/env bash
# Apply optional demo rows from backend/scripts/postgres/dev_seed.sql
# (moved out of docker-entrypoint-initdb.d so empty volumes start with schema only).
#
# Usage:
#   bash backend/scripts/apply_dev_seed.sh
# Env (non-Docker):
#   PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SEED="$ROOT/scripts/postgres/dev_seed.sql"

psql_query() {
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'local-postgres-1'; then
    docker exec local-postgres-1 psql -U postgres -d data4cyber -tAc "$1" | tr -d '[:space:]'
  else
    export PGPASSWORD="${PGPASSWORD:-postgres}"
    psql -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}" \
      -d "${PGDATABASE:-data4cyber}" -tAc "$1" | tr -d '[:space:]'
  fi
}

ok="$(psql_query "SELECT EXISTS (SELECT 1 FROM assets WHERE asset_id = 'aset0001')" || true)"
if [ "$ok" = "t" ] || [ "$ok" = "1" ]; then
  echo "apply_dev_seed: skip — demo asset aset0001 already present"
  exit 0
fi

echo "apply_dev_seed: applying $SEED ..."
if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'local-postgres-1'; then
  # Host path is not visible inside the container; stream SQL on stdin.
  cat "$SEED" | docker exec -i local-postgres-1 psql -U postgres -d data4cyber -v ON_ERROR_STOP=1
else
  export PGPASSWORD="${PGPASSWORD:-postgres}"
  psql -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}" \
    -d "${PGDATABASE:-data4cyber}" -v ON_ERROR_STOP=1 -f "$SEED"
fi
echo "apply_dev_seed: done"
