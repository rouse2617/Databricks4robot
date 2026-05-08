# Local CDC Stack

This directory holds the local infrastructure scaffold for the long-term
CDC/WAL architecture. It includes Debezium connector configurations and a
registration script for the local development environment.

## Components

CDC runs as Compose **profile `cdc`** and is also included in **profile `full`** inside `../docker-compose.yml`:

- **Postgres** (base stack): same service as `make dev-up`; must use `wal_level=logical` (already set there).
- **redpanda**: lightweight Kafka-compatible bus.
- **connect**: Debezium Kafka Connect (REST **8083 inside the container**, published as **8084 on the host**).

## Connectors

### `connectors/postgres-unified-cdc.json` (default — registered by `register-connectors.sh`)

Use **one** PostgreSQL connector for all CDC tables. Running two separate
`PostgresConnector` instances against the same database reused internal naming and,
locally, left `debezium_current_state` inactive while only `debezium_asset_events`
streamed — current-state topics stayed empty.

| Setting | Value |
|---------|-------|
| Tables | `public.assets`, `public.asset_tags`, `public.asset_algo_latest`, `public.mcap_files`, `public.asset_events` |
| Replication slot | `debezium_unified` |
| Publication | `debezium_unified_pub` |
| Topic prefix | `data4cyber` → e.g. `data4cyber.public.assets` |
| Plugin | `pgoutput` |

The legacy split JSON files (`postgres-asset-events.json`,
`postgres-current-state.json`) remain as references but are **not** registered by the script.

## Prerequisites

1. **Debezium Connect must be running** — start the local CDC stack first:

   ```bash
   cd deploy/local
   docker compose --profile cdc up -d
   ```

2. **PostgreSQL WAL level must be `logical`** — verify with:

   ```bash
   psql -h localhost -p 5432 -U postgres -d data4cyber \
     -c "SHOW wal_level;"
   # Expected output: logical
   ```

   If it shows `replica` or `minimal`, set `wal_level = logical` in
   `postgresql.conf` and restart PostgreSQL.

3. **Backend and Debezium must use the same Postgres instance** — Connect reads the `postgres` **container**. A host backend using `DB_HOST=127.0.0.1` may hit a **different** Postgres also bound to `:5432` (Homebrew install, SSH tunnel). Verify row-count parity:

   ```bash
   psql "postgresql://postgres:postgres@127.0.0.1:5432/data4cyber" -tAc "SELECT count(*) FROM assets;"
   docker exec local-postgres-1 psql -U postgres -d data4cyber -tAc "SELECT count(*) FROM assets;"
   ```

   Counts **must** match before trusting CDC → Kafka → ES.

4. **Debezium Connect is healthy** — check before registering:

   ```bash
   curl -s http://localhost:8084/ | jq .version
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

The script registers `connectors/postgres-unified-cdc.json` to `${CONNECT_URL}/connectors`.
If the connector already exists, the script leaves it in place and exits successfully.

## Verifying Connectors Are Registered

List all registered connectors:

```bash
curl -s http://localhost:8084/connectors | jq .
# Expected: ["postgres-unified-cdc"]
```

Check the status of a specific connector:

```bash
curl -s http://localhost:8084/connectors/postgres-unified-cdc/status | jq .
# connector.state should be "RUNNING"
# tasks[0].state should be "RUNNING"
```

If a connector is in `FAILED` state, inspect the Debezium Connect logs:

```bash
cd deploy/local && docker compose --profile cdc logs connect
```

## Endpoints

| Service | Address |
|---------|---------|
| PostgreSQL | `localhost:5432` |
| Redpanda Kafka bootstrap | `localhost:19092` |
| Redpanda HTTP proxy | `localhost:18082` |
| Debezium Connect REST | `http://localhost:8084` |

## Relationship to `backend/internal/cdc`

The app-side CDC runtime (`backend/internal/cdc`) consumes from the Kafka
topics that these connectors produce:

- `BronzeConsumer` reads from `data4cyber.public.asset_events` (topic prefix from Debezium; override via `CDC_TOPIC_ASSET_EVENTS`) → writes
  JSONL staging files for Iceberg ingestion
- `ESConsumer` reads from the current-state topics → rebuilds Elasticsearch
  search projections

Enable the runtime in the backend by setting `CDC_ENABLED=true` in
`backend/.env`. The `full` Compose profile already injects the required
`CDC_*` variables and waits for connector registration before starting the backend.
See `backend/.env.example` for all CDC configuration variables.
