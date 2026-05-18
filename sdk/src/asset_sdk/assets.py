"""AssetClient: CRUD operations for assets."""

from __future__ import annotations

from typing import Optional

import httpx

from asset_sdk.types.asset import Asset, AssetCreate, AssetList, AssetUpdate


class AssetClient:
    _base = "/api/v1/assets"

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def get(self, asset_id: str) -> Asset:
        r = self._http.get(f"{self._base}/{asset_id}")
        r.raise_for_status()
        return Asset.model_validate(r.json())

    def list(
        self,
        *,
        mcap_file_id: Optional[str] = None,
        status: Optional[str] = None,
        tag: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> AssetList:
        params: dict[str, object] = {"page": page, "page_size": page_size}
        if mcap_file_id:
            params["mcap_file_id"] = mcap_file_id
        if status:
            params["status"] = status
        if tag:
            params["tag"] = tag
        r = self._http.get(self._base, params=params)
        r.raise_for_status()
        return AssetList.model_validate(r.json())

    def create(self, payload: AssetCreate) -> Asset:
        r = self._http.post(self._base, json=payload.model_dump())
        r.raise_for_status()
        return Asset.model_validate(r.json())

    def update(self, asset_id: str, payload: AssetUpdate) -> Asset:
        r = self._http.patch(f"{self._base}/{asset_id}", json=payload.model_dump(exclude_none=True))
        r.raise_for_status()
        return Asset.model_validate(r.json())

    def delete(self, asset_id: str) -> None:
        r = self._http.delete(f"{self._base}/{asset_id}")
        r.raise_for_status()
