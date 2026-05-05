# Tasks — cf_* Legacy Layer Removal

> **Spec ID**: P2-9
> **设计依据**: 
> - #[[file:.kiro/specs/cf-legacy-removal/requirements.md]]
> - #[[file:.kiro/specs/cf-legacy-removal/design.md]]
>
> **编码与提交规范**：严格遵循 #[[file:CLAUDE.md]]
> - Conventional Commits 格式
> - 每次提交前：`go build ./...` + `go vet ./...` + `go test ./...`
> - 只改任务要求的代码

---

## Milestone 1: 验证无运行时依赖

> 目标：确认生产代码不依赖 `cf_*` 列

- [x] 1.1 确认 `postgres/repos.go` 的 `Get`/`Set`/`List` 方法不读写 `cf_*` 列
- [x] 1.2 确认 `searchindex/builder.go` 不从 `cf_*` 读取数据
- [x] 1.3 确认 `outbox/` 包不依赖 `cf_*`
- [x] 1.4 检查 `scripts/` 目录下脚本不依赖 `cf_*`
- [x] 1.5 运行 `go build ./...` + `go test ./...` 确认无错误

---

## Milestone 2: 更新 Seed 数据

> 目标：`migrations/002_seed.sql` 改用 typed columns + projection tables

- [x] 2.1 `mcap_files` INSERT 语句移除 `cf_meta`, `cf_process`，改用 `metadata`, `process_state`
- [ ] 2.2 `assets` INSERT 语句：
  - 移除 `cf_meta`, `cf_algo`, `cf_tag`, `cf_files`
  - 使用 typed columns: `owner`, `reviewer`, `retention_tier`, `duration_ms`, etc.
- [x] 2.3 新增 `asset_tags` INSERT 语句（从 `cf_tag` 数据迁移）
- [x] 2.4 新增 `asset_algo_latest` INSERT 语句（从 `cf_algo` 数据迁移）
- [x] 2.5 `deliveries` INSERT 语句移除 `cf_meta`，改用 `metadata`
- [ ] 2.6 验证 `docker compose down -v && docker compose up -d` 后 seed 数据正确

---

## Milestone 3: 创建 Migration 删除列和索引

> 目标：新 migration 文件删除 `cf_*` 列和相关索引

- [x] 3.1 创建 `migrations/013_drop_cf_legacy_columns.sql`
- [x] 3.2 删除 `cf_meta` 表达式索引 (6 个)
- [x] 3.3 删除 `cf_tag` 表达式索引 (5 个)
- [x] 3.4 删除 GIN 索引 (5 个)
- [x] 3.5 删除 `assets.cf_*` 列 (4 个)
- [x] 3.6 删除 `mcap_files.cf_*` 列 (2 个)
- [ ] 3.7 删除 `deliveries.cf_meta` 列
- [ ] 3.8 验证 migration 幂等（可重复执行）

---

## Milestone 4: 更新 Schema 文档

> 目标：schema 文档不再包含 `cf_*` 列

- [ ] 4.1 `schemas/pg-phase0.sql` 删除 `cf_*` 列定义
- [ ] 4.2 `schemas/pg-phase0.sql` 删除 `cf_*` GIN 索引定义
- [ ] 4.3 `schemas/pg-phase0.sql` 更新注释，标注 `cf_*` 已移除
- [ ] 4.4 `docs/review/sql.md` 更新字段表（如需）
- [ ] 4.5 `backend/README.md` 更新架构说明

---

## Milestone 5: 验证与提交

> 目标：全部测试通过，提交变更

- [ ] 5.1 `go build ./...` 编译通过
- [ ] 5.2 `go vet ./...` 无警告
- [ ] 5.3 `go test ./...` 全部通过
- [ ] 5.4 本地 docker compose 验证：
  - 新环境：seed + migration 正确执行
  - 现有环境：migration 正确执行
- [ ] 5.5 提交变更，commit message: `refactor(db): remove cf_* legacy columns and indexes`

---

## 完成标准

- [ ] `migrations/002_seed.sql` 不包含 `cf_*` 列
- [ ] `migrations/013_drop_cf_legacy_columns.sql` 存在且幂等
- [ ] `schemas/pg-phase0.sql` 不包含 `cf_*` 列定义
- [ ] `go build ./...` + `go test ./...` 全部通过
- [ ] 本地 docker compose 环境正常启动

---

## Bigtable 说明

`internal/bigtable/` 包中的 `cf_*` 引用**不修改**：
- Bigtable 已 DEPRECATED，启动时 fail fast
- 代码保留作为历史参考
- 任务范围仅限 PostgreSQL 层
