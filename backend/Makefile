.PHONY: all run run-server build test fmt vet tidy deps bt-bootstrap migrate-local clean swagger

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

clean:
	rm -rf bin/

swagger:
	swag init -g cmd/server/main.go -o docs/swagger
