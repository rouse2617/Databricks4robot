# Task: 验证批次一部署 + 修复烟雾测试 + 前端 E2E

## 背景

批次一（资产模型扩展 Phase 1 + 统一检索 P1）已实现、合入 `feat/pipeline-integration`、部署到 Cloud Run dev 环境。但烟雾测试有一个失败，前端 E2E 测试未完成。需要你验证部署状态、修复测试问题、完成 E2E 验证，并生成结论文档。

## 环境

- **Repo**: `/Users/rick/cyber-databrew`
- **分支**: `feat/pipeline-integration`
- **Cloud Run**: `cyber-databrew-backend-dev` (region: us-central-1, project: green-valley-442103)
- **后端端点**: `https://cyber-databrew-backend-dev-uc.a.run.app`
- **认证**: `X-Databrew-Token` header + Google IAP ID token (`gcloud auth print-identity-token`)
- **前端 URL**: `https://cyber-databrew-dev-uc.a.run.app`
- 使用 `source scripts/dev-backend-env.sh` 获取环境变量

## 步骤

### 1. 验证 Cloud Run revision 和 API

1.1 列出当前 Cloud Run revision，确认运行的是最新构建的镜像（最近部署的 revision）

1.2 直接 curl 验证以下端点（使用 dev-backend-env.sh 的 BASE 和 TOKEN）：
   - `GET /api/v1/asset-types/dataset/schema` → 应返回 200 + JSON Schema
   - `GET /api/v1/asset-types/annotation_result/schema` → 应返回 200
   - `GET /api/v1/asset-types/unknown/schema` → 应返回 404
   - `POST /api/v1/queries/run` → 搜索已有资产

1.3 重新跑烟雾测试:
   ```
   source scripts/dev-backend-env.sh && bash scripts/api-guide-smoke.sh
   ```
   如果 `asset type schema dataset` 还是 404，debug 原因（可能是路由注册问题、handler 未初始化、或镜像未正确部署）。

### 2. 前端 E2E 测试

2.1 用 Chrome DevTools MCP 打开 `https://cyber-databrew-dev-uc.a.run.app`
2.2 登录（如果需要，用 dev-token 或已有 session）
2.3 导航到资产搜索页
2.4 验证：
   - 搜索结果页是否能加载
   - dataset 类型资产是否能被搜索到
   - 搜索筛选/类型过滤功能
   - 如果有血缘关系数据，验证血缘关系可视化

### 3. 生成结论文档

3.1 把所有发现写入 `docs/review/verify-batch-one-deploy.md`，格式：
   - 部署验证结论
   - API 验证结果（每个端点通过/失败）
   - 烟雾测试结果（24passed/5failed 明细）
   - 前端 E2E 测试结果（截图/观察）
   - 发现的问题（如果有）及修复建议
   - 总体结论：批次一是否可视为完成

3.2 如果有代码 bug 需要修复：
   - 直接修复并提交
   - 在结论文档中记录修复内容

## 注意事项

- 用 Chrome DevTools MCP 时必须保证浏览器已正确打开页面
- 烟雾脚本中有 skip 的测试项（RUN_WRITES=1）不需要处理，那不是问题
- ES 在 dev 环境可能不可用，搜索相关测试如果因 ES 失败要记录但不算 blocker
- 文档要写清楚：验证了什么、结果怎样、有什么问题
