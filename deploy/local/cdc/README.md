# Local CDC Stack

This directory holds the local infrastructure scaffold for the long-term
CDC/WAL architecture branch.

## Components

- `postgres`: PostgreSQL with `wal_level=logical`
- `redpanda`: lightweight Kafka-compatible bus
- `connect`: Debezium Kafka Connect runtime

## Start

```bash
cd /Users/rick/Databricks4robot-longterm-sync/deploy/local
docker compose -f docker-compose.cdc.yml up -d
```

## Endpoints

- PostgreSQL: `localhost:5432`
- Redpanda Kafka bootstrap: `localhost:19092`
- Redpanda HTTP proxy: `localhost:18082`
- Debezium Connect REST: `http://localhost:8084`

## Notes

- This stack is intentionally separate from the current MVP stack.
- It is for long-term CDC experiments and adapter work.
- The branch does not yet auto-register connectors; connector JSON lives under
  `deploy/local/cdc/connectors/`.

## Connector registration example

Register the `asset_events` connector:

```bash
curl -sS -X POST http://localhost:8084/connectors \
  -H 'Content-Type: application/json' \
  --data @/Users/rick/Databricks4robot-longterm-sync/deploy/local/cdc/connectors/postgres-asset-events.json
```

Register the current-state tables connector:

```bash
curl -sS -X POST http://localhost:8084/connectors \
  -H 'Content-Type: application/json' \
  --data @/Users/rick/Databricks4robot-longterm-sync/deploy/local/cdc/connectors/postgres-current-state.json
```

## Relationship to `backend/internal/cdc`

The app-side skeleton currently provides:

- `DecodeDebeziumMessage(...)`
- `KafkaSource`
- `BronzeConsumer`
- `ESConsumer`

What is still missing:

- a real Kafka client implementation for `KafkaPoller`
- connector bootstrapping automation
- production checkpoint / metrics / retry policy
