"""SDK version checker and hot update support.

Checks the backend for newer SDK versions and provides
update metadata to the caller.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass

_logger = logging.getLogger(__name__)


@dataclass
class UpdateInfo:
    """Information about an available SDK update."""
    version: str
    url: str | None = None
    changelog: str | None = None
    critical: bool = False


def check_update(current_version: str | None = None) -> UpdateInfo | None:
    """Check if a newer SDK version is available.

    Args:
        current_version: Current SDK version.  Auto-detected if omitted.

    Returns:
        UpdateInfo if an update is available, ``None`` otherwise.
    """
    # TODO: query backend for latest SDK version
    _logger.debug("update check skipped — not yet implemented")
    return None
