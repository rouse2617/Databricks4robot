"""ConfigManager — unified configuration with multi-source merging.

Usage::

    from cyber_databrew_sdk.config import ConfigManager

    # Auto-load from all sources (env → file → remote → defaults)
    config = ConfigManager.load(base_url="http://localhost:8080")

    # Or with explicit overrides (highest priority)
    config = ConfigManager.load(
        base_url="https://api.example.com",
        token="g-xxx",
    )

    # Resolve endpoint paths
    path = config.resolve("asset_get", asset_id="abc")
    url = config.build_url("asset_get", asset_id="abc")
"""

from __future__ import annotations

import logging
import os
from typing import Any
from urllib.parse import urljoin

import httpx

from cyber_databrew_sdk.config.contract import ConfigSource
from cyber_databrew_sdk.config.endpoints import ENDPOINTS
from cyber_databrew_sdk.config.sources import DefaultSource, EnvSource, FileSource, RemoteSource

_logger = logging.getLogger(__name__)


class ConfigManager:
    """Central configuration manager.

    Loads configuration from multiple sources with a strict priority order
    (highest wins):

    1. Explicit constructor args (``token``, ``base_url``, ``endpoint_overrides``)
    2. Environment variables (``CYBER_DATABREW_*``)
    3. Config file (``~/.cyber-databrew/config.yaml``)
    4. Remote discovery (``GET /api/v1/sdk-config``)
    5. Built-in defaults (:py:mod:`config.endpoints`)

    Each source is an implementation of :py:class:`ConfigSource` — new
    sources can be added without changing the manager.
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        timeout: float = 30.0,
        endpoint_overrides: dict[str, str] | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._endpoint_overrides = endpoint_overrides or {}
        self._sources: list[ConfigSource] = []
        self._rebuild()

    def _rebuild(self) -> None:
        endpoints = dict(ENDPOINTS)
        endpoints.update(self._endpoint_overrides)
        self._endpoints = endpoints

    # ── Builder ───────────────────────────────────────────────────────

    @classmethod
    def load(
        cls,
        base_url: str | None = None,
        token: str | None = None,
        email: str | None = None,
        timeout: float | None = None,
        *,
        endpoint_overrides: dict[str, str] | None = None,
        auth_headers: dict[str, str] | None = None,
        http_client: httpx.Client | None = None,
    ) -> ConfigManager:
        """Build a ``ConfigManager`` by merging all standard sources.

        Args:
            base_url: API base URL (highest priority override).
            token: Grace token override.
            email: User email override.
            timeout: Request timeout override.
            endpoint_overrides: Individual endpoint path overrides.
            auth_headers: Auth headers for the remote config fetch.
                          If omitted, built from ``token`` + ``email``.
            http_client: Inject ``httpx.Client`` (for testing).
                         When provided, the remote fetch uses this client.
        """
        # 1. Collect explicit overrides (highest priority)
        overrides: dict[str, Any] = {}
        if base_url is not None:
            overrides["base_url"] = base_url
        if timeout is not None:
            overrides["timeout"] = timeout
        if endpoint_overrides:
            overrides["endpoints"] = endpoint_overrides

        # 2. Build auth headers from provided token/email
        if auth_headers is None:
            auth_headers = {}
            t = token
            e = email
            if t is None:
                t = os.environ.get("CYBER_DATABREW_TOKEN") or os.environ.get("GRACE_TOKEN")
            if e is None:
                e = os.environ.get("CYBER_DATABREW_EMAIL")
            if t:
                auth_headers["X-Grace-Token"] = t
            if e:
                auth_headers["X-User-Email"] = e

        # 3. Determine bootstrap base_url for remote source
        bootstrap_base = (
            base_url
            or os.environ.get("CYBER_DATABREW_BASE_URL")
            or "http://localhost:8080"
        )

        # 4. Assemble sources in priority order (lowest first, merged sequentially)
        sources: list[ConfigSource] = [
            DefaultSource(),
            RemoteSource(bootstrap_base, auth_headers, http_client=http_client),
            FileSource(),
            EnvSource(),
        ]

        # 5. Merge all sources
        merged: dict[str, Any] = {}
        for source in sources:
            data = source.load()
            _logger.debug("Config source %r → %s", source.source_name, data)
            merged.update(data)

        # 6. Explicit overrides win over everything
        merged.update(overrides)

        # 7. Build ConfigManager
        return cls(
            base_url=merged.get("base_url", "http://localhost:8080"),
            timeout=float(merged.get("timeout", 30.0)),
            endpoint_overrides=merged.get("endpoints") or {},
        )

    # ── Properties ────────────────────────────────────────────────────

    @property
    def base_url(self) -> str:
        return self._base_url

    @property
    def timeout(self) -> float:
        return self._timeout

    # ── Endpoint resolution ──────────────────────────────────────────

    def resolve(self, name: str, **path_params: str) -> str:
        """Resolve an endpoint name to a URL path.

        Args:
            name: Logical endpoint name (e.g. ``"asset_get"``).
            **path_params: Values for ``{param}`` placeholders.

        Returns:
            The resolved URL path (e.g. ``"/api/v1/assets/abc"``).

        If the endpoint name is unknown, formats it as ``/api/v1/{name}``
        as a fallback (with a warning).
        """
        template = self._endpoints.get(name)
        if template is None:
            _logger.warning("Unknown endpoint %r, falling back to /api/v1/%s", name, name)
            template = f"/api/v1/{name}"
        if path_params:
            return template.format(**path_params)
        return template

    def build_url(self, endpoint_name: str, **path_params: str) -> str:
        """Resolve an endpoint name to a full absolute URL."""
        path = self.resolve(endpoint_name, **path_params)
        return urljoin(self._base_url + "/", path.lstrip("/"))

    # ── Refresh ──────────────────────────────────────────────────────

    def refresh(
        self,
        base_url: str | None = None,
        auth_headers: dict[str, str] | None = None,
    ) -> None:
        """Re-fetch remote config and update endpoints.

        Call this after the backend has deployed new endpoints.
        """
        source = RemoteSource(
            base_url or self._base_url,
            auth_headers,
        )
        data = source.load()
        remote_eps = data.get("endpoints")
        if isinstance(remote_eps, dict):
            self._endpoint_overrides.update(remote_eps)
            self._rebuild()
            _logger.info("Config refreshed with %d remote endpoint overrides", len(remote_eps))

    # ── Internal ─────────────────────────────────────────────────────

    def _get_sources(self) -> list[ConfigSource]:
        """Return the current source list (for testing/inspection)."""
        return list(self._sources)
