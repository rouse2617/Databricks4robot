"""RegistryManager — algo, tag, metric, and lifecycle registries."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class RegistryManager(BaseManager):
    """Read-only registries for algos, tags, metrics, and lifecycle states."""

    def list_algos(self) -> dict[str, Any]:
        """List all registered algorithms."""
        return self._request("GET", self._endpoint("registry_algos"))

    def list_tags(self) -> dict[str, Any]:
        """List all registered tags."""
        return self._request("GET", self._endpoint("registry_tags"))

    def list_metrics(self) -> dict[str, Any]:
        """List all registered metrics."""
        return self._request("GET", self._endpoint("registry_metrics"))

    def list_action_labels(self) -> dict[str, Any]:
        """List all action labels."""
        return self._request("GET", self._endpoint("registry_action_labels"))

    def list_lifecycle_states(self) -> dict[str, Any]:
        """List all lifecycle states."""
        return self._request("GET", self._endpoint("registry_lifecycle_states"))
