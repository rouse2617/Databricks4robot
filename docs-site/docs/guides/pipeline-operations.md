# Pipeline Operations

本章介绍流水线操作的实践指南，涵盖算法运行的管理和流水线组件的使用。

## 算法运行管理

### 创建与启动

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

# 登记运行
run = client.algo_runs.create({
    "run_id": "R001abc123def456",
    "algo_name": "hand_track",
    "algo_version": "2.0",
    "algo_kind": "processing",
    "triggered_by": "manual:ops",
})

# 启动
client.algo_runs.start(run["run_id"])
```

### 完成与取消

```python
# 成功
client.algo_runs.finish(run["run_id"], {
    "status": "ok",
    "assets_processed": 10,
})

# 失败
client.algo_runs.finish(run["run_id"], {
    "status": "failed",
    "reason": "OOM",
})

# 取消
client.algo_runs.cancel(run["run_id"])
```

### 查询

```python
# 获取单个运行
run = client.algo_runs.get("R001abc123def456")
print(run["status"], run["algo_name"])

# 列表
result = client.algo_runs.list(page=1, page_size=20)
for item in result["items"]:
    print(item["run_id"], item["status"])

# 受影响资产
result = client.algo_runs.get_affected_assets("R001abc123def456")
```

## 注册中心

```python
# 列表查询（返回 dict，取 items 遍历）
algos = client.registry.list_algos()
for algo in algos["items"]:
    print(algo["key"], algo["version"])

tags = client.registry.list_tags()
for tag in tags["items"]:
    print(tag["key"])

states = client.registry.list_lifecycle_states()
for s in states["items"]:
    print(s)
```

## Pipeline Components

```python
# 列出组件
result = client.pipeline_components.list(q="camera", source="registry")
for comp in result.get("items", []):
    print(comp)
```

## 工作流

```python
workflows = client.workflows.list()
for wf in workflows.get("items", []):
    print(wf["name"])
```

## MCAP 文件存储

```python
# 列出文件
result = client.storage.list_files(page=1, page_size=20)
for f in result["items"]:
    print(f["mcap_file_id"], f["size_bytes"])

# 获取文件信息
info = client.storage.get_file_info("mcap0001")

# 下载 MCAP（自动处理 302 重定向到 GCS 签名 URL）
client.storage.download_mcap("mcap0001", output_path="./data.mcap")

# 获取资产关联的 MCAP 定位信息
locator = client.assets.get_mcap_locator("aset0001")
print(locator["asset_id"], locator["mcap"])

# 上传最终确认
client.storage.finalize_upload({
    "mcap_file_id": "mcap0001",
    "storage_path": "gs://bucket/path/to/file",
})
```
