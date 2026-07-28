-- atlas:txmode none

-- CYB-4297: 让 "打开资产看运行了哪些流水线" 反查可用。
--
-- pipeline_runs.asset_ids 是 TEXT[](每个 run 记录参与的 asset,业务侧就是
-- grace_video_id)。之前只有正向读(run → asset),反查 "asset → runs" 走
-- WHERE $1 = ANY(asset_ids) 会全表扫;dev 已 45 万行 / 2.5 GB,prod 更大。
--
-- GIN 索引让 = ANY / && / @> 命中 Bitmap Index Scan,反查毫秒级。
--
-- CONCURRENTLY 不锁写,不会卡 dispatcher/watcher/webhook。要求文件不在
-- 事务里跑 —— 顶部 `atlas:txmode none` 告诉 Atlas 逐语句提交。
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pipeline_runs_asset_ids_gin
  ON pipeline_runs USING GIN (asset_ids);
