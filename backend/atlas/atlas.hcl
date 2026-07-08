# Atlas versioned-migration config for the cyber-databrew backend.
#
# We keep a single env ("migrate") because databrew does not use an ORM as the
# schema source of truth — migrations are hand-written SQL in backend/migrations/.
# URL resolution order matches cyber-grace's `migrate` env:
#
#   1. `--url` flag (Makefile / CI) — always overrides.
#   2. `DB_URL`, if provided (full connection string).
#   3. Otherwise composed from DB_* env vars. The password is NOT put in the
#      URL — the driver reads it from `DB_PASSWORD` (or `PGPASSWORD`) — so the
#      Cloud Run migration Job reuses the EXISTING password secret with no
#      duplication and no URL-encoding issues for special characters.
#   4. Otherwise empty — keeps no-DB commands (migrate hash / validate) working.
#
# databrew talks to private-IP CloudSQL over a Serverless VPC connector, so
# `sslmode=disable` is the default (see backend/internal/postgres/client.go).
env "migrate" {
  url = getenv("DB_URL") != "" ? getenv("DB_URL") : (
    getenv("DB_HOST") != "" ?
      "postgres://${getenv("DB_USER")}:${getenv("DB_PASSWORD")}@${getenv("DB_HOST")}:${getenv("DB_PORT")}/${getenv("DB_NAME")}?sslmode=disable"
      : ""
  )

  migration {
    dir = "file://migrations"
  }
}
