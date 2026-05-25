"""ConfigSource Protocol — cross-language contract for config providers.

Any language implementing an SDK for cyber-databrew can implement this
Protocol (or its equivalent in that language) to provide configuration
from any source.
"""

from __future__ import annotations

from typing import Any, Protocol, runtime_checkable


@runtime_checkable
class ConfigSource(Protocol):
    """Interface for a configuration source.

    Each source implements ``load()`` and returns a dict with any of::

        {"base_url": str, "timeout": float,
         "endpoints": {"name": "path_template", ...}}

    The ``source_name`` attribute identifies the source for debugging.
    """

    source_name: str

    def load(self) -> dict[str, Any]:
        """Load configuration from this source.

        Returns:
            A dict of config values.  Can return partial config —
            the manager merges all sources, later sources override earlier ones.
        """
        ...
