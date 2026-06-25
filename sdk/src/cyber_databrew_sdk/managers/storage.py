"""DataBrewFS — POSIX-like file interface backed by Arrow fs.

Usage::

    from cyber_databrew_sdk import CyberDatabrew

    sdk = CyberDatabrew(token="...")

    # Read asset by ID
    data = sdk.read("asset://019ed9c6-ab86-7cec-ba4b-83e580b2ff31")

    # Direct GCS path (POSIX-like)
    with sdk.open("gs://my-bucket/video.mp4", "rb") as f:
        chunk = f.read(8 * 1024 * 1024)

    sdk.write("gs://my-bucket/output/result.json", data)
    info = sdk.stat("gs://my-bucket/file.bin")
    entries = sdk.listdir("gs://my-bucket/prefix/")
    sdk.copy("gs://bucket-a/input.mov", "gs://bucket-b/output.mov")
    sdk.delete("gs://bucket/temp/file.bin")
"""

from __future__ import annotations

import io
import logging
import datetime
from dataclasses import dataclass
import os
from pathlib import Path
from typing import IO, Any, BinaryIO

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


class _SignedURLFile(io.RawIOBase):
    """Read a file via signed URL using HTTP."""
    def __init__(self, url: str, mode: str = "rb") -> None:
        import httpx
        self._client = httpx.Client(timeout=None)
        self._resp = self._client.stream("GET", url, follow_redirects=True)
        self._resp.__enter__()
        self._pos = 0

    def readable(self) -> bool: return True
    def read(self, n: int = -1) -> bytes:
        data = self._resp.read() if n == -1 else self._resp.read(n)
        self._pos += len(data)
        return data
    def seek(self, offset: int, whence: int = os.SEEK_SET) -> int:
        if not self._resp: raise OSError("closed")
        self._resp.close()
        # Range-based reseek not implemented for simplicity
        return self._pos
    def tell(self) -> int: return self._pos
    def close(self) -> None:
        if self._resp:
            try: self._resp.__exit__(None, None, None)
            except: pass
        if self._client: self._client.close()
    def __enter__(self) -> _SignedURLFile: return self
    def __exit__(self, *a: Any) -> None: self.close()

class StorageManager(BaseManager):
    """POSIX-like file operations backed by Arrow fs."""

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._gcs: pa_fs.FileSystem | None = None
        self._gcs_token: str | None = None
        self._last_signed_url: str | None = None

    @property
    def fs(self) -> pa_fs.FileSystem:
        """Lazily-initialized Arrow GcsFileSystem."""
        if self._gcs is None:
            kw = {}
            if self._gcs_token:
                kw["access_token"] = self._gcs_token
                kw["credential_token_expiration"] = datetime.datetime.now() + datetime.timedelta(hours=1)
            self._gcs = pa_fs.GcsFileSystem(**kw)
        return self._gcs

    def _arrow_path(self, uri: str) -> str:
        if uri.startswith("gs://"):
            return uri[5:]
        if uri.startswith("asset://"):
            resolved = self._resolve_asset(uri)
            if not resolved:
                raise FileNotFoundError(f"cannot resolve asset URI: {uri}")
            return resolved
        if uri.startswith("file://"):
            return uri[7:]
        return uri

    def open(self, uri: str, mode: str = "rb") -> BinaryIO:
        if uri.startswith("gs://") or uri.startswith("asset://"):
            path = self._arrow_path(uri)
            # If a signed URL was resolved, use HTTP download
            if self._last_signed_url:
                url = self._last_signed_url
                self._last_signed_url = None
                return _SignedURLFile(url, mode)
            if "r" in mode:
                return self.fs.open_input_stream(path)
            return self.fs.open_output_stream(path)
        return open(self._arrow_path(uri), mode)

    def read(self, uri: str) -> bytes:
        path = self._arrow_path(uri)
        if uri.startswith("gs://") or uri.startswith("asset://"):
            return self.fs.open_input_stream(path).read()
        return Path(self._arrow_path(uri)).read_bytes()

    def write(self, uri: str, data: bytes | str) -> int:
        if isinstance(data, str):
            data = data.encode("utf-8")
        if uri.startswith("gs://") or uri.startswith("asset://"):
            with self.fs.open_output_stream(self._arrow_path(uri)) as f:
                f.write(data)
            return len(data)
        Path(self._arrow_path(uri)).write_bytes(data)
        return len(data)

    def stat(self, uri: str) -> FileInfo:
        path = self._arrow_path(uri)
        try:
            info = self.fs.get_file_info(path)
        except Exception as exc:
            if uri.startswith("gs://") or uri.startswith("asset://"):
                raise FileNotFoundError(str(uri)) from exc
            p = Path(path)
            if not p.exists():
                raise FileNotFoundError(str(uri)) from exc
            st = p.stat()
            return FileInfo(name=uri, size=st.st_size, mtime=st.st_mtime,
                            type="dir" if p.is_dir() else "file")
        if info.type == pa_fs.FileType.NotFound:
            raise FileNotFoundError(str(uri))
        return FileInfo(
            name=uri,
            size=info.size,
            mtime=info.mtime_ns / 1e9 if info.mtime_ns is not None else None,
            type="dir" if info.type == pa_fs.FileType.Directory else "file",
        )

    def listdir(self, uri: str) -> list[FileInfo]:
        path = self._arrow_path(uri)
        if uri.startswith("gs://") or uri.startswith("asset://"):
            selector = pa_fs.FileSelector(path.rstrip("/") + "/")
            infos = self.fs.get_file_info(selector)
            return [FileInfo(
                name=info.path or "",
                size=info.size,
                mtime=info.mtime_ns / 1e9 if info.mtime_ns is not None else None,
                type="dir" if info.type == pa_fs.FileType.Directory else "file",
            ) for info in infos]
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

    def copy(self, src: str, dst: str) -> None:
        src_path = self._arrow_path(src)
        dst_path = self._arrow_path(dst)
        if (src.startswith("gs://") or src.startswith("asset://")) and \
           (dst.startswith("gs://") or dst.startswith("asset://")):
            self.fs.copy_file(src_path, dst_path)
            return
        data = self.read(src)
        self.write(dst, data)

    def delete(self, uri: str) -> None:
        self.fs.delete_file(self._arrow_path(uri))

    def exists(self, uri: str) -> bool:
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

    # --- internal ---

    def _resolve_asset(self, uri: str) -> str | None:
        # asset://grace:<video_id> → backend resolve → signed URL
        # We return the signed URL directly and let Arrow fs handle it.
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
                url = result.get("url", "")
                if url:
                    # Arrow fs can't read signed URLs directly, so return
                    # the bucket/object path and use signed URL for open().
                    self._last_signed_url = url
                    return f"{result['bucket']}/{result['object']}"
            except Exception as exc:
                _logger.warning("resolve failed for %s: %s", uri, exc)
            return None

        # Fallback: legacy asset ID (mcap-based)
        asset_id = rest.strip("/")
        try:
            locator = self._request("GET", "asset_mcap_locator", asset_id=asset_id)
            mcap_id = locator.get("mcap_file_id")
            if mcap_id:
                info = self._request("GET", "storage_file_info", mcap_id=mcap_id)
                gcs_path = info.get("gcs_path") or info.get("storage_path", "")
                if gcs_path:
                    return gcs_path[5:] if gcs_path.startswith("gs://") else gcs_path
        except Exception as exc:
            _logger.warning("failed to resolve asset %s: %s", asset_id, exc)
        return None
