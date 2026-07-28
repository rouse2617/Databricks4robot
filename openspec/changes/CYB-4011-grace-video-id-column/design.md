# Design — CYB-4011

## Context

已验证 DataBrew mcap ↔ Grace video 的可靠 join key 是 `raw_hash_md5`（= GCS 文件名），可 1:1 命中。本次只**预留显式列**承载 Grace video 主键，不做任何数据填充。设计目标：与已有「flatten mcap 列到 assets」（迁移 `20260722100000`）模式完全对齐，最小惊喜。

## Decisions

### D1 — 列类型用 `text`，不用 `uuid`
- **决定**：`grace_video_id text`（可空），绑定走 `nullableText()`，与 `camera_model`/`data_source` 同类。
- **理由**：Grace video id 是 UUID，但用户可能拿到非规范/遗留值；text 不会因单个坏值导致写入失败（对照迁移里 `device_id uuid` 需要 per-row 异常兜底）。前端/契约按 UUID 语义描述即可。
- **备选**：`uuid` 列 —— 需要 `NULLIF(x,'')::uuid` 兜底，且对空/坏值更脆，放弃。

### D2 — 单一事实源在 `mcap_files`，镜像到 `assets`
- **决定**：`grace_video_id` 主列加在 `mcap_files`；沿用 flatten 模式在 `assets` 上加同名镜像列 + 部分索引；raw_mcap 资产创建时（`handlers/mcap/handler.go` 的 placeholder asset）把值一并镜像。
- **理由**：`/assets` 查询页与 mcap 详情页都要显示；flatten 模式已经让 `/queries/run` 能直接 select/filter assets 列，无需 join mcap_files。
- **注意**：本次**不含回填 DO block**，所以既有行两张表都为 NULL；新写入路径会同时落 mcap_files 与镜像 asset 列。

### D3 — filter-only，不 facet
- **决定**：注册进 `filter/assets_fields.go` 的 `exactFieldSpecs`（可 `grace_video_id:eq:`），**不**加入 `queryplan/planner.go` 的 `PGSupportedFacetFields`、`postgres/asset_facets.go`、`elasticsearch/query_ir.go` 的 facet 路径。
- **理由**：Grace UUID 高基数，facet 会产生海量桶且无意义 —— 与迁移注释里 `device_id`/`collector_id`/`scene_id` 被刻意排除 facet 的判断一致。
- **前端**：加进 `AddFilterPopover`（string 类型可过滤），**不**加进 facet 侧栏。

### D4 — 索引
- **决定**：仅在 `assets` 上建部分索引 `idx_assets_grace_video_id ... WHERE is_deleted=FALSE AND grace_video_id IS NOT NULL`（与 flatten 索引一致）；`mcap_files` 是否建索引视 D2 事实源查询需要——本次 mcap_files 侧不加索引（无按 grace_video_id 查 mcap_files 的路径），保持最小。

## Data Flow

```
mcap create (POST) ──req.grace_video_id──▶ models.McapFile.GraceVideoID
                                              │
                          McapFileRepo.Set ───┼──▶ mcap_files.grace_video_id
                                              │
        placeholder raw_mcap asset (mirror) ──┴──▶ assets.grace_video_id
                                                      │
     GET mcap / list / /queries/run ◀── select+scan ─┘ (Asset/McapFile JSON: grace_video_id)
                                                      │
                            searchindex.builder ──────┴──▶ ES doc["grace_video_id"] (top-level + mcap nested)

Frontend McapDetailDrawer ◀── McapFile.grace_video_id ── (copy + link to grace.cyberorigin.ai/videos/<id>, else 「未关联」)
Frontend AddFilterPopover ── grace_video_id (string, filter-only)
```

## Migration & Risk

| 项 | 说明 | 风险 | 缓解 |
|---|---|---|---|
| ALTER 加 2 列 | `mcap_files` + `assets` 可空 text | 低（非破坏，NULL 默认） | 无回填，既有行不动 |
| 部分索引 | assets 上 1 个 | 低 | `WHERE grace_video_id IS NOT NULL`，空表段几乎零成本 |
| off-limits `backend/migrations/` | 需授权 | — | Linear CYB-4011 记录发起人口头授权 + decisions.md |
| scan/column 索引错位 | select 列表与 Scan 参数必须同序 | 中（运行时 panic） | 每处 select+scan+插入占位符同步核对（见 tasks 校验项） |
| 部署顺序 | 引用新列的代码需迁移先行 | 中（500） | `apply-migration-dev.sh` 先跑，再部署 backend |

## Verification Tier
**Tier L** — 触及 `backend/internal/postgres` + `handlers/` + OpenAPI 契约 + `Frontend/` UI 路由。含 API 契约同步，前端改动需 Chrome DevTools MCP 在 dev 验证。
