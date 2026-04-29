# Requirements — P1-3~P1-6: Iceberg 湖仓上线

## Introduction

启用 Iceberg REST Catalog + PyIceberg Bronze 入湖 + Compact + Trino 查询接入。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §5.6（数据索引与同步）、§5.6.2（两段式入湖）、§5.6.4（Iceberg 选型）
- #[[file:docs/review/next-steps-tasks.md]] P1-3 ~ P1-6

## Requirements

### R1: Iceberg REST Catalog 选型 + 部署（P1-3）
- Polaris（首选）或 Lakekeeper 部署完成
- PyIceberg / Trino 都能 list / read 一张冒烟表
- ADR 写明选型理由

### R2: PyIceberg Bronze 入湖 CronJob（P1-4）
- 每 5–10 min 扫 staging parquet → `MERGE INTO bronze.asset_events`（按 event_seq 去重）
- 24h 跑无丢失；事件去重率验证

### R3: PyIceberg Compact CronJob（P1-5）
- 合并小文件、过期 snapshot 清理
- 跑 7 天后 bronze 表小文件数稳定

### R4: Trino 查询接入（P1-6）
- `/api/v1/lakehouse/*` handler 对接真实 catalog
- sync-status / training-assets / quality-distribution 三个核心端点返回真实数据
