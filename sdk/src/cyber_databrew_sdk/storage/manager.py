"""StorageManager — pure IO layer. No business logic.

Knows nothing about ``asset://``, users, auth, or path mapping.
It only routes ``gs://``, ``s3://``, ``file://``, and local paths to
the right backend for read/write.

Business-layer concerns (auth, path resolution, ``asset://``) are
handled by ``client.py`` before calling this class.

Usage::

    from cyber_databrew_sdk.storage.manager import StorageManager

    mgr = StorageManager()
    mgr.read("gs://bucket/file.mcap")       # ← already resolved path
    mgr.read("/local/path/file.bin")
"""

from __future__ import annotations

import io
import logging
from pathlib import Path
from typing import IO, Any

from cyber_databrew_sdk.storage.backend import Backend, FileInfo
from cyber_databrew_sdk.storage.gcs import GCSBackend
from cyber_databrew_sdk.storage.local import LocalBackend
from cyber_databrew_sdk.storage.proxy import ProxyBackend

_logger = logging.getLogger(__name__)


def _parse_uri(uri: str) -> tuple[str, str]:
    """Split ``protocol://path`` → ``(protocol, path)``."""
    if "://" in uri:
        proto, _, path = uri.partition("://")
        return proto, path
    return "", uri


class StorageManager:
    """Unified storage layer — POSIX IO + URI resolution.

    +----------------+------------------+--------------------+
    | Protocol       | Backend          | POSIX (seek/tell)  |
    +----------------+------------------+--------------------+
    | ``gs://``      | GCSBackend       | ✅ gcsfs            |
    | ``s3://``      | S3Backend (WIP)  | ✅ s3fs             |
    | ``asset://``   | resolve → GCS    | ✅ (经 gcsfs)       |
    | local / ``file://`` | LocalBackend | ✅ built-in        |
    +----------------+------------------+--------------------+

    ``asset://`` URIs are resolved via the backend API and delegated
    to the appropriate protocol backend.
    """

    def __init__(self, requestor: Any = None, config: Any = None) -> None:
        # requestor + config needed for asset:// resolution and legacy MCAP
        self._requestor = requestor
        self._cfg = config
        self._gcs_backend: GCSBackend | None = None
        self._local_backend: LocalBackend | None = None
        self._proxy_backend: ProxyBackend | None = None
        self._gcs_token: str | None = None

    # ── backends ─────────────────────────────────────────────────────

    @property
    def gcs(self) -> GCSBackend:
        if self._gcs_backend is None:
            self._gcs_backend = GCSBackend(token=self._gcs_token)
        return self._gcs_backend

    @property
    def local(self) -> LocalBackend:
        if self._local_backend is None:
            self._local_backend = LocalBackend()
        return self._local_backend

    def set_gcs_token(self, token: str) -> None:
        """Set GCS access token (local dev without ADC)."""
        self._gcs_token = token
        self._gcs_backend = None

    # ── URI → backend routing (protocol only, no business) ───────────

    def resolve(self, uri: str) -> tuple[Backend, str]:
        """Resolve URI → ``(backend, path)``.

        - ``gs://bucket/obj`` → (GCSBackend, "bucket/obj")
        - ``asset://grace:id`` → resolve → (GCSBackend, "bucket/obj")
        - ``/local/path`` → (LocalBackend, "/local/path")
        """
        if uri.startswith("asset://"):
            resolved = self._resolve_asset(uri)
            if not resolved:
                raise FileNotFoundError(f"cannot resolve: {uri}")
            return self.gcs, resolved

        proto, path = _parse_uri(uri)
        if proto == "gs":
            return self.gcs, path
        if proto == "s3":
            raise NotImplementedError("S3 backend not yet implemented")
        if proto == "file" or not proto:
            return self.local, path
        raise ValueError(f"unknown protocol: {proto}")

    def _resolve_asset(self, uri: str) -> str | None:
        """Resolve ``asset://source:id/subpath`` → ``bucket/object``."""
        rest = uri[len("asset://"):]
        if ":" in rest:
            source, _, asset_id = rest.partition(":")
            source = source.strip()
            asset_id = asset_id.strip("/")
            try:
                result = self._requestor.request("POST", "storage_resolve", json_body={
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
            locator = self._requestor.request("GET", self._cfg.resolve("asset_mcap_locator", asset_id=asset_id))
            mcap_id = locator.get("mcap_file_id")
            if mcap_id:
                info = self._requestor.request("GET", self._cfg.resolve("storage_file_info", mcap_id=mcap_id))
                gcs_path = info.get("gcs_path") or info.get("storage_path", "")
                if gcs_path:
                    return gcs_path[5:] if gcs_path.startswith("gs://") else gcs_path
        except Exception as exc:
            _logger.warning("legacy asset resolve failed for %s: %s", uri, exc)
        return None

    @staticmethod
    def _local_path(uri: str) -> str:
        if uri.startswith("file://"):
            return uri[7:]
        return uri

    # ── public API (pure IO, no business logic) ──────────────────────

    def open(self, uri: str, mode: str = "rb") -> IO[Any]:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        return backend.open(path, mode)

    def read(self, uri: str) -> bytes:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        return backend.read(path)

    def write(self, uri: str, data: bytes | str) -> int:
        if isinstance(data, str):
            data = data.encode("utf-8")
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        return backend.write(path, data)

    def stat(self, uri: str) -> FileInfo:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        return backend.stat(path)

    def listdir(self, uri: str) -> list[FileInfo]:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        return backend.listdir(path)

    def copy(self, src: str, dst: str) -> None:
        backend_src, path_src = self.resolve(src)
        backend_dst, path_dst = self.resolve(dst)
        if isinstance(backend_src, type(backend_dst)):
            backend_src.copy(path_src, path_dst)
        else:
            data = backend_src.read(path_src)
            backend_dst.write(path_dst, data)

    def delete(self, uri: str) -> None:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            path = self._local_path(uri)
        backend.delete(path)

    def exists(self, uri: str) -> bool:
        try:
            self.stat(uri)
            return True
        except (FileNotFoundError, NotADirectoryError):
            return False

    def download(self, uri: str, output_path: str | Path) -> Path:
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            import shutil
            output_path = Path(output_path)
            output_path.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(self._local_path(uri), output_path)
            return output_path
        return backend.download(path, output_path)

    def upload(self, local_path: str | Path, uri: str) -> None:
        local_path = Path(local_path)
        if not local_path.exists():
            raise FileNotFoundError(f"local file not found: {local_path}")
        backend, path = self.resolve(uri)
        if isinstance(backend, LocalBackend):
            import shutil
            shutil.copy2(local_path, Path(self._local_path(uri)))
            return
        backend.upload(local_path, path)

    # ── legacy MCAP methods ──────────────────────────────────────────

    def list_files(self, **kwargs: Any) -> dict[str, Any]:
        return self._requestor.request("GET", "storage_files_list", params=kwargs)

    def get_file_info(self, mcap_id: str) -> dict[str, Any]:
        return self._requestor.request("GET", self._cfg.resolve("storage_file_info", mcap_id=mcap_id))

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
        locator = self._requestor.request("GET", self._cfg.resolve("asset_mcap_locator", asset_id=asset_id))
        return self.download_mcap(locator["mcap_file_id"], output_path)

    def open_mcap(self, mcap_file_id: str) -> IO[bytes]:
        import httpx
        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {"Content-Type": "application/json", **self._requestor._auth_headers}
        resp = httpx.get(url, headers=headers, follow_redirects=True)
        resp.raise_for_status()
        return io.BytesIO(resp.content)

    def finalize_upload(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._requestor.request("POST", "storage_upload_finalize", json_body=payload)

    def get_messages(self, mcap_id: str) -> dict[str, Any]:
        return self._requestor.request("GET", self._cfg.resolve("storage_mcap_messages", mcap_id=mcap_id))

