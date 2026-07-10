# Tasks — CYB-3296

## Checkpoint
- [x] 实测确认 P0-1 可利用（dev：只读 key 打 4 条路由，返回 400/422 而非 403）
- [x] 确认 dev 现有合法写入方 key 都带 `assets:write`（test / databrew- public / hrp）→ 修这 4 条不影响
- [ ] 算法 worker / SDK 的 API key scope（决定 :265-267 / :270,274 是否纳入）—— 留作跟进
- [ ] 第二审阅人 merge（本 PR 不自动合并，等二审）

## 实现
- [x] routes.go: 4 条路由挂 `RequireScope("assets:write")`（actions POST/PATCH/DELETE + eval-results POST）
- [ ] 算法 start/finish/reset 与 mcap 写路由 —— 本 PR **不含**（需先确认 worker/ingest key scope，否则可能断流水线）；跟进
- [x] 路由授权测试 `routes_authz_test.go`：只读 key 打这 4 条路由断言 403（堵 P1-A 盲区）
- [x] `go test ./routes/`（CGO_ENABLED=0）通过；`go build ./...` OK

## 验证
- [x] 单测：只读 key → 4 条路由均 403
- [ ] dev 复测（合并后）：只读 key 403；正常前端 JWT / 合法 key 仍能写（无回归）
