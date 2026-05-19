# Lakehouse Incremental Ingestion (PG → Iceberg Bronze)

Status: deployed (2026-05-14) — Cloud Run Job + Cloud Scheduler in `green-valley-442103/us-central1`
Owners: cyber-databrew backend / data
Supersedes: `data-platform-design.md` §5.6.2 (two-stage Polaris+staging approach) for the **current GCP-managed BigLake REST** path.

## Why this doc

The data platform design doc assumes a self-hosted Iceberg catalog (Polaris/Lakekeeper) and a "Bronze Sink + GCS staging + PyIceberg MERGE CronJob" two-stage pipeline. That blueprint is correct for self-managed lakes but **does not match what we actually run on GCP**.

What we actually have on `green-valley-442103`:

- **BigLake REST Iceberg Catalog** (Google-managed) at `https://biglake.googleapis.com/iceberg/v1/restcatalog` — no Polaris / Nessie to operate.
- **GCS warehouse** `gs://cyber-databrew-iceberg-warehouse-prod`.
- **Three tables already created** in namespace `robot`: `bronze_asset_events`, `silver_asset_events_current`, `gold_asset_event_daily`.
- A working **one-off backfill K8s Job** (`deploy/k8s/jobs/biglake-bronze-asset-events-backfill-once.yaml`) that:
  - Reads `MAX(event_seq)` from Bronze (acts as cursor).
  - Pulls `asset_events WHERE event_seq > cursor` from Postgres in batches.
  - Casts to PyArrow with a fixed schema.
  - `table.append()` directly into Bronze via BigLake REST.
  - Has been executed once (2026-05-13) and produced ~25 parquet files; the path is **proven**.

Because BigLake handles catalog + file layout, there is no need for the staging-then-merge dance. This doc documents the simplified design.

## Goals

1. **Continuous incremental ingestion** of `asset_events` into Bronze with bounded lag (≤ 5 min P95).
2. **Same operational shape** as the ES sync side: per-table cursor (high-water mark), idempotent re-runs, watermark gauges + alerts.
3. **No new long-running services**: piggyback on what's already proven (the one-off backfill script becomes a K8s CronJob).
4. **No regression** of the design doc's correctness guarantees (transactional outbox, single source of truth in PG, no business state that lives only in the lake).

## Non-goals

- Streaming / sub-minute latency (Bronze is for **analytical** workloads; ES already covers low-latency search).
- Silver / Gold incremental compute (out of scope for v1 — keep Silver/Gold as periodic full-rebuild jobs until v1.1).
- Strict cross-engine consistency (BigQuery reads can lag Bronze writes by one snapshot; acceptable).
- Removing the design-doc §5.6.2 blueprint (kept as fallback in case BigLake becomes unavailable / cost-prohibitive).

## Architecture

```
┌──────────────────┐
│ PostgreSQL       │
│ asset_events     │── source of truth, owned by backend
│ (event_seq mono) │
└──────┬───────────┘
       │ SELECT event_seq > $cursor
       │ ORDER BY event_seq ASC LIMIT 5000
       ▼
┌──────────────────────────────────────┐
│ Cloud Run Job  bronze-incremental    │
│ region:    us-central1               │
│ trigger:   Cloud Scheduler */5 * * * │  ← short-term: 5 min
│ image:     python:3.11 + pyiceberg   │
│ network:   VPC connector cr-central- │
│            conn  (Cloud SQL priv IP) │
│ logic:                               │
│   1. cursor = MAX(event_seq) Bronze  │
│   2. rows   = SELECT FROM PG > cursor│
│   3. PyArrow batch                   │
│   4. table.append(arrow_batch)       │
│   5. structured stdout logs (scraped │
│      by Cloud Logging)               │
└──────┬───────────────────────────────┘
       │ append (parquet)
       ▼
┌──────────────────┐         ┌─────────────────────┐
│ GCS              │ <─────► │ BigLake REST        │
│ bronze table     │   meta  │ Catalog             │
│ (parquet files)  │         │ (Google-managed)    │
└──────────────────┘         └─────────────────────┘
       │ SELECT
       ▼
┌──────────────────┐
│ BigQuery (in-cluster│
│ → BigLake REST)  │── reads from same table; backend hits BigQuery
└──────────────────┘
```

### Cursor & idempotency

- The cursor is **not a separate table or file** — it's `MAX(event_seq)` read directly from the Bronze table at job start.
- This makes the job **stateless**: any run reads the lake to find its own starting point, so:
  - Re-runs after a crash are safe (worst case: read 0 new rows and exit).
  - Concurrent runs are safe-ish (both compute the same cursor; one will append rows the other still sees as unseen — duplicate rows possible). Mitigation: Cloud Run Job `--max-retries=1` + Cloud Scheduler single-fire prevents most overlap; even with overlap, see "Duplicates in Bronze" below.

### Duplicates in Bronze are EXPECTED — Silver de-duplicates

**Observed 2026-05-14**: the first production run of `bronze-incremental` ingested 129,780 distinct `event_seq` values across 144,780 physical rows — i.e. ~11.5% duplicates. Root cause: BigLake REST Catalog returned `429 RESOURCE_EXHAUSTED` mid-commit on 3 batches; PyIceberg's internal `tenacity` retry succeeded on the metadata commit, but the exception still bubbled up to our outer `append_with_retry`, which re-called `table.append(arrow_batch)` and wrote the same batch a second time.

**Design decision: accept Bronze duplicates; rely on Silver to deduplicate on `event_id`.** This matches the standard Medallion pattern (Bronze = raw / replay-safe; Silver = canonical / deduped).

Why we don't try to make Bronze exactly-once:

| Reason | Detail |
|---|---|
| Bronze cursor stays correct | Cursor = `MAX(event_seq)`. Duplicate rows do not advance the cursor past their own `event_seq`, so re-runs never *miss* events; they only ever *duplicate* them. |
| Silver dedup is required anyway | Schema drift, late-arriving events, payload version upgrades all need a `ROW_NUMBER() OVER (PARTITION BY event_id ORDER BY event_seq DESC) = 1` filter. Adding 429-retry dedup on top is redundant. |
| Cost impact is bounded | 429s only fire under burst load. Steady-state cron pulls ≤ 1 batch / run → 1 commit / run → cannot collide with itself. Backfill / catch-up runs are the only ones that ever produce dupes. |
| The fix is non-trivial | "Verify after 429" requires scanning Iceberg with a filter; getting it race-free with PyIceberg's internal retry is fragile. Not worth the complexity. |

**Consumer contract (Silver / Gold / ad-hoc BigQuery)**: never `COUNT(*) FROM bronze_asset_events` to count business events — always `COUNT(DISTINCT event_id)` or join through a Silver view that dedupes. The Silver job (`silver_asset_events_current`) MUST apply:

```sql
SELECT *
FROM (
  SELECT *, ROW_NUMBER() OVER (PARTITION BY event_id ORDER BY event_seq DESC, _ingested_at DESC) AS rn
  FROM iceberg.robot.bronze_asset_events
) WHERE rn = 1
```

The `event_seq` global monotonic sequence guarantees ordering across all duplicates, so picking any one row per `event_id` is safe.

### Schema

Locked to the schema used by `biglake-bronze-asset-events-backfill-once.yaml` (22 columns + `_ingested_at`, `_source_file`). Changing the schema is an Iceberg DDL change and outside this doc's scope.

### Concurrency / failure

| Scenario | Behavior |
|---|---|
| Job runs every 5m, normal load | Reads N≈5000 rows / run; total elapsed ~30s; appends one parquet file per run. |
| Burst: 50k rows produced in 1m | Next run pulls them all (LIMIT bumped if needed); ~3s of PG load. |
| Network blip mid-append | PyIceberg fails the commit; no partial snapshot. Next run picks up from same cursor. |
| Job pod OOM | Lost in-flight batch; next run replays since cursor unchanged. |
| Postgres down | Job fails fast; alert fires on `bronze_lag_events`; events queue up in PG harmlessly. |
| BigLake REST 5xx | PyIceberg retries (default backoff); permanent failure surfaces in pod logs + lag metric. |

### Observability

Three Prometheus gauges, exposed by **backend** (not by the CronJob — backend already has a metrics endpoint and BigQuery client):

| Gauge | Source | Meaning |
|---|---|---|
| `lakehouse_bronze_max_event_seq` | `SELECT max(event_seq) FROM iceberg.robot.bronze_asset_events` (BigQuery) | High-water mark for Bronze |
| `lakehouse_bronze_lag_events` | derived: `outbox_published_max_seq - lakehouse_bronze_max_event_seq` | Backlog in events |
| `lakehouse_bronze_last_seen_at` | `SELECT max(occurred_at) FROM iceberg...bronze_asset_events` | Bronze "data freshness" timestamp |

Refresh model: backend runs a 5-minute background ticker that issues these two BigQuery queries and updates the gauges. Cheap (≤ 100 ms each on a partitioned Iceberg scan).

Recommended alert (mirrors the ES side):

| Alert | Condition | Severity |
|---|---|---|
| `lakehouse-bronze-lag` | `lakehouse_bronze_lag_events > 50_000` for 15m | warning |
| `lakehouse-bronze-stale` | `time() - lakehouse_bronze_last_seen_at > 1800` (30 min) | warning |
| `lakehouse-bronze-cronjob-failed` | k8s `kube_job_failed{job_name=~"bronze-incremental.*"} > 0` | warning |

(Numbers calibrated for 5-min CronJob + dev throughput; revisit at scale.)

## Phasing

| Step | Scope | Owner | Status |
|---|---|---|---|
| 1 | Cloud Run Job `bronze-incremental` + Cloud Scheduler `*/5 * * * *` (image at `us-central1-docker.pkg.dev/.../bronze-incremental:dev-latest`; code under `deploy/cloudrun/bronze-incremental/`). Chose Cloud Run Job over K8s CronJob to align with backend/frontend already on Cloud Run. | backend | **DONE 2026-05-14** |
| 2 | First production run `bronze-incremental-jvzkr`: 129,780 distinct event_seq ingested in 70s (cursor 0 → 508,859); next scheduled run `9gd69` started at cursor=508,859 and exited with `processed=0` — cursor recovery via `MAX(event_seq)` proven end-to-end. | ops | **DONE 2026-05-14** |
| 3 | Add 3 Prometheus gauges + 5-min ticker to backend; export to existing `/metrics` | backend | TODO |
| 4 | Add alert rules (mirror `outbox-watermark-alerts.md`) | ops | TODO |
| 5 | One-off resync: re-run the existing backfill Job once to close the gap between when ES sync started and when this CronJob starts | ops | optional |
| 6 | Silver / Gold incremental — out of scope; keep one-off jobs until business needs increase | — | DEFER |

## Open questions

- **Should the CronJob be 1 min instead of 5 min?** 5 is plenty for analytics today; raising to 1 is one-line change later.
- **Where do schema changes get owned?** Today the schema is duplicated between the K8s Job script and the Iceberg table DDL; need a single source of truth (probably a `schemas/iceberg/bronze_asset_events.json` file or `pyiceberg` migration script).
- **Backend ↔ BigQuery auth**: backend already has a BigQuery client for `/api/v1/lakehouse/*`; reusing it for watermark queries is free. Confirmed available.
- **Silver/Gold cadence**: once Bronze is live, do we run Silver as `INSERT OVERWRITE` every hour or move to incremental MERGE? Defer until v1.1.

## What this doc does NOT change

- `asset_events` outbox semantics, `publish_state` lifecycle, ES sync path — all unchanged.
- `data-platform-design.md` §5.6.2 itself — kept as the self-managed alternative blueprint.
- BigQuery / Iceberg query layer in backend (`/api/v1/lakehouse/*`) — already works with the existing tables.
