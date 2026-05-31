# Pipelines

**流水线（Pipeline）** 是 Cyber Databrew 中编排数据处理流程的核心概念。Pipeline 将数据摄入、算法处理、资产生成等步骤串联成可重复执行的自动化流程。

> **注意**：Pipeline 组件正在建设中。当前版本主要通过 **Algo Runs（算法运行）** 来执行算法任务。完整的 Pipeline 编排功能将在后续版本发布。

## Algo Runs（算法运行）

Algo Run 是 Pipeline 体系中的执行单元，代表一次算法在指定资产上的运行实例。

### 创建与启动

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 创建算法运行
run = client.algo_runs.create(
    algo_key="some-algo",
    asset_ids=["asset-1", "asset-2"],
)

# 启动运行
client.algo_runs.start(run.run_id)
```

### 完成与取消

```python
# 标记为完成
client.algo_runs.finish(run.run_id)

# 取消运行
client.algo_runs.cancel(run.run_id)
```

### 查询运行状态

```python
# 获取单个运行详情
run = client.algo_runs.get("run-id-xxx")

# 列出运行
runs = client.algo_runs.list(page=1, page_size=20)

# 获取受影响资产
assets = client.algo_runs.get_affected_assets("run-id-xxx")
```

## Pipeline Components（流水线组件）

SDK 提供了 Pipeline 组件管理器，用于后续流水线配置管理：

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 操作 Pipeline 组件
components = client.pipeline_components.list()
```

## 注册中心

注册中心提供了平台中可用的算法、标签、指标等元数据注册信息：

```python
# 列出可用算法
algos = client.registry.list_algos()

# 列出可用标签
tags = client.registry.list_tags()

# 列出可用指标
metrics = client.registry.list_metrics()

# 列出生命周期状态
states = client.registry.list_lifecycle_states()

# 列出操作标签
labels = client.registry.list_action_labels()
```

## 相关工作流

平台还支持 Workflow 管理，用于更复杂的工作流编排：

```python
# Workflow 操作
workflows = client.workflows.list()
```
