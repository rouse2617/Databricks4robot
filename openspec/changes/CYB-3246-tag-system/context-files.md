# Context files — CYB-3246

## Phase 1（开放词汇 + 展示）
backend/internal/config/tag_registry.go            # Validate() 校验入口，Phase 1 放开处
backend/internal/config/tag_registry_test.go       # 校验单测
backend/internal/usecase/asset/usecase.go          # UpsertTag/Create 调用 Validate 的位置（~224/952/1076）
backend/config/tag_registry.yaml                   # 现有注册定义（Phase 2 seed 来源）
Frontend/src/components/assets/                     # 资产详情页标签展示组件（Task A）
docs/review/api-guide.md                            # tags 写入 API 文档

## Phase 2（注册表 DB 化）
backend/routes/routes.go                            # admin 路由组（admin / adminRO=AdminTokenOrAdminRole）
backend/internal/handlers/admin/                    # 现有 admin handler 模式参考（search reindex）
backend/internal/config/watcher.go                  # 现有 ConfigWatcher / Reload 机制
backend/migrations/                                 # off-limits：新建 tag_registry 表
Frontend/src/pages/SettingsPage.tsx                 # 标签管理界面挂载点
Frontend/src/api/                                   # 类型化 API client 目录

## 流程
docs/agents/AI-RULES.md                             # §API contract sync / §Schema changes / §Verification tiers
docs/agents/deploy-verification.md                  # 部署后针对性验证
