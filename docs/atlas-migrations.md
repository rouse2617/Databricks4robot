# Atlas Migrations (cyber-databrew)

cyber-databrew 用 [Atlas](https://atlasgo.io) 管 Postgres schema 变更。迁移 SQL **手写**，Atlas 只做版本化 + 指纹校验 + fresh-DB 重放 + 部署前 apply。

- **Stage 1**：atlas.hcl + atlas.sum + CI lint（hash、fresh-DB apply、排序、Pro lint）
- **Stage 2**：部署前自动 apply（dev/prod 两个 Cloud Run Job）

> **基线重置（2026-07-08）**：早先尝试用 GORM struct 作 schema 真源 + `atlas migrate diff --env gorm` 自动生成迁移（cyber-grace 模式），实践证明对本仓库不成立——GORM model 只能表达「表 + 列」骨架，真实 schema 的索引 / CHECK / 触发器 / 函数 / 分区 / GENERATED 列它都表达不了，自动 diff 会生成把这些真实对象一并 `DROP` 的破坏性迁移。现改为：**基线从 dev 线上库反推**（`20260708104125_baseline_from_dev.sql`），**新迁移一律手写 SQL**；`migrate diff --env gorm` 停用。**dev 为 schema 真源**，prod 落后、向此基线对齐。

## 文件结构

```
backend/
├── atlas/
│   ├── atlas.hcl                # env "migrate"(手写迁移) + env "gorm"(已停用,留作参考)
│   └── Dockerfile.migrate       # arigaio/atlas distroless,部署 Cloud Run Job 用
├── migrations/
│   ├── 20260708104125_baseline_from_dev.sql  # 从 dev 反推的基线(全量 schema)
│   ├── <后续时间戳>_*.sql                      # 之后每次手写的增量迁移
│   └── atlas.sum                # 每个文件 h1 SHA-256 指纹
└── Makefile                     # db-migrate-{hash,hash-check,apply,status}

.github/workflows/
├── db-migrate-lint.yml          # PR: hash + fresh-PG17 apply + 排序 + Pro lint
├── deploy-dev.yml               # deploy-migrate 在 deploy-backend 之后(迁移完才路由)
└── deploy-prod.yml              # deploy-migrate 在 deploy-backend 之前(expand-contract)

deploy/cloudrun/create-migrate-jobs.sh   # 一次性:创建 dev/prod 两个 migrate Job
scripts/setup-atlas-secrets.sh           # gh secret set ATLAS_TOKEN
```

## 日常流程：加迁移（手写）

1. 新建 `backend/migrations/<YYYYMMDDHHMMSS>_<name>.sql`（时间戳前缀，严格晚于现有最新），手写 DDL。
2. 重算指纹：`cd backend && atlas migrate hash --env migrate --config file://atlas/atlas.hcl`（或 `make db-migrate-hash`）。
3. 本地验证：`atlas migrate validate --env migrate --dev-url "docker://postgres/17/dev?search_path=public"`（在全新 PG17 上重放全部迁移 + 校验指纹）。
4. PR → `db-migrate-lint.yml`：hash + fresh-PG17 apply + 排序校验 + Pro lint（配了 `ATLAS_TOKEN` 才跑，没配跳过、不 fail）。
5. 合 dev → `deploy-dev.yml` 自动 build migrate image、apply、路由流量。
6. tag `vX.Y.Z` 推 main → `deploy-prod.yml` apply（pin 同 tag）后再 deploy 后端。

> 排序校验：CI 比 PR 新文件的数字前缀 vs main 最新。时间戳前缀天然递增，别用比现有更小的前缀。

## Postgres 版本 & 扩展

- dev/prod CloudSQL 都是 **PostgreSQL 17** → atlas.hcl 的 `dev-url` 必须 `docker://postgres/17`，CI service 用 `postgres:17`（写 16 会出假阳性 diff）。
- live 装了扩展 `pg_trgm` + `pgcrypto`。**atlas 不 emit `CREATE EXTENSION`**，基线开头已手补这两行；将来新迁移若用到新扩展，同样手写 `CREATE EXTENSION IF NOT EXISTS`。

## 一次性启用：Cloud Run migrate Job（dev/prod 各一次）

```bash
ENV=dev PROJECT_ID=green-valley-442103 REGION=us-central1 \
IMAGE=us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-migrate:init \
DB_SECRET=cyber-databrew-dev-postgres-password \
DB_HOST=172.27.160.7 DB_PORT=5432 DB_USER=postgres DB_NAME=cyber_databrew_dev \
bash deploy/cloudrun/create-migrate-jobs.sh
# prod 同款：DB_SECRET=…-prod-…、DB_HOST=172.27.160.9、DB_NAME=cyber_databrew_prod
```

## 在 live 上启用重置后的基线（⚠️ 写 live，需明确授权）

基线重置后，dev/prod 的 `atlas_schema_revisions` 仍停在旧的 `064` 记录。要让 atlas 认识新基线，须在每个 env 上把新基线 **baseline** 进去（记为已应用、不重跑）：

```bash
gcloud run jobs execute cyber-databrew-migrate-<env> \
  --region=us-central1 --project=green-valley-442103 --wait \
  --args="--baseline,20260708104125_baseline_from_dev.sql"
```

> ⚠️ 这一步写 live 的 `atlas_schema_revisions` 表。prod 现有 schema 与 dev 基线有差异（见下），**先把 prod schema 对齐到基线，再 baseline**，否则 baseline 会掩盖真实 drift。

## dev / prod 差异（2026-07-08 实测，只读 diff）

基线取自 **dev**。prod 与之相差约 274 行，主要：

- **prod 有、dev 无**：`databrew_runs` / `component_build_runs` / `component_releases` / `rag_build_runs` + 同步触发器、`mcap_files.summary_index_*`（代码在用，dev 待补）。
- **dev 有、prod 无**：`node_runs` / `node_attempts` / `pipeline_definitions` / `pipeline_revisions` / `pipeline_shards`（代码 0 引用，疑似死表）、grace_video 约束、trgm 索引 + `pg_trgm`、`pipeline_runs` 若干列。

prod 向 dev 基线对齐 = 单独的 forward 迁移（手写），逐项 review 后 apply。

## dev / prod 部署顺序差异

| env | 顺序 | 原因 |
|---|---|---|
| **dev** | backend 部署（不路由）→ migrate apply → 路由流量 | 新代码短暂空跑不接流量 |
| **prod** | migrate apply → backend 部署 | expand-contract：schema 先更新，再 deploy 新代码 |

## 紧急救援

```bash
DB_URL=postgres://user:pass@host/db?sslmode=disable make db-migrate-status
psql "$DB_URL" -c "SELECT * FROM atlas_schema_revisions ORDER BY version"
DB_URL=... make db-migrate-apply            # 重跑最新一条
```

## 与 cyber-grace 对齐点

- env URL 解析优先级一致（`--url` > `DB_URL` > 组合 `DB_*`）
- Serverless VPC connector 接 CloudSQL 私网
- 迁移不在 service 启动时跑，由独立 Cloud Run Job 在 deploy 前预跑
- **CI 只留 `db-migrate-lint`（fresh-apply + hash + 排序），不做 GORM drift 检查**（cyber-grace 本身也没有）

## Pro 套餐说明

Atlas Cloud Registry / hosted drift detection 需 Team 套餐；本仓库用 Pro，仅解锁 `migrate lint --web`（报告上传 Atlas Cloud 出可分享 URL）。migrations 留在 repo 内的 `backend/migrations/`。
