"""Backend-proxy mode — all GCS ops go through DataBrew backend sign-url.

Use this when the user does NOT have direct GCP credentials.  The DataBrew
backend signs GCS URLs on the caller's behalf.
"""

from __future__ import annotations

import io
import logging
from pathlib import Path
from typing import IO, Any

import httpx

from cyber_databrew_sdk._base_manager import BaseManager
from cyber_databrew_sdk.storage.backend import Backend, FileInfo

_logger = logging.getLogger(__name__)


class ProxyBackend(Backend):
    """Backend-proxy — delegates all GCS operations to DataBrew backend APIs.

    The backend must expose ``POST /api/v1/storage/sign-url``.
    """

    def __init__(self, requestor: BaseManager) -> None:
        self._requestor = requestor

    # ── helpers ──────────────────────────────────────────────────────

    @staticmethod
    def _parse(path: str) -> tuple[str, str]:
        bucket, _, obj = path.partition("/")
        return bucket, obj

    def _sign_url(self, bucket: str, obj: str, method: str = "GET") -> str:
        """Get a signed URL from the backend."""
        result = self._requestor._request("POST", "storage_sign_url", json_body={
            "bucket": bucket,
            "object": obj,
            "method": method,
            "ttl": 3600,
        })
        return result["url"]

    # ── public ───────────────────────────────────────────────────────

    def open(self, path: str, mode: str = "rb") -> IO[Any]:
        if "r" in mode:
            bucket, obj = self._parse(path)
            url = self._sign_url(bucket, obj, "GET")
            resp = httpx.get(url, follow_redirects=True)
            resp.raise_for_status()
            return io.BytesIO(resp.content)
        raise NotImplementedError("proxy backend does not support write-mode open")

    def read(self, path: str) -> bytes:
        bucket, obj = self._parse(path)
        url = self._sign_url(bucket, obj, "GET")
        resp = httpx.get(url, follow_redirects=True)
        resp.raise_for_status()
        return resp.content

    def write(self, path: str, data: bytes) -> int:
        bucket, obj = self._parse(path)
        url = self._sign_url(bucket, obj, "PUT")
        resp = httpx.put(url, content=data, follow_redirects=True)
        resp.raise_for_status()
        return len(data)

    def stat(self, path: str) -> FileInfo:
        # Minimal stat via head request on the signed URL
        try:
            bucket, obj = self._parse(path)
            url = self._sign_url(bucket, obj, "GET")
            resp = httpx.head(url, follow_redirects=True)
            resp.raise_for_status()
            size = int(resp.headers.get("Content-Length", 0))
            return FileInfo(name=path, size=size, mtime=None, type="file")
        except Exception as exc:
            raise FileNotFoundError(path) from exc

    def listdir(self, path: str) -> list[FileInfo]:
        raise NotImplementedError("proxy backend does not support listdir")

    def copy(self, src: str, dst: str) -> None:
        data = self.read(src)
        self.write(dst, data)

    def delete(self, path: str) -> None:
        raise NotImplementedError("proxy backend does not support delete")

    def exists(self, path: str) -> bool:
        try:
            self.stat(path)
            return True
        except FileNotFoundError:
            return False

    def download(self, path: str, local_path: str | Path) -> Path:
        local_path = Path(local_path)
        local_path.parent.mkdir(parents=True, exist_ok=True)
        data = self.read(path)
        local_path.write_bytes(data)
        return local_path

    def upload(self, local_path: str | Path, path: str) -> None:
        self.write(path, Path(local_path).read_bytes())
