# CYB-3422 P3.1a Tasks

## Backend

- [ ] 1. `backend/internal/handlers/workflow/elastic_quota.go` 新文件:`ListElasticQuotas` handler
  - Deps:注入 K8s client(和 `resource_quota.go` 同模式)
  - List `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota` 全 namespace(用 dynamic client 或 controller-runtime client)
  - 解析 `.spec.min/max` 和 `.status.used`(resource.Quantity 转字符串)
  - 计算 `utilizationPercent`(used / max × 100,memory 用 bytes 计算)
  - 处理 CRD 不存在错误(`meta.IsNoMatchError`)→ 返回 200 空列表
- [ ] 2. `backend/routes/routes.go` 注册路由 `GET /api/v1/elastic-quotas`(和 `/api/v1/resource-quotas` 同权限组)
- [ ] 3. `backend/internal/handlers/workflow/elastic_quota_test.go` 单测
  - Mock K8s client 返回 3 个 ElasticQuota
  - 断言 JSON 输出字段完整
  - 断言 utilizationPercent 计算正确
  - 断言 CRD 不存在时返回 200 空列表

## Frontend

- [ ] 4. `Frontend/src/api/pipelineApi.ts` 加类型 `ElasticQuota` + 函数 `listElasticQuotas(): Promise<{items: ElasticQuota[]}>`
- [ ] 5. `Frontend/src/components/pipeline/PoolManager.tsx` 加 ElasticQuota panel
  - 在现有 ExecutionTarget 表格下方新 section
  - antd `<Table>` 展示 name / namespace / cpu-min-max-used-% / memory-min-max-used-%
  - 15s `useEffect` 轮询(和现有面板一致的 timer)
  - 空列表时隐藏整个 section(未装 Koordinator 兼容)
- [ ] 6. `Frontend/src/components/pipeline/__tests__/PoolManager.test.tsx` 加 case
  - Mock listElasticQuotas 返回 3 quota
  - 断言 table 渲染 3 行 + usage bar 显示
  - 断言空列表隐藏 section

## 验证

- [ ] 7. Tier L(build + vet + lint + 单测全过)
- [ ] 8. 本地 dev 起前端 + 后端,curl `/api/v1/elastic-quotas` 拿到 3 个 quota
- [ ] 9. 前端 dev 起,浏览器访问 `/registry` → pools tab 看 ElasticQuota panel(要在能通 GKE 集群的环境)
- [ ] 10. PR 到 dev,监听 auto-merge + deploy-dev
- [ ] 11. dev 环境 Chrome MCP 验证:https://cyber-databrew-dev.cyberorigin.ai/registry 打开看 pools tab 里的 ElasticQuota 面板显示 3 个池 + 实时状态
