# Tasks — P1-3~P1-6: Iceberg 湖仓上线

> Source of truth: #[[file:docs/review/data-platform-design.md]] §5.6, §5.11, §6.2

## Milestone 1: Iceberg REST Catalog（P1-3）

- [ ] 1.1 选型 ADR：Polaris vs Lakekeeper，写明理由
- [ ] 1.2 `deploy/local/docker-compose.iceberg.yml` 加 Catalog 服务
- [ ] 1.3 PyIceberg 冒烟：list / read 一张测试表
- [ ] 1.4 Trino 冒烟：`SELECT * FROM iceberg.test.smoke LIMIT 1`

## Milestone 2: PyIceberg Bronze 入湖（P1-4）

- [ ] 2.1 Outbox Worker Bronze Sink：写 staging parquet（文件名带 event_seq 区间）
- [ ] 2.2 PyIceberg CronJob 脚本：扫 staging → `MERGE INTO bronze.asset_events USING staging ON event_seq`
- [ ] 2.3 K8s CronJob 配置（每 5–10 min）
- [ ] 2.4 24h 跑无丢失验证
- [ ] 2.5 事件去重率验证

## Milestone 3: Compact（P1-5）

- [ ] 3.1 Expire Snapshots CronJob（保留 7 天）
- [ ] 3.2 Rewrite Data Files（compaction）
- [ ] 3.3 Remove Orphan Files
- [ ] 3.4 跑 7 天后小文件数稳定

## Milestone 4: Trino 查询接入（P1-6）

- [ ] 4.1 `internal/trino/client.go` 对接真实 Iceberg catalog
- [ ] 4.2 `/api/v1/lakehouse/sync-status` 返回真实数据
- [ ] 4.3 `/api/v1/lakehouse/training-assets` 返回真实数据
- [ ] 4.4 `/api/v1/lakehouse/quality-distribution` 返回真实数据
- [ ] 4.5 E2E 验证
