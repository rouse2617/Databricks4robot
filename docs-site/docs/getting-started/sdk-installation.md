# SDK Installation

Cyber Databrew Python SDK 是平台的主要编程接口，支持 Python 3.10+。

## 快速安装（公开 PyPI）

```bash
pip install cyber-databrew-sdk
```

## 内部安装（Artifact Registry）

如果你的团队使用 Google Artifact Registry 管理内部包，请按以下步骤操作：

### 前置条件

- Python 3.11+
- `gcloud` 已登录并有权限访问目标 GCP 项目

### 安装步骤

```bash
# 建一个项目用的虚拟环境（如果还没有）
python3 -m venv .venv
source .venv/bin/activate

# 仅首次：安装 Artifact Registry 认证插件
pip install keyrings.google-artifactregistry-auth

# 安装 SDK
pip install --extra-index-url \
  https://us-central1-python.pkg.dev/green-valley-442103/python-packages/simple/ \
  cyber-databrew-sdk
```

### 验证安装

```bash
python -c "from cyber_databrew_sdk import CyberDatabrewClient; print('OK')"
```

## 版本管理

SDK 版本由 Git 自动管理（基于 `hatch-vcs`）：

| 场景 | 版本号示例 | 方式 |
|------|-----------|------|
| Dev（无 tag） | `0.0.1.dev120` | Commit 数自动递增 |
| Release | `0.1.0` | 打 tag `sdk/v0.1.0` 后自动变为正式版 |

### 查看版本

```bash
pip show cyber-databrew-sdk

# 或
python -c "from cyber_databrew_sdk import _version; print(_version.__version__)"
```

### 升级 SDK

```bash
pip install --upgrade cyber-databrew-sdk
```

## 依赖说明

SDK 的核心依赖：

| 依赖 | 用途 |
|------|------|
| `httpx>=0.27.0` | HTTP 客户端（支持异步） |
| `pydantic>=2.11.0` | 数据模型验证 |
| `mcap>=1.1.1` | MCAP 文件操作 |
| `python-dateutil>=2.9.0` | 日期时间处理 |

可选依赖：

| 依赖 | 安装方式 | 用途 |
|------|---------|------|
| PyYAML | `pip install cyber-databrew-sdk[yaml]` | YAML 配置支持 |
| OpenTelemetry | `pip install cyber-databrew-sdk[otel]` | 分布式追踪 |

## Context Manager 用法（推荐）

```python
with CyberDatabrewClient(base_url="...", token="g-xxx") as client:
    assets = client.assets.list()
    for asset in assets:
        print(asset.asset_id)
# 退出时自动关闭 HTTP 连接
```
