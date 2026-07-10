# Tasks — CYB-3296

## Checkpoint（当前停在此，待用户批准 + 第二审阅人）
- [ ] 确认算法 worker / SDK 使用的 API key 当前 scope（决定 :265-267 / :270,274 是否纳入本 PR）
- [ ] 确认所有合法写入方都带 `assets:write`（或 `*`），避免加 scope 后打断流水线
- [ ] 用户批准触碰 backend/routes + 授权语义

## 实现（批准后）
- [ ] routes.go: 4 条路由挂 `RequireScope("assets:write")`（:391/:393/:394/:410）
- [ ] （按 checkpoint 决策）算法/ mcap 写路由的 scope 处理
- [ ] 表驱动路由授权测试：每个 mutating 路由都要求写 scope
- [ ] `go test`（CGO_ENABLED=0）通过

## 验证
- [ ] dev 复测：只读 key 打这 4 条路由现在返回 403（回归上面的实测脚本）
- [ ] dev 复测：正常前端 JWT / 合法 key 仍能写（无回归）
