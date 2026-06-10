"""ActionManager — child asset annotations (actions)."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class ActionManager(BaseManager):
    """Manage action annotations on assets."""

    def list(self, asset_id: str, **params: Any) -> dict[str, Any]:
        """List actions for an asset."""
        return self._request(
            "GET", self._endpoint("action_list", asset_id=asset_id), params=params
        )

    def create(self, asset_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Create an action on an asset."""
        return self._request(
            "POST", self._endpoint("action_create", asset_id=asset_id), json_body=payload
        )

    def update(self, asset_id: str, action_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Update an action by ID."""
        return self._request(
            "PATCH",
            self._endpoint("action_update", asset_id=asset_id, action_id=action_id),
            json_body=payload,
        )

    def delete(self, asset_id: str, action_id: str) -> dict[str, Any]:
        """Delete an action by ID."""
        return self._request(
            "DELETE",
            self._endpoint("action_delete", asset_id=asset_id, action_id=action_id),
        )
