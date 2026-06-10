# Dagster 接入方案

本文档描述如何将 Dagster 与 cyber-databrew 数据平台对接，
让 Dagster 作为算法调度层，通过 `grace_sdk` 调用 Backend API 管理 segment 的算法生命周期。

## 前提

- Backend 已部署，算法生命周期 API 可用（start/finish/reset/algo-events）
- `grace_sdk` 已发布，包含算法生命周期方法
- `algo_registry.yaml` 已配置好所有算法和依赖关系

## 架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│ Dagster (调度层)                                                 │
│                                                                  │
│  sensor ──→ "有 pending 的 segment 吗？"                         │
│                │                                                 │
│                ▼                                                 │
│  asset  ──→ start_algo → 跑算法 → finish_algo                   │
│                │              │           │                      │
│                ▼              ▼           ▼                      │
│           grace_sdk      Ray/本地    grace_sdk                   │
└────────────────┬──────────────────────────┬─────────────────────┘
                 │                          │
                 ▼                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Backend (状态层)                                                 │
│                                                                  │
│  POST /assets/:id/algo/:key/start   → cf_algo status=running    │
│  POST /assets/:id/algo/:key/finish  → cf_algo status=ok/failed  │
│                                     → cf_files[key]=output_uri  │
│                                     → tryUnblockDownstream()    │
│  GET  /assets?filter=cf_algo.X:status:eq:pending                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 第 1 步：给 grace_sdk 补算法生命周期方法

当前 `grace_sdk/assets.py` 只有 CRUD 方法，需要加 `start_algo`、`finish_algo`、`reset_algo`、`list_algo_events`，
以及支持 `filter` 参数的 `list` 方法。

### `sdk/src/grace_sdk/assets.py` 需要新增的方法

```python
def list(
    self,
    *,
    filter: list[str] | None = None,   # 新增：支持 filter 参数
    mcap_file_id: str | None = None,
    page: int = 1,
    page_size: int = 20,
    sort_by: str | None = None,
) -> AssetList:
    params: dict[str, object] = {"page": page, "page_size": page_size}
    if mcap_file_id:
        params["mcap_file_id"] = mcap_file_id
    if sort_by:
        params["sort_by"] = sort_by
    r = self._http.get(self._base, params={**params, "filter": filter or []})
    r.raise_for_status()
    return AssetList.model_validate(r.json())

def start_algo(
    self,
    asset_id: str,
    algo_key: str,
    *,
    method: str,
    run_id: str | None = None,
) -> dict:
    """POST /assets/:id/algo/:algo_key/start"""
    body: dict = {"method": method}
    if run_id:
        body["run_id"] = run_id
    r = self._http.post(f"{self._base}/{asset_id}/algo/{algo_key}/start", json=body)
    r.raise_for_status()
    return r.json()

def finish_algo(
    self,
    asset_id: str,
    algo_key: str,
    *,
    status: str,
    output_uri: str | None = None,
    run_id: str | None = None,
    reason: str | None = None,
    result_size_bytes: int | None = None,
    extra_fields: dict | None = None,
) -> dict:
    """POST /assets/:id/algo/:algo_key/finish"""
    body: dict = {"status": status}
    if output_uri:
        body["output_uri"] = output_uri
    if run_id:
        body["run_id"] = run_id
    if reason:
        body["reason"] = reason
    if result_size_bytes is not None:
        body["result_size_bytes"] = result_size_bytes
    if extra_fields:
        body["extra_fields"] = extra_fields
    r = self._http.post(f"{self._base}/{asset_id}/algo/{algo_key}/finish", json=body)
    r.raise_for_status()
    return r.json()

def reset_algo(self, asset_id: str, algo_key: str) -> dict:
    """POST /assets/:id/algo/:algo_key/reset"""
    r = self._http.post(f"{self._base}/{asset_id}/algo/{algo_key}/reset")
    r.raise_for_status()
    return r.json()

def list_algo_events(self, asset_id: str, *, algo_key: str | None = None) -> dict:
    """GET /assets/:id/algo-events"""
    params = {}
    if algo_key:
        params["algo_key"] = algo_key
    r = self._http.get(f"{self._base}/{asset_id}/algo-events", params=params)
    r.raise_for_status()
    return r.json()
```

### `sdk/src/grace_sdk/types/asset.py` 需要扩展的字段

```python
class Asset(BaseModel):
    asset_id: str
    mcap_file_id: str
    start_timestamp_ns: int
    end_timestamp_ns: int
    duration_sec: float
    reviewer: str
    status: AssetStatus
    owner: str
    version: int
    created_at: datetime
    updated_at: datetime
    # 新增
    algo_results: dict[str, str] = {}   # cf_algo 的内容
    tags: dict[str, str] = {}           # cf_tag 的内容
    files: dict[str, str] = {}          # cf_files 的内容
```

---

## 第 2 步：在 dagster_playground 里加 GraceClient resource

### `dagster_playground/src/resources.py` 新增

```python
import dagster as dg
from grace_sdk import GraceClient

class GraceResource(dg.ConfigurableResource):
    """数据平台 Backend 客户端，作为 Dagster resource 注入到 asset 中。"""
    base_url: str = "http://localhost:8080"
    token: str = "dev-token"

    def create_client(self) -> GraceClient:
        return GraceClient(base_url=self.base_url, token=self.token)
```

### `dagster_playground/src/definitions.py` 注册 resource

```python
from .resources import GraceResource

defs = dg.Definitions(
    assets=[...],
    jobs=[...],
    sensors=[...],
    resources={
        "io_manager": _io_manager,
        "gcs": gcs_resource,
        "ray_cluster": _ray_resource,
        "grace": GraceResource(                    # 新增
            base_url=os.getenv("GRACE_BASE_URL", "http://localhost:8080"),
            token=os.getenv("DATABREW_TOKEN", "dev-token"),
        ),
    },
)
```

---

## 第 3 步：写算法 asset（通用模板）

每个算法的 Dagster asset 都遵循同一个模板：查 pending → start → 跑算法 → finish。
只有"跑算法"这一步不同。

### `dagster_playground/src/assets/algo_template.py`

```python
"""通用算法 asset 工厂函数。

用法：
    from src.assets.algo_template import make_algo_asset

    hand_tracking_results, hand_tracking_job = make_algo_asset(
        algo_key="hand_tracking@1.2.0",
        run_fn=run_hand_tracking,       # 你的实际算法函数
        batch_size=50,
    )
"""

import os
from typing import Callable

import dagster as dg

from src.gcs_helpers import gcs_download, gcs_upload_file
from src.resources import GraceResource


def make_algo_asset(
    algo_key: str,
    run_fn: Callable,
    batch_size: int = 50,
    pool: str | None = None,
):
    """为一个算法创建 Dagster asset + job。

    Args:
        algo_key: 算法标识，如 "hand_tracking@1.2.0"
        run_fn: 实际算法函数，签名 run_fn(asset_info, work_dir) -> result_path
                asset_info 是从 Backend 拿到的完整 asset 信息（含 files）
                work_dir 是本地临时目录
                返回结果文件的本地路径
        batch_size: 每次处理多少个 pending segment
        pool: Dagster concurrency pool 名称（如 "ray_gpu"）
    """
    # 把 algo_key 转成合法的 Python 标识符作为 asset 名
    asset_name = algo_key.replace("@", "_v").replace(".", "_") + "_results"

    asset_kwargs = {}
    if pool:
        asset_kwargs["pool"] = pool

    @dg.asset(name=asset_name, **asset_kwargs)
    def _asset(context: dg.AssetExecutionContext, grace: GraceResource):
        client = grace.create_client()

        # 1. 查 pending 的 segment
        pending = client.assets.list(
            filter=[f"cf_algo.{algo_key}:status:eq:pending"],
            page_size=batch_size,
        )

        if not pending.items:
            context.log.info(f"No pending segments for {algo_key}")
            context.add_output_metadata({"segments_processed": 0})
            return {"count": 0}

        context.log.info(f"Found {len(pending.items)} pending segments for {algo_key}")

        success_count = 0
        fail_count = 0
        work_dir = f"/tmp/algo_work/{algo_key}"
        os.makedirs(work_dir, exist_ok=True)

        for seg in pending.items:
            seg_id = seg.asset_id

            # 2. 标记开始
            try:
                client.assets.start_algo(seg_id, algo_key,
                    method="dagster", run_id=context.run_id)
            except Exception as e:
                # 可能已经被别的 worker 抢走了（409），跳过
                context.log.warning(f"Skip {seg_id}: start_algo failed: {e}")
                continue

            # 3. 跑算法
            try:
                result_path = run_fn(seg, work_dir)

                # 4. 上传结果到 GCS
                algo_name = algo_key.split("@")[0]
                gcs_dest = f"gs://grace-algo/{algo_name}/{seg_id}/{os.path.basename(result_path)}"
                result_uri = gcs_upload_file(result_path, gcs_dest)
                result_size = os.path.getsize(result_path)

                # 5. 标记成功
                client.assets.finish_algo(seg_id, algo_key,
                    status="ok",
                    output_uri=result_uri,
                    run_id=context.run_id,
                    result_size_bytes=result_size)

                success_count += 1
                context.log.info(f"✅ {seg_id} {algo_key} ok → {result_uri}")

            except Exception as e:
                # 6. 标记失败
                client.assets.finish_algo(seg_id, algo_key,
                    status="failed",
                    reason=str(e)[:500],
                    run_id=context.run_id)

                fail_count += 1
                context.log.error(f"❌ {seg_id} {algo_key} failed: {e}")

        context.add_output_metadata({
            "segments_processed": success_count + fail_count,
            "success": success_count,
            "failed": fail_count,
        })
        return {"success": success_count, "failed": fail_count}

    # 创建对应的 job
    _job = dg.define_asset_job(
        name=f"{asset_name}_job",
        selection=dg.AssetSelection.assets(_asset),
    )

    return _asset, _job
```

---

## 第 4 步：用模板注册每个算法

### `dagster_playground/src/assets/tracking.py`

```python
"""Hand/Head/Body tracking 算法 assets。"""

from src.assets.algo_template import make_algo_asset
from src.gcs_helpers import gcs_download


def run_hand_tracking(seg, work_dir):
    """实际的 hand_tracking 算法。"""
    # 从 seg.files 拿到原始 MCAP 的 GCS URI
    raw_mcap_uri = seg.files.get("raw_mcap", "")
    local_mcap = gcs_download(raw_mcap_uri, work_dir)

    # 跑你的算法（替换成实际代码）
    import your_hand_tracking_lib
    result_path = your_hand_tracking_lib.process(str(local_mcap), work_dir)
    return result_path


def run_head_tracking(seg, work_dir):
    raw_mcap_uri = seg.files.get("raw_mcap", "")
    local_mcap = gcs_download(raw_mcap_uri, work_dir)
    import your_head_tracking_lib
    return your_head_tracking_lib.process(str(local_mcap), work_dir)


def run_body_tracking(seg, work_dir):
    raw_mcap_uri = seg.files.get("raw_mcap", "")
    local_mcap = gcs_download(raw_mcap_uri, work_dir)
    import your_body_tracking_lib
    return your_body_tracking_lib.process(str(local_mcap), work_dir)


# 注册 assets
hand_tracking_results, hand_tracking_job = make_algo_asset(
    algo_key="hand_tracking@1.2.0",
    run_fn=run_hand_tracking,
    pool="ray_gpu",
)

head_tracking_results, head_tracking_job = make_algo_asset(
    algo_key="head_tracking@1.0.0",
    run_fn=run_head_tracking,
    pool="ray_gpu",
)

body_tracking_results, body_tracking_job = make_algo_asset(
    algo_key="body_tracking@1.0.0",
    run_fn=run_body_tracking,
    pool="ray_gpu",
)
```

### `dagster_playground/src/assets/action_annotation.py`

```python
"""Action annotation — 依赖 hand + head + body 的产物。"""

from src.assets.algo_template import make_algo_asset
from src.gcs_helpers import gcs_download


def run_action_annotation(seg, work_dir):
    """读取三个 tracking 的产物，跑 action annotation。"""
    # 从 cf_files 拿到上游产物的 URI
    ht_uri = seg.files.get("hand_tracking@1.2.0", "")
    head_uri = seg.files.get("head_tracking@1.0.0", "")
    body_uri = seg.files.get("body_tracking@1.0.0", "")

    # 下载上游产物
    ht_local = gcs_download(ht_uri, work_dir)
    head_local = gcs_download(head_uri, work_dir)
    body_local = gcs_download(body_uri, work_dir)

    # 跑算法
    import your_action_annotation_lib
    return your_action_annotation_lib.process(
        str(ht_local), str(head_local), str(body_local), work_dir
    )


action_annotation_results, action_annotation_job = make_algo_asset(
    algo_key="action_annotation@1.0.0",
    run_fn=run_action_annotation,
)
```

注意：action_annotation 的 sensor 查到 pending 时，`seg.files` 里已经有了三个 tracking 的 URI，
因为 Backend 的 `tryUnblockDownstream` 只有在三个都 ok 后才会把 action_annotation 从 blocked 变成 pending。

---

## 第 5 步：写通用 sensor

一个 sensor 覆盖所有算法，不需要每个算法一个 sensor：

### `dagster_playground/src/sensors/pending_algo.py`

```python
"""通用算法 sensor：轮询 Backend 查 pending 的 segment，触发对应的算法 job。"""

import dagster as dg
from src.resources import GraceResource

# 算法 → job 名称的映射
ALGO_JOB_MAP = {
    "hand_tracking@1.2.0": "hand_tracking_v1_2_0_results_job",
    "head_tracking@1.0.0": "head_tracking_v1_0_0_results_job",
    "body_tracking@1.0.0": "body_tracking_v1_0_0_results_job",
    "deface@2.0.0": "deface_v2_0_0_results_job",
    "action_annotation@1.0.0": "action_annotation_v1_0_0_results_job",
    "env_analysis@1.0.0": "env_analysis_v1_0_0_results_job",
}


@dg.sensor(minimum_interval_seconds=30)
def pending_algo_sensor(context: dg.SensorEvaluationContext):
    """每 30 秒检查所有算法是否有 pending 的 segment。"""
    from grace_sdk import GraceClient
    import os

    client = GraceClient(
        base_url=os.getenv("GRACE_BASE_URL", "http://localhost:8080"),
        token=os.getenv("DATABREW_TOKEN", "dev-token"),
    )

    for algo_key, job_name in ALGO_JOB_MAP.items():
        try:
            pending = client.assets.list(
                filter=[f"cf_algo.{algo_key}:status:eq:pending"],
                page_size=1,  # 只查有没有，不需要拿全部
            )
            if pending.total > 0:
                # 有 pending 的 segment，触发对应的 job
                yield dg.RunRequest(
                    run_key=f"{algo_key}:{context.cursor or 'init'}",
                    job_name=job_name,
                )
                context.log.info(
                    f"{algo_key}: {pending.total} pending segment(s), triggered {job_name}"
                )
        except Exception as e:
            context.log.warning(f"Failed to check {algo_key}: {e}")

    client.close()
```

---

## 第 6 步：注册到 definitions.py

```python
# dagster_playground/src/definitions.py

from .assets.tracking import (
    hand_tracking_results, hand_tracking_job,
    head_tracking_results, head_tracking_job,
    body_tracking_results, body_tracking_job,
)
from .assets.action_annotation import action_annotation_results, action_annotation_job
from .assets.deface import deface_results, deface_job
from .assets.env_analysis import env_analysis_results, env_analysis_job
from .sensors.pending_algo import pending_algo_sensor
from .resources import GraceResource

defs = dg.Definitions(
    assets=[
        # 已有的
        hello_tekton,
        sam2_segmentation,
        transcode_video,
        # 新增的
        hand_tracking_results,
        head_tracking_results,
        body_tracking_results,
        deface_results,
        action_annotation_results,
        env_analysis_results,
    ],
    jobs=[
        # 已有的
        hello_tekton_job,
        sam2_segmentation_job,
        transcode_video_job,
        # 新增的
        hand_tracking_job,
        head_tracking_job,
        body_tracking_job,
        deface_job,
        action_annotation_job,
        env_analysis_job,
    ],
    sensors=[
        new_video_sensor,
        pending_algo_sensor,    # 新增
    ],
    resources={
        "io_manager": _io_manager,
        "gcs": gcs_resource,
        "ray_cluster": _ray_resource,
        "grace": GraceResource(
            base_url=os.getenv("GRACE_BASE_URL", "http://localhost:8080"),
            token=os.getenv("DATABREW_TOKEN", "dev-token"),
        ),
    },
)
```

---

## 第 7 步：本地跑起来

```bash
# 终端 1：启动你的 Backend
cd cyber-databrew/backend
STORAGE_BACKEND=postgres go run cmd/server/main.go

# 终端 2：启动 Dagster
cd tekton-playground/dagster_playground
export DAGSTER_HOME=/tmp/dagster_demo
export DAGSTER_ENV=local
export GRACE_BASE_URL=http://localhost:8080
export DATABREW_TOKEN=dev-token
mkdir -p "$DAGSTER_HOME"
uv run dagster dev -m src.definitions --port 3333
```

打开 http://localhost:3333，你会看到：
- 6 个新的算法 asset（hand_tracking_results, head_tracking_results, ...）
- 1 个 pending_algo_sensor
- 6 个对应的 job

---

## 数据流总结

```
1. MCAP 上传到 GCS
2. （手动或 Pub/Sub sensor）调 Backend API 创建 mcap_files + assets
3. Backend 初始化 cf_algo：hand=pending, head=pending, ..., action=blocked
4. pending_algo_sensor 发现 hand_tracking 有 pending
5. 触发 hand_tracking_job
6. hand_tracking asset 执行：
   a. list(filter=pending) → 拿到 seg-001, seg-002, ...
   b. start_algo(seg-001, hand_tracking) → Backend: status=running
   c. 下载 raw_mcap，跑算法，上传结果
   d. finish_algo(seg-001, hand_tracking, ok, uri) → Backend:
      - cf_algo status=ok
      - cf_files["hand_tracking@1.2.0"] = uri
      - tryUnblockDownstream: 检查 action_annotation 的依赖
7. 当 hand+head+body 都 ok → action_annotation 从 blocked 变 pending
8. 下一轮 sensor 发现 action_annotation 有 pending
9. 触发 action_annotation_job
10. action_annotation asset 执行：
    a. list(filter=pending) → 拿到 seg-001
    b. seg-001.files 里已有三个 tracking 的 URI
    c. 下载三个产物，跑算法，上传结果
    d. finish_algo → 完成
```

---

## 加新算法的流程

1. `algo_registry.yaml` 加算法定义（Backend 侧）
2. 写一个 `run_xxx(seg, work_dir)` 函数（算法代码）
3. 调 `make_algo_asset(algo_key, run_fn)` 生成 asset + job
4. 在 `definitions.py` 注册
5. 在 `pending_algo.py` 的 `ALGO_JOB_MAP` 加一行
6. 重启 Backend + Dagster
