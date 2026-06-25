"""DataBrewFS — POSIX-like file interface backed by Arrow GcsFileSystem.

Pods in GKE have a service account with GCS access, so ``gs://`` paths
are handled directly via PyArrow ``GcsFileSystem``.  ``asset://`` URIs
(such as ``asset://grace:video_id/raw``) are resolved through the DataBrew
backend, which calls the external source API (e.g. Grace) and returns a
GCS path that Arrow fs then reads directly.

Usage::

    from cyber_databrew_sdk import CyberDatabrew

    sdk = CyberDatabrew(token="...")

    # Read GCS file directly (pod SA has permissions)
    data = sdk.read("gs://co-prod-gv-cybercap/raw/e6c57180dab96076b923999acb45ad59.mcap")

    # Read resolved asset
    data = sdk.read("asset://grace:019ed9c5-.../raw")

    # Write
    sdk.write("gs://my-bucket/output/result.json", data)

    # Stream read
    with sdk.open("gs://my-bucket/video.mp4", "rb") as f:
        chunk = f.read(8 * 1024 * 1024)
"""

from __future__ import annotations

import io
import logging
import datetime
from dataclasses import dataclass
from pathlib import Path
from typing import IO, Any

import pyarrow.fs as pa_fs

from cyber_databrew_sdk._base_manager import BaseManager

_logger = logging.getLogger(__name__)


@dataclass
class FileInfo:
    """File metadata (similar to ``os.stat_result``)."""
    name: str
    size: int
    mtime: float | None
    type: str  # "file" | "dir"


class StorageManager(BaseManager):
    """File operations backed by Arrow GcsFileSystem (for GCS) + backend
    resolve (for ``asset://`` URIs)."""

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._gcs: pa_fs.FileSystem | None = None
        self._gcs_token: str | None = None

    # ------------------------------------------------------------------
    # Arrow GcsFileSystem
    # ------------------------------------------------------------------

    def set_gcs_token(self, token: str) -> None:
        """Set a GCS access token for Arrow fs.

        Call this on your dev machine when ADC is unavailable::

            import subprocess
            token = subprocess.check_output(
                ["gcloud", "auth", "print-access-token"]
            ).decode().strip()
            sdk.storage.set_gcs_token(token)

        In GKE, skip this — the Pod's service account is used automatically.
        """
        self._gcs_token = token
        self._gcs = None  # force re-init

    @property
    def fs(self) -> pa_fs.FileSystem:
        """Lazily-initialized Arrow GcsFileSystem.

        In GKE, uses the Pod's service account (ADC) automatically.
        On dev machines, call ``set_gcs_token()`` first.
        """
        if self._gcs is None:
            kw = {}
            if self._gcs_token:
                kw["access_token"] = self._gcs_token
                kw["credential_token_expiration"] = (
                    datetime.datetime.now() + datetime.timedelta(hours=1)
                )
            self._gcs = pa_fs.GcsFileSystem(**kw)
        return self._gcs

    # ------------------------------------------------------------------
    # GCS URI helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _gcs_path(uri: str) -> str:
        """Strip ``gs://`` prefix → ``bucket/object`` for Arrow fs."""
        return uri[5:]

    @staticmethod
    def _bucket_object(uri: str) -> tuple[str, str]:
        """Parse ``gs://bucket/object`` → ``(bucket, object)``."""
        path = uri[5:]
        bucket, _, obj = path.partition("/")
        return bucket, obj

    @staticmethod
    def _local_path(uri: str) -> str:
        """Strip ``file://`` prefix → local path."""
        return uri[7:] if uri.startswith("file://") else uri

    # ------------------------------------------------------------------
    # Asset URI resolution
    # ------------------------------------------------------------------

    def _resolve_asset(self, uri: str) -> str | None:
        """Resolve ``asset://source:id/subpath`` → GCS path (``bucket/object``).

        Returns None on failure.
        """
        rest = uri[len("asset://"):]
        if ":" in rest:
            source, _, asset_id = rest.partition(":")
            source = source.strip()
            asset_id = asset_id.strip("/")
            try:
                result = self._request("POST", "storage_resolve", json_body={
                    "source": source,
                    "id": asset_id,
                    "env": "dev",  # TODO: read from config
                })
                gcs_path = result.get("gcs_path", "")
                if gcs_path:
                    return gcs_path[5:] if gcs_path.startswith("gs://") else gcs_path
            except Exception as exc:
                _logger.warning("resolve failed for %s: %s", uri, exc)
            return None

        # Legacy: plain asset://<id> → MCAP locator
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
        """Open a file for reading/writing.

        ``gs://`` → Arrow GcsFileSystem
        ``asset://`` → backend resolve → Arrow GcsFileSystem
        local → built-in ``open()``
        """
        if uri.startswith("gs://"):
            path = self._gcs_path(uri)
            if "r" in mode:
                return self.fs.open_input_file(path)
            return self.fs.open_output_stream(path)
        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve asset URI: {uri}")
            if "r" in mode:
                return self.fs.open_input_file(path)
            return self.fs.open_output_stream(path)
        return open(self._local_path(uri), mode)

    def read(self, uri: str) -> bytes:
        """Read entire file into memory."""
        if uri.startswith("gs://") or uri.startswith("asset://"):
            path = self._gcs_path(uri) if uri.startswith("gs://") else self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            return self.fs.open_input_stream(path).read()
        return Path(self._local_path(uri)).read_bytes()

    def write(self, uri: str, data: bytes | str) -> int:
        """Write entire file."""
        if isinstance(data, str):
            data = data.encode("utf-8")
        if uri.startswith("gs://"):
            with self.fs.open_output_stream(self._gcs_path(uri)) as f:
                f.write(data)
            return len(data)
        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            with self.fs.open_output_stream(path) as f:
                f.write(data)
            return len(data)
        Path(self._local_path(uri)).write_bytes(data)
        return len(data)

    def stat(self, uri: str) -> FileInfo:
        """Get file metadata."""
        if uri.startswith("gs://") or uri.startswith("asset://"):
            path = self._gcs_path(uri) if uri.startswith("gs://") else self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            try:
                info = self.fs.get_file_info(path)
            except Exception as exc:
                raise FileNotFoundError(str(uri)) from exc
            if info.type == pa_fs.FileType.NotFound:
                raise FileNotFoundError(str(uri))
            return FileInfo(
                name=uri,
                size=info.size,
                mtime=info.mtime_ns / 1e9 if info.mtime_ns is not None else None,
                type="dir" if info.type == pa_fs.FileType.Directory else "file",
            )
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
        if uri.startswith("gs://") or uri.startswith("asset://"):
            path = self._gcs_path(uri) if uri.startswith("gs://") else self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            selector = pa_fs.FileSelector(path.rstrip("/") + "/")
            infos = self.fs.get_file_info(selector)
            return [FileInfo(
                name=info.path or "",
                size=info.size,
                mtime=info.mtime_ns / 1e9 if info.mtime_ns is not None else None,
                type="dir" if info.type == pa_fs.FileType.Directory else "file",
            ) for info in infos]
        p = Path(self._local_path(uri))
        if not p.is_dir():
            raise NotADirectoryError(str(uri))
        results = []
        for entry in p.iterdir():
            st = entry.stat()
            results.append(FileInfo(
                name=entry.name, size=st.st_size, mtime=st.st_mtime,
                type="dir" if entry.is_dir() else "file",
            ))
        return results

    def copy(self, src: str, dst: str) -> None:
        """Copy file via Arrow fs (GCS server-side copy when both paths are GCS)."""
        src_path = self._gcs_path(src) if src.startswith("gs://") else (
            self._resolve_asset(src) if src.startswith("asset://") else self._local_path(src)
        )
        dst_path = self._gcs_path(dst) if dst.startswith("gs://") else (
            self._resolve_asset(dst) if dst.startswith("asset://") else self._local_path(dst)
        )
        if not src_path or not dst_path:
            raise FileNotFoundError(f"cannot resolve: {src} or {dst}")
        is_gcs = src.startswith("gs://") or dst.startswith("gs://")
        if is_gcs:
            self.fs.copy_file(src_path, dst_path)
            return
        data = self.read(src)
        self.write(dst, data)

    def delete(self, uri: str) -> None:
        """Delete file."""
        if uri.startswith("gs://"):
            self.fs.delete_file(self._gcs_path(uri))
        elif uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            self.fs.delete_file(path)
        else:
            Path(self._local_path(uri)).unlink()

    def exists(self, uri: str) -> bool:
        """Check file existence."""
        try:
            self.stat(uri)
            return True
        except (FileNotFoundError, NotADirectoryError):
            return False

    # --- legacy MCAP methods ---

    def list_files(self, **kwargs: Any) -> dict[str, Any]:
        return self._request("GET", "storage_files_list", params=kwargs)

    def get_file_info(self, mcap_id: str) -> dict[str, Any]:
        return self._request("GET", "storage_file_info", mcap_id=mcap_id)

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
        locator = self._request("GET", "asset_mcap_locator", asset_id=asset_id)
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
        return self._request("GET", "storage_mcap_messages", mcap_id=mcap_id)
