.PHONY: all run run-server build test fmt vet tidy deps bt-bootstrap migrate-local clean swagger db-migrate-apply db-migrate-status db-migrate-hash db-migrate-hash-check

SERVER_BIN := bin/server

all: build

build:
	go build -o $(SERVER_BIN) ./cmd/server

run: run-server

run-server:
	go run ./cmd/server

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

deps:
	go get github.com/gin-gonic/gin@latest
	go get github.com/golang-jwt/jwt/v5@latest
	go get github.com/google/uuid@latest
	go get github.com/joho/godotenv@latest
	go get github.com/stretchr/testify@latest
	go get github.com/jackc/pgx/v5/pgxpool@latest
	go get cloud.google.com/go/bigtable@latest
	go get cloud.google.com/go/storage@latest
	go get cloud.google.com/go/pubsub@latest
	go mod tidy

bt-bootstrap:
	bash ./scripts/bootstrap_bigtable.sh

# Replay incremental SQL (004–012) on an existing DB; see scripts/apply_pg_deltas.sh.
migrate-local:
	bash ./scripts/apply_pg_deltas.sh

# ---- Atlas versioned migrations ----------------------------------------------
# db-migrate-* targets use Atlas (https://atlasgo.io) for hash / validate / apply.
# They coexist with the legacy bash-based `migrate-local` target above; both
# write to independent tracking tables (atlas_schema_revisions vs schema_migrations).
# New migrations should be applied via these targets; `migrate-local` is kept as
# a rescue path for environments that pre-date the Atlas baseline.

ATLAS ?= atlas
ATLAS_CONFIG ?= atlas/atlas.hcl
ATLAS_MIGRATE_ENV ?= migrate
ATLAS_COMMON_FLAGS ?= --config $(ATLAS_CONFIG)

db-migrate-hash:
	$(ATLAS) migrate hash $(ATLAS_COMMON_FLAGS) --env $(ATLAS_MIGRATE_ENV)

db-migrate-hash-check:
	$(ATLAS) migrate validate $(ATLAS_COMMON_FLAGS) --env $(ATLAS_MIGRATE_ENV)

db-migrate-apply:
	@if [ -z "$(DB_URL)" ]; then \
		echo "ERROR: DB_URL is required. Usage: make db-migrate-apply DB_URL=postgres://HOST:5432/db?sslmode=disable"; \
		exit 1; \
	fi
	$(ATLAS) migrate apply $(ATLAS_COMMON_FLAGS) --env $(ATLAS_MIGRATE_ENV) --url "$(DB_URL)"

db-migrate-status:
	@if [ -z "$(DB_URL)" ]; then \
		echo "ERROR: DB_URL is required. Usage: make db-migrate-status DB_URL=postgres://HOST:5432/db?sslmode=disable"; \
		exit 1; \
	fi
	$(ATLAS) migrate status $(ATLAS_COMMON_FLAGS) --env $(ATLAS_MIGRATE_ENV) --url "$(DB_URL)"

clean:
	rm -rf bin/

swagger:
	swag init -g cmd/server/main.go -o docs/swagger
