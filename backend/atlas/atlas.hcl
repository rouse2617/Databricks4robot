# Atlas config for cyber-databrew.
#
# Two envs:
#   - `migrate`: hand-written SQL migrations. This is the current source of
#     truth. Used by the Cloud Run migration Job to apply pending changes.
#   - `gorm`:    GORM model source of truth. Used by `atlas migrate diff` to
#     generate new SQL migrations from Go-side schema changes (cyber-grace
#     pattern). Initially enabled for the `assets` table only (PoC); expand
#     to all 47 tables in subsequent migrations.
#
# URL resolution order (both envs):
#   1. `--url` flag (Makefile / CI) — always overrides.
#   2. `DB_URL`, if provided (full connection string).
#   3. Otherwise composed from DB_* env vars. DB password is embedded in
#      the URL (read from DB_PASSWORD, matching the cyber-databrew Cloud
#      Run service convention).
#   4. Otherwise empty — keeps no-DB commands (migrate hash / validate)
#      working.
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

# GORM env: schema source = registered models in backend/internal/dbschema.
# Used by `atlas migrate diff <name> --env gorm` to auto-generate a new SQL
# migration when a GORM model changes.
#
# The provider runs `ariga.io/atlas-provider-gorm` (prebuilt), pointing at
# the dbschema package. Models register themselves via dbschema.AllModels().
#
# `dev` is a docker-based dev DB Atlas spins up to validate the diff applies
# (matches cyber-grace exactly).
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/17/dev?search_path=public"

  migration {
    dir = "file://migrations"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/dbschema",
    "--dialect", "postgres"
  ]
}
