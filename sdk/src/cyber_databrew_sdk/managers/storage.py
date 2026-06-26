"""DataBrewFS — POSIX-like file interface backed by gcsfs + transfer_manager.

Pods in GKE have a service account with GCS access. ``gs://`` paths use
``gcsfs.GCSFileSystem`` (POSIX semantics, including seek/tell). Bulk
reads/writes (``read()``, ``write()``) use ``transfer_manager`` for
maximum throughput.  ``asset://`` URIs are resolved through the DataBrew
backend.

Usage::

    from cyber_databrew_sdk import CyberDatabrew

    sdk = CyberDatabrew(token="...")

    # Full file download (fast path)
    data = sdk.read("gs://bucket/file.mcap")

    # Stream with seek (POSIX)
    with sdk.open("gs://bucket/file.mcap", "rb") as f:
        f.seek(1024)
        chunk = f.read(8192)

    sdk.write("gs://bucket/out.bin", data)
    info = sdk.stat("gs://bucket/file.bin")
"""

from __future__ import annotations

import io
import logging
import os
from dataclasses import dataclass
from pathlib import Path
from typing import IO, Any

import gcsfs

from cyber_databrew_sdk._base_manager import BaseManager

_logger = logging.getLogger(__name__)

# Files below this threshold use simple upload/download;
# above it, use transfer_manager for parallel throughput.
_TRANSFER_THRESHOLD = 100 * 1024 * 1024  # 100 MiB


@dataclass
class FileInfo:
    """File metadata (similar to ``os.stat_result``)."""
    name: str
    size: int
    mtime: float | None
    type: str  # "file" | "dir"


# ---------------------------------------------------------------------------
# Transfer-manager helpers (lazy import — heavy deps)
# ---------------------------------------------------------------------------

def _transfer_download(bucket_name: str, object_path: str, local_path: str) -> None:
    """Parallel chunked download via ``transfer_manager``."""
    from google.cloud import storage
    from google.cloud.storage import transfer_manager

    client = storage.Client()
    blob = client.bucket(bucket_name).blob(object_path)
    transfer_manager.download_chunks_concurrently(blob, local_path)


def _transfer_upload(local_path: str, bucket_name: str, object_path: str) -> None:
    """Parallel chunked upload via ``transfer_manager``."""
    from google.cloud import storage
    from google.cloud.storage import transfer_manager

    client = storage.Client()
    blob = client.bucket(bucket_name).blob(object_path)
    transfer_manager.upload_chunks_concurrently(local_path, blob)


def _transfer_read(bucket_name: str, object_path: str) -> bytes:
    """Download via transfer_manager and return in-memory bytes."""
    import tempfile
    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp_name = tmp.name
    try:
        _transfer_download(bucket_name, object_path, tmp_name)
        with open(tmp_name, "rb") as f:
            return f.read()
    finally:
        os.unlink(tmp_name)


def _transfer_write(bucket_name: str, object_path: str, data: bytes) -> None:
    """Upload bytes via transfer_manager using a temp file."""
    import tempfile
    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp.write(data)
        tmp_name = tmp.name
    try:
        _transfer_upload(tmp_name, bucket_name, object_path)
    finally:
        os.unlink(tmp_name)


# ---------------------------------------------------------------------------
# StorageManager
# ---------------------------------------------------------------------------

class StorageManager(BaseManager):
    """File operations backed by gcsfs + transfer_manager."""

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._gcsfs: gcsfs.GCSFileSystem | None = None

    # ------------------------------------------------------------------
    # gcsfs — POSIX-style filesystem with seek support
    # ------------------------------------------------------------------

    @property
    def fs(self) -> gcsfs.GCSFileSystem:
        """Lazily-initialized gcsfs filesystem.

        In GKE, uses the Pod's service account (ADC) automatically.
        For local dev, set ``GOOGLE_APPLICATION_CREDENTIALS``.
        """
        if self._gcsfs is None:
            self._gcsfs = gcsfs.GCSFileSystem()
        return self._gcsfs

    def set_gcs_token(self, token: str) -> None:
        """Set an explicit GCS access token (local development)."""
        self._gcsfs = gcsfs.GCSFileSystem(token=token)

    # ------------------------------------------------------------------
    # URI helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _parse_gcs_uri(uri: str) -> tuple[str, str]:
        """Parse ``gs://bucket/path/to/obj`` → ``(bucket, object)``."""
        path = uri[5:]
        bucket, _, obj = path.partition("/")
        return bucket, obj

    @staticmethod
    def _gcs_path(uri: str) -> str:
        """Strip ``gs://`` prefix."""
        return uri[5:]

    @staticmethod
    def _local_path(uri: str) -> str:
        """Strip ``file://`` prefix."""
        return uri[7:] if uri.startswith("file://") else uri

    # ------------------------------------------------------------------
    # Asset URI resolution
    # ------------------------------------------------------------------

    def _resolve_asset(self, uri: str) -> str | None:
        """Resolve ``asset://source:id/subpath`` → ``bucket/object``."""
        rest = uri[len("asset://"):]
        if ":" in rest:
            source, _, asset_id = rest.partition(":")
            source = source.strip()
            asset_id = asset_id.strip("/")
            try:
                result = self._request("POST", "storage_resolve", json_body={
                    "source": source,
                    "id": asset_id,
                    "env": "dev",
                })
                gcs_path = result.get("gcs_path", "")
                if gcs_path:
                    return gcs_path[5:] if gcs_path.startswith("gs://") else gcs_path
            except Exception as exc:
                _logger.warning("resolve failed for %s: %s", uri, exc)
            return None

        # Legacy: asset://<plain_id>
        asset_id = rest.strip("/")
        try:
            locator = self._request("GET", self._cfg.resolve("asset_mcap_locator", asset_id=asset_id))
            mcap_id = locator.get("mcap_file_id")
            if mcap_id:
                info = self._request("GET", self._cfg.resolve("storage_file_info", mcap_id=mcap_id))
                gcs_path = info.get("gcs_path") or info.get("storage_path", "")
                if gcs_path:
                    return gcs_path[5:] if gcs_path.startswith("gs://") else gcs_path
        except Exception as exc:
            _logger.warning("legacy asset resolve failed for %s: %s", uri, exc)
        return None

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def open(self, uri: str, mode: str = "rb") -> IO[Any]:
        """Open a file for reading/writing (supports seek/tell with gcsfs)."""
        if uri.startswith("gs://"):
            return self.fs.open(self._gcs_path(uri), mode)
        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve asset URI: {uri}")
            return self.fs.open(path, mode)
        return open(self._local_path(uri), mode)

    def read(self, uri: str) -> bytes:
        """Read entire file — uses transfer_manager for files > 100 MiB."""
        if uri.startswith("gs://"):
            bucket, obj = self._parse_gcs_uri(uri)
            # Fast check via gcsfs
            try:
                info = self.fs.info(self._gcs_path(uri))
                size = info.get("size", 0)
            except Exception:
                size = 0
            if size > _TRANSFER_THRESHOLD:
                return _transfer_read(bucket, obj)
            return self.fs.open(self._gcs_path(uri), "rb").read()

        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            return self.fs.open(path, "rb").read()

        return Path(self._local_path(uri)).read_bytes()

    def write(self, uri: str, data: bytes | str) -> int:
        """Write entire file — uses transfer_manager for data > 100 MiB."""
        if isinstance(data, str):
            data = data.encode("utf-8")

        if uri.startswith("gs://"):
            if len(data) > _TRANSFER_THRESHOLD:
                bucket, obj = self._parse_gcs_uri(uri)
                _transfer_write(bucket, obj, data)
            else:
                with self.fs.open(self._gcs_path(uri), "wb") as f:
                    f.write(data)
            return len(data)

        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            with self.fs.open(path, "wb") as f:
                f.write(data)
            return len(data)

        Path(self._local_path(uri)).write_bytes(data)
        return len(data)

    def stat(self, uri: str) -> FileInfo:
        """Get file metadata."""
        if uri.startswith("gs://"):
            try:
                info = self.fs.info(self._gcs_path(uri))
            except FileNotFoundError:
                raise
            except Exception as exc:
                raise FileNotFoundError(str(uri)) from exc
            if info.get("type") == "directory":
                return FileInfo(name=uri, size=0, mtime=None, type="dir")
            return FileInfo(
                name=uri,
                size=info.get("size", 0),
                mtime=info.get("mtime", None),
                type="file",
            )

        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            return self._gcsfs_info(uri, path)

        p = Path(self._local_path(uri))
        if not p.exists():
            raise FileNotFoundError(str(uri))
        st = p.stat()
        return FileInfo(
            name=uri,
            size=st.st_size,
            mtime=st.st_mtime,
            type="dir" if p.is_dir() else "file",
        )

    def listdir(self, uri: str) -> list[FileInfo]:
        """List directory entries."""
        if uri.startswith("gs://"):
            path = self._gcs_path(uri).rstrip("/") + "/"
            try:
                entries = self.fs.ls(path)
            except Exception as exc:
                raise FileNotFoundError(str(uri)) from exc
            results = []
            for e in entries:
                if isinstance(e, dict):
                    name = e.get("name", "")
                    results.append(FileInfo(
                        name=name,
                        size=e.get("size", 0),
                        mtime=e.get("mtime", None),
                        type="dir" if e.get("type") == "directory" else "file",
                    ))
                else:
                    results.append(FileInfo(name=str(e), size=0, mtime=None, type="file"))
            return results

        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            return self._gcsfs_listdir(uri, path)

        p = Path(self._local_path(uri))
        if not p.is_dir():
            raise NotADirectoryError(str(uri))
        results = []
        for entry in p.iterdir():
            st = entry.stat()
            results.append(FileInfo(
                name=entry.name,
                size=st.st_size,
                mtime=st.st_mtime,
                type="dir" if entry.is_dir() else "file",
            ))
        return results

    def copy(self, src: str, dst: str) -> None:
        """Copy file — uses gcsfs cp for GCS paths, fallback read+write."""
        if src.startswith("gs://") or dst.startswith("gs://"):
            src_path = self._gcs_path(src) if src.startswith("gs://") else self._resolve_asset(src)
            dst_path = self._gcs_path(dst) if dst.startswith("gs://") else self._resolve_asset(dst)
            if not src_path or not dst_path:
                raise FileNotFoundError(f"cannot resolve: {src} or {dst}")
            self.fs.cp(src_path, dst_path)
            return
        data = self.read(src)
        self.write(dst, data)

    def delete(self, uri: str) -> None:
        """Delete file."""
        if uri.startswith("gs://"):
            self.fs.rm(self._gcs_path(uri))
        elif uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            self.fs.rm(path)
        else:
            Path(self._local_path(uri)).unlink()

    def download(self, uri: str, output_path: str | Path) -> Path:
        """Download a file to local disk via transfer_manager (no memory buffering).

        Uses parallel chunked download for files > 100 MiB.
        Returns the output path.
        """
        output_path = Path(output_path)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        if uri.startswith("gs://"):
            bucket, obj = self._parse_gcs_uri(uri)
            _transfer_download(bucket, obj, str(output_path))
            return output_path

        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            # path is "bucket/object"
            bucket, obj = path.split("/", 1)
            _transfer_download(bucket, obj, str(output_path))
            return output_path

        # Local file: copy
        import shutil
        shutil.copy2(self._local_path(uri), output_path)
        return output_path

    def exists(self, uri: str) -> bool:
        """Check file existence."""
        try:
            self.stat(uri)
            return True
        except (FileNotFoundError, NotADirectoryError):
            return False

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _gcsfs_info(self, uri: str, path: str) -> FileInfo:
        try:
            info = self.fs.info(path)
        except Exception as exc:
            raise FileNotFoundError(str(uri)) from exc
        return FileInfo(
            name=uri,
            size=info.get("size", 0),
            mtime=info.get("mtime", None),
            type="dir" if info.get("type") == "directory" else "file",
        )

    def _gcsfs_listdir(self, uri: str, path: str) -> list[FileInfo]:
        try:
            entries = self.fs.ls(path.rstrip("/") + "/")
        except Exception as exc:
            raise FileNotFoundError(str(uri)) from exc
        results = []
        for e in entries:
            if isinstance(e, dict):
                results.append(FileInfo(
                    name=e.get("name", ""),
                    size=e.get("size", 0),
                    mtime=e.get("mtime", None),
                    type="dir" if e.get("type") == "directory" else "file",
                ))
            else:
                results.append(FileInfo(name=str(e), size=0, mtime=None, type="file"))
        return results

    # --- legacy MCAP methods ---

    def list_files(self, **kwargs: Any) -> dict[str, Any]:
        return self._request("GET", "storage_files_list", params=kwargs)

    def get_file_info(self, mcap_id: str) -> dict[str, Any]:
        return self._request("GET", self._cfg.resolve("storage_file_info", mcap_id=mcap_id))

    def download_mcap(self, mcap_file_id: str, output_path: str | Path) -> Path:
        import httpx
        output_path = Path(output_path)
        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {"Content-Type": "application/json", **self._requestor._auth_headers}
        with httpx.Client().stream("GET", url, headers=headers, follow_redirects=True) as resp:
            resp.raise_for_status()
            output_path.parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, "wb") as f:
                for chunk in resp.iter_bytes(chunk_size=8192):
                    f.write(chunk)
        return output_path

    def download_asset_mcap(self, asset_id: str, output_path: str | Path) -> Path:
        locator = self._request("GET", self._cfg.resolve("asset_mcap_locator", asset_id=asset_id))
        return self.download_mcap(locator["mcap_file_id"], output_path)

    def open_mcap(self, mcap_file_id: str) -> IO[bytes]:
        import httpx
        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {"Content-Type": "application/json", **self._requestor._auth_headers}
        resp = httpx.get(url, headers=headers, follow_redirects=True)
        resp.raise_for_status()
        return io.BytesIO(resp.content)

    def finalize_upload(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", "storage_upload_finalize", json_body=payload)

    def get_messages(self, mcap_id: str) -> dict[str, Any]:
        return self._request("GET", self._cfg.resolve("storage_mcap_messages", mcap_id=mcap_id))
