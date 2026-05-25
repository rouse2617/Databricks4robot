"""Configuration management for cyber-databrew SDK.

Provides multi-source configuration merging with a strict priority order:

    1. Explicit constructor arguments
    2. Environment variables (``CYBER_DATABREW_*``)
    3. Config file (``~/.cyber-databrew/config.yaml``)
    4. Remote discovery (``GET /api/v1/sdk-config``)
    5. Built-in defaults (:py:mod:`config.endpoints`)

Usage::

    from cyber_databrew_sdk.config import ConfigManager

    config = ConfigManager.load(base_url="https://api.example.com")
    path = config.resolve("asset_get", asset_id="abc")
"""

from cyber_databrew_sdk.config.contract import ConfigSource
from cyber_databrew_sdk.config.endpoints import ENDPOINTS
from cyber_databrew_sdk.config.manager import ConfigManager
from cyber_databrew_sdk.config.sources import DefaultSource, EnvSource, FileSource, RemoteSource

__all__ = [
    "ENDPOINTS",
    "ConfigManager",
    "ConfigSource",
    "DefaultSource",
    "EnvSource",
    "FileSource",
    "RemoteSource",
]
