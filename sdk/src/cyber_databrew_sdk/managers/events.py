"""EventManager — asset events and SSE stream."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class EventManager(BaseManager):
    """Asset and system events."""

    def list_global(
        self,
        *,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """List all events across the system."""
        return self._request(
            "GET",
            self._endpoint("event_list"),
            params={"page": page, "page_size": page_size},
        )

    def list_for_asset(
        self,
        asset_id: str,
        *,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """List events for a specific asset."""
        return self._request(
            "GET",
            self._endpoint("event_list_asset", asset_id=asset_id),
            params={"page": page, "page_size": page_size},
        )

    def stream_for_asset(
        self,
        asset_id: str,
    ) -> dict[str, Any]:
        """Open an SSE stream of events for a specific asset."""
        return self._request(
            "GET",
            self._endpoint("event_stream_asset", asset_id=asset_id),
        )
