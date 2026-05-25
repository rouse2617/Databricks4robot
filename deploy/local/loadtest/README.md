# HTTP load / smoke tests (full Compose backend)

Use these against the API behind **`docker compose --profile full`** (backend on port **8080** by default).

**Warnings**

- Do not aim high concurrency at shared production without approval.
- Writes (`k6-write-smoke.js`) create real rows in Postgres / side effects; use a **throwaway** environment or low `--iterations**.

## Auth

All `/api/v1/*` routes need header **`X-Databrew-Token`** (Compose default: `dev-token`).

```bash
export BASE=http://localhost:8080
export TOKEN=dev-token
```

## k6 (recommended)

Install: [k6.io/docs/getting-started/installation](https://k6.io/docs/getting-started/installation/)

```bash
cd deploy/local/loadtest

# Read-heavy mix: default soak plan — ramp 30s → hold 10m → ramp down 60s
BASE="$BASE" TOKEN="$TOKEN" VUS=15 k6 run k6-read-mix.js

# Quick smoke (~75s total): short ramp + short plateau
SOAK_DURATION=45s RAMP_UP=15s RAMP_DOWN=15s BASE="$BASE" TOKEN="$TOKEN" VUS=15 k6 run k6-read-mix.js

# Optional: a few write iterations (mcap-file + asset), single-digit load only
BASE="$BASE" TOKEN="$TOKEN" k6 run --vus 1 --iterations 3 k6-write-smoke.js
```

### `k6-read-mix.js` timing (env)

| Env | Default | Meaning |
|-----|---------|---------|
| `VUS` | `10` | Peak virtual users (max **200** in script) |
| `RAMP_UP` | `30s` | Ramp from 0 → `VUS` |
| `SOAK_DURATION` | `10m` | Steady load at `VUS` (soak) |
| `RAMP_DOWN` | `60s` | Ramp from `VUS` → 0 |

Tune **`VUS`** and durations for longer/soak or shorter smoke without editing the script.

### k6 via Docker (no local install; Linux VM friendly)

Use **`--network host`** so the container can reach **`127.0.0.1:8080`** on the host.

```bash
cd deploy/local/loadtest
docker pull grafana/k6:latest

docker run --rm --network host -v "$PWD:/scripts:ro" -w /scripts \
  -e BASE=http://127.0.0.1:8080 \
  -e TOKEN=dev-token \
  -e VUS=15 \
  -e SOAK_DURATION=10m \
  -e RAMP_UP=30s \
  -e RAMP_DOWN=60s \
  grafana/k6:latest run k6-read-mix.js
```

Request IDs use **`__VU` / `__ITER`** inside header helpers (required for current k6); do not move those into module-level constants.

## Apache Bench fallback (no k6)

macOS ships **`ab`** at `/usr/sbin/ab`.

```bash
cd deploy/local/loadtest
chmod +x run-ab-mix.sh
BASE=http://YOUR_HOST:8080 TOKEN=dev-token N=300 C=25 ./run-ab-mix.sh
```

## What was validated in-repo

- **`k6-read-mix.js`**: mixes **healthz** + registry + assets + search + lakehouse + deliveries + mcap-files + metrics registry.
- **`k6-write-smoke.js`**: one **POST /mcap-files** + **POST /assets** chain per iteration with unique IDs (avoids `raw_hash_md5` collisions).

For capacity planning, also watch **VM CPU/RAM**, **Postgres** connections, and **Elasticsearch** latency while scaling **`VUS`**.
