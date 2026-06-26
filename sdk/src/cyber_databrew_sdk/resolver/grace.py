"""Grace API resolver — resolves video IDs to GCS paths via Grace REST API.

Credentials (from environment, or mounted Secret Manager in GKE)::

    CYBER_DATABREW_GRACE_URL     https://dev.cyber-grace.pages.dev/api
    CYBER_DATABREW_GRACE_USERNAME  grace-service-dev
    CYBER_DATABREW_GRACE_PASSWORD  <password>
"""

from __future__ import annotations

import base64
import logging
import os
from dataclasses import dataclass

import httpx

_logger = logging.getLogger(__name__)

# Default endpoints
# API via nexus backend (no /api prefix — the user sets GRACE_URL directly)
_DEV_URL = "https://nexus.cyberorigin.ai"
_PROD_URL = "https://nexus.cyberorigin.ai"


@dataclass
class GraceConfig:
    """Grace API connection configuration."""
    url: str
    username: str
    password: str

    @property
    def auth_header(self) -> str:
        raw = f"{self.username}:{self.password}"
        encoded = base64.b64encode(raw.encode()).decode()
        return f"Basic {encoded}"


class GraceResolver:
    """Resolve Grace video IDs to GCS paths.

    Usage::

        resolver = GraceResolver(env="dev")
        gcs_uri = resolver.resolve("019ed9c5-...", sub_path="raw")
        # → "gs://bucket/path/file.mcap"
    """

    def __init__(
        self,
        env: str = "dev",
        url: str | None = None,
        username: str | None = None,
        password: str | None = None,
    ) -> None:
        # Prefer explicit params, fallback to env vars, then defaults
        prefix = "CYBER_DATABREW_GRACE"
        self._cfg = GraceConfig(
            url=url or os.environ.get(f"{prefix}_URL") or (_DEV_URL if env == "dev" else _PROD_URL),
            username=username or os.environ.get(f"{prefix}_USERNAME") or "",
            password=password or os.environ.get(f"{prefix}_PASSWORD") or "",
        )
        self._client = httpx.Client(timeout=30)
        self._headers = {"Authorization": self._cfg.auth_header}

    def resolve(self, video_id: str, sub_path: str = "raw") -> str:
        """Resolve a Grace video ID to a GCS URI.

        Args:
            video_id: Grace video UUID (e.g. "019ed9c5-...")
            sub_path: "raw" for raw video, "algo_input" for algo input

        Returns:
            ``gs://bucket/path/file``

        Raises:
            FileNotFoundError: If the video ID is not found.
            PermissionError: If credentials are invalid.
        """
        if sub_path == "algo_input":
            return self._resolve_algo_input(video_id)
        return self._resolve_raw(video_id)

    def _resolve_raw(self, video_id: str) -> str:
        """GET /grace/videos/{id}/raw-gcs-uri → gcs_uri."""
        url = f"{self._cfg.url.rstrip('/')}/grace/videos/{video_id}/raw-gcs-uri"
        resp = self._client.get(url, headers=self._headers)
        if resp.status_code == 404:
            raise FileNotFoundError(f"grace video not found: {video_id}")
        if resp.status_code == 403:
            raise PermissionError("grace credentials invalid or insufficient")
        resp.raise_for_status()
        data = resp.json()
        gcs_uri = data.get("gcs_uri") or data.get("uri", "")
        if not gcs_uri:
            raise ValueError(f"no gcs_uri in Grace response: {data}")
        return gcs_uri

    def _resolve_algo_input(self, video_id: str) -> str:
        """GET /grace/videos/{id}/algo-input → uri."""
        url = f"{self._cfg.url.rstrip('/')}/grace/videos/{video_id}/algo-input"
        resp = self._client.get(url, headers=self._headers)
        if resp.status_code == 404:
            raise FileNotFoundError(f"grace algo-input not found: {video_id}")
        resp.raise_for_status()
        data = resp.json()
        uri = data.get("uri", "")
        if not uri:
            raise ValueError(f"no uri in Grace algo-input response: {data}")
        return uri

    def close(self) -> None:
        self._client.close()
