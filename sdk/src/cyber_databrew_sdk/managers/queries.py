"""QueryManager — validate, run, and manage saved queries."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class QueryManager(BaseManager):
    """Query execution and saved query management."""

    # ------------------------------------------------------------------
    # Execution
    # ------------------------------------------------------------------

    def validate(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Validate a query without executing it."""
        return self._request("POST", self._endpoint("query_validate"), json_body=payload)

    def run(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Execute a query and return results."""
        return self._request("POST", self._endpoint("query_run"), json_body=payload)

    # ------------------------------------------------------------------
    # Saved queries CRUD
    # ------------------------------------------------------------------

    def list_saved(self, *, page: int = 1, page_size: int = 20) -> dict[str, Any]:
        """List saved queries."""
        return self._request(
            "GET",
            self._endpoint("query_saved_list"),
            params={"page": page, "page_size": page_size},
        )

    def get_saved(self, query_id: str) -> dict[str, Any]:
        """Get a saved query by ID."""
        return self._request("GET", self._endpoint("query_saved_get", query_id=query_id))

    def create_saved(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Create a saved query."""
        return self._request("POST", self._endpoint("query_saved_create"), json_body=payload)

    def update_saved(self, query_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Update a saved query."""
        return self._request("PATCH", self._endpoint("query_saved_update", query_id=query_id), json_body=payload)

    def delete_saved(self, query_id: str) -> dict[str, Any]:
        """Delete a saved query."""
        return self._request("DELETE", self._endpoint("query_saved_delete", query_id=query_id))
