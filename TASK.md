IMPLEMENT Asset Model Phase 2 基础层。

项目背景：cyber-databrew，Go 1.25 + Gin + PostgreSQL + Elasticsearch。架构：handler → usecase → repository。

当前分支：feat/asset-model-p2（基于 feat/pipeline-integration）
工作目录：/tmp/worktree-asset-p2

任务：

1. 创建 migration 044：backend/migrations/044_asset_model_p2.sql
   - 扩展 asset_relations.relation_type CHECK，新增：trained_from, evaluated_on, validated_on, configured_by, fine_tuned_from, features_from, tested_on, evaluates, compares_to, calibrated_from, generated_by
   - 给 asset_relations 加 metadata JSONB 列
   - 扩展 assets 约束允许 ml_model/evaluation_report 类型没有 mcap_file_id

2. 在 backend/internal/models/asset_type_schema.go 中注册 ml_model 和 evaluation_report 类型：
   - ml_model 字段：framework(string), architecture(string), metrics(object), quantization(string), artifact_uri(string), training_run_id(string), base_model(string)
   - evaluation_report 字段：model_id(string), dataset_id(string), metrics(object), evaluated_at(string), tool(string), report_uri(string)
   - 添加对应的 validate 函数

3. 在 backend/internal/models/asset_type_schema_test.go 添加 ml_model/evaluation_report 的单元测试

强制要求：
- 每次修改后运行 make fmt && make vet（在 backend/ 目录）
- 为新逻辑写单元测试
- 分层：handler → usecase → repository
- 提交：git add -A && git commit -m "feat(asset-model): phase 2 ml_model and evaluation_report types"
