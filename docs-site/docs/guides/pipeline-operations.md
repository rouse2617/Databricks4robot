# Pipeline Operations

本章介绍流水线操作的完整指南，涵盖算法运行的管理和流水线组件的使用。

## 算法运行管理

### 创建算法运行

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 创建运行（指定算法和资产）
run = client.algo_runs.create(
    algo_key="some-algo",
    asset_ids=["asset-1", "asset-2", "asset-3"],
)
print(f"Created run: {run.run_id}")
```

### 生命周期管理

```python
# 启动运行
client.algo_runs.start(run.run_id)

# 标记完成
client.algo_runs.finish(run.run_id)

# 取消运行
client.algo_runs.cancel(run.run_id)
```

### 状态查询

```python
# 获取单个运行详情
run = client.algo_runs.get("run-id-xxx")
print(f"Status: {run.status}, Algo: {run.algo_key}")

# 列出所有运行
runs = client.algo_runs.list(page=1, page_size=20)

# 查询受影响资产
assets = client.algo_runs.get_affected_assets("run-id-xxx")
```

## 注册中心

了解平台支持哪些算法、标签和指标：

```python
# 算法列表
for algo in client.registry.list_algos():
    print(f"Algo: {algo}")

# 标签列表
for tag in client.registry.list_tags():
    print(f"Tag: {tag}")

# 指标列表
for metric in client.registry.list_metrics():
    print(f"Metric: {metric}")
```

## 工作流

```python
# 操作工作流
workflows = client.workflows.list()
```

## MCAP 文件存储

### 列出文件

```python
files = client.storage.list_files(page_size=20)
```

### 获取文件信息

```python
info = client.storage.get_file_info("file-001")
```

### 下载 MCAP 文件

```python
# 下载文件（自动跟随 302 到 GCS 签名 URL）
client.storage.download_mcap("file-001", output_path="./data.mcap")

# 从资产下载 MCAP
client.storage.download_asset_mcap("asset-id-xxx", output_path="./asset.mcap")
```

### 遍历消息

```python
messages = client.storage.get_messages("file-001", start_ns=0, end_ns=1000000)
```

### 处理上传

```python
# 上传最终确认
client.storage.finalize_upload(
    mcap_file_id="file-001",
    storage_path="gs://bucket/path/to/file",
)
```
