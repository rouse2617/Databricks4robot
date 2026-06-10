-- Adds asynchronous MCAP summary indexing state on mcap_files.
--
-- Context (docs/review/mcap-index-metadata-v1.md):
--   - V1 索引器（异步 worker）从对象存储的 MCAP 文件读取 summary / channel 信息，
--     然后回写到 mcap_files：标量列（start/end/duration/channel_count/...）+
--     metadata.channels JSON 数组 + metadata.mcap_* / process_state.mcap_* 约定键。
--   - 几百万存量行需要一个稳定的 "做没做、做到哪、失败原因" 的状态机，
--     不复用 ingest_state（业务状态机），单独加一个维度。
--
-- 列设计：
--   summary_index_state      'pending' | 'running' | 'done' | 'failed'
--                            存量行默认 'pending' = 全部待办；DEFAULT 在 PG 11+
--                            是元数据级变更，几百万行也不会重写表。
--   summary_index_version    当前实现版本（如 'v1'）。以后改字段集时，把旧版本
--                            的 'done' 行重置回 'pending' 即可强制重跑。
--   summary_indexed_at       最近一次成功时间。
--   summary_index_attempts   累计尝试次数，worker 用于限制重试。
--   summary_index_error      最近一次失败的错误（截断到合理长度由应用层控制）。
--
-- 索引：
--   只对"待办"行建部分索引，避免在 done 占多数时索引膨胀。
--   worker claim 使用 `WHERE summary_index_state IN ('pending','failed')`
--   配合 SELECT ... FOR UPDATE SKIP LOCKED 抢占行。
--
-- 兼容：
--   - 现有读路径不读这些列，无需代码同步上线。
--   - 现有 ingest_state 不动；indexer 只更新 summary_index_* + metadata/process_state。

ALTER TABLE mcap_files
    ADD COLUMN IF NOT EXISTS summary_index_state    TEXT        NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS summary_index_version  TEXT,
    ADD COLUMN IF NOT EXISTS summary_indexed_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS summary_index_attempts INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS summary_index_error    TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_mcap_files_summary_index_state'
    ) THEN
        ALTER TABLE mcap_files
            ADD CONSTRAINT chk_mcap_files_summary_index_state
            CHECK (summary_index_state IN ('pending','running','done','failed'));
    END IF;
END$$;

-- 部分索引：worker 扫待办行；done 行不进索引，几百万规模也不臃肿。
CREATE INDEX IF NOT EXISTS idx_mcap_files_summary_index_pending
    ON mcap_files (summary_index_state, created_at)
    WHERE is_deleted = FALSE
      AND summary_index_state <> 'done';

COMMENT ON COLUMN mcap_files.summary_index_state    IS 'V1 MCAP summary 索引状态：pending/running/done/failed。详见 docs/review/mcap-index-metadata-v1.md。';
COMMENT ON COLUMN mcap_files.summary_index_version  IS '索引实现版本（如 v1）；用于以后按版本重跑。';
COMMENT ON COLUMN mcap_files.summary_indexed_at     IS '最近一次成功写入 summary 的时间。';
COMMENT ON COLUMN mcap_files.summary_index_attempts IS 'worker 累计领取次数；用于失败限流。';
COMMENT ON COLUMN mcap_files.summary_index_error    IS '最近一次失败的错误摘要；成功后由 worker 清空或保留。';
