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
