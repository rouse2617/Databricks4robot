# Design — P1-3~P1-6: Iceberg 湖仓上线

## Source of Truth

- 两段式入湖架构：#[[file:docs/review/data-platform-design.md]] §5.6.2
- Iceberg 选型理由：#[[file:docs/review/data-platform-design.md]] §5.6.4
- 技术栈选型：#[[file:docs/review/data-platform-design.md]] §5.11
- 部署形态：#[[file:docs/review/data-platform-design.md]] §6.2
- 闸口验收：#[[file:docs/review/data-platform-design.md]] §9.1.2（2.0 → 3.x 闸口）

## 架构

```
Outbox Worker → staging parquet → PyIceberg CronJob → MERGE INTO bronze.asset_events
                                                        ↓
                                                   Iceberg REST Catalog (Polaris/Lakekeeper)
                                                        ↓
                                                   Trino → /api/v1/lakehouse/*
```

## 变更范围

### P1-3: Catalog 部署
- `deploy/local/docker-compose.iceberg.yml` 加 Polaris/Lakekeeper 服务
- ADR 文档 `docs/review/adr-iceberg-catalog.md`

### P1-4: Bronze 入湖
- `dagster/` 或独立 Python 脚本：扫 staging → MERGE INTO bronze
- K8s CronJob 配置

### P1-5: Compact
- 独立 CronJob：expire snapshots + rewrite data files + remove orphan files

### P1-6: Trino 接入
- `internal/trino/client.go` 对接真实 catalog
- `internal/handlers/lakehouse/handler.go` 返回真实数据
