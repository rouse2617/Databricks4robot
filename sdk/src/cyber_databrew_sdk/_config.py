"""Backward-compat re-export — moved to :py:mod:`config`.

Use ``from cyber_databrew_sdk.config import ConfigManager`` instead.
"""

from cyber_databrew_sdk.config.manager import ConfigManager as SDKConfig

__all__ = ["SDKConfig"]
