# Context files — CYB-3268

# Read these paths before implementing.

# One repo-relative path per line. Trailing # reason.

backend/routes/routes.go:385-390  # /assets/:id/actions 4 方法路由全切到 assetHandler
backend/internal/handlers/asset/handler.go:736-798  # createChildAssetRequest + CreateAction + 新增 ListActions/UpdateAction/DeleteAction
backend/internal/handlers/action/handler.go:42,105,189,244  # 老 Create/List/Patch/Delete (退役,代码保留供 backfill)
backend/internal/usecase/asset/usecase.go:760-768, 880-924, 1006-1050  # CreateChildAssetInput + Create + CreateChildAsset
backend/internal/usecase/asset/usecase.go  # 新增 ListActionsByParent/UpdateActionAsset/SoftDeleteActionAsset
backend/internal/repository/asset_repository.go  # 新增 ListByParentAndType/UpdateWithParentCheck/SoftDeleteWithParentCheck
backend/internal/usecase/action/usecase.go  # 老 actionUC(退役后保留供 backfill 复用)
backend/cmd/server/core.go:97,238  # actionLabelReg 注入(assetUC 是否复用?)
backend/internal/deliveryrules/asset_validator.go:189-204  # checkAction(已就绪,不动)
backend/internal/config/action_label_registry.go  # Validate(primary, labels)
backend/internal/searchindex/builder.go:42-244  # ES doc builder (lifecycle_state skip + actions[] 删)
backend/internal/elasticsearch/query_ir.go:163-174  # facetFieldPath(扩展 metadata.X facet)
backend/internal/elasticsearch/client.go:404-449, 670-700  # buildFilterClause / buildScalarClause
backend/deploy/local/elasticsearch/init-index.sh:55-69  # ES mapping(确认 metadata=flattened 不变)
backend/migrations/archive/016_add_actions.sql  # 老 actions 表 schema(字段映射参考 + backfill 用)
docs/agents/deploy-before-commit.md  # deploy 流程
docs/agents/deploy-verification.md  # deploy 后 smoke 规范
api/openapi.yaml  # POST/GET/PATCH/DELETE /assets/{id}/actions 全标 v2 / BREAKING
docs/review/api-guide.md  # actions 段重写为 first-class
sdk/src/cyber_databrew_sdk/assets.py  # SDK 4 个 action 方法对齐 first-class
scripts/smoke-asset-actions-dev.sh  # 现有 smoke 脚本(扩展覆盖 4 方法 + BREAKING 验证)
