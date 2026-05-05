# Local CDC Stack

This directory holds the local infrastructure scaffold for the long-term
CDC/WAL architecture. It includes Debezium connector configurations and a
registration script for the local development environment.

## Components

- `postgres`: PostgreSQL with `wal_level=logical`
- `redpanda`: lightweight Kafka-compatible bus
- `connect`: Debezium Kafka Connect runtime (exposes REST API on port 8083)

## Connectors

### `connectors/postgres-asset-events.json`

Captures every INSERT/UPDATE/DELETE on `public.asset_events` and publishes
change events to the `dataplatform.public.asset_events` Kafka topic.

| Setting | Value |
|---------|-------|
| Replication slot | `debezium_asset_events` |
| Publication | `debezium_asset_events_pub` |
| Topic prefix | `dataplatform` |
| Plugin | `pgoutput` |

### `connectors/postgres-current-state.json`

Captures changes on the four current-state tables and publishes them to
individual topics under the `dataplatform` prefix.

| Setting | Value |
|---------|-------|
| Tables | `public.assets`, `public.asset_tags`, `public.asset_algo_latest`, `public.mcap_files` |
| Replication slot | `debezium_current_state` |
| Publication | `debezium_current_state_pub` |
| Topic prefix | `dataplatform` |
| Plugin | `pgoutput` |

## Prerequisites

1. **Debezium Connect must be running** — start the local CDC stack first:

   ```bash
   cd deploy/local
   docker compose -f docker-compose.cdc.yml up -d
   ```

2. **PostgreSQL WAL level must be `logical`** — verify with:

   ```bash
   psql -h localhost -U postgres -d dataplatform \
     -c "SHOW wal_level;"
   # Expected output: logical
   ```

   If it shows `replica` or `minimal`, set `wal_level = logical` in
   `postgresql.conf` and restart PostgreSQL.

3. **Debezium Connect is healthy** — check before registering:

   ```bash
   curl -s http://localhost:8083/ | jq .version
   ```

## Registering Connectors

Run the registration script from any directory:

```bash
bash deploy/local/cdc/register-connectors.sh
```

Or set a custom Connect URL if your stack uses a different port:

```bash
CONNECT_URL=http://localhost:8084 bash deploy/local/cdc/register-connectors.sh
```

The script POSTs each connector JSON to `${CONNECT_URL}/connectors` using:

```bash
curl -X POST http://localhost:8083/connectors \
  -H 'Content-Type: application/json' \
  --data @connectors/postgres-asset-events.json

curl -X POST http://localhost:8083/connectors \
  -H 'Content-Type: application/json' \
  --data @connectors/postgres-current-state.json
```

## Verifying Connectors Are Registered

List all registered connectors:

```bash
curl -s http://localhost:8083/connectors | jq .
# Expected: ["postgres-asset-events","postgres-current-state"]
```

Check the status of a specific connector:

```bash
curl -s http://localhost:8083/connectors/postgres-asset-events/status | jq .
# connector.state should be "RUNNING"
# tasks[0].state should be "RUNNING"

curl -s http://localhost:8083/connectors/postgres-current-state/status | jq .
```

If a connector is in `FAILED` state, inspect the Debezium Connect logs:

```bash
docker compose -f deploy/local/docker-compose.cdc.yml logs connect
```

## Endpoints

| Service | Address |
|---------|---------|
| PostgreSQL | `localhost:5432` |
| Redpanda Kafka bootstrap | `localhost:19092` |
| Redpanda HTTP proxy | `localhost:18082` |
| Debezium Connect REST | `http://localhost:8083` |

## Relationship to `backend/internal/cdc`

The app-side CDC runtime (`backend/internal/cdc`) consumes from the Kafka
topics that these connectors produce:

- `BronzeConsumer` reads from `dataplatform.public.asset_events` → writes
  JSONL staging files for Iceberg ingestion
- `ESConsumer` reads from the current-state topics → rebuilds Elasticsearch
  search projections

Enable the runtime in the backend by setting `CDC_ENABLED=true` in
`backend/.env`. See `backend/.env.example` for all CDC configuration variables.
