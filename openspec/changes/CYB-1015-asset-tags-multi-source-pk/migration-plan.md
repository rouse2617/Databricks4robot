# Migration Plan — CYB-1015

`asset_tags` PK refactor. Destructive. HITL approval required before apply.

---

## 1. Current state

```sql
-- 001_init.sql
CREATE TABLE asset_tags (
  asset_id   TEXT NOT NULL REFERENCES assets(asset_id),
  tag_key    TEXT NOT NULL,
  tag_value  TEXT NOT NULL,
  tag_type   TEXT NOT NULL DEFAULT 'string',
  source_type TEXT NOT NULL,
  source_name TEXT,
  source_version TEXT,
  run_id     TEXT,
  tenant_id  TEXT,
  project_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_id, tag_key)
);
```

Repo `Upsert` writes only 5 cols and `ON CONFLICT (asset_id, tag_key) DO UPDATE`.

---

## 2. Target state

```sql
ALTER TABLE asset_tags DROP CONSTRAINT asset_tags_pkey;

ALTER TABLE asset_tags
  ADD COLUMN id BIGSERIAL PRIMARY KEY,
  ADD COLUMN applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN source_version_norm TEXT
    GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED;

ALTER TABLE asset_tags
  ADD CONSTRAINT uq_asset_tags_identity
    UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm);

CREATE INDEX IF NOT EXISTS idx_atags_lookup
  ON asset_tags(tag_key, tag_value, asset_id);
CREATE INDEX IF NOT EXISTS idx_atags_source
  ON asset_tags(source_type, source_name, source_version);
CREATE INDEX IF NOT EXISTS idx_atags_propagation
  ON asset_tags(tag_key) WHERE tag_key LIKE 'compliance.%';
CREATE INDEX IF NOT EXISTS idx_atags_run
  ON asset_tags(run_id) WHERE run_id IS NOT NULL;
```

Add FK on `run_id` (deferred — `algo_runs` not yet on dev; CYB-1018):

```sql
-- when CYB-1018 lands:
ALTER TABLE asset_tags
  ADD CONSTRAINT fk_atags_run FOREIGN KEY (run_id)
  REFERENCES algo_runs(run_id) ON DELETE SET NULL;
```

---

## 3. Pre-flight (read-only)

Run on dev before applying:

```sql
-- 3.1 row count
SELECT COUNT(*) FROM asset_tags;

-- 3.2 distinct source_type values currently in use
SELECT source_type, COUNT(*) FROM asset_tags GROUP BY 1 ORDER BY 2 DESC;

-- 3.3 source_name / source_version coverage
SELECT
  COUNT(*) FILTER (WHERE source_name IS NOT NULL)    AS with_source_name,
  COUNT(*) FILTER (WHERE source_version IS NOT NULL) AS with_source_version,
  COUNT(*) FILTER (WHERE run_id IS NOT NULL)         AS with_run_id,
  COUNT(*)                                           AS total
FROM asset_tags;

-- 3.4 sanity: should be empty under current PK
SELECT asset_id, tag_key, COUNT(*)
FROM asset_tags GROUP BY 1,2 HAVING COUNT(*) > 1;

-- 3.5 simulated post-state duplicate check (must be 0)
SELECT asset_id, tag_key, tag_value, source_type, COALESCE(source_version,''),
       COUNT(*)
FROM asset_tags
GROUP BY 1,2,3,4,5 HAVING COUNT(*) > 1;
```

**Decision gate**: §3.4 must return 0 rows (it will under current PK). §3.5
must also return 0 — if it does not, repo bug already produced ambiguous data
and we need a merge rule before adding UNIQUE.

---

## 4. Apply procedure (dev)

1. Take logical backup of `asset_tags`:
   ```bash
   gcloud sql export sql cyber-databrew-pg-dev \
     gs://cyber-databrew-backups/asset_tags_pre_cyb1015.sql.gz \
     --database=cyber_databrew --table=asset_tags
   ```
2. Stop tag-write traffic OR accept dual-write inconsistency for < 60s:
   - Dev: deploy backend with `TAG_WRITES_DISABLED=true` env (one-line guard
     in `UpsertTag` / `DeleteTag`), or simply note that dev write volume is ~0
     and skip.
   - Production (future): proper feature flag + drain.
3. Apply `030_asset_tags_multisource.sql` via `scripts/apply-migration-dev.sh`.
4. Verify §3.5 returns 0.
5. Deploy backend with new repo / handler code.
6. Re-enable tag writes.

Total expected downtime on dev: < 2 minutes. No table rewrite (only column
adds + constraint swap; `BIGSERIAL` add on a small table rewrites in place).

---

## 5. Rollback

**Hard rollback** (data preserved):

```sql
BEGIN;
ALTER TABLE asset_tags DROP CONSTRAINT uq_asset_tags_identity;
ALTER TABLE asset_tags DROP COLUMN source_version_norm;
ALTER TABLE asset_tags DROP COLUMN applied_at;
ALTER TABLE asset_tags DROP COLUMN id;
-- restore old PK; relies on no multi-source rows having been inserted yet
ALTER TABLE asset_tags ADD PRIMARY KEY (asset_id, tag_key);
COMMIT;
```

If multi-source rows were inserted between forward apply and rollback:
restore from §4.1 backup (drop table → reload). Document this is the only
safe path; do not attempt manual merge.

Roll back backend to previous Cloud Run revision in parallel.

---

## 6. Production checklist (future, not this slice)

- Snapshot prod `asset_tags` to GCS
- Feature flag drains tag-write traffic for ~5 minutes
- Apply migration in maintenance window
- Smoke test multi-source upsert
- Monitor `pg_stat_user_indexes` to confirm new indexes used
- Keep dev migration soaking for ≥ 1 week before prod cutover

---

## 7. Risk register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Generated column not supported on PG < 12 | Low (dev/prod on 15) | High | Verify `SELECT version()` before migration |
| Existing rows have non-normalized `source_version` (whitespace, case) | Low | Medium | Pre-flight §3.5 catches; ETL fix if found |
| ES `tags_flat` last-write-wins surfaces stale value after split | Medium | Low | Document; followup ticket if facet UX needs change |
| `Frontend` TagsTab broken until rebuilt | High during dev | Low | Ship UI in same PR; keep flat `tags` map as fallback |
| Surrogate PK collides with future logical sharding | Low | Low | `BIGSERIAL` enough for current scale; UUID alternative noted |
| `applied_at` default `now()` differs from `created_at` for existing rows | Low | Low | Set `applied_at = created_at` in same migration |
