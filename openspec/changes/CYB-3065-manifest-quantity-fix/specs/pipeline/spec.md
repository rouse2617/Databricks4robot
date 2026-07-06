## MODIFIED Requirements

### Requirement: 终态 run 查询 workflow 详情不再直连 Argo
- **Before**: The system SHALL avoid querying the live Argo API for a workflow's detail view when the corresponding run has already reached a terminal state and the underlying workflow object still exists — but manifest serialization silently dropped CPU/memory resource quantities, so reconstruction failed (and fell back to querying Argo) for essentially every run whose pipeline declared resource requests/limits.
- **After**: The system SHALL correctly reconstruct the response from a persisted manifest for runs whose pipeline declares resource requests/limits, not only for resource-free pipelines.
- **Reason**: manifest 序列化(yaml.v3 反射)无法正确处理 K8s `resource.Quantity` 类型,导致几乎所有真实业务 pipeline 的 manifest 反序列化都会失败,使前一个需求的覆盖范围在实践中远小于预期。

#### Scenario: 终态 run 的 pipeline 声明了资源 requests/limits
- **Given** 一个已终态的 run,其 pipeline 定义中某个节点声明了 CPU 和/或内存的 requests/limits
- **When** 系统尝试从该 run 的 manifest 重建响应
- **Then** 重建成功,资源数值与提交时的原始声明一致,不触发对 Argo 的实时调用
