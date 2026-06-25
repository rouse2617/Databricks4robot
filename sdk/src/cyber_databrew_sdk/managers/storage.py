"""DataBrewFS — POSIX-like file interface for GCS and asset:// URIs.

Usage::

    from cyber_databrew_sdk import CyberDatabrew

    sdk = CyberDatabrew(token="...")

    # Read asset by ID
    with sdk.open("asset://019ed9c6-ab86-7cec-ba4b-83e580b2ff31", "rb") as f:
        data = f.read()

    # Direct GCS path
    with sdk.open("gs://my-bucket/videos/xxx.mp4", "rb") as f:
        chunk = f.read(8 * 1024 * 1024)

    # Write
    with sdk.open("gs://my-bucket/output/result.json", "wb") as f:
        f.write(b'{"key": "value"}')

    # Shortcuts
    data = sdk.read("gs://bucket/file.bin")
    sdk.write("gs://bucket/file.bin", data)
    info = sdk.stat("gs://bucket/file.bin")
    entries = sdk.listdir("gs://bucket/prefix/")
"""

from __future__ import annotations

import io
import logging
import os
import time
from dataclasses import dataclass
from pathlib import Path
from typing import IO, Any, BinaryIO, Iterator
from urllib.parse import urlparse

from cyber_databrew_sdk._base_manager import BaseManager

_logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Types
# ---------------------------------------------------------------------------


@dataclass
class FileInfo:
    """File metadata (similar to ``os.stat_result``)."""

    name: str
    size: int
    mtime: float | None
    type: str  # "file" | "dir"


class DataBrewFile(io.RawIOBase):
    """A file-like object for reading/writing GCS objects via signed URLs.

    Wraps a ``httpx`` streaming response.  Supports ``read()``,
    ``readinto()``, ``seek()``, ``tell()``, and context manager.
    """

    def __init__(self, url: str, mode: str = "rb") -> None:
        self._url = url
        self._mode = mode
        self._pos = 0
        self._closed = False
        self._response = None
        self._client = None
        self._open()

    def _open(self) -> None:
        import httpx

        self._client = httpx.Client(timeout=None)
        if "r" in self._mode:
            self._response = self._client.stream("GET", self._url, follow_redirects=True)
            self._response.__enter__()
        elif "w" in self._mode:
            self._response = None  # PUT is handled by _upload

    def readable(self) -> bool:
        return "r" in self._mode

    def writable(self) -> bool:
        return "w" in self._mode

    def seekable(self) -> bool:
        return True

    def read(self, n: int = -1) -> bytes:
        if not self._response:
            raise OSError("not opened for reading")
        if n == -1:
            data = self._response.read()
        else:
            data = self._response.read(n)
        self._pos += len(data)
        return data

    def readinto(self, b: bytearray) -> int:
        data = self.read(len(b))
        n = len(data)
        b[:n] = data
        return n

    def seek(self, offset: int, whence: int = os.SEEK_SET) -> int:
        if not self._response:
            raise OSError("not opened for reading")
        if whence == os.SEEK_SET:
            self._pos = offset
        elif whence == os.SEEK_CUR:
            self._pos += offset
        elif whence == os.SEEK_END:
            raise NotImplementedError("SEEK_END not supported")
        # Reopen with Range header for seek
        self._response.close()
        import httpx

        self._client = httpx.Client(timeout=None)
        headers = {"Range": f"bytes={self._pos}-"} if self._pos > 0 else {}
        self._response = self._client.stream(
            "GET", self._url, headers=headers, follow_redirects=True
        )
        self._response.__enter__()
        return self._pos

    def tell(self) -> int:
        return self._pos

    def close(self) -> None:
        if self._closed:
            return
        self._closed = True
        if self._response:
            try:
                self._response.__exit__(None, None, None)
            except Exception:
                pass
        if self._client:
            self._client.close()

    def __enter__(self) -> DataBrewFile:
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


# ---------------------------------------------------------------------------
# StorageManager
# ---------------------------------------------------------------------------


class StorageManager(BaseManager):
    """POSIX-like file operations on GCS and DataBrew assets.

    This manager is accessed via ``client.storage`` but also aliased
    as the default filesystem for the SDK — so ``client.open(...)``
    delegates here.
    """

    # ------------------------------------------------------------------
    # open — the core primitive
    # ------------------------------------------------------------------

    def open(self, uri: str, mode: str = "rb") -> DataBrewFile | BinaryIO:
        """Open a file for reading or writing.

        Args:
            uri: ``asset://<id>``, ``gs://<bucket>/<path>``, or local ``/path``.
            mode: ``"rb"`` (read) or ``"wb"`` (write).

        Returns:
            A file-like object supporting ``read()``, ``seek()``, ``tell()``.
        """
        url = self._resolve(uri, mode)
        if url is None:
            # Local file
            return open(self._strip_scheme(uri), mode)  # type: ignore[return-value]
        return DataBrewFile(url, mode)

    # ------------------------------------------------------------------
    # Convenience shortcuts
    # ------------------------------------------------------------------

    def read(self, uri: str) -> bytes:
        """Read the entire file into memory.

        For large files, prefer ``open(uri, "rb")`` and stream.
        """
        with self.open(uri, "rb") as f:
            return f.read()

    def write(self, uri: str, data: bytes | str) -> int:
        """Write bytes (or string) to a file."""
        if isinstance(data, str):
            data = data.encode("utf-8")
        url = self._resolve(uri, "wb")
        if url is None:
            with open(self._strip_scheme(uri), "wb") as f:
                return f.write(data)
        import httpx

        resp = httpx.put(url, content=data)
        resp.raise_for_status()
        return len(data)

    # ------------------------------------------------------------------
    # File info & listing
    # ------------------------------------------------------------------

    def stat(self, uri: str) -> FileInfo:
        """Get file metadata (size, mtime, type).

        Raises ``FileNotFoundError`` if the path does not exist.
        """
        gs_path = self._parse_gs(uri)
        if gs_path:
            bucket, blob_name = gs_path
            from google.cloud import storage as gcs_storage

            client = gcs_storage.Client()
            bucket_obj = client.bucket(bucket)
            blob = bucket_obj.get_blob(blob_name)
            if blob is None:
                # Check if it's a "directory" by listing prefix
                blobs = list(client.list_blobs(bucket, prefix=blob_name.rstrip("/") + "/", max_results=1))
                if blobs:
                    return FileInfo(name=uri, size=0, mtime=None, type="dir")
                raise FileNotFoundError(f"gs://{bucket}/{blob_name}")
            return FileInfo(
                name=uri,
                size=blob.size or 0,
                mtime=blob.updated.timestamp() if blob.updated else None,
                type="file",
            )

        # asset:// — resolve then stat
        gs_path = self._resolve_asset(uri)
        if gs_path:
            return self.stat(gs_path)

        # Local
        path = Path(self._strip_scheme(uri))
        if not path.exists():
            raise FileNotFoundError(str(path))
        st = path.stat()
        return FileInfo(name=uri, size=st.st_size, mtime=st.st_mtime, type="dir" if path.is_dir() else "file")

    def listdir(self, uri: str) -> list[FileInfo]:
        """List entries under a GCS prefix or local directory."""
        gs_path = self._parse_gs(uri)
        if gs_path:
            bucket, prefix = gs_path
            from google.cloud import storage as gcs_storage

            client = gcs_storage.Client()
            blobs = client.list_blobs(bucket, prefix=prefix.rstrip("/") + "/")
            results: list[FileInfo] = []
            seen_prefixes: set[str] = set()
            for blob in blobs:
                name = blob.name
                if name == prefix.rstrip("/") + "/":
                    continue
                # Check if this is a "directory" marker
                rest = name[len(prefix.rstrip("/")) + 1:] if prefix else name
                if "/" in rest.rstrip("/"):
                    dir_name = rest.split("/")[0]
                    if dir_name not in seen_prefixes:
                        seen_prefixes.add(dir_name)
                        results.append(FileInfo(name=dir_name, size=0, mtime=None, type="dir"))
                else:
                    results.append(FileInfo(
                        name=name,
                        size=blob.size or 0,
                        mtime=blob.updated.timestamp() if blob.updated else None,
                        type="file",
                    ))
            return results

        # Local
        path = Path(self._strip_scheme(uri))
        if not path.is_dir():
            raise NotADirectoryError(str(path))
        results = []
        for entry in path.iterdir():
            st = entry.stat()
            results.append(FileInfo(
                name=entry.name,
                size=st.st_size if entry.is_file() else 0,
                mtime=st.st_mtime,
                type="dir" if entry.is_dir() else "file",
            ))
        return results

    # ------------------------------------------------------------------
    # Copy, delete, exists
    # ------------------------------------------------------------------

    def copy(self, src: str, dst: str) -> None:
        """Copy a file between GCS paths (or local paths).

        For GCS-to-GCS copies, the backend performs a server-side copy
        (no data transferred through the client).
        """
        src_gs = self._parse_gs(src)
        dst_gs = self._parse_gs(dst)

        if src_gs and dst_gs:
            from google.cloud import storage as gcs_storage

            client = gcs_storage.Client()
            src_bucket = client.bucket(src_gs[0])
            src_blob = src_bucket.blob(src_gs[1])
            dst_bucket = client.bucket(dst_gs[0])
            dst_blob = dst_bucket.blob(dst_gs[1])
            dst_blob.rewrite(src_blob)
            return

        # Fallback: download + upload
        data = self.read(src)
        self.write(dst, data)

    def delete(self, uri: str) -> None:
        """Delete a file."""
        gs_path = self._parse_gs(uri)
        if gs_path:
            from google.cloud import storage as gcs_storage

            client = gcs_storage.Client()
            bucket = client.bucket(gs_path[0])
            blob = bucket.blob(gs_path[1])
            blob.delete()
            return
        Path(self._strip_scheme(uri)).unlink()

    def exists(self, uri: str) -> bool:
        """Check if a file or asset exists."""
        try:
            self.stat(uri)
            return True
        except (FileNotFoundError, NotADirectoryError):
            return False

    # ------------------------------------------------------------------
    # Backward-compat: keep existing methods
    # ------------------------------------------------------------------

    def list_files(self, **kwargs: Any) -> dict[str, Any]:
        """List MCAP files (legacy)."""
        return self._request("GET", "storage_files_list", params=kwargs)

    def get_file_info(self, mcap_id: str) -> dict[str, Any]:
        """Get MCAP file metadata (legacy)."""
        return self._request("GET", "storage_file_info", mcap_id=mcap_id)

    def download_mcap(self, mcap_file_id: str, output_path: str | Path) -> Path:
        """Download MCAP (legacy, kept for compat)."""
        output_path = Path(output_path)
        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {"Content-Type": "application/json", **self._requestor._auth_headers}
        import httpx

        with httpx.Client().stream("GET", url, headers=headers, follow_redirects=True) as resp:
            resp.raise_for_status()
            output_path.parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, "wb") as f:
                for chunk in resp.iter_bytes(chunk_size=8192):
                    f.write(chunk)
        return output_path

    def download_asset_mcap(self, asset_id: str, output_path: str | Path) -> Path:
        """Download MCAP by asset ID (legacy)."""
        locator = self._request("GET", "asset_mcap_locator", asset_id=asset_id)
        return self.download_mcap(locator["mcap_file_id"], output_path)

    def open_mcap(self, mcap_file_id: str) -> IO[bytes]:
        """Open MCAP as stream (legacy)."""
        import io
        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {"Content-Type": "application/json", **self._requestor._auth_headers}
        import httpx

        resp = httpx.get(url, headers=headers, follow_redirects=True)
        resp.raise_for_status()
        return io.BytesIO(resp.content)

    def finalize_upload(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Finalize multipart upload (legacy)."""
        return self._request("POST", "storage_upload_finalize", json_body=payload)

    def get_messages(self, mcap_id: str) -> dict[str, Any]:
        """Get MCAP messages (legacy)."""
        return self._request("GET", "storage_mcap_messages", mcap_id=mcap_id)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _resolve(self, uri: str, mode: str = "rb") -> str | None:
        """Resolve a URI to a signed GCS URL, or None for local paths."""
        if uri.startswith("asset://"):
            gs_path = self._resolve_asset(uri)
            if not gs_path:
                raise FileNotFoundError(f"cannot resolve asset URI: {uri}")
            return self._sign_url(gs_path, method="GET" if "r" in mode else "PUT")

        gs_path = self._parse_gs(uri)
        if gs_path:
            return self._sign_url(gs_path, method="GET" if "r" in mode else "PUT")

        # Local path — return None
        return None

    def _resolve_asset(self, uri: str) -> str | None:
        """Resolve ``asset://<id>`` to a ``gs://bucket/path`` string.

        Calls the backend asset locator endpoint.
        """
        asset_id = uri[len("asset://"):].strip("/")
        try:
            locator = self._request("GET", "asset_mcap_locator", asset_id=asset_id)
            mcap_id = locator.get("mcap_file_id")
            if mcap_id:
                info = self._request("GET", "storage_file_info", mcap_id=mcap_id)
                gcs_path = info.get("gcs_path") or info.get("storage_path", "")
                if gcs_path:
                    return gcs_path
        except Exception as exc:
            _logger.warning("failed to resolve asset %s: %s", asset_id, exc)
        return None

    def _sign_url(self, gs_path: str, method: str = "GET") -> str:
        """Sign a GCS URL via the backend, or fall back to local signing.

        Args:
            gs_path: ``bucket/path`` (without ``gs://`` prefix).
            method: ``"GET"`` or ``"PUT"``.
        """
        bucket, _, obj = gs_path.partition("/")
        try:
            result = self._request("POST", "storage_sign_url", json_body={
                "bucket": bucket,
                "object": obj,
                "method": method,
            })
            return result["url"]
        except Exception as exc:
            _logger.info("backend sign-url failed (%s), falling back to local signing", exc)
            return self._sign_url_locally(bucket, obj, method)

    def _sign_url_locally(self, bucket: str, obj: str, method: str = "GET") -> str:
        """Fallback: sign URL using local ADC credentials."""
        from google.cloud import storage as gcs_storage

        client = gcs_storage.Client()
        url, _ = client.blob(bucket, obj).generate_signed_url(
            version="v4",
            expiration=900,
            method=method,
        )
        return url

    @staticmethod
    def _parse_gs(uri: str) -> tuple[str, str] | None:
        """Parse ``gs://bucket/path`` into ``(bucket, path)``."""
        if not uri.startswith("gs://"):
            return None
        path = uri[5:]
        bucket, _, obj = path.partition("/")
        if not bucket:
            return None
        return bucket, obj

    @staticmethod
    def _strip_scheme(uri: str) -> str:
        """Strip ``file://`` prefix for local paths."""
        if uri.startswith("file://"):
            return uri[7:]
        return uri
