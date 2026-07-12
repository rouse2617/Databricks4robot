# Decisions — CYB-3309

## 2026-07-12 — 方案从"物化列(A)"改为"删前端降级(Path 0)"

- **Context**: issue 最初假设"后端筛不了 algo_status,需物化列"。dev 运行时抓包(`POST /api/v1/queries/run` where `algo_status:eq:failed`)显示后端 `engine=postgres,mode=filter`、`total:0` 精确,list 与 facets 查询都带该 where。
- **Decision**: 后端零改动,删前端过时降级(client 过滤 + 黄条)。保留 EXISTS-any 语义。
- **Alternatives**: 物化列 + 触发器(A2)—— 过度工程,数据量用不上,且触 `migrations/` off-limits。
- **Rationale**: 最小化;信任已验证的后端行为。用户在知情下拍板 Path 0。详见 issue comment。

## 2026-07-12 — 本地组件测试环境损坏(非本 PR 引入)

- **Context**: `npm run test -- --run` 报 **42 failed test files / 0 failed 断言 / 267 passed**。失败是 `@testing-library/react/dist/pure.js` **加载即挂**(import 阶段),所有 `render()`/`renderHook()` 类测试文件受影响;纯逻辑套件(如 `assetsDiscoveryReducer.test.ts` 39 passed)正常。
- **验证非本 PR 引入**: `npm ci`(干净 lockfile)后仍 42 failed;`git stash` 我的 2 文件改动后跑,**同样 42 failed / 267 passed**,与改动后逐字一致 → 本 PR **0 引入新失败**。
- **Decision**: 不在 CYB-3309 内修前端测试环境(scope creep + 违反最小化)。本 PR 代码正确性依据:`npm run build` 通过、无残留引用、逻辑单测通过、后端行为运行时实证、**dev 部署后 Chrome DevTools MCP 真机验收**(deploy-verification 规定 MCP 验收 > 单测)。
- **Follow-up**: 仓库级前端测试环境(RTL 加载失败 / lockfile 漂移)建议单独开 CYB 跟踪修复。
