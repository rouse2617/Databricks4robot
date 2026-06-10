"""CustomerManager — customer CRUD."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class CustomerManager(BaseManager):
    """Manage delivery customers."""

    def list(
        self,
        *,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        """List customers."""
        return self._request(
            "GET",
            self._endpoint("customer_list"),
            params={"page": page, "page_size": page_size},
        )

    def get(self, customer_id: str) -> dict[str, Any]:
        """Get a customer by ID."""
        return self._request("GET", self._endpoint("customer_get", customer_id=customer_id))

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Create a customer."""
        return self._request("POST", self._endpoint("customer_create"), json_body=payload)

    def update(self, customer_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Update a customer."""
        return self._request("PATCH", self._endpoint("customer_update", customer_id=customer_id), json_body=payload)
