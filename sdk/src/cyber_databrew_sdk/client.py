"""Top-level client entry point.

Adapted from Stripe's StripeClient — central facade that creates
a shared APIRequestor and lazily instantiates managers via
__getattr__ (like Stripe's V1Services pattern).

Configuration is loaded dynamically via the :py:mod:`config` module —
the SDK merges env vars, config files, and remote discovery
(``GET /api/v1/sdk-config``) with built-in defaults.
"""

from __future__ import annotations

import logging
import os
from importlib import import_module
from typing import Any

import httpx

from cyber_databrew_sdk._requestor import APIRequestor
from cyber_databrew_sdk.auth import AuthProvider, CompositeAuth, DatabrewTokenAuth, EmailAuth
from cyber_databrew_sdk.config import ConfigManager

_logger = logging.getLogger(__name__)

# Map attribute name → [module_path, class_name]
# Adapted from Stripe's _subservices pattern in V1Services.
# Each manager is lazily imported on first access.
_managers: dict[str, tuple[str, str]] = {
    "assets": ("cyber_databrew_sdk.managers.assets", "AssetManager"),
    "storage": ("cyber_databrew_sdk.managers.storage", "StorageManager"),
    "delivery": ("cyber_databrew_sdk.managers.delivery", "DeliveryManager"),
    "algo_runs": ("cyber_databrew_sdk.managers.algo_runs", "AlgoRunManager"),
    "search": ("cyber_databrew_sdk.managers.search", "SearchManager"),
    "queries": ("cyber_databrew_sdk.managers.queries", "QueryManager"),
    "customers": ("cyber_databrew_sdk.managers.customers", "CustomerManager"),
    "lakehouse": ("cyber_databrew_sdk.managers.lakehouse", "LakehouseManager"),
    "events": ("cyber_databrew_sdk.managers.events", "EventManager"),
    "registry": ("cyber_databrew_sdk.managers.registry", "RegistryManager"),
    "audit": ("cyber_databrew_sdk.managers.audit", "AuditManager"),
    "actions": ("cyber_databrew_sdk.managers.actions", "ActionManager"),
    "eval_metrics": ("cyber_databrew_sdk.managers.eval_metrics", "EvalMetricsManager"),
    "admin_search": ("cyber_databrew_sdk.managers.admin_search", "AdminSearchManager"),
}


class CyberDatabrewClient:
    """Entry point for the cyber-databrew Python SDK.

    Usage::

        from cyber_databrew_sdk import CyberDatabrewClient

        client = CyberDatabrewClient(email="alice@company.com")
        asset = client.assets.get("abc12345")

    Configuration is loaded automatically from env vars, config files,
    and remote discovery.  Pass a pre-built :class:`ConfigManager` to
    fully control the config::

        from cyber_databrew_sdk.config import ConfigManager

        config = ConfigManager.load(base_url="https://my-host")
        client = CyberDatabrewClient(config=config)

    All 11 managers are lazily loaded on first access.
    """

    _requestor: APIRequestor
    _config: ConfigManager
    _closed: bool

    def __init__(
        self,
        token: str | None = None,
        email: str | None = None,
        *,
        auth: AuthProvider | None = None,
        base_url: str | None = None,
        timeout: float | None = None,
        http_client: httpx.Client | None = None,
        config: ConfigManager | None = None,
        enable_tracing: bool = True,
    ) -> None:
        """Initialize the client.

        Args:
            token: Databrew token for X-Databrew-Token header.
                   Defaults to CYBER_DATABREW_TOKEN, then DATABREW_TOKEN env var.
            email: User email for X-User-Email header (audit-only).
                   Defaults to CYBER_DATABREW_EMAIL env var.
            auth: Custom AuthProvider (overrides token+email default).
            base_url: API base URL. Defaults to CYBER_DATABREW_BASE_URL
                      or http://localhost:8080.
            timeout: Request timeout in seconds.  Defaults to 30.0.
            http_client: Inject custom httpx.Client (for testing).
            config: Pre-built ConfigManager.  When provided, ``base_url``,
                    ``timeout``, ``token``, ``email`` are ignored in favor
                    of the config's values (except ``auth`` still applies).
        """
        self._closed = False

        # 1. Resolve config
        if config is not None:
            self._config = config
        else:
            # Build auth headers for remote config fetch
            if auth is None:
                auth = CompositeAuth(DatabrewTokenAuth(token), EmailAuth(email))
            auth_headers = auth.get_headers()

            self._config = ConfigManager.load(
                base_url=base_url,
                token=token,
                email=email,
                timeout=timeout,
                auth_headers=auth_headers,
                http_client=http_client,
            )

        # 2. Resolve auth (config may override token/email, but explicit auth wins)
        if auth is not None:
            auth_headers = auth.get_headers()
        else:
            auth_headers = {}
            t = token or os.environ.get("CYBER_DATABREW_TOKEN") or os.environ.get("DATABREW_TOKEN")
            e = email or os.environ.get("CYBER_DATABREW_EMAIL")
            if t:
                auth_headers["X-Databrew-Token"] = t
            if e:
                auth_headers["X-User-Email"] = e

        # 3. Create shared requestor
        self._requestor = APIRequestor(
            base_url=self._config.base_url,
            auth_headers=auth_headers,
            timeout=self._config.timeout,
            http_client=http_client,
            enable_tracing=enable_tracing,
        )

    # ------------------------------------------------------------------
    # Lazy manager access — Stripe's __getattr__ + _subservices pattern
    # ------------------------------------------------------------------

    def __getattr__(self, name: str) -> Any:
        """Lazily import and instantiate manager on first access."""
        try:
            import_from, service_class = _managers[name]
        except KeyError as err:
            raise AttributeError(
                f"{self.__class__.__name__!r} object has no attribute {name!r}"
            ) from err

        module = import_module(import_from)
        manager_cls = getattr(module, service_class)
        instance = manager_cls(self._requestor, self._config)
        setattr(self, name, instance)
        return instance

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    def close(self) -> None:
        """Close the underlying HTTP client."""
        if not self._closed:
            self._requestor.close()
            self._closed = True

    def __enter__(self) -> CyberDatabrewClient:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def __del__(self) -> None:
        self.close()


# Alias for friendly import
CyberDatabrew = CyberDatabrewClient
