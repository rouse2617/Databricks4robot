"""AuditManager — operation audit search and lineage search."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class AuditManager(BaseManager):
    """Audit log search and lineage queries."""

    def search(
        self,
        *,
        actor: str | None = None,
        event_type: str | None = None,
        asset_id: str | None = None,
        run_id: str | None = None,
        time_from: str | None = None,
        time_to: str | None = None,
        limit: int = 50,
        cursor: str | None = None,
    ) -> dict[str, Any]:
        """Search audit logs with optional filters."""
        return self._request(
            "GET",
            self._endpoint("audit_search"),
            params={
                "actor": actor,
                "event_type": event_type,
                "asset_id": asset_id,
                "run_id": run_id,
                "time_from": time_from,
                "time_to": time_to,
                "limit": limit,
                "cursor": cursor,
            },
        )

    def lineage_search(
        self,
        asset_id: str,
        *,
        limit: int = 50,
        cursor: str | None = None,
    ) -> dict[str, Any]:
        """Search audit lineage for an asset."""
        return self._request(
            "GET",
            self._endpoint("audit_lineage_search"),
            params={
                "asset_id": asset_id,
                "limit": limit,
                "cursor": cursor,
            },
        )
