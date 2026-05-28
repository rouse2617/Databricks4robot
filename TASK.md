IMPLEMENT ES typed projection for Asset Model Phase 2。

项目背景：cyber-databrew，Go 1.25 + Gin + PostgreSQL + Elasticsearch。架构：handler → usecase → repository。

当前分支：feat/es-p2（基于 feat/pipeline-integration）
工作目录：/tmp/worktree-es-p2

任务：

在 backend/internal/searchindex/builder.go 中增加 ml_model.* 和 evaluation_report.* 的 typed projection 字段：

ml_model 投影字段：
- ml_model.framework (keyword)
- ml_model.architecture (keyword)  
- ml_model.metrics (flattened)
- ml_model.quantization (keyword)
- ml_model.artifact_uri (keyword)

evaluation_report 投影字段：
- evaluation_report.model_id (keyword)
- evaluation_report.dataset_id (keyword)
- evaluation_report.metrics (flattened)
- evaluation_report.tool (keyword)

参考现有 dataset.* 和 annotation_result.* 的实现模式。
需要写对应的单元测试。

强制要求：
- 每次修改后运行 make fmt && make vet（在 backend/ 目录）
- 为新逻辑写单元测试
- 提交：git add -A && git commit -m "feat(es): ml_model and evaluation_report typed projection"
