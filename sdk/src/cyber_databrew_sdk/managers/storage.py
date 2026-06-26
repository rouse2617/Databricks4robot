"""DataBrewFS — unified filesystem backed by fsspec.

Supports GCS, S3, and local files with a single POSIX-like interface::

    from cyber_databrew_sdk import CyberDatabrew

    sdk = CyberDatabrew(token="...")

    # Any protocol — same API
    sdk.read("gs://bucket/file.mcap")
    sdk.read("s3://bucket/file.mcap")
    sdk.open("/local/path/file.bin", "rb")

    with sdk.open("gs://bucket/file.mcap", "rb") as f:
        f.seek(1024)
        chunk = f.read(8192)

    sdk.download("gs://bucket/file.mcap", "/tmp/local.mcap")
"""

from __future__ import annotations

import io
import logging
import os
from dataclasses import dataclass
from pathlib import Path
from typing import IO, Any

import fsspec

from cyber_databrew_sdk._base_manager import BaseManager

_logger = logging.getLogger(__name__)

# Large-file threshold — above this use the cloud-specific high-speed channel
_TRANSFER_THRESHOLD = 100 * 1024 * 1024  # 100 MiB


@dataclass
class FileInfo:
    """File metadata (similar to ``os.stat_result``)."""
    name: str
    size: int
    mtime: float | None
    type: str  # "file" | "dir"


# ---------------------------------------------------------------------------
# Cloud-specific high-speed bulk transfer
# ---------------------------------------------------------------------------

def _gcs_transfer_download(bucket: str, obj: str, local_path: str) -> None:
    """GCS parallel chunked download via ``transfer_manager`` (thread pool)."""
    from google.cloud import storage
    from google.cloud.storage import transfer_manager

    client = storage.Client()
    blob = client.bucket(bucket).blob(obj)
    transfer_manager.download_chunks_concurrently(
        blob, local_path,
        worker_type="thread", max_workers=8,
    )


def _gcs_transfer_upload(local_path: str, bucket: str, obj: str) -> None:
    """GCS parallel chunked upload via ``transfer_manager`` (thread pool)."""
    from google.cloud import storage
    from google.cloud.storage import transfer_manager

    client = storage.Client()
    blob = client.bucket(bucket).blob(obj)
    transfer_manager.upload_chunks_concurrently(
        local_path, blob,
        worker_type="thread", max_workers=8,
    )


def _gcs_transfer_read(bucket: str, obj: str) -> bytes:
    """Download GCS via ``transfer_manager`` → in-memory bytes."""
    import tempfile
    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp_name = tmp.name
    try:
        _gcs_transfer_download(bucket, obj, tmp_name)
        with open(tmp_name, "rb") as f:
            return f.read()
    finally:
        os.unlink(tmp_name)


def _gcs_transfer_write(bucket: str, obj: str, data: bytes) -> None:
    """Upload bytes to GCS via ``transfer_manager``."""
    import tempfile
    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp.write(data)
        tmp_name = tmp.name
    try:
        _gcs_transfer_upload(tmp_name, bucket, obj)
    finally:
        os.unlink(tmp_name)


# ---------------------------------------------------------------------------
# Protocol — URI helpers
# ---------------------------------------------------------------------------

_KNOWN_PROTOCOLS = {"gs", "s3", "file"}


def _parse_uri(uri: str) -> tuple[str, str]:
    """Split ``protocol://path`` → ``(protocol, path)``.

    Local paths (no protocol) return ``("", raw_path)``.
    """
    if "://" in uri:
        proto, _, path = uri.partition("://")
        return proto, path
    return "", uri


def _is_cloud(proto: str) -> bool:
    return proto in {"gs", "s3"}


# ---------------------------------------------------------------------------
# StorageManager
# ---------------------------------------------------------------------------

class StorageManager(BaseManager):
    """Unified filesystem backed by fsspec + cloud-specific high-speed channels.

    +------------+------------------+---------------------+
    | Protocol   | POSIX (open/rw)  | Bulk (read/write)   |
    +------------+------------------+---------------------+
    | ``gs://``  | gcsfs            | transfer_manager    |
    | ``s3://``  | s3fs             | boto3 transfer      |
    | local      | built-in open    | —                   |
    | ``asset://``| backend resolve  | → delegated         |
    +------------+------------------+---------------------+
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._fsspec_caches: dict[str, fsspec.AbstractFileSystem] = {}
        self._gcs_token: str | None = None

    def set_gcs_token(self, token: str) -> None:
        """Set an explicit GCS access token (local development)."""
        self._gcs_token = token
        self._fsspec_caches.pop("gs", None)  # force re-init

    def _fs(self, protocol: str) -> fsspec.AbstractFileSystem:
        if protocol not in self._fsspec_caches:
            kwargs = {}
            if protocol == "gs" and self._gcs_token:
                kwargs["token"] = self._gcs_token
            self._fsspec_caches[protocol] = fsspec.filesystem(protocol, **kwargs)
        return self._fsspec_caches[protocol]

    # ------------------------------------------------------------------
    # fsspec filesystem resolution
    # ------------------------------------------------------------------

    def _fs(self, protocol: str) -> fsspec.AbstractFileSystem:
        """Get or create an fsspec filesystem for the given protocol."""
        if protocol not in self._fsspec_caches:
            self._fsspec_caches[protocol] = fsspec.filesystem(protocol)
        return self._fsspec_caches[protocol]

    def _resolve(self, uri: str) -> tuple[fsspec.AbstractFileSystem, str, str]:
        """Resolve URI → (filesystem, path, protocol).

        ``asset://`` is resolved via the backend first, then delegated to
        the underlying protocol (gs:/s3:).
        """
        if uri.startswith("asset://"):
            path = self._resolve_asset(uri)
            if not path:
                raise FileNotFoundError(f"cannot resolve asset URI: {uri}")
            proto, _, obj = path.partition("/")
            return self._fs(proto), obj, proto

        proto, path = _parse_uri(uri)
        if not proto:
            return self._fs("file"), path, ""
        return self._fs(proto), path, proto

    def _cloud_path(self, proto: str, path: str) -> tuple[str, str]:
        """For cloud protocols, split ``bucket/object`` → ``(bucket, object)``."""
        bucket, _, obj = path.partition("/")
        return bucket, obj

    # ------------------------------------------------------------------
    # Bulk transfer helpers
    # ------------------------------------------------------------------

    def _bulk_read(self, proto: str, bucket: str, obj: str) -> bytes:
        """High-speed bulk read."""
        if proto == "gs":
            return _gcs_transfer_read(bucket, obj)
        if proto == "s3":
            return self._fs("s3").open(f"{bucket}/{obj}", "rb").read()
        raise NotImplementedError(f"no bulk reader for {proto}")

    def _bulk_write(self, proto: str, bucket: str, obj: str, data: bytes) -> None:
        """High-speed bulk write."""
        if proto == "gs":
            _gcs_transfer_write(bucket, obj, data)
        elif proto == "s3":
            self._fs("s3").open(f"{bucket}/{obj}", "wb").write(data)
        else:
            raise NotImplementedError(f"no bulk writer for {proto}")

    def _bulk_download(self, proto: str, bucket: str, obj: str, local_path: str) -> None:
        """High-speed download to local file."""
        if proto == "gs":
            _gcs_transfer_download(bucket, obj, local_path)
        elif proto == "s3":
            self._fs("s3").get(f"{bucket}/{obj}", local_path)
        else:
            raise NotImplementedError(f"no bulk download for {proto}")

    # ------------------------------------------------------------------
    # Asset URI resolution
    # ------------------------------------------------------------------

    def _resolve_asset(self, uri: str) -> str | None:
        """Resolve ``asset://source:id/subpath`` → ``protocol/bucket/object``."""
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
                    return gcs_path  # returns "gs/bucket/object" or "s3/bucket/object"
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
                    return gcs_path
        except Exception as exc:
            _logger.warning("legacy asset resolve failed for %s: %s", uri, exc)
        return None

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def open(self, uri: str, mode: str = "rb") -> IO[Any]:
        """Open a file (supports seek/tell on cloud protocols)."""
        fs, path, proto = self._resolve(uri)
        if proto == "file" or not proto:
            return open(path, mode)
        return fs.open(path, mode)

    def read(self, uri: str) -> bytes:
        """Read entire file — uses bulk channel for large cloud files."""
        if uri.startswith("file://") or not uri.startswith(("gs://", "s3://", "asset://")):
            return Path(_parse_uri(uri)[1]).read_bytes()

        fs, path, proto = self._resolve(uri)

        # Small files → fsspec (fast path, no temp file)
        if _is_cloud(proto):
            try:
                info = fs.info(path)
                size = info.get("size", 0) if isinstance(info, dict) else 0
            except Exception:
                size = 0
            if size and size < _TRANSFER_THRESHOLD:
                return fs.open(path, "rb").read()
            bucket, obj = self._cloud_path(proto, path)
            return self._bulk_read(proto, bucket, obj)

        return fs.open(path, "rb").read()

    def write(self, uri: str, data: bytes | str) -> int:
        """Write entire file — uses bulk channel for large data."""
        if isinstance(data, str):
            data = data.encode("utf-8")

        if not uri.startswith(("gs://", "s3://", "asset://")):
            Path(_parse_uri(uri)[1]).write_bytes(data)
            return len(data)

        fs, path, proto = self._resolve(uri)

        if len(data) > _TRANSFER_THRESHOLD and _is_cloud(proto):
            bucket, obj = self._cloud_path(proto, path)
            self._bulk_write(proto, bucket, obj, data)
        else:
            with fs.open(path, "wb") as f:
                f.write(data)
        return len(data)

    def upload(self, local_path: str | Path, uri: str) -> None:
        """Upload a local file to cloud storage.

        Uses transfer_manager for large files (>100 MiB).
        Avoids loading the entire file into memory.
        """
        local_path = Path(local_path)
        if not local_path.exists():
            raise FileNotFoundError(f"local file not found: {local_path}")

        if not uri.startswith(("gs://", "s3://", "asset://")):
            import shutil
            shutil.copy2(local_path, Path(_parse_uri(uri)[1]))
            return

        fs, path, proto = self._resolve(uri)
        bucket, obj = self._cloud_path(proto, path)

        size = local_path.stat().st_size
        if size > _TRANSFER_THRESHOLD and proto == "gs":
            _gcs_transfer_upload(str(local_path), bucket, obj)
        elif size > _TRANSFER_THRESHOLD and proto == "s3":
            self._bulk_write(proto, bucket, obj, local_path.read_bytes())
        else:
            with fs.open(path, "wb") as f:
                f.write(local_path.read_bytes())

    def download(self, uri: str, output_path: str | Path) -> Path:
        """Download file to local disk via high-speed channel."""
        output_path = Path(output_path)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        if not uri.startswith(("gs://", "s3://", "asset://")):
            import shutil
            shutil.copy2(_parse_uri(uri)[1], output_path)
            return output_path

        fs, path, proto = self._resolve(uri)
        bucket, obj = self._cloud_path(proto, path)
        self._bulk_download(proto, bucket, obj, str(output_path))
        return output_path

    def stat(self, uri: str) -> FileInfo:
        """Get file metadata."""
        if not uri.startswith(("gs://", "s3://", "asset://", "file://")):
            p = Path(_parse_uri(uri)[1])
            if not p.exists():
                raise FileNotFoundError(str(uri))
            st = p.stat()
            return FileInfo(name=uri, size=st.st_size, mtime=st.st_mtime,
                            type="dir" if p.is_dir() else "file")

        fs, path, proto = self._resolve(uri)
        try:
            info = fs.info(path)
        except Exception as exc:
            raise FileNotFoundError(str(uri)) from exc

        if isinstance(info, dict):
            return FileInfo(
                name=uri,
                size=info.get("size", 0),
                mtime=info.get("mtime", None) or info.get("updated", None),
                type="dir" if info.get("type") in ("directory", "dir") else "file",
            )
        # Fallback for fsspec backends that return scalar
        return FileInfo(name=uri, size=0, mtime=None, type="file")

    def listdir(self, uri: str) -> list[FileInfo]:
        """List directory entries."""
        if not uri.startswith(("gs://", "s3://", "asset://", "file://")):
            _uri = uri
            path = _parse_uri(uri)[1]
            p = Path(path)
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

        fs, path, proto = self._resolve(uri)
        try:
            entries = fs.ls(path.rstrip("/") + "/")
        except Exception as exc:
            raise FileNotFoundError(str(uri)) from exc

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
        """Copy file — uses fsspec cp for cloud, shutil for local."""
        if not src.startswith(("gs://", "s3://")):
            import shutil
            shutil.copy2(_parse_uri(src)[1], self._local_path(dst) if "://" not in dst else dst)
            return

        fs_src, path_src, _ = self._resolve(src)
        if dst.startswith(("gs://", "s3://")):
            fs_dst, path_dst, _ = self._resolve(dst)
            if fs_src is fs_dst:
                fs_src.cp(path_src, path_dst)
                return
        # Cross-protocol or mixed: download + upload
        data = self.read(src)
        self.write(dst, data)

    def delete(self, uri: str) -> None:
        """Delete file."""
        if not uri.startswith(("gs://", "s3://", "file://", "asset://")):
            Path(_parse_uri(uri)[1]).unlink()
            return
        fs, path, _ = self._resolve(uri)
        fs.rm(path)

    def exists(self, uri: str) -> bool:
        """Check file existence."""
        try:
            self.stat(uri)
            return True
        except (FileNotFoundError, NotADirectoryError):
            return False

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    @staticmethod
    def _local_path(uri: str) -> str:
        if uri.startswith("file://"):
            return uri[7:]
        return uri

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
