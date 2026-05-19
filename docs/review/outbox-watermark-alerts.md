# Outbox watermark alert thresholds

Status: draft (2026-05-14)
Owners: cyber-databrew backend

Companion to the 5 Prometheus gauges introduced in `feat(backend): export
outbox watermark gauges to prometheus` (commit 5ef74f3). These gauges are the
single source of truth for `PG → MQ → ES` health; this doc captures the
recommended alert wiring and the reasoning behind the thresholds so on-call
doesn't have to re-derive them at 3am.

## Gauges (recap)

| Metric | Meaning | Cost |
|---|---|---|
| `outbox_pg_max_event_seq` | `MAX(event_seq)` over `asset_events` | O(1) on index |
| `outbox_published_max_seq` | `MAX(event_seq) WHERE publish_state='published'` | O(1) on partial index |
| `outbox_seq_lag` | `pg_max - published_max` — PG→MQ backlog | derived |
| `outbox_es_applied_min_seq` | `MIN(applied_seq)` across `es_sync_checkpoint` shards, with idle-shard advance applied | O(shards), ≤ 16 rows by default |
| `outbox_consumer_lag` | `published_max - es_applied_min` when both > 0 — MQ→ES backlog | derived |

Refresh cadence: gauges only re-compute when `GET /api/v1/search/sync-progress`
is invoked. The frontend dashboard polls this on a timer; a Cloud Monitoring
uptime check or a tiny cron should hit it every minute in prod so scrape
freshness ≤ 60s.

## Alert thresholds (recommended, not enforced)

Pick the **stricter** of "absolute backlog" and "rate of change" depending on
which is noisier in your environment. Start with absolute; switch to rate
if you get false positives from steady-state burst loads.

| Alert | Condition | Severity | Rationale |
|---|---|---|---|
| `outbox-pg-mq-backlog` | `outbox_seq_lag > 1000` for 5m | warning | Relay is keeping up if `seq_lag` oscillates near 0; sustained >1k = relay stalled or PG write spike outrunning relay batch size. 1k ≈ 5 relay cycles at default 200 batch / 500ms interval. |
| `outbox-pg-mq-backlog-critical` | `outbox_seq_lag > 50_000` for 5m | critical | At 50k there is real risk of bumping into outbox retention / lease expiry. Page on-call. |
| `outbox-mq-es-backlog` | `outbox_consumer_lag > 1000` for 5m | warning | ES subscriber bulk loop healthy → consumer_lag near 0. Sustained 1k = ES bulk degraded or hot shard, but no user-visible search lag yet. |
| `outbox-mq-es-backlog-critical` | `outbox_consumer_lag > 50_000` for 15m | critical | Search results materially stale. Investigate ES cluster health, bulk reject rate, subscriber pod CPU. |
| `outbox-sync-progress-stale` | `time() - timestamp(outbox_pg_max_event_seq)` > 5m | warning | `/sync-progress` not being scraped → gauges frozen → all other alerts blind. |
| `outbox-no-events-flowing` | `rate(outbox_pg_max_event_seq[15m]) == 0` AND env=prod | info | Either nothing is writing (legitimate at 3am) or the API is wedged. Demote to info; let humans interpret. |

### Why these specific numbers

- **1000 / 50000**: empirically the relay drains 200 rows per 500ms = 24k/min steady state. 1k = ~2.5s drain backlog (acceptable; below human-perception of "search is fresh"). 50k = ~2min drain — search now visibly behind for new writes.
- **5m / 15m windows**: avoids paging on the 1–2 min spikes that happen every time the relay is restarted or a 10k bulk insert lands.
- **rate ≠ 0**: protects against the failure mode "everything looks 0 but that's because the API is returning a cached `progress` object". Catches frozen-state bugs.

### What not to alert on

- `outbox_es_applied_min_seq == 0`: legitimately 0 during cold start (no shard has reported yet). Already handled by the "warming up" gate in `MinAppliedSeq`; alerting on it creates false pages on every deploy.
- `outbox_pg_max_event_seq` absolute value: only the deltas matter.

## Cloud Monitoring (GCP) wiring

Backend exposes Prometheus metrics at `/metrics`. With managed Prometheus on
GKE / sidecar collector on Cloud Run, the rules look like:

```yaml
# rules.yaml fragment
groups:
- name: cyber-databrew-outbox
  rules:
  - alert: OutboxPGMQBacklog
    expr: outbox_seq_lag > 1000
    for: 5m
    labels:
      severity: warning
      service: cyber-databrew-backend
    annotations:
      summary: "PG outbox backlog ({{ $value }}) above 1k for 5m"
      runbook: "https://wiki/runbooks/outbox-pg-mq-backlog"

  - alert: OutboxMQESBacklog
    expr: outbox_consumer_lag > 1000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "ES subscriber backlog ({{ $value }}) above 1k for 5m"
      runbook: "https://wiki/runbooks/outbox-mq-es-backlog"

  - alert: OutboxProgressStale
    expr: time() - timestamp(outbox_pg_max_event_seq) > 300
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "sync-progress gauge not refreshing"
```

For the Cloud Run dev environment specifically, an uptime check hitting
`https://cyber-databrew-backend-dev-…/api/v1/search/sync-progress` every 60s
is the simplest scrape path (auth bypass for the metrics endpoint is already
configured).

## Runbook stubs (TODO)

- `outbox-pg-mq-backlog`: check relay pod logs for `lease lost` or
  `publish failed`; check `OUTBOX_RELAY_BATCH_SIZE` / `_INTERVAL_MS`; tail
  `asset_events WHERE publish_state='processing'`.
- `outbox-mq-es-backlog`: check ES cluster yellow/red; check bulk reject
  rate (`indices.indexing.index_total` vs `_failed`); check subscriber pod
  CPU/memory.
- `outbox-sync-progress-stale`: check Cloud Run revision health; check PG
  connection pool exhaustion (since `/sync-progress` runs two `MAX` queries).

## Lakehouse Bronze (PG → Iceberg)

Companion pipeline, owned by the Cloud Run Job `bronze-incremental` (Cloud
Scheduler `*/5 * * * *`). Three gauges introduced alongside this section:

| Metric | Meaning | Cost |
|---|---|---|
| `lakehouse_bronze_max_event_seq` | `MAX(event_seq)` in `iceberg.robot.bronze_asset_events` (read via BigQuery — Iceberg metadata, no parquet scan) | O(1) |
| `lakehouse_bronze_lag_events` | `outbox_published_max_seq - lakehouse_bronze_max_event_seq` | derived |
| `lakehouse_bronze_last_ingested_unix_seconds` | `MAX(_ingested_at)` as Unix epoch | O(1) |

Refresh cadence: gauges re-compute when `GET /api/v1/lakehouse/sync-progress`
is invoked. Frontend SettingsPage polls every 60s; same uptime-check pattern
as the ES side will refresh them in prod.

### Bronze alert thresholds

| Alert | Condition | Severity | Rationale |
|---|---|---|---|
| `lakehouse-bronze-stale` | `time() - lakehouse_bronze_last_ingested_unix_seconds > 600` for 5m | warning | Cron is `*/5 * * * *` → expected stale ≤ 5 min steady. 10 min = at least two consecutive runs missed or hung. |
| `lakehouse-bronze-stale-critical` | `time() - lakehouse_bronze_last_ingested_unix_seconds > 1800` for 5m | critical | 30 min without an ingest = job consistently failing; Bronze actively diverging from PG. |
| `lakehouse-bronze-backlog` | `lakehouse_bronze_lag_events > 5000` for 15m | warning | At steady write rate ~50/min, 5k = ~100min of un-ingested events. Cron should clear this in one run; sustained = job stuck or PG burst. |
| `lakehouse-bronze-backlog-critical` | `lakehouse_bronze_lag_events > 50000` for 15m | critical | Same scale as ES side. 50k = analytical queries materially behind. |
| `lakehouse-bronze-job-failed` | `run.googleapis.com/job/completed_task_attempt_count{result="failed",job_name="bronze-incremental"} > 0` for 10m | warning | Cloud Run Job natively exports execution outcomes; doesn't need our gauges. Two-tier alerting (job-level + watermark-level) catches "Job runs successfully but produces 0 rows" vs "Job crashes". |

### What not to alert on

- `lakehouse_bronze_max_event_seq == 0` for a fresh deployment / empty Bronze. Already filtered by the watermark equations (lag is 0 when max is 0).
- BigLake `429 RESOURCE_EXHAUSTED` events in the Job logs — internal retry already handles them; only the watermark-level alert matters.

### Cloud Monitoring rules

```yaml
# rules.yaml fragment — appended to the cyber-databrew-outbox group
  - alert: LakehouseBronzeStale
    expr: time() - lakehouse_bronze_last_ingested_unix_seconds > 600
    for: 5m
    labels:
      severity: warning
      service: cyber-databrew-bronze
    annotations:
      summary: "Bronze stale {{ $value }}s; expected ≤ 300s"
      runbook: "https://wiki/runbooks/lakehouse-bronze-stale"

  - alert: LakehouseBronzeBacklog
    expr: lakehouse_bronze_lag_events > 5000
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "Bronze {{ $value }} events behind PG-published"
```

For the failure-mode alert at the Cloud Run Job level, use Cloud Monitoring's
built-in metric (no Prometheus needed):

```
metric.type="run.googleapis.com/job/completed_task_attempt_count"
resource.labels.job_name="bronze-incremental"
metric.labels.result="failed"
```

### Runbook stubs (TODO)

- `lakehouse-bronze-stale`: `gcloud run jobs executions list --job=bronze-incremental --region=us-central1`; tail latest execution logs; if PG empty / wedged check `/api/v1/search/sync-progress` first (upstream).
- `lakehouse-bronze-backlog`: usually self-clears on next tick; investigate if persists 3+ ticks. Check BigLake quota (`bq ls --project_id=… --max_creation_time=…`).
- `lakehouse-bronze-job-failed`: read Job logs in Cloud Logging filtered to `severity>=ERROR`; common causes are PG timeout, Secret Manager rotation lag, BigLake REST 5xx.
