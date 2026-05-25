"""LakehouseManager — lakehouse health, metrics, and growth reports."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class LakehouseManager(BaseManager):
    """Lakehouse monitoring and reporting."""

    def get_report(self, *, time_from: str | None = None, time_to: str | None = None) -> dict[str, Any]:
        """Get lakehouse report."""
        return self._request(
            "GET",
            self._endpoint("lakehouse_report"),
            params={"time_from": time_from, "time_to": time_to},
        )

    def get_status(self) -> dict[str, Any]:
        """Get lakehouse system status."""
        return self._request("GET", self._endpoint("lakehouse_status"))

    def get_tables(self) -> dict[str, Any]:
        """List lakehouse tables."""
        return self._request("GET", self._endpoint("lakehouse_tables"))

    def get_overview(self) -> dict[str, Any]:
        """Get lakehouse overview."""
        return self._request("GET", self._endpoint("lakehouse_overview"))

    def get_event_daily(self) -> dict[str, Any]:
        """Get daily event counts."""
        return self._request("GET", self._endpoint("lakehouse_event_daily"))

    def get_event_type_share(self) -> dict[str, Any]:
        """Get event type distribution."""
        return self._request("GET", self._endpoint("lakehouse_event_type_share"))

    def get_asset_growth(self) -> dict[str, Any]:
        """Get asset growth metrics."""
        return self._request("GET", self._endpoint("lakehouse_asset_growth"))

    def get_sync_status(self) -> dict[str, Any]:
        """Get lakehouse sync status."""
        return self._request("GET", self._endpoint("lakehouse_sync_status"))

    def get_sync_progress(self) -> dict[str, Any]:
        """Get lakehouse sync progress."""
        return self._request("GET", self._endpoint("lakehouse_sync_progress"))

    def get_failure_clusters(self) -> dict[str, Any]:
        """Get failure cluster analysis."""
        return self._request("GET", self._endpoint("lakehouse_failure_clusters"))

    def get_quality_distribution(self) -> dict[str, Any]:
        """Get data quality distribution."""
        return self._request("GET", self._endpoint("lakehouse_quality_distribution"))

    def get_customer_replay(self) -> dict[str, Any]:
        """Get customer replay metrics."""
        return self._request("GET", self._endpoint("lakehouse_customer_replay"))
