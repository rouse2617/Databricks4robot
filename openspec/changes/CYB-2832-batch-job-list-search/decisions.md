# decisions — CYB-2832

## 2026-06-30 — Chrome DevTools MCP 不可用

- **Context**: diff 触及 `Frontend/`，规则要求用 Chrome DevTools MCP 验证 UI 行为
- **Decision**: MCP server 在 profile lock 清理后断连，无法继续；改用 `npm run build` 做 Tier L 验证
- **Alternatives**: 让用户本地打开 http://localhost:5174/pipeline?tab=executions&executionView=batch 手动验证
- **Rationale**: build 可确认类型和打包无误；功能验证需用户协助
