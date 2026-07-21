# Decisions — CYB-3709

## 2026-07-21 — 单次执行列表排除全部批量(产品选择)

- **Context**: 单次执行 tab 被 6 万+ 批量子任务淹没(cyb-3392b 刻意让子任务带徽章显示)。这是产品取舍,不是纯 bug。
- **Decision**: 用户经选项明确选择「排除全部批量」→ 单次 tab 用 `excludeBatch: true`,只显 ~4.4k 单次 run;批量子任务保留在批量 tab。
- **Alternatives**: 保持现状(子任务淹没);默认排除+加开关(前端改动更大)。用户选最简洁的排除全部。
- **Rationale**: 6 万子任务使单次 tab 失去意义;子任务在批量 tab 完整可见,无信息丢失;一行参数改。推翻 cyb-3392b 的前端选择(其"badge 无行可渲染"的顾虑不再适用 —— 现在就是不要批量行)。

## 2026-07-21 — 部署验证走 CI + 合并后 Chrome DevTools MCP

- **Context**: 改动含 `Frontend/`,AI-RULES 要求 Chrome DevTools MCP 验收。用户偏好 CI 部署(前端 push dev 自动部署 Cloudflare)。
- **Decision**: 走 PR→merge→CI 自动部署,**合并后**用 Chrome DevTools MCP 在 dev 验收(单次 tab ≈4.4k 无批量行;批量 tab 仍列子任务),而非预提交手动 `wrangler deploy`。
- **Rationale**: 与本轮既定的 CI 部署路径一致;改动极小(参数+注释),tsc build + 单测已覆盖行为。Rule precedence #1(用户偏好)。仍会做 MCP 验收,不省略。
