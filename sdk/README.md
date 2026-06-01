# cyber-databrew-sdk

Python SDK for the [cyber-databrew](https://github.com/CyberOrigin2077/cyber-databrew) data platform.

---

## Install

### 前置条件

- Python 3.10+
- `gcloud` 已登录并有权限访问 `green-valley-442103` 项目

### 安装命令（算法工程师用）

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

验证安装：

```bash
python -c "from cyber_databrew_sdk import CyberDatabrewClient; print('OK')"
```

### 查看版本 / 升级

```bash
# 查看当前安装版本
pip show cyber-databrew-sdk

# 升级到最新版
pip install --upgrade --extra-index-url \
  https://us-central1-python.pkg.dev/green-valley-442103/python-packages/simple/ \
  cyber-databrew-sdk

# 安装指定版本
pip install --extra-index-url \
  https://us-central1-python.pkg.dev/green-valley-442103/python-packages/simple/ \
  'cyber-databrew-sdk==0.0.1.dev120'
```

### 版本管理

SDK 版本由 git 自动管理（`hatch-vcs`）：

| 场景 | 版本号示例 | 方式 |
|------|-----------|------|
| Dev（无 tag） | `0.0.1.dev120` | git commit 数自动递增，每次 `make publish` 出新版 |
| Release | `0.1.0` | 打 tag `sdk/v0.1.0` 后自动变成正式版 |

查看当前版本：

```bash
pip show cyber-databrew-sdk
# 或
python -c "from cyber_databrew_sdk import _version; print(_version.__version__)"
```


---

## 配置

有三种方式配置 SDK，优先级从高到低：

### 1. 构造参数（最高优先级）

```python
client = CyberDatabrewClient(
    base_url="https://your-api.run.app",
    token="g-xxx",
    timeout=60.0,
)
```

### 2. 环境变量

```bash
export CYBER_DATABREW_BASE_URL="https://your-api.run.app"
export CYBER_DATABREW_TOKEN="g-xxx"
export CYBER_DATABREW_EMAIL="user@company.com"   # 仅审计用
export CYBER_DATABREW_TIMEOUT=60
```

### 3. 配置文件

`~/.cyber-databrew/config.yaml`：

```yaml
base_url: https://your-api.run.app
default_token: g-xxx
default_email: user@company.com
timeout: 60
```

### 4. 远端获取

SDK 启动时会自动调用后端 `GET /api/v1/sdk-config` 获取配置覆盖。后端不可用时静默回退。

### 5. 内置默认值

| 参数 | 默认值 |
|------|--------|
| `base_url` | `http://localhost:8080` |
| `timeout` | 30.0 秒 |
| 所有 endpoint 路径 | 内置 `/api/v1/...` 模板 |

### 认证方式

SDK 默认使用 `X-Databrew-Token` 进行程序化调用：

```python
client = CyberDatabrewClient(base_url="...", token="g-xxx")
```

或通过环境变量配置：

```bash
export CYBER_DATABREW_TOKEN="g-xxx"
```

浏览器端不是直接保存 SDK token；前端通过 `POST /api/v1/auth/email-login` 提交邮箱，后端校验域名后写入 `databrew_session` JWT cookie。后续浏览器请求依赖该 HttpOnly cookie 访问受保护接口。

```bash
curl -X POST "$BASE/api/v1/auth/email-login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@cyberorigin.ai"}' \
  -c /tmp/databrew.cookie

curl "$BASE/api/v1/auth/me" -b /tmp/databrew.cookie
```

`POST /api/v1/auth/login` 仍保留给静态 token 会话登录和兼容场景；SDK 直接传 `token` 时不需要先调用登录接口。

---

## 使用场景

### 资产管理

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 查询资产（支持过滤、分页）
assets = client.assets.list_all(
    page=1,
    page_size=20,
)
print(assets["items"])

# 获取单个资产详情
asset = client.assets.get("asset-id-xxx")
print(asset["asset_id"], asset["status"])

# 创建资产
asset = client.assets.create({
    "mcap_file_id": "file-001",
    "t_start": 1000000000,
    "t_end": 2000000000,
})

# 更新资产
asset = client.assets.update("asset-id-xxx", {"status": "reviewed"})

# 批量获取
assets = client.assets.batch_get(["id1", "id2", "id3"])

# 血缘 / 谱系
lineage = client.assets.get_lineage("asset-id-xxx")
provenance = client.assets.get_provenance("asset-id-xxx")

# 标签管理
client.assets.set_tags("asset-id-xxx", [{"key": "priority", "value": "high"}])
client.assets.delete_tag("asset-id-xxx", key="priority")

# 记录浏览 / 收藏
client.assets.record_view("asset-id-xxx")
client.assets.set_favorite("asset-id-xxx", True)
```

### MCAP 文件存储

```python
# 列出文件
files = client.storage.list_files(page_size=20)

# 获取文件信息
info = client.storage.get_file_info("file-001")

# 下载 MCAP 文件（自动跟随 302 到 GCS 签名 URL）
client.storage.download_mcap("file-001", output_path="./data.mcap")

# 从资产下载 MCAP
client.storage.download_asset_mcap("asset-id-xxx", output_path="./asset.mcap")

# 上传最终确认
client.storage.finalize_upload({
    "mcap_file_id": "file-001",
    "storage_path": "gs://...",
})

# 遍历消息
messages = client.storage.get_messages("file-001")
```

### 交付管理（Delivery）

```python
# 创建交付
delivery = client.delivery.create({
    "customer_id": "cust-001",
    "items": [
        {"asset_id": "asset-1", "mcap_file_id": "file-1"},
        {"asset_id": "asset-2", "mcap_file_id": "file-2"},
    ],
})

# 两步提交：草稿 → 添加内容 → 提交
draft = client.delivery.draft({"customer_id": "cust-001"})
client.delivery.add_items(draft["delivery_id"], {"items": [...]})
client.delivery.commit(draft["delivery_id"])

# 取消 / 重试 / 确认
client.delivery.cancel("delivery-id-xxx")
client.delivery.retry("delivery-id-xxx")
client.delivery.ack("delivery-id-xxx")

# 查询
deliveries = client.delivery.list(page=1, page_size=20)
delivery = client.delivery.get("delivery-id-xxx")
items = client.delivery.get_items("delivery-id-xxx")

# 规则列表
rules = client.delivery.list_rules()
```

### 算法运行（Algo Run）

```python
# 创建 & 启动
run = client.algo_runs.create({
    "algo_key": "some-algo",
    "asset_ids": ["asset-1", "asset-2"],
})
client.algo_runs.start(run["run_id"])

# 完成 / 取消
client.algo_runs.finish(run["run_id"])
client.algo_runs.cancel(run["run_id"])

# 查询
run = client.algo_runs.get("run-id-xxx")
runs = client.algo_runs.list(page=1, page_size=20)

# 受影响资产
assets = client.algo_runs.get_affected_assets("run-id-xxx")
```

### 客户管理

```python
client.customers.create({"customer_id": "cust-001", "name": "ACME Corp"})
client.customers.list()
client.customers.get("cust-001")
client.customers.update("cust-001", {"name": "ACME Inc."})
```

### 湖仓（Lakehouse）

```python
client.lakehouse.get_report()
client.lakehouse.get_status()
client.lakehouse.get_tables()
client.lakehouse.get_overview()
client.lakehouse.get_sync_progress()
```

### 事件

```python
# 全局事件
events = client.events.list_global(page=1, page_size=20)

# 资产事件
events = client.events.list_for_asset("asset-id-xxx")

# SSE 事件流
stream = client.events.stream_for_asset("asset-id-xxx")
print(stream)
```

### 注册中心（只读）

```python
client.registry.list_algos()
client.registry.list_tags()
client.registry.list_metrics()
client.registry.list_action_labels()
client.registry.list_lifecycle_states()
```

### 审计

```python
# 审计搜索
results = client.audit.search(
    asset_id="asset-xxx",
    event_type="delivery.commit",
    time_from="2026-01-01",
    time_to="2026-05-25",
)

# 血缘审计
results = client.audit.lineage_search(asset_id="asset-xxx")
```

### 查询（Query IR）

```python
# 验证查询
result = client.queries.validate({"query_ir": {...}})

# 运行查询
result = client.queries.run({"query_ir": {...}})

# 保存查询管理
client.queries.create_saved({"name": "my query", "query_ir": {...}})
client.queries.list_saved()
client.queries.get_saved("query-id-xxx")
client.queries.update_saved("query-id-xxx", {"name": "new name"})
client.queries.delete_saved("query-id-xxx")
```

### 搜索

```python
client.search.get_sync_status()
client.search.get_sync_progress()
```

---

## 错误处理

```python
from cyber_databrew_sdk import CyberDatabrewClient
from cyber_databrew_sdk.exceptions import (
    NotFoundError,
    AuthenticationError,
    BadRequestError,
    ServerError,
    RateLimitError,
)

client = CyberDatabrewClient(base_url="...", token="g-xxx")

try:
    asset = client.assets.get("nonexistent-id")
except NotFoundError as e:
    print(f"资产不存在: {e}")
except AuthenticationError as e:
    print(f"鉴权失败: {e}")
except BadRequestError as e:
    print(f"请求参数错误: {e}")
except RateLimitError as e:
    print(f"被限流，稍后重试: {e}")
except ServerError as e:
    print(f"服务端错误，请联系管理员: {e}")
```

---

## Context Manager（推荐用法）

```python
with CyberDatabrewClient(base_url="...", token="g-xxx") as client:
    assets = client.assets.list_all()
    for asset in assets["items"]:
        print(asset["asset_id"])
# 退出自动关闭 HTTP 连接
```

---

## 远端配置热更新

```python
# 运行时刷新配置（重新调用 /api/v1/sdk-config）
client._config.refresh(base_url="...")
```

---

## 所有 Manager

| 属性 | 类 | 用途 |
|------|-----|------|
| `client.assets` | `AssetManager` | 增删改查、标签、血缘、收藏 |
| `client.storage` | `StorageManager` | MCAP 文件列表、下载、上传确认 |
| `client.delivery` | `DeliveryManager` | 草稿、提交、取消、确认 |
| `client.algo_runs` | `AlgoRunManager` | 算法运行全生命周期 |
| `client.customers` | `CustomerManager` | 客户 CRUD |
| `client.lakehouse` | `LakehouseManager` | 湖仓报表、状态、表信息 |
| `client.events` | `EventManager` | 事件列表、SSE 流 |
| `client.registry` | `RegistryManager` | 各类注册中心（只读） |
| `client.audit` | `AuditManager` | 审计搜索 |
| `client.search` | `SearchManager` | ES 同步状态 |
| `client.queries` | `QueryManager` | Query IR 验证、执行、保存 |
| `client.actions` | `ActionManager` | 资产 action 列表、创建、更新、删除 |
| `client.eval_metrics` | `EvalMetricsManager` | 评测结果、指标列表、指标检索 |
| `client.admin_search` | `AdminSearchManager` | 搜索重建与重建任务管理 |
| `client.workflows` | `WorkflowManager` | 工作流列表、详情、日志、操作 |
| `client.pipeline_components` | `PipelineComponentManager` | Pipeline 组件列表与 CRUD |

---

## 开发

```bash
cd sdk
uv sync --dev
make validate    # ruff → mypy → 150 tests → build → smoke-install
make publish     # 发布到 Artifact Registry
```

---

## License

Internal — CyberOrigin
