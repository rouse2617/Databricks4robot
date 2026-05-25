"""Base class for all semantic managers.

Adapted from Stripe's StripeService — each manager receives
the shared APIRequestor and delegates HTTP calls through
self._request().

Key design decision: managers are stateless accessors (OpenAI/Stripe pattern),
not domain objects with cached state (Nucleus/GCS pattern).
"""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._requestor import APIRequestor
from cyber_databrew_sdk.config import ConfigManager


class BaseManager:
    """Every manager holds a reference to the shared APIRequestor + config."""

    _requestor: APIRequestor
    _cfg: ConfigManager

    def __init__(self, requestor: APIRequestor, config: ConfigManager) -> None:
        self._requestor = requestor
        self._cfg = config

    def _endpoint(self, name: str, **path_params: str) -> str:
        """Resolve an endpoint name to a URL path via the SDK config.

        Example::

            self._endpoint("asset_get", asset_id="abc")
            # → "/api/v1/assets/abc"
        """
        return self._cfg.resolve(name, **path_params)

    def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Thin wrapper — delegates to APIRequestor.

        Exists so subclasses can override for pre/post-processing
        (e.g. progress bars, auto-pagination, parameter flattening).
        """
        return self._requestor.request(method, path, params=params, json_body=json_body)
