# Cloud Run 开发环境独有环境变量

本文件记录了 `cyber-databrew-backend-dev` Cloud Run 服务上配置的、**不在 K8s ConfigMap/Secret 中的环境变量**。

`backend-dev.sh` deploy 时会从 K8s ConfigMap `cyber-databrew-config` + Secret `cyber-databrew-secrets` 拉取环境变量，然后通过以下机制确保独有变量不丢失：

1. **`_OVERRIDE`** 显式覆盖（优先级最高）
2. **`preserve_all_cloudrun_env_vars`** 自动保留所有 Cloud Run 上已有的、不在排除列表中的 env var（兜底）

以下表格仅供参考，**新变量无需登记**，脚本会自动保留。| 变量 | 典型值 | 保护机制 | 必要性 |
|------|--------|----------|--------|
| `K8S_USE_METADATA_TOKEN` | `true` | 脚本 preserve（自动从当前 Cloud Run 读取） | **必须** — 启用 Workload Identity 自动 token |
| `K8S_AUDIENCE` | `https://34.59.48.233` | 脚本 preserve | **必须** — K8s API audience，与 `K8S_API_ENDPOINT` 一致 |
| `K8S_INSECURE_SKIP_VERIFY` | `false` | 脚本 preserve | **必须** — 通过 Secret Manager CA 证书验证 K8s API |

> `K8S_CA_DATA` / `K8S_CA_B64` 通过 Secret Manager 绑定（`cyber-databrew-dev-k8s-ca-data`），不由 env var 直接配置。
> `K8S_BEARER_TOKEN` 也通过 Secret Manager 绑定。
> `K8S_API_ENDPOINT` 由 `K8S_API_ENDPOINT_OVERRIDE` 控制，默认值 `https://34.59.48.233`。

## 流水线调度与资源限制

| 变量 | 典型值 | 保护机制 | 必要性 |
|------|--------|----------|--------|
| `PIPELINE_TEMPLATE_TOLERATIONS_JSON` | `[{"key":"nvidia.com/gpu",...}]` | `PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE` 或云上 preserve | **建议** — GPU 节点 toleration；缺失 → Pod 永久 Pending |
| `PIPELINE_RESOURCE_MAX_CPU` | `8` | `PIPELINE_RESOURCE_MAX_CPU_OVERRIDE` | 默认已覆盖 |
| `PIPELINE_RESOURCE_MAX_MEMORY` | `28Gi` | `PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE` | 默认已覆盖 |
| `PIPELINE_RESOURCE_MAX_DISK` | `250Gi` | `PIPELINE_RESOURCE_MAX_DISK_OVERRIDE` | 默认已覆盖 |
| `PIPELINE_RESOURCE_MAX_GPU` | `1` | `PIPELINE_RESOURCE_MAX_GPU_OVERRIDE` | 默认已覆盖 |
| `PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD` | `15m` | `PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD_OVERRIDE` | 默认已覆盖 |

## 运行时挂载（Secret、ConfigMap）

| 变量 | 典型值 | 保护机制 | 必要性 |
|------|--------|----------|--------|
| `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON` | `[{"id":"video-proc-dev-db",...},...]` | `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE` 或云上 preserve | **必须** — runtime secret 挂载；缺失 → `runtime secret ... is not available` |
| `PIPELINE_RUNTIME_SECRET_TARGET_IDS` | `default,cyber-databrew-dev` | `PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE` 或云上 preserve | 默认已覆盖 |

## 其他

| 变量 | 典型值 | 保护机制 | 必要性 |
|------|--------|----------|--------|
| `PRICING_CONFIG_PATH` | `/app/config/gcp_pricing.yaml` | `PRICING_CONFIG_PATH_OVERRIDE` | 默认已覆盖 |
| `ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION` | `2592000` | `ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION_OVERRIDE` | 默认已覆盖 |

## 保护机制说明

脚本 `backend-dev.sh` 使用两种保护机制：

### 1. `_OVERRIDE` 环境变量（推荐）

```bash
export PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE='[{"key":"nvidia.com/gpu","operator":"Equal","value":"present","effect":"NoSchedule"}]'
export PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE='[{"id":"video-proc-dev-db",...}]'
bash deploy/cloudrun/backend-dev.sh
```

显式可控、可审计。适用于首次部署或需要变更值时使用。

### 2. 自动全局保留（兜底）

`preserve_all_cloudrun_env_vars()` 在 deploy 前自动执行：
1. 读取当前 Cloud Run 上 **所有** env var
2. 跳过排除列表中的变量（`PORT`、`K8S_CA_DATA`、Secret Manager 引用等）
3. 跳过已经在 deploy 文件中的变量
4. 其余全部追加到 deploy 文件

**这意味着：无论你在 Cloud Run 上手动加了什么变量，下次 deploy 都不会丢失。** 不需要在脚本里注册、不需要记文档。

### 最佳实践

1. **首次部署**或**更换 AI 工具**时，总是用 `_OVERRIDE` 把必需的独有变量传进去
2. 日常部署可用 preserve 机制，前提是 Cloud Run 上已有正确的值
3. 部署后验证：`gcloud run services describe cyber-databrew-backend-dev --region=us-central1 --format=json | jq '.spec.template.spec.containers[0].env[] | select(.name | startswith("K8S_") or startswith("PIPELINE"))'`

## 出现过的问题

| 日期 | 症状 | 根因 | 修复 |
|------|------|------|------|
| 2026-06-27 | `0/73 nodes: 67 untolerated taint(s)` | `PIPELINE_TEMPLATE_TOLERATIONS_JSON` 被清 | `backend-dev.sh` 已加 preserve |
| 2026-06-27 | `runtime secret ... is not available` | `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON` 被 `--env-vars-file` 清掉 | 通过 `_OVERRIDE` 恢复 |
| 2026-06-27 | `runtime config projection: Unauthorized` | `K8S_USE_METADATA_TOKEN` 和 `K8S_AUDIENCE` 被清 | `backend-dev.sh` 已加 preserve |
