# 回归测试规约 (Ralph 每轮 VERIFY 必读)

> 循环策略见 `.agent/loop/RALPH-STRATEGY.md`(iter 23 起,DISCOVER 事件驱动优先)

## 环境
- **前端**: https://cyber-databrew-dev.cyberorigin.ai
- **后端**: https://cyber-databrew-backend-dev-234851712830.us-central1.run.app
- **回归入口页**: `/runs?executionView=batch`
- **用户身份**: ruipeng.huang@cyberorigin.ai(邮箱记账用,自动化跳 SSO)
- **Auth**: `X-Databrew-Token: dev-token`
  - 后端已确认接受(config.go:221 默认值)
  - 前端调 `/api/v1/*` 时也识别该 header([pipelineClient.ts:17-38](Frontend/src/api/pipelineClient.ts))
  - 浏览器自动化:在页面加载前用 `localStorage.setItem('databrew_token','dev-token')` 或 `document.cookie` 注入

## 已知稳定夹具(用于 smoke,勿删)
- **完成的 batch**: `8f5cd676-5641-4759-ab1d-a6b056fad0bb`(paused,10000 items,9835 completed)
  - `GET /api/v1/backfill/{id}` 期望 200 + `total_count=10000`
  - `GET /api/v1/backfill/{id}/node-summary` 期望 200 + 非空 nodes 数组

## VERIFY 步骤(每轮 CI merge 后跑,或 DISCOVER 空转轮次跑)
按下列顺序,任一失败即写入 backlog 新建 P1 「回归失败: <case>」并 CURRENT_STATE→IDLE。

### T1 [API smoke,后端存活]
```bash
curl -sS -o /dev/null -w "%{http_code}" -H "X-Databrew-Token: dev-token" \
  "$BACKEND/api/v1/backfill?pageSize=1"
```
期望 200。非 200 → P0 后端 down。

### T2 [已知 batch 详情,契约]
响应形状为 `{job:{...}}` 包裹,字段 camelCase。
```bash
curl -sS -H "X-Databrew-Token: dev-token" \
  "$BACKEND/api/v1/backfill/8f5cd676-5641-4759-ab1d-a6b056fad0bb" | jq '.job.totalCount, .job.completedCount, .job.failedCount, .job.status'
```
期望 totalCount=10000, completedCount≥9835, status ∈ {paused,failed,completed}。

### T3 [node-summary 字段完整]
顶层 `{nodes:[...], subtasks, dataCoverage, ...}`;node 字段 camelCase。
```bash
curl -sS -H "X-Databrew-Token: dev-token" \
  "$BACKEND/api/v1/backfill/8f5cd676-5641-4759-ab1d-a6b056fad0bb/node-summary" | jq '.nodes | length, (.nodes[0] | keys)'
```
期望 length>0,`.nodes[0] | keys` 严格等于 `["attempted","counts","dagOrder","displayName","failureRate","pipelineNodeId"]`;并断言 `.nodes[0].attempted >= 0` 且 `.nodes[0].counts | type == "object"`。

### T4 [数据一致性,item 汇总 ≈ job 计数]
调 backfill/{id} 拿到 completed_count + failed_count,再调 /attempts?limit=1 抽查一个 item。若 completed+failed > total_count → 数据一致性 bug,P0。

### T5 [前端 UI 渲染 /runs?executionView=batch]
使用 in-app browser (`mcp__Claude_Browser__navigate`):
1. 打开 `${FRONTEND}/runs?executionView=batch`
2. 通过 `mcp__Claude_Browser__javascript_tool` 注入 `localStorage.setItem('databrew_token','dev-token'); location.reload()`
3. `mcp__Claude_Browser__read_page` 验证:
   - 页面标题或表格 header 包含 "Batch" / "批量"
   - 至少 1 行 batch 数据(有 UUID pattern)
   - 无红色 error banner
4. 点击第一个 batch → URL 变为 `/pipeline/batch/{uuid}` → 进度条 SVG/元素存在

### T6 [控制台无 error]
`mcp__Claude_Browser__read_console_messages onlyErrors:true` — 期望空(允许 warning)。

### T7 [regression for #464: gone workflow logs → 404, not 500]
锚 commit: 1a7b81cd `fix(backend): return graceful not-found for logs of gone workflows/pods`。
**契约细节确认 2026-07-18 UTC**:`nodeId` 查询参数是**必需的**——不带 nodeId 返回 `HTTP=400 code=INVALID_ARGUMENT`("workflow name and nodeId are required"),不进入 workflow 查找路径,不能验证 #464。必须带 nodeId 才能触发 argo 查询并得到 404 分支。
```bash
curl -sS -o /tmp/t7.json -w "%{http_code}" -H "X-Databrew-Token: dev-token" \
  "$BACKEND/api/v1/workflows/ralph-nonexistent-workflow-t7/logs?nodeId=nonexistent-node-t7"
```
断言:`HTTP=404` 且 `jq -r .code /tmp/t7.json` == `WORKFLOW_NOT_FOUND`。
非 404 或 500 → P1「#464 回归」。若忘带 nodeId 得到 400,不是回归,是命令写错。GET only,幂等,<2s。

## DISCOVER 第一信号源 (M001)
- DISCOVER 前先跑 `.agent/loop/scripts/discover-signals.sh`,任一 section 非空 → 优先响应(review-requested / failed CI / stalled 自己 PR / backend ERROR / stranded 日志)。
- **openspec 幽灵过滤 (M002)**:DISCOVER `openspec/changes/*` 时,每个候选先跑 `.agent/loop/scripts/openspec-ghost-filter.sh <change-name>`,非零退出 (GHOST) 直接跳过——代码已合入但目录未归档,不算真信号。自测:CYB-3386/1013/1014 均正确判定为 GHOST。
- 全空 → 才进入 backlog 扫描。

## Ralph 自行扩充测试用例的规则
- 每 5 轮 DISCOVER **必须**新增至少 1 条基于当前代码 diff 的场景测试(如新 feature、新 route),追加到本文件 T7、T8...
- 新测试必须:客观可验证(HTTP 码 / 输出 grep) + 幂等 + <30s。
- 涉及写入(POST /pause /resume /rerun)必须先在 dev 环境自建**新 sandbox batch**,禁止在生产已有 batch 上试。

## 失败上报
任何 Tn 失败 → journal.md LOG 顶部 + backlog P1 新增,附:请求命令 + 实际响应 + 期望。
