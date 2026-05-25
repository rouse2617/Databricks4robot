"""Built-in ConfigSource implementations.

Each source reads configuration from a different origin.  They all
implement the ``ConfigSource`` Protocol so they are interchangeable
and testable independently.
"""

from __future__ import annotations

import logging
import os
from pathlib import Path
from typing import Any
from urllib.parse import urljoin

import httpx

from cyber_databrew_sdk.config.endpoints import ENDPOINTS

_logger = logging.getLogger(__name__)

_CONFIG_ENDPOINT = "/api/v1/sdk-config"


class DefaultSource:
    """Built-in endpoint defaults from :py:mod:`config.endpoints`."""

    source_name = "defaults"

    def load(self) -> dict[str, Any]:
        return {"endpoints": dict(ENDPOINTS)}


class EnvSource:
    """Read configuration from environment variables.

    Mappings::

        CYBER_DATABREW_BASE_URL   → base_url
        CYBER_DATABREW_TOKEN      → token (also GRACE_TOKEN)
        CYBER_DATABREW_EMAIL      → email
        CYBER_DATABREW_TIMEOUT    → timeout (float)
    """

    source_name = "env"

    def __init__(self, prefix: str = "CYBER_DATABREW") -> None:
        self._prefix = prefix

    def load(self) -> dict[str, Any]:
        data: dict[str, Any] = {}

        base_url = os.environ.get(f"{self._prefix}_BASE_URL")
        if base_url:
            data["base_url"] = base_url

        token = os.environ.get(f"{self._prefix}_TOKEN") or os.environ.get("GRACE_TOKEN")
        if token:
            data["token"] = token

        email = os.environ.get(f"{self._prefix}_EMAIL")
        if email:
            data["email"] = email

        timeout = os.environ.get(f"{self._prefix}_TIMEOUT")
        if timeout:
            try:
                data["timeout"] = float(timeout)
            except ValueError:
                _logger.warning("Invalid %s_TIMEOUT: %r", self._prefix, timeout)

        return data


class FileSource:
    """Read configuration from a YAML file.

    Searches these locations in order (first found wins)::

        1. ``path`` argument (explicit)
        2. ``./.cyber-databrew.yaml``
        3. ``~/.cyber-databrew/config.yaml``
    """

    source_name = "file"

    def __init__(self, path: str | Path | None = None) -> None:
        self._path = Path(path) if path else self._find()

    @staticmethod
    def _find() -> Path | None:
        for candidate in (
            Path.cwd() / ".cyber-databrew.yaml",
            Path.home() / ".cyber-databrew" / "config.yaml",
        ):
            if candidate.is_file():
                return candidate
        return None

    def load(self) -> dict[str, Any]:
        if self._path is None or not self._path.is_file():
            return {}
        try:
            import yaml
        except ImportError:
            _logger.warning(
                "PyYAML not installed — skipping config file %s. "
                "Install with: uv add pyyaml",
                self._path,
            )
            return {}
        try:
            with open(self._path) as f:
                data: dict[str, Any] = yaml.safe_load(f) or {}
            _logger.debug("Loaded config from %s", self._path)
            return data
        except Exception as exc:  # noqa: BLE001
            _logger.warning("Failed to load config file %s: %s", self._path, exc)
            return {}


class RemoteSource:
    """Fetch configuration from the backend ``/api/v1/sdk-config`` endpoint.

    Requires bootstrap ``base_url`` and ``auth_headers`` for the HTTP call.
    If the backend is unreachable, returns empty dict (no crash).
    """

    source_name = "remote"

    def __init__(
        self,
        base_url: str,
        auth_headers: dict[str, str] | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._auth_headers = auth_headers or {}
        self._client = http_client

    def load(self) -> dict[str, Any]:
        client = self._client or httpx.Client(timeout=15.0)
        try:
            url = urljoin(self._base_url + "/", _CONFIG_ENDPOINT.lstrip("/"))
            resp = client.get(url, headers=self._auth_headers)
            resp.raise_for_status()
            data: dict[str, Any] = resp.json()
            _logger.info("Fetched remote SDK config from %s", url)
            return data
        except Exception as exc:  # noqa: BLE001
            _logger.debug("Remote config fetch failed (%s), using defaults", exc)
            return {}
        finally:
            if self._client is None:
                client.close()
