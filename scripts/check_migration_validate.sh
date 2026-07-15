#!/usr/bin/env bash
# Local pre-commit check: every backend/migrations/*.sql (and atlas.sum) must
# replay cleanly on a fresh PostgreSQL 17 and match its declared hash.
#
# This is the same check that CI's db-migrate-lint.yml runs, run locally
# before commit so you don't find out your migration is broken at PR time.
# It does NOT connect to any live database.
#
# Requires on PATH: atlas, docker, go (the latter only for the rare first run
# that compiles atlas-provider-gorm; usually cached).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT}/backend"

echo "check_migration_validate: running atlas migrate validate on fresh PG17..." >&2
OUT="$(atlas migrate validate \
    --env migrate \
    --config file://atlas/atlas.hcl \
    --dev-url "docker://postgres/17/dev?search_path=public" 2>&1)" || true
echo "$OUT"

# atlas sometimes prints failures but exits 0; treat Error/FAIL as failure.
if echo "$OUT" | grep -qiE '^Error|^FAIL|hint: |Error: '; then
  cat >&2 <<'EOF'

migrations/ does not replay cleanly on fresh PG17 (or atlas.sum does not match).
  cat >&2 <<'EOF'

migrations/ does not replay cleanly on fresh PG17 (or atlas.sum does not match).

Fix locally:
  1. Re-read docs/atlas-migrations.md ("日常流程:加迁移")
  2. If you forgot to regenerate atlas.sum after editing a .sql file:
       cd backend && make db-migrate-hash
  3. If a new migration uses a new PostgreSQL extension, add a manual
     CREATE EXTENSION IF NOT EXISTS in the migration (atlas does not emit these).
  4. Re-run:
       cd backend && atlas migrate validate --env migrate --config file://atlas/atlas.hcl \
         --dev-url "docker://postgres/17/dev?search_path=public"
EOF
  exit 1
fi
echo "check_migration_validate: OK"
