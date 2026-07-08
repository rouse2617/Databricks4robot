# Atlas Migrations (cyber-databrew)

cyber-databrew 用 [Atlas](https://atlasgo.io) 管 Postgres schema 变更。

- **Stage 1**：atlas.hcl + atlas.sum + CI lint（hash、fresh-DB apply、排序、Pro lint）
- **Stage 2**：部署前自动 apply（dev/prod 两个 Cloud Run Job）

> 注：Atlas Cloud 的 Registry / drift detection 需 Team 及以上套餐，本仓库只用 Pro 套餐，所以 **没有 drift detection workflow**。Pro 套餐解锁的 `migrate lint --web`（把 lint 报告上传 Atlas Cloud 生成可分享链接）已接入。

## 文件结构

```
backend/
├── atlas/                       # Atlas 工具相关（hcl config + migrate image）
│   ├── atlas.hcl                # 单 env "migrate"，URL 解析对齐 cyber-grace
│   └── Dockerfile.migrate       # arigaio/atlas distroless，部署 Cloud Run Job 用
├── migrations/
│   ├── 000_initial.sql
│   ├── 039_*.sql ... 064_*.sql
│   └── atlas.sum                # 每个文件 h1 SHA-256 指纹
└── Makefile                     # db-migrate-{hash,hash-check,apply,status}

.github/workflows/
├── db-migrate-lint.yml          # PR 时跑：hash + fresh-DB apply + 排序 + Pro lint
├── deploy-dev.yml               # deploy-migrate 跑在 deploy-backend 之后（迁移完才路由流量）
└── deploy-prod.yml              # deploy-migrate 跑在 deploy-backend 之前（严格 expand-contract）

deploy/cloudrun/
└── create-migrate-jobs.sh       # 一次性脚本：创建 dev/prod 两个 Cloud Run Job

scripts/
└── setup-atlas-secrets.sh       # `gh secret set ATLAS_TOKEN` 一键脚本
```

## 一次性启用

### 1) 建 Cloud Run Job（dev + prod 各一次）

```bash
# dev 环境
ENV=dev \
PROJECT_ID=green-valley-442103 \
REGION=us-central1 \
IMAGE=us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-migrate:init \
DB_SECRET=cyber-databrew-dev-postgres-password \
DB_HOST=172.27.160.7 DB_PORT=5432 DB_USER=postgres DB_NAME=cyber_databrew_dev \
bash deploy/cloudrun/create-migrate-jobs.sh

# prod 环境（env vars 不同，DB_SECRET 用 prod 的）
ENV=prod \
PROJECT_ID=green-valley-442103 \
REGION=us-central1 \
IMAGE=us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-migrate:init \
DB_SECRET=cyber-databrew-prod-postgres-password \
DB_HOST=172.27.160.9 DB_PORT=5432 DB_USER=postgres DB_NAME=cyber_databrew_prod \
bash deploy/cloudrun/create-migrate-jobs.sh
```

### 2) 首次 baseline（每个 env 各一次）

```bash
gcloud run jobs execute cyber-databrew-migrate-dev \
  --region=us-central1 --project=green-valley-442103 --wait \
  --args="--baseline,000_initial.sql"
```

`--baseline 000_initial.sql` 告诉 Atlas "000 之后的所有文件都认为已应用，别重跑"。首次成功后，给 Job 加 `ATLAS_BASELINED=1` 环境变量（以后 deploy workflow 会自动跳过 --baseline 参数）：

```bash
gcloud run jobs update cyber-databrew-migrate-dev \
  --region=us-central1 --project=green-valley-442103 \
  --set-env-vars=ATLAS_BASELINED=1
# prod 同款
```

### 3) Pro 接入（PR lint）

Pro 套餐只解锁 `migrate lint --web`（把报告上传到 Atlas Cloud 出可分享 URL），其他 Pro CLI 特性本地也能跑。

```bash
# 一键设 secret
ATLAS_TOKEN=<your-pro-token> bash scripts/setup-atlas-secrets.sh

# 验证
bash scripts/setup-atlas-secrets.sh --check
```

设好后 `db-migrate-lint.yml` 的 "Atlas lint (Pro)" 步骤会自动启用，把 lint 报告 URL 写到 PR 检查的 Annotation 里。**没设 ATLAS_TOKEN 也不会让 CI 失败** —— 那步直接跳过，只跑 hash + fresh-DB apply + 排序校验（这三项都是 free 的）。

## 日常流程

### 加新迁移

1. 新建文件 `backend/migrations/065_xxx.sql`（编号必须**严格大于 main 最新**，CI 排序检查会拦）
2. 重新生成 `atlas.sum`：`cd backend && make db-migrate-hash`
3. 提交 PR。CI 会跑：
   - `atlas migrate validate`（指纹校验）
   - fresh Postgres 上 `atlas migrate apply`（确认能跑通）
   - 排序校验（防 063/063 撞号重演）
   - Pro lint（如果 `ATLAS_TOKEN` 已配，输出 web 报告 URL）
4. 合 dev → GHA `deploy-dev.yml` 自动 build migrate image、apply 迁移、路由流量
5. 打 tag `vX.Y.Z` 推 main → GHA `deploy-prod.yml` 自动 build migrate image（pin 在同 tag 上）、apply 迁移、再 deploy 后端

### 排序校验为什么重要

`backend/migrations/` 用 `XXX_*.sql` 命名，CI 比较 PR 新文件的数字前缀 vs main 上**最大前缀**。任何新文件的前缀 ≤ main 最大值都会被挡下 —— **063 撞号事故永久免疫**。

### 与现有 bash 脚本的关系

- **`scripts/apply_pg_deltas.sh`（homegrown bash）保留**：作为"老 DB 救援"路径，独立的 `schema_migrations` 表追踪
- **Atlas 自己维护 `atlas_schema_revisions` 表**：每次 apply/rollback 都更新
- 两套机制**互不干扰**。新迁移一律走 Atlas；老 DB 已经应用过的 039-064 在首次 `--baseline 000_initial.sql` 时被标记为已应用

## 紧急救援

迁移出错或卡住：

```bash
# 查状态
DB_URL=postgres://user:pass@host/db?sslmode=disable make db-migrate-status

# 看某个 DB 上的 atlas_schema_revisions 表
psql "$DB_URL" -c "SELECT * FROM atlas_schema_revisions ORDER BY version"

# 重跑最新一条（一般会自动检测）
DB_URL=... make db-migrate-apply

# 强制把当前 schema dump 重新同步（drift 修复）
DB_URL=... atlas migrate apply --config atlas/atlas.hcl --env migrate
```

## 与 cyber-grace 对齐点

- 两 env 解析 URL 优先级一致（`--url` > `DB_URL` > 组合 `DB_*`）
- 用 Serverless VPC connector 接 CloudSQL 私网
- 不在 Cloud Run service 启动时跑迁移，全部由独立 Cloud Run Job 在 deploy 前预跑
- 不改文件名（保留 3 位编号），降低迁移摩擦

## dev / prod 部署顺序差异

| env | 顺序 | 原因 |
|---|---|---|
| **dev** | backend 部署（不路由）→ migrate apply → 路由流量 | dev 迭代快，migrate 在 revision 创建后、流量切换前完成；新代码短暂空跑不接流量 |
| **prod** | migrate apply → backend 部署 | 严格 expand-contract：DB schema 先更新，再 deploy 新代码，确保新代码永远跑在最新 schema 上 |

## Pro 套餐限制说明

Atlas Cloud **Registry**（`atlas migrate push` 把迁移目录上传 + drift detection）**只在 Team 及以上套餐可用**，Pro 不行。本仓库当前只用 Pro，所以：

- 没有 drift detection workflow（db-drift-check.yml 已删除）
- 没有 hosted migration directory（migrations 仍在 repo 内的 `backend/migrations/`）
- 唯一 Pro-unlock 的特性：`migrate lint --web`（报告上传 Atlas Cloud）

如果未来想开 drift detection，需要公司把 Atlas Cloud 升到 Team。