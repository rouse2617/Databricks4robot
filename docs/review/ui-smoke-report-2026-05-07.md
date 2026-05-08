# UI 深钻与 API 冒烟报告（2026-05-07）

范围：前端侧边栏逐页深钻（每页可见控件/弹窗按钮/主要异常路径覆盖一次），同时记录 `network` 与 `console` 证据。

结论：本报告已按“历史问题（已修复）+ 后续建议（非阻塞）”重排，不再保留“仍失败待办”表述。

---

## 已修复并验证（已从问题列表剔除）

1. `资产页` structured 模式携带 `_fulltext` 时的查询链路
   - 预期：structured 含全文 token 时，不再混用列表接口，统一走检索主链路（Query API）。
   - 证据（代码落地）：`Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts` 在 structured + fulltext 场景走专门检索分支。

2. 后端空 between 输入的处理（避免 500）
   - 预期：`filter=duration_ms:between:,` 这类 bounds 非空缺失输入返回 `400 INVALID_FILTER`。
   - 证据（代码落地）：`backend/internal/filter/build.go` 的 `parseBetweenValuePair` 在 lo/hi 为空时返回错误 `filter: between bounds must be non-empty`，并在资产 handler 路径中转为 `BadRequest(CodeInvalidFilter)`。

3. 交付弹窗 `asset_ids` 手动输入 + 去重 + 幂等提交
   - 预期：支持逗号/换行分隔手动填入 `asset_ids`，自动去重；继续要求 `Idempotency-Key`。
   - 证据（代码落地）：`Frontend/src/components/deliveries/CreateDeliveryModal.tsx` 对手动输入 `split(/[\n,]/)`、`trim`、`Set` 去重，并提交 `asset_ids: effectiveAssetIds`；后端 delivery handler 中对每个 asset id 做 `id.ValidateAssetID` 校验且强制 `Idempotency-Key`。
   - 本轮回归（network）：`reqid=507 POST /api/v1/deliveries` 返回 `201 Created`

4. 事件流页 `asset_id` 规则统一为 8 位字母数字
   - 预期：事件流页只接受后端 canonical 的 8 位 `^[0-9A-Za-z]{8}$` asset id；文案与测试同步。
   - 证据（代码落地）：`Frontend/src/pages/EventsPage.tsx` 使用 `isCanonicalAssetId` 校验并相应提示/不触发请求；后端 `backend/internal/id/assetid.go` 的 `ValidateAssetID` 亦为 8 位 alnum。
   - 本轮回归（network）：`reqid=516 GET /api/v1/assets/7VBGimAO/events?limit=100` 返回 `200 OK`

---

## 历史问题（本轮已修复）

### P0：湖仓验证关键接口曾返回 500（已修复）

1. `湖仓验证` - Training Assets
   - 页面：`湖仓验证`（`/analytics` 内侧边栏项“bar-chart 湖仓验证”）
   - 请求：`GET /api/v1/lakehouse/training-assets?snapshot_id=mvp_hand_tracking_quality_v1`
   - 历史状态码：`500 Internal Server Error`
   - `network reqid`：本轮回归，`reqid=13` → `200 OK`
   - 期望：该接口 2xx 且前端展示“训练候选/关键查询验证”相关结果。
   - 结果：当前不再复现同类 500。

2. `湖仓验证` - Recompute Candidates
   - 页面：`湖仓验证`
   - 请求：`GET /api/v1/lakehouse/recompute-candidates?algo_key=hand_tracking%401.2.0&target_version=next`
   - 历史状态码：`500 Internal Server Error`
   - `network reqid`：本轮回归，`reqid=14` → `200 OK`
   - 期望：该接口 2xx 且“哪些历史 asset 要重算？”面板能正常展示候选。

结论：P0 两个接口已恢复可用。

---

### P1：湖仓验证 Trino 语义错误（已修复）

在 `湖仓验证` 页展开的“关键查询验证”面板里，本轮曾出现 Trino semantic error（200 OK 但 query failed），现已修复：

- 触发示例：`Column 'env' cannot be resolved`
- 触发示例：`Column 'algo.algo_key' cannot be resolved`

修复方式：对齐 `backend/internal/handlers/lakehouse/handler.go` 中 Trino SQL 列名与 Iceberg MVP schema（去掉不存在的 env/task、用 algo_name/algo_version 代替 algo_key），并对 `algo_key` 参数做解析。

---

### P2：前端可访问性（已修复）

- 现象：`[issue] A form field element should have an id or name attribute`（已修复）
- 验证：重新 `new_page` 加载 `/events` 后，Chrome DevTools MCP `console` 未再出现该告警（不再复现 `msgid=83/88/90`）。

结论：对应告警在本轮回归中已消失。

---

## 后续建议（非阻塞）

1. 持续做 `/analytics` 回归，防止 Trino 列名漂移再次引入 500/语义错误
2. 在 UI 自动化中保留 `/events` 表单可访问性断言，避免回归

---
## 本轮逐页深钻补齐证据（页面级）

1. `dashboard 概览`（`/dashboard`）
   - 覆盖：顶部分段切换 `资产维度/算法维度/交付维度`（含告警提示的 `close`）

2. `file MCAP 文件`（`/mcap-files`）
   - 覆盖：`状态` 下拉切到 `pending` 后触发空状态（`file MCAP 文件(0)` + `暂无 MCAP 文件记录`）
     - 证据：`reqid=580 GET /api/v1/mcap-files?...&ingest_state=pending` → `200 OK`
   - 覆盖：切回 `全部状态` 并点击分页 `下一页`
     - 证据：`reqid=582 GET /api/v1/mcap-files?page=2&page_size=20` → `200 OK`

3. `robot 算法处理`（`/algo`）
   - 覆盖：状态过滤复选框（`成功/失败/运行中/待处理/已阻塞`）组合切换
   - 覆盖：行内主要动作 `check-circle` 打开任务详情面板，并点击 `reload 重置`
     - 证据：`reqid=576 POST /api/v1/assets/7VBGimAO/algo/env_analysis@1.0.0/reset` → `200 OK`
   - 覆盖：矩阵单元格动作图标 `minus-circle` / `lock` / `sync` 各点击一次打开详情面板
   - 证据（以 `sync(运行中)` 为例）：在详情面板点击 `reload 重置` 后触发 `POST /api/v1/assets/1UPTZPT9/algo/body_tracking@1.0.0/reset` → `409`（本轮 `reqid=14`）

