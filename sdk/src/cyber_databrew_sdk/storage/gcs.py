"""Google Cloud Storage backend — gcsfs + transfer_manager.

In GKE Pods the service account is picked up automatically via ADC.
Local development requires ``set_gcs_token()`` or ``GOOGLE_APPLICATION_CREDENTIALS``.
"""

from __future__ import annotations

import logging
import os
import tempfile
from pathlib import Path
from typing import IO, Any

import gcsfs

from cyber_databrew_sdk.storage.backend import Backend, FileInfo

_logger = logging.getLogger(__name__)

# Files above this threshold use transfer_manager for parallel throughput.
_TRANSFER_THRESHOLD = 100 * 1024 * 1024  # 100 MiB


# ---------------------------------------------------------------------------
# Transfer-manager helpers (lazy import)
# ---------------------------------------------------------------------------

def _gcs_download(bucket: str, obj: str, local_path: str) -> None:
    """Download via ``blob.download_to_filename`` (single-stream ~200 MB/s)."""
    from google.cloud import storage

    client = storage.Client()
    blob = client.bucket(bucket).blob(obj)
    blob.download_to_filename(local_path)


def _gcs_upload(local_path: str, bucket: str, obj: str) -> None:
    """Upload via ``transfer_manager`` (thread pool, ~246 MB/s)."""
    from google.cloud import storage
    from google.cloud.storage import transfer_manager

    client = storage.Client()
    blob = client.bucket(bucket).blob(obj)
    transfer_manager.upload_chunks_concurrently(
        local_path, blob,
        worker_type="thread", max_workers=8,
    )


# ---------------------------------------------------------------------------
# GCS backend
# ---------------------------------------------------------------------------

class GCSBackend(Backend):
    """GCS backend — POSIX-like interface on top of gcsfs."""

    def __init__(self, token: str | None = None) -> None:
        kwargs = {}
        if token:
            kwargs["token"] = token
        self._fs = gcsfs.GCSFileSystem(**kwargs)

    # ── helpers ──────────────────────────────────────────────────────

    @staticmethod
    def _parse(path: str) -> tuple[str, str]:
        """Split ``bucket/object`` → ``(bucket, object)``."""
        bucket, _, obj = path.partition("/")
        return bucket, obj

    def _size(self, path: str) -> int:
        try:
            return self._fs.info(path).get("size", 0)
        except Exception:
            return 0

    # ── public ───────────────────────────────────────────────────────

    def open(self, path: str, mode: str = "rb") -> IO[Any]:
        return self._fs.open(path, mode)

    def read(self, path: str) -> bytes:
        size = self._size(path)
        if size > _TRANSFER_THRESHOLD:
            bucket, obj = self._parse(path)
            with tempfile.NamedTemporaryFile(delete=False) as tmp:
                tmp_name = tmp.name
            try:
                _gcs_download(bucket, obj, tmp_name)
                with open(tmp_name, "rb") as f:
                    return f.read()
            finally:
                os.unlink(tmp_name)
        return self._fs.open(path, "rb").read()

    def write(self, path: str, data: bytes) -> int:
        if len(data) > _TRANSFER_THRESHOLD:
            bucket, obj = self._parse(path)
            with tempfile.NamedTemporaryFile(delete=False) as tmp:
                tmp.write(data)
                tmp_name = tmp.name
            try:
                _gcs_upload(tmp_name, bucket, obj)
            finally:
                os.unlink(tmp_name)
        else:
            with self._fs.open(path, "wb") as f:
                f.write(data)
        return len(data)

    def stat(self, path: str) -> FileInfo:
        try:
            info = self._fs.info(path)
        except Exception as exc:
            raise FileNotFoundError(path) from exc
        if isinstance(info, dict):
            return FileInfo(
                name=path,
                size=info.get("size", 0),
                mtime=info.get("mtime", None) or info.get("updated", None),
                type="dir" if info.get("type") in ("directory", "dir") else "file",
            )
        return FileInfo(name=path, size=0, mtime=None, type="file")

    def listdir(self, path: str) -> list[FileInfo]:
        try:
            entries = self._fs.ls(path.rstrip("/") + "/")
        except Exception as exc:
            raise FileNotFoundError(path) from exc
        results = []
        for e in entries:
            if isinstance(e, dict):
                results.append(FileInfo(
                    name=e.get("name", ""),
                    size=e.get("size", 0),
                    mtime=e.get("mtime", None) or e.get("updated", None),
                    type="dir" if e.get("type") in ("directory", "dir") else "file",
                ))
            else:
                results.append(FileInfo(name=str(e), size=0, mtime=None, type="file"))
        return results

    def copy(self, src: str, dst: str) -> None:
        self._fs.cp(src, dst)

    def delete(self, path: str) -> None:
        self._fs.rm(path)

    def exists(self, path: str) -> bool:
        try:
            self.stat(path)
            return True
        except FileNotFoundError:
            return False

    def download(self, path: str, local_path: str | Path) -> Path:
        """Download directly to file — no memory buffering."""
        local_path = Path(local_path)
        local_path.parent.mkdir(parents=True, exist_ok=True)
        bucket, obj = self._parse(path)
        _gcs_download(bucket, obj, str(local_path))
        return local_path

    def upload(self, local_path: str | Path, path: str) -> None:
        """Upload from local file — uses transfer_manager for large files."""
        local_path = Path(local_path)
        bucket, obj = self._parse(path)
        size = local_path.stat().st_size
        if size > _TRANSFER_THRESHOLD:
            _gcs_upload(str(local_path), bucket, obj)
        else:
            with self._fs.open(path, "wb") as f:
                f.write(local_path.read_bytes())
