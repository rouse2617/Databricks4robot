# CYB-4323 decisions

## 2026-07-28 — 跳过合并前 frontend dev 部署 + Chrome DevTools MCP 验证

- **Context**: 本 PR 触及 `Frontend/`,按 [`deploy-before-commit.md`](../../../docs/agents/deploy-before-commit.md)
  + [`deploy-verification.md`](../../../docs/agents/deploy-verification.md) §1.3,合并前应部署
  frontend dev(共享 Cloudflare Worker `cyber-databrew-dev.cyberorigin.ai`,**无隔离预览**)
  并由 Agent 跑 Chrome DevTools MCP 目视验收。用户在本次会话中明确指示:「ui 修复好了那就
  直接提交 pr 到 dev」,即跳过部署验证直接开 PR。
- **Decision**: 跳过合并前的 frontend dev 部署与 MCP 验收,直接提 PR 到 `dev`。合并后由
  `.github/workflows/deploy-dev.yml`(npm run build → dist→site → `wrangler deploy --env dev`)
  自动部署到 dev 前端,MCP 目视验收顺延到彼时补做。
- **Alternatives**:
  - (A) 部署共享 dev Worker + MCP 验真实数据 —— 完全合规,但把未合并改动推给所有 dev 用户。
  - (B) 本地 `wrangler dev` + MCP —— localhost 非 `*-dev.*`,`_worker.js` 会代理到 **prod**
    后端,拿不到 dev 真实数据;且不算「已部署 dev」。
- **Rationale**:
  - [Rule precedence](../../../docs/agents/AI-RULES.md#rule-precedence) 优先级 1(用户当前消息
    显式指示)高于优先级 4(deploy-before-commit)。用户是平台 owner,已知情裁定。
  - 改动为**纯视觉**(柱宽/渐变、KPI 位置、双 Y 轴刻度配色、X 轴标签对比度),已通过
    Tier M 本地验证:biome(零新增问题)、vitest 4/4、`npm run build` ✓。运行时行为、接口、
    数据口径均未变,回归风险低。
  - dev 为自动部署环境,合并即触发 GHA build+deploy;PR 中已注明 MCP 验收顺延,可追溯。
