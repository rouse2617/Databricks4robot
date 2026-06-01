# Pipelines

**流水线（Pipeline）** 是 Cyber Databrew 中编排数据处理流程的核心概念。Pipeline 将数据摄入、算法处理、资产生成等步骤串联成可重复执行的自动化流程。

> **注意**：Pipeline 组件正在建设中。当前版本主要通过 **Algo Runs（算法运行）** 来执行算法任务。完整的 Pipeline 编排功能将在后续版本发布。

## Algo Runs（算法运行）

Algo Run 是 Pipeline 体系中的执行单元，代表一次算法在指定资产上的运行实例。

### 状态机

```
blocked → pending → running → ok
                            ↘ failed
```

`ok` 和 `failed` 状态可通过 reset 回到 `pending`。

### 核心字段

| 字段 | 说明 |
|------|------|
| `run_id` | 16 位字母数字，全局唯一 |
| `algo_name` | 算法名称（如 hand_tracking） |
| `algo_version` | 算法版本（如 2.0.0） |
| `algo_kind` | 算法类别（processing 等） |
| `status` | blocked / pending / running / ok / failed |
| `triggered_by` | 触发来源 |

### 可用算法

| algo_key | 说明 | 依赖 |
|----------|------|------|
| `env_analysis@1.0.0` | AI 场景分析 | 无 |
| `hand_tracking@1.2.0` | 手部追踪 | 无 |
| `deface@2.0.0` | 去人脸 | 无 |
| `action_annotation@1.0.0` | 动作标注 | hand_tracking + head_tracking + body_tracking |

### 依赖链

`action_annotation@1.0.0` 依赖三个算法。当所有依赖都完成（status=ok）后，系统自动将 `action_annotation` 从 `blocked` 变为 `pending`。

## Pipeline Components（建设中）

流水线组件管理器用于后续流水线配置管理，当前提供基础 CRUD。

## 注册中心

注册中心提供平台中可用的算法、标签、指标、生命周期状态和操作标签的元数据注册信息。

## 代码示例

具体的 SDK 和 API 调用示例见 [Pipeline 操作指南](../guides/pipeline-operations.md)。
