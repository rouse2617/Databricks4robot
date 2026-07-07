# pipeline Specification

## Purpose

Transpile and execute pipeline runs as Argo Workflows, including cost attribution for GKE-native billing tooling.

## Requirements

### Requirement: Pipeline run pods carry cost-tracking labels

The system SHALL attach labels identifying the batch job, template, and owner to every pod created for a submitted pipeline run, whenever the corresponding identifier is known at submission time — including for backfill batch items, whose owner is sourced from the owning job's creator.

### Requirement: Cost-tracking label values remain valid Kubernetes labels

The system SHALL sanitize any identifier before using it as a label value, so that no pipeline run submission fails or is rejected by Kubernetes due to an invalid label value.

### Requirement: 终态 run 查询 workflow 详情不再直连 Argo

The system SHALL avoid querying the live Argo API for a workflow's detail view when the corresponding run has already reached a terminal state and the underlying workflow object still exists. This SHALL hold for pipelines that declare CPU/memory resource requests or limits on any node, not only for resource-free pipelines.

### Requirement: 降级路径数据不完整时安全回退,不返回残缺响应

The system SHALL fall back to querying the live Argo API when the data needed to construct a degraded (non-Argo) response is missing or invalid, rather than returning a response with silently missing or incorrect fields.

### Requirement: WorkflowDetailPage 加载 run ledger 数据不重复请求

The system SHALL fetch a run's ledger sub-resources (events/asset-nodes/cost-summary/inputs/outputs/runtime) at most once per mount, manual refresh, or poll tick, rather than once per independent trigger path.

### Requirement: 批量任务终态飞书通知

The system SHALL send a Feishu text notification exactly once when a batch (backfill) job transitions from a non-terminal status to a terminal status (`completed` or `failed`), regardless of outcome, including a summary of total/succeeded/failed item counts and a link to the job detail page. This SHALL hold even when multiple backend instances observe the same transition concurrently — exactly one notification is sent. The feature SHALL be a no-op (no error, no network call) when no webhook is configured.

### Requirement: 确定性失败的 run 不被复活逻辑倒退覆盖

The system SHALL treat a run that has reached a definitive terminal failure — a `Failed`/`Error` status carrying a real, non-transient error message (e.g. rejected by the resource guard before any workflow is created) — as final, and SHALL NOT regress it back to an active status (`Pending`/`Running`) via any "waiting for workflow creation" or misclassification-recovery heuristic. A terminal state carrying a known stale/transient "workflow unavailable" message is a misclassification, not a definitive failure, and MAY still be recovered.
