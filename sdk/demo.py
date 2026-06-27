#!/usr/bin/env python3
"""DataBrew SDK Demo — 快速了解所有接口和用法。

运行方式：:

    # 先确保已安装 SDK
    pip install cyber-databrew-sdk

    # Pod 中直接运行（SA 自动认证）
    python3 demo.py

    # 本地开发（用 gcloud 临时 token）
    GCS_TOKEN=$(gcloud auth print-access-token) python3 demo.py
"""

from __future__ import annotations

import os
import subprocess
import time
from pathlib import Path

from cyber_databrew_sdk import CyberDatabrewClient

# ── 配置 ──────────────────────────────────────────────────────────
BASE_URL = os.environ.get("DATABREW_URL", "http://databrew-backend:8080")
TOKEN = os.environ.get("DATABREW_TOKEN", "dev-token")
DEMO_BUCKET = "gs://cyber-databrew-dev/tmp"


def demo(sdk: CyberDatabrewClient) -> None:
    print("=" * 60)
    print("DataBrew SDK Demo")
    print("=" * 60)
    print()

    # ── 1. 基本文件操作 ────────────────────────────────────────────
    print("─── 1. 基本文件操作 ───")

    # stat
    uri = f"{DEMO_BUCKET}/cyber_databrew_dev_001_init.sql"
    info = sdk.stat(uri)
    print(f"  stat:      size={info.size}, type={info.type}")

    # exists
    print(f"  exists:    {sdk.exists(uri)}")
    print(f"  !exists:   {sdk.exists('gs://cyber-databrew-dev/tmp/nope')}")

    # listdir
    entries = sdk.listdir(f"{DEMO_BUCKET}/")
    print(f"  listdir:   {len(entries)} entries")

    # ── 2. 读写 ────────────────────────────────────────────────────
    print()
    print("─── 2. 读写 ───")

    # write
    sdk.write(f"{DEMO_BUCKET}/hello.txt", b"Hello DataBrew!")
    print("  write:     OK")

    # read
    data = sdk.read(f"{DEMO_BUCKET}/hello.txt")
    print(f"  read:      {data}")

    # ── 3. 流式读取 (seek/tell) ────────────────────────────────────
    print()
    print("─── 3. 流式读取 (seek/tell) ───")

    with sdk.open(f"{DEMO_BUCKET}/cyber_databrew_dev_001_init.sql", "rb") as f:
        print(f"  read(10):  {f.read(10)}")
        print(f"  tell:      {f.tell()}")
        f.seek(0)
        print(f"  seek(0):   tell={f.tell()}")
        f.seek(5000)
        print(f"  seek(5000): read(10)={f.read(10)}")

    # ── 4. 大文件下载 (直接到文件，不进内存) ─────────────────────────
    print()
    print("─── 4. 大文件下载 (直接到文件) ───")

    local_path = Path("/tmp/sdk_demo_downloaded.txt")
    t0 = time.time()
    sdk.download(f"{DEMO_BUCKET}/cyber_databrew_dev_001_init.sql", local_path)
    elapsed = time.time() - t0
    print(f"  download:  {local_path.stat().st_size} bytes in {elapsed:.2f}s")
    local_path.unlink(missing_ok=True)

    # ── 5. 大文件上传 (从文件直传，transfer_manager 加速) ──────────
    print()
    print("─── 5. 大文件上传 (从文件直传) ───")

    local_file = Path("/tmp/sdk_demo_upload.txt")
    local_file.write_text("upload test data")
    t0 = time.time()
    sdk.upload(local_file, f"{DEMO_BUCKET}/uploaded.txt")
    elapsed = time.time() - t0
    print(f"  upload:    {local_file.stat().st_size} bytes in {elapsed:.3f}s")
    print(f"  verified:  {sdk.read(f'{DEMO_BUCKET}/uploaded.txt')}")
    local_file.unlink(missing_ok=True)

    # ── 6. 复制 & 删除 ────────────────────────────────────────────
    print()
    print("─── 6. 复制 & 删除 ───")

    sdk.copy(f"{DEMO_BUCKET}/hello.txt", f"{DEMO_BUCKET}/hello_copy.txt")
    print("  copy:      OK")
    print(f"  copy exists: {sdk.exists(f'{DEMO_BUCKET}/hello_copy.txt')}")

    sdk.delete(f"{DEMO_BUCKET}/hello.txt")
    sdk.delete(f"{DEMO_BUCKET}/hello_copy.txt")
    sdk.delete(f"{DEMO_BUCKET}/uploaded.txt")
    print("  delete:    OK")

    # ── 7. asset:// URI 解析 ──────────────────────────────────────
    print()
    print("─── 7. asset:// URI (需后端 GraceResolver) ───")
    try:
        data = sdk.read("asset://grace:019ed9c5-f88a-78fc-878e-a2a2420b1f26/raw")
        print(f"  asset://grace: OK, {len(data)} bytes")
    except Exception as e:
        print(f"  asset://grace: SKIP (backend not available: {e})")

    # ── 8. 本地文件 ───────────────────────────────────────────────
    print()
    print("─── 8. 本地文件 ───")

    hello = Path("/tmp/sdk_demo_local.txt")
    hello.write_text("local test")
    print(f"  read:      {sdk.read(str(hello))}")
    print(f"  exists:    {sdk.exists(str(hello))}")
    hello.unlink()
    print(f"  deleted:   {not hello.exists()}")

    # ── 总结 ──────────────────────────────────────────────────────
    print()
    print("=" * 60)
    print("所有接口验证通过！")
    print()
    print("接口汇总:")
    print("  sdk.open(uri, 'rb')    流式读 (seek/tell)")
    print("  sdk.read(uri)         全量读 → bytes")
    print("  sdk.write(uri, data)  全量写")
    print("  sdk.download(uri, path)  大文件下载到本地")
    print("  sdk.upload(path, uri)    本地文件上传到云端")
    print("  sdk.stat(uri)         文件信息")
    print("  sdk.listdir(uri)      列目录")
    print("  sdk.copy(src, dst)    复制")
    print("  sdk.delete(uri)       删除")
    print("  sdk.exists(uri)       是否存在")
    print()
    print("支持的 URI 前缀: gs://  s3://  file://  asset://  本地路径")
    print("=" * 60)


def main() -> None:
    sdk = CyberDatabrewClient()
    # 自动从环境变量读 EMAIL / TOKEN / BASE_URL
    # 有 email 无 token 时自动调用 email_login

    # 本地开发: 从 gcloud 获取临时 GCS token
    gcs_token = os.environ.get("GCS_TOKEN")
    if gcs_token:
        sdk.set_gcs_token(gcs_token)
    else:
        try:
            token = subprocess.check_output(
                ["gcloud", "auth", "print-access-token"]
            ).decode().strip()
            sdk.set_gcs_token(token)
        except Exception:
            pass  # 在 GKE Pod 中不需要 token

    demo(sdk)


if __name__ == "__main__":
    main()
