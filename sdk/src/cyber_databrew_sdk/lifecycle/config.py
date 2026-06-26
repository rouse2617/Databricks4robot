"""Dynamic runtime configuration — remote and local config sources.

Supports:
- Remote config from backend (``GET /api/v1/sdk-config``)
- Local config file
- Environment variable overrides
"""

from __future__ import annotations

import logging
import os
from typing import Any

_logger = logging.getLogger(__name__)


class DynamicConfig:
    """Hierarchical dynamic configuration with fallback.

    Config sources (highest priority first)::

        1. Environment variable ``SDK_<KEY>``
        2. Remote backend config (sdk-config endpoint)
        3. Local config file
        4. Hardcoded default
    """

    def __init__(self) -> None:
        self._remote: dict[str, Any] = {}
        self._local: dict[str, Any] = {}

    def get(self, key: str, default: Any = None) -> Any:
        """Get a config value by dot-separated key (e.g. ``storage.threshold``)."""
        env_key = f"SDK_{key.upper().replace('.', '_')}"
        if env_val := os.environ.get(env_key):
            return env_val

        if key in self._remote:
            return self._remote[key]
        if key in self._local:
            return self._local[key]
        return default

    def refresh(self) -> None:
        """Pull latest config from backend."""
        # TODO: implement remote config fetch
        _logger.debug("dynamic config refresh skipped — not yet implemented")
