# Export MCAP rows from `collector-db` database `postgres`

## Policy: collector-db is read-only

Do **not** run `INSERT`, `UPDATE`, `DELETE`, `DDL`, or any migration against `collector-db`. Use only **`SELECT`** / read-only export (as below). All writes go to **Cyber Databrew** via its **HTTP API** (`/api/v1/mcap-files`, `/api/v1/assets`, etc.), not back into collector.

---

Run **inside the VPC** (e.g. temporary GKE `postgres:17` pod in `cyber-databrew-dev`, same as earlier `psql` checks).

Set `PGPASSWORD` to a user that can read `grace_videos` and `annotation_segmentations`.

## 1) MCAP videos (one JSON object per line)

```bash
psql -h 10.159.176.3 -p 5432 -U postgres -d postgres -At -c "
COPY (
  SELECT to_jsonb(v)
  FROM grace_videos v
  WHERE v.raw_hash_md5 IS NOT NULL
    AND COALESCE(v.storage_meta::text, '') ILIKE '%gs://%.mcap%'
    AND NOT COALESCE(v.is_deleted, false)
  ORDER BY v.id
) TO STDOUT;
" > grace_videos_mcap.jsonl
```

> If your `psql` rejects `COPY (...) TO STDOUT` over the client, use instead:

```bash
psql -h 10.159.176.3 -p 5432 -U postgres -d postgres -At -c "
SELECT to_jsonb(v)::text
FROM grace_videos v
WHERE v.raw_hash_md5 IS NOT NULL
  AND COALESCE(v.storage_meta::text, '') ILIKE '%gs://%.mcap%'
  AND NOT COALESCE(v.is_deleted, false)
ORDER BY v.id;
" > grace_videos_mcap.jsonl
```

## 2) Segmentations for those videos (one JSON per line)

```bash
psql -h 10.159.176.3 -p 5432 -U postgres -d postgres -At -c "
SELECT to_jsonb(s)::text
FROM annotation_segmentations s
JOIN grace_videos v ON v.id = s.video_id
WHERE v.raw_hash_md5 IS NOT NULL
  AND COALESCE(v.storage_meta::text, '') ILIKE '%gs://%.mcap%'
  AND NOT COALESCE(v.is_deleted, false)
  AND NOT COALESCE(s.is_deleted, false)
  AND s.start_timestamp > 0
  AND s.end_timestamp > s.start_timestamp
ORDER BY s.video_id, s.start_timestamp, s.end_timestamp,
  CASE WHEN COALESCE(NULLIF(trim(s.env), ''), '-') IN ('-', '') THEN 1 ELSE 0 END,
  s.segmentation_id;
" > annotation_segmentations_mcap.jsonl
```

Collector often stores **two rows per time window** (e.g. one with real `env` like a site name, one with `env` = `-` which the importer maps to `metadata.source_db`, e.g. `postgres`). The import script **dedupes by `(video_id, start_timestamp, end_timestamp)`** and keeps the first row after this ordering so the meaningful `env` wins. Use `--no-dedupe-segment-windows` only if you truly need one API asset per DB row.

## 3) Import via API

```bash
export DATABREW_BASE_URL='https://cyber-databrew-backend-dev-234851712830.us-central1.run.app'
export GRACE_TOKEN='dev-token'

python3 backend/scripts/import_collector_postgres_to_databrew.py \
  --videos-jsonl grace_videos_mcap.jsonl \
  --segments-jsonl annotation_segmentations_mcap.jsonl \
  --reviewer collector-import \
  --owner collector-import \
  --import-batch collector-postgres-mcap-full
```

Options:

- `--max-videos N` / `--max-segments N` — smoke test without full 120k+ run.
- `--dry-run` — print payloads only.
- `--videos-only` — skip segment assets.
- `--assets-only` — **不**调用 `POST /mcap-files`；按 collector 行推导 `mcap_file_id`（与已写入 Databrew 的规则一致），只跑 segment 的 `POST /assets`。集群里可设 Secret 键 `IMPORT_ASSETS_ONLY=1`（与 `IMPORT_VIDEOS_ONLY` 互斥）。
- `--source-db postgres` — stored in `metadata.source_db` (default: `postgres`).
- `--base-url` / `--token` — override `DATABREW_BASE_URL` / `GRACE_TOKEN`.
- `--mcap-workers` / `--asset-workers` — parallel `POST` 并发数（默认 **8**）；可用 `IMPORT_MCAP_WORKERS` / `IMPORT_ASSET_WORKERS` 覆盖。若后端开启 Gin 限流（`RATE_LIMIT_RPS`>0）出现 429，请调低并发或关闭/放宽限流。
- 日志会带 **`[timing]`**：每 500 条进度带 `last500_wall` / 吞吐；每若干批 HTTP 有 `mcap_http_batch` / `asset_http_batch`；阶段结束有 `mcap_phase` / `asset_phase`；`main` 结束有 `run_total`。

## 4) In-cluster (recommended): Pod / Job, no kubectl redirect

Run the importer **inside GKE** so you reach collector **private IP** and Cloud Run over HTTPS without piping `kubectl` logs into JSONL.

1. Install driver is handled inside the Job (`pip install -r requirements-collector-import.txt`).
2. Create a Secret (once) in your namespace, e.g. `cyber-databrew-dev`:

```bash
kubectl -n cyber-databrew-dev create secret generic collector-databrew-import-env \
  --from-literal=COLLECTOR_PG_DSN='postgresql://postgres:YOUR_PASSWORD@10.159.176.3:5432/postgres?sslmode=disable' \
  --from-literal=DATABREW_BASE_URL='https://cyber-databrew-backend-dev-234851712830.us-central1.run.app' \
  --from-literal=GRACE_TOKEN='dev-token' \
  --dry-run=client -o yaml | kubectl apply -f -
```

Optional keys on the same Secret (or add with `kubectl edit secret`): `IMPORT_BATCH`, `IMPORT_REVIEWER`, `IMPORT_OWNER`, `IMPORT_MAX_VIDEOS`, `IMPORT_MAX_SEGMENTS`, `IMPORT_VIDEOS_ONLY=1`, `IMPORT_ASSETS_ONLY=1`.

3. Apply ConfigMap + Job from repo root:

```bash
./backend/scripts/k8s/run_collector_import_in_cluster.sh
kubectl -n cyber-databrew-dev logs -f job/collector-databrew-import
```

4. Local alternative (same binary, needs `pip install -r backend/scripts/requirements-collector-import.txt`):

```bash
export COLLECTOR_PG_DSN='postgresql://...'
export DATABREW_BASE_URL='https://...'
export GRACE_TOKEN='...'
python3 backend/scripts/import_collector_postgres_to_databrew.py \
  --import-batch collector-postgres-direct \
  --reviewer collector-import --owner collector-import
```

`--pg-dsn` is optional if `COLLECTOR_PG_DSN` is set. The script sets `default_transaction_read_only=on` and only runs `SELECT` SQL.
