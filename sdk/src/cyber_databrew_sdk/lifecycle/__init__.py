"""Lifecycle — SDK version management, hot updates, and dynamic config.

Usage::

    from cyber_databrew_sdk.lifecycle import check_update, dynamic_config

    # Check if a newer SDK version is available
    update = check_update()
    if update:
        print(f"SDK {update.version} available")

    # Get runtime configuration (from backend or local file)
    cfg = dynamic_config.get("storage.transfer_threshold", 100)
"""

from cyber_databrew_sdk.lifecycle.updater import check_update, UpdateInfo
from cyber_databrew_sdk.lifecycle.config import DynamicConfig

__all__ = [
    "check_update",
    "UpdateInfo",
    "DynamicConfig",
]
