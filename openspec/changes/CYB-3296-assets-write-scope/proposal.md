# CYB-3296 — 写操作路由补 assets:write scope（P0 安全）

## Why
已在 dev 实测确认：一个 `scopes:["assets:read"]` 的只读 API key 可以调用多条**改数据**的路由（创建/改/删子资产、写评测），全程未被授权层拦截（返回 400/422 业务错误而非 403）。根因是这些路由漏挂 `RequireScope("assets:write")`，而 `RequireScope` 是唯一的 scope 执行点，handler 内部不兜底。API key 创建接受任意 scopes，故只读 key 可被合法铸造。

## What Changes
1. 给以下路由挂 `RequireScope("assets:write")`（`backend/routes/routes.go`）：
   - `POST /assets/:id/actions`（:391）
   - `PATCH /assets/:id/actions/:action_id`（:393）
   - `DELETE /assets/:id/actions/:action_id`（:394）
   - `POST /assets/:id/eval-results`（:410）
2. **待定（checkpoint 决策）**：算法生命周期写 `POST /assets/:id/algo/:algo_key/start|finish|reset`（:265-267）与 mcap 写（:270,274）同样未挂 scope。是否纳入本 PR 取决于算法 worker / ingest 使用的 API key 当前 scope——若它们没有 `assets:write`，直接加会打断流水线。需确认后决定：一并加 / 引入 `algo:write`、`mcap:write` / 暂缓另立。
3. 新增**表驱动路由授权测试**：断言每个 mutating（POST/PATCH/DELETE）路由都要求某个写 scope。堵住 P1-A 测试盲区（`fakeDB` 不校验、路由 scope 无整体断言），防止再漏。

## Impact
- Affected specs: 授权语义（谁能写资产）
- Affected code:
  - `backend/routes/routes.go`（⚠️ off-limits：路由/授权，需第二审阅人）
  - 新增 `backend/routes/*_authz_test.go`（表驱动）
- 风险：给路由加 scope 后，**只有携带 `assets:write`（或 `*`）的调用方**才能写。需确认所有合法写入方（前端 JWT user 已含 assets:write；SDK 静态 token=admin=*；算法 worker / SDK 的 API key 需逐一确认 scope）。这是本变更的主要回归面，须在 checkpoint 与部署前核对。

## 边界（必须）
本变更触碰 `backend/routes` + 授权语义（off-limits 区）：需用户批准 + 第二审阅人后方可实现。当前停在 checkpoint。
