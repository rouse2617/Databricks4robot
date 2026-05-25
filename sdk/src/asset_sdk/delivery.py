"""DeliveryClient: commit and inspect deliveries."""

from __future__ import annotations

from typing import Optional
from uuid import uuid4

import httpx


class DeliveryClient:
    _base = "/api/v1/deliveries"

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def commit(
        self,
        asset_ids: list[str],
        customer_id: str,
        *,
        note: Optional[str] = None,
        idempotency_key: Optional[str] = None,
    ) -> dict:
        """Create a delivery record for a set of assets.

        The backend requires an Idempotency-Key header for this endpoint.
        If not provided, a UUID is generated automatically.
        """
        key = idempotency_key or str(uuid4())
        r = self._http.post(
            self._base,
            json={"asset_ids": asset_ids, "customer_id": customer_id, "note": note},
            headers={"Idempotency-Key": key},
        )
        r.raise_for_status()
        return r.json()

    def get(self, delivery_id: str) -> dict:
        r = self._http.get(f"{self._base}/{delivery_id}")
        r.raise_for_status()
        return r.json()

    def list_for_customer(self, customer_id: str, *, page: int = 1, page_size: int = 20) -> dict:
        r = self._http.get(
            f"/api/v1/customers/{customer_id}/deliveries",
            params={"page": page, "page_size": page_size},
        )
        r.raise_for_status()
        return r.json()
