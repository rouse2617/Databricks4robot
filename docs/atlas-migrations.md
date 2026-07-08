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

Atlas Cloud **Registry**（`atlas migrate push` 把迁移目录上传 + hosted drift detection）**只在 Team 及以上套餐可用**，Pro 不行。本仓库当前只用 Pro，所以：

- 没有 hosted migration directory（migrations 仍在 repo 内的 `backend/migrations/`）
- 没有 hosted drift detection；改用本地 `atlas migrate diff` + CI drift check（见下）
- 唯一 Pro-unlock 的特性：`migrate lint --web`（报告上传 Atlas Cloud）

如果未来想开 hosted drift detection，需要公司把 Atlas Cloud 升到 Team。

## Schema 改动流程（GORM 模式）

cyber-databrew 用 [GORM](https://gorm.io) struct 作为 schema 真源（cyber-grace 模式）。`backend/internal/dbschema/*.go` 里的 struct 反映数据库表的最终状态；`atlas migrate diff --env gorm` 会基于这些 struct 自动生成对应的 SQL migration。

### 加一个新列

1. 改 `backend/internal/dbschema/*.go` 里对应 struct，加新字段 + gorm tag：

   ```go
   type Asset struct {
       // ... 已有字段 ...
       E2ETestColumn string `gorm:"column:e2e_test_column;type:text" json:"e2e_test_column,omitempty"`
   }
   ```

2. 本地跑 `atlas migrate diff <name> --env gorm` 生成 SQL：

   ```bash
   cd backend
   atlas migrate diff add_e2e_test_column --env gorm \
     --config file://atlas/atlas.hcl
   ```

   会在 `migrations/` 下生成 `YYYYMMDDHHMMSS_add_e2e_test_column.sql`，含 `ALTER TABLE assets ADD COLUMN e2e_test_column text;`。

3. **重要**：检查生成的 SQL 是否符合预期。GORM 表达不了的东西（CHECK 约束、INDEX、TRIGGER、FUNCTION、PARTITION）Atlas 会忽略——这些得**手动**写在 migration SQL 里。

4. Commit 三个一起：
   - 改了的 GORM model
   - 生成的 migration SQL
   - 更新的 `migrations/atlas.sum`（跑 `cd backend && atlas migrate hash --env migrate --config file://atlas/atlas.hcl`）

5. PR 触发：
   - `db-migrate-lint.yml`（hash + fresh-DB apply + 排序校验 + Pro lint）
   - `db-drift-check.yml`（新增表才 fail；修改已有表只发 PR comment）

### 加一张新表

1. 在 `backend/internal/dbschema/` 写新 struct（参考其他表的风格）
2. 在 `dbschema/registry.go` 的 `AllModels()` 注册
3. `atlas migrate diff <name> --env gorm`
4. CI drift check 会因为检测到新表而 fail —— 正常，**确认这是预期**后给 PR 加 comment 说明

### 改 enum / CHECK 约束

GORM **不能**表达 CHECK 约束。手写 migration：

```sql
-- 064_change_lifecycle_enum.sql
ALTER TABLE assets DROP CONSTRAINT IF EXISTS chk_lifecycle_state;
ALTER TABLE assets ADD CONSTRAINT chk_lifecycle_state CHECK (
  lifecycle_state IN ('created', 'processing', 'ready', 'delivered',
                      'archived', 'superseded', 'failed', 'rejected',
                      'new_state')  -- 加上新值
);
```

同时**必须**更新 `internal/dbschema/` 里相关 struct 的注释（GORM tag 表达不了），让团队知道 CHECK 约束存在。

### 改外键 / 索引 / 触发器 / 函数

GORM 同样表达不了。**完全手写** migration，对应修改 GORM struct 加注释说明。

### 删字段

GORM model 删字段 → `migrate diff` 生成 `DROP COLUMN` SQL。**注意**：如果代码还在用这个字段，编译会挂。删之前先：

1. 找代码里所有引用（`internal/postgres/`、`internal/usecase/`、`internal/handlers/`）
2. 改成不读这个字段
3. compile + 跑测试
4. 然后 GORM model 删字段 + 跑 `migrate diff`

### Atlas Cloud drift check (CI)

`.github/workflows/db-drift-check.yml` 在 PR 改 dbschema/migrations/atlas 时跑：

- 提取 PR diff 的 `CREATE TABLE` 列表
- 对比 baseline（`.atlas-drift-baseline.sql`）
- **新表**（PR 有 baseline 没有）→ fail
- 已有表的修改 → post PR comment（**不 fail**），让团队 review

baseline 当前包含 45 张表 + 3 个 uniqueIndex（共 48 个 CREATE statement）。`asset_events`/`asset_events_default`（partition）不在 GORM models 里。

修了一张表（缩 drift）后，重新生成 baseline：

```bash
cd backend
rm -rf /tmp/atlas-baseline && mkdir -p /tmp/atlas-baseline
atlas migrate diff baseline --env gorm \
  --config file://atlas/atlas.hcl \
  --dir file:///tmp/atlas-baseline
cp /tmp/atlas-baseline/2026*.sql ../.atlas-drift-baseline.sql
# commit
```

### 验证 workflow（手动 e2e 测试）

```bash
# 1. 改 GORM model
# 2. 跑 diff 到 SCRATCH（不污染 repo migrations/）
cd backend
atlas migrate diff test_name --env gorm \
  --config file://atlas/atlas.hcl \
  --dir file:///tmp/atlas-e2e

# 3. 看生成的 SQL
less /tmp/atlas-e2e/*.sql

# 4. 撤销 GORM 改动（如果只是测试 workflow）
git checkout -- internal/dbschema/
```

### 已知 gap（GORM 表达不了，保留在 SQL）

- **CHECK 约束**（lifecycle_state enum、asset_id regex 等）
- **外键**（GORM 不会在 diff 里 emit FK）
- **索引**（非 unique 索引，unique 索引通过 `uniqueIndex` tag 可以）
- **触发器**（set_updated_at、trg_logical_assets_type_immutable）
- **函数**（event_retention_cleanup、sync_databrew_run_from_pipeline）
- **表分区**（`asset_events` + 它的 monthly partition）
- **GENERATED 列**（`source_version_norm` in asset_tags、`duration_ns` in algo_runs）

这些在 SQL migration 里手写维护，不出现在 GORM models 里。CI drift check 的 baseline 模式会让这些不卡 PR，但 PR comment 还会展示。