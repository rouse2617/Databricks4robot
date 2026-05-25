"""StorageManager — MCAP file download via backend proxy."""

from __future__ import annotations

from pathlib import Path
from typing import IO, Any

from cyber_databrew_sdk._base_manager import BaseManager


class StorageManager(BaseManager):
    """MCAP storage operations proxied through the backend.

    Never accesses GCS directly — all operations go through
    the backend for centralized auth and audit.
    """

    def list_files(self, *, page: int = 1, page_size: int = 20) -> dict[str, Any]:
        """List MCAP files."""
        return self._request(
            "GET",
            self._endpoint("storage_files_list"),
            params={"page": page, "page_size": page_size},
        )

    def get_file_info(self, mcap_id: str) -> dict[str, Any]:
        """Get metadata for a single MCAP file."""
        return self._request("GET", self._endpoint("storage_file_info", mcap_id=mcap_id))

    def download_mcap(self, mcap_file_id: str, output_path: str | Path) -> Path:
        """Download MCAP file by file id via backend-signed GCS URL.

        Calls the storage download endpoint and follows
        the 302 redirect to the signed GCS URL, streaming bytes to disk.

        Returns the resolved Path where the file was written.
        """
        output_path = Path(output_path)

        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {
            "Content-Type": "application/json",
            **self._requestor._auth_headers,
        }

        with self._requestor._client.stream(
            "GET", url, headers=headers, follow_redirects=True
        ) as resp:
            resp.raise_for_status()
            output_path.parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, "wb") as f:
                for chunk in resp.iter_bytes(chunk_size=8192):
                    f.write(chunk)

        return output_path

    def download_asset_mcap(self, asset_id: str, output_path: str | Path) -> Path:
        """Resolve asset MCAP locator, then download the referenced MCAP file.

        Two-step: resolve asset_mcap_locator endpoint → extract
        mcap_file_id → download_mcap().
        """
        locator = self._request(
            "GET", self._endpoint("asset_mcap_locator", asset_id=asset_id)
        )
        mcap_file_id = locator["mcap_file_id"]
        return self.download_mcap(mcap_file_id, output_path)

    def open_mcap(self, mcap_file_id: str) -> IO[bytes]:
        """Open an MCAP file as a readable byte stream.

        For large files, prefer download_mcap() with streaming
        to avoid loading everything into memory.
        """
        import io

        url = self._cfg.build_url("storage_mcap_download", mcap_file_id=mcap_file_id)
        headers = {
            "Content-Type": "application/json",
            **self._requestor._auth_headers,
        }
        resp = self._requestor._client.get(url, headers=headers, follow_redirects=True)
        resp.raise_for_status()
        return io.BytesIO(resp.content)

    def finalize_upload(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Finalize a multipart upload."""
        return self._request("POST", self._endpoint("storage_upload_finalize"), json_body=payload)

    def get_messages(self, mcap_id: str) -> dict[str, Any]:
        """Get messages from an MCAP file."""
        return self._request("GET", self._endpoint("storage_mcap_messages", mcap_id=mcap_id))
