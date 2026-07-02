"""SearchManager — asset search plus search sync status and progress."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class SearchManager(BaseManager):
    """Elasticsearch-backed asset search and sync status."""

    def assets(
        self,
        *,
        q: str | None = None,
        mode: str | None = None,
        filter: list[str] | None = None,  # noqa: A002 - public SDK keyword
        page: int | None = None,
        page_size: int | None = None,
        lineage_with: str | None = None,
        lineage_direction: str | None = None,
        lineage_depth: int | None = None,
        relation_types: list[str] | None = None,
    ) -> dict[str, Any]:
        """Search assets with optional lineage filtering."""
        params: dict[str, Any] = {
            "q": q,
            "mode": mode,
            "page": page,
            "page_size": page_size,
            "lineage_with": lineage_with,
            "lineage_direction": lineage_direction,
            "lineage_depth": lineage_depth,
        }
        if filter:
            params["filter"] = filter
        if relation_types:
            params["relation_types"] = ",".join(relation_types)
        return self._request("GET", self._endpoint("search_assets"), params=params)

    def get_sync_status(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("search_sync_status"))

    def get_sync_progress(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("search_sync_progress"))
