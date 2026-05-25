"""AssetManager — asset CRUD, tags, favorites, lineage, provenance, timeline."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class AssetManager(BaseManager):
    """Manage cyber-databrew assets."""

    # ------------------------------------------------------------------
    # CRUD
    # ------------------------------------------------------------------

    def get(self, asset_id: str) -> dict[str, Any]:
        """Retrieve a single asset by ID."""
        return self._request("GET", self._endpoint("asset_get", asset_id=asset_id))

    def list_all(
        self,
        *,
        status: str | None = None,
        tag: str | None = None,
        mcap_file_id: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """List assets with optional filters."""
        return self._request(
            "GET",
            self._endpoint("asset_list"),
            params={
                "status": status,
                "tag": tag,
                "mcap_file_id": mcap_file_id,
                "page": page,
                "page_size": page_size,
            },
        )

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Create a new asset."""
        return self._request("POST", self._endpoint("asset_create"), json_body=payload)

    def update(self, asset_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Update an existing asset."""
        return self._request("PATCH", self._endpoint("asset_update", asset_id=asset_id), json_body=payload)

    def delete(self, asset_id: str) -> dict[str, Any]:
        """Delete an asset."""
        return self._request("DELETE", self._endpoint("asset_delete", asset_id=asset_id))

    def batch_get(self, asset_ids: list[str]) -> dict[str, Any]:
        """Retrieve multiple assets by ID."""
        return self._request("POST", self._endpoint("asset_batch_get"), json_body={"ids": asset_ids})

    # ------------------------------------------------------------------
    # Tags
    # ------------------------------------------------------------------

    def get_tags(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_tags_list", asset_id=asset_id))

    def set_tags(self, asset_id: str, tags: list[dict[str, Any]]) -> dict[str, Any]:
        return self._request("POST", self._endpoint("asset_tags_set", asset_id=asset_id), json_body={"tags": tags})

    def get_tag(self, asset_id: str, key: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_tags_get", asset_id=asset_id, key=key))

    def delete_tag(self, asset_id: str, key: str) -> dict[str, Any]:
        return self._request("DELETE", self._endpoint("asset_tags_delete", asset_id=asset_id, key=key))

    def get_tag_history(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_tags_history", asset_id=asset_id))

    # ------------------------------------------------------------------
    # Lineage / Provenance / Timeline
    # ------------------------------------------------------------------

    def get_lineage(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_lineage", asset_id=asset_id))

    def get_provenance(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_provenance", asset_id=asset_id))

    def get_timeline(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_timeline", asset_id=asset_id))

    # ------------------------------------------------------------------
    # Misc
    # ------------------------------------------------------------------

    def get_mcap_locator(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_mcap_locator", asset_id=asset_id))

    def get_foxglove_source(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_foxglove_source", asset_id=asset_id))

    def get_deliveries(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_deliveries", asset_id=asset_id))

    def get_events(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_events", asset_id=asset_id))

    def record_view(self, asset_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("asset_record_view", asset_id=asset_id))

    def set_favorite(self, asset_id: str, favorite: bool = True) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("asset_set_favorite", asset_id=asset_id),
            json_body={"favorite": favorite},
        )

    def get_revisions(self, asset_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("asset_revisions", asset_id=asset_id))
