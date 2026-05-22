#!/usr/bin/env bash
# Apply one backend/migrations/*.sql file to dev PostgreSQL (private Cloud SQL via GKE tool pod).
#
# Usage:
#   bash scripts/apply-migration-dev.sh backend/migrations/029_customers.sql
#   K8S_NAMESPACE=cyber-databrew-dev bash scripts/apply-migration-dev.sh 028_asset_versioning.sql
#
# Requires: kubectl (context with cyber-databrew-dev), gcloud (for optional secret fallback)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NAMESPACE="${K8S_NAMESPACE:-cyber-databrew-dev}"
SECRET_NAME="${K8S_SECRET:-cyber-databrew-secrets}"
PGHOST="${DEV_PG_HOST:-172.27.160.7}"
PGPORT="${DEV_PG_PORT:-5432}"
PGUSER="${DEV_PG_USER:-postgres}"
PGDATABASE="${DEV_PG_DATABASE:-cyber_databrew_dev}"
POD_PREFIX="${MIGRATE_POD_PREFIX:-pg-migrate}"

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <migration.sql|NNN_name.sql>" >&2
  exit 1
fi

ARG="$1"
if [[ -f "$ARG" ]]; then
  MIGRATION="$ARG"
elif [[ -f "$ROOT/backend/migrations/$ARG" ]]; then
  MIGRATION="$ROOT/backend/migrations/$ARG"
else
  echo "migration not found: $ARG" >&2
  exit 1
fi

case "$MIGRATION" in
  "$ROOT"/backend/migrations/*.sql) ;;
  *)
    echo "refusing path outside backend/migrations/: $MIGRATION" >&2
    exit 1
    ;;
esac

BASENAME="$(basename "$MIGRATION" .sql)"
# K8s names: lowercase RFC 1123 (no underscores).
POD_SUFFIX="$(echo "$BASENAME" | tr '[:upper:]' '[:lower:]' | tr '_' '-')"
POD="${POD_PREFIX}-${POD_SUFFIX}"

cleanup() {
  kubectl -n "$NAMESPACE" delete pod "$POD" --ignore-not-found=true >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> migration: $MIGRATION"
echo "==> namespace: $NAMESPACE  database: $PGDATABASE @ $PGHOST"

kubectl -n "$NAMESPACE" delete pod "$POD" --ignore-not-found=true >/dev/null 2>&1
kubectl -n "$NAMESPACE" run "$POD" \
  --image=postgres:17 \
  --restart=Never \
  --overrides="$(cat <<EOF
{"spec":{"containers":[{"name":"$POD","image":"postgres:17","command":["sleep","600"],"env":[{"name":"PGHOST","value":"$PGHOST"},{"name":"PGPORT","value":"$PGPORT"},{"name":"PGUSER","value":"$PGUSER"},{"name":"PGDATABASE","value":"$PGDATABASE"},{"name":"PGPASSWORD","valueFrom":{"secretKeyRef":{"name":"$SECRET_NAME","key":"DB_PASSWORD"}}}]}],"restartPolicy":"Never"}}
EOF
)"
kubectl -n "$NAMESPACE" wait --for=condition=Ready "pod/$POD" --timeout=120s

REMOTE="/tmp/${BASENAME}.sql"
kubectl -n "$NAMESPACE" cp "$MIGRATION" "$POD:$REMOTE"
kubectl -n "$NAMESPACE" exec "$POD" -- psql -v ON_ERROR_STOP=1 -f "$REMOTE"

echo "==> apply finished OK: $(basename "$MIGRATION")"
