#!/usr/bin/env bash
# event_retention.sh — Clean up old published asset_events.
# Usage: ./event_retention.sh [RETENTION_DAYS]
#   Default RETENTION_DAYS = 90
#
# Requires: psql, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME env vars.

set -euo pipefail

RETENTION_DAYS="${1:-90}"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-data_platform}"

export PGPASSWORD="${DB_PASSWORD}"

echo "🗑️  Running event retention cleanup (${RETENTION_DAYS} days)..."

DELETED=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -A -c \
  "SELECT event_retention_cleanup('${RETENTION_DAYS} days'::interval);")

echo "✅ Deleted ${DELETED} old published events."
