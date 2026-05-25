"""SearchManager — search sync status and progress."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class SearchManager(BaseManager):
    """Elasticsearch sync status and progress."""

    def get_sync_status(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("search_sync_status"))

    def get_sync_progress(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("search_sync_progress"))
