"""ActionClient: seg-internal time-bounded annotations (mcap → seg → action).

The action API is rooted at the parent seg: every endpoint takes the seg
asset id in the path. See backend/internal/handlers/action and
docs/review/api-guide.md §2.7.
"""

from __future__ import annotations

from typing import Optional

import httpx

from asset_sdk.types.action import Action, ActionCreate, ActionList


class ActionClient:
    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def _base(self, asset_id: str) -> str:
        return f"/api/v1/assets/{asset_id}/actions"

    def create(self, asset_id: str, payload: ActionCreate) -> Action:
        r = self._http.post(self._base(asset_id), json=payload.model_dump(exclude_none=True))
        r.raise_for_status()
        return Action.model_validate(r.json())

    def list(
        self,
        asset_id: str,
        *,
        at: Optional[int] = None,
        from_ns: Optional[int] = None,
        to_ns: Optional[int] = None,
        label: Optional[str] = None,
        limit: int = 200,
    ) -> ActionList:
        """List actions for a seg.

        Filters mirror the backend:
          - ``at``: point-in-time ns; returns actions whose interval covers it.
          - ``from_ns`` / ``to_ns``: overlap window (ns).
          - ``label``: matches ``primary_label`` OR membership in ``labels[]``.
        """
        params: dict[str, object] = {"limit": limit}
        if at is not None:
            params["at"] = at
        if from_ns is not None:
            params["from"] = from_ns
        if to_ns is not None:
            params["to"] = to_ns
        if label:
            params["label"] = label
        r = self._http.get(self._base(asset_id), params=params)
        r.raise_for_status()
        return ActionList.model_validate(r.json())

    def at(self, asset_id: str, timestamp_ns: int) -> ActionList:
        """Convenience for point-in-time query (`?at=`)."""
        return self.list(asset_id, at=timestamp_ns)
