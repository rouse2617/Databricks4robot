"""DeliveryManager — delivery lifecycle (create, commit, cancel, ack, draft)."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class DeliveryManager(BaseManager):
    """Manage delivery workflows."""

    # ------------------------------------------------------------------
    # Delivery CRUD
    # ------------------------------------------------------------------

    def get(self, delivery_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("delivery_get", delivery_id=delivery_id))

    def list(
        self,
        *,
        status: str | None = None,
        customer_id: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("delivery_list"),
            params={
                "status": status,
                "customer_id": customer_id,
                "page": page,
                "page_size": page_size,
            },
        )

    # ------------------------------------------------------------------
    # Single-step create + commit (legacy / C1)
    # ------------------------------------------------------------------

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_create"), json_body=payload)

    # ------------------------------------------------------------------
    # Two-step draft → commit (C2)
    # ------------------------------------------------------------------

    def draft(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_draft"), json_body=payload)

    def commit(self, delivery_id: str, payload: dict[str, Any] | None = None) -> dict[str, Any]:
        return self._request(
            "POST", self._endpoint("delivery_commit", delivery_id=delivery_id), json_body=payload
        )

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    def cancel(self, delivery_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_cancel", delivery_id=delivery_id))

    def retry(self, delivery_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_retry", delivery_id=delivery_id))

    def ack(self, delivery_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_ack", delivery_id=delivery_id))

    # ------------------------------------------------------------------
    # Items
    # ------------------------------------------------------------------

    def get_items(self, delivery_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("delivery_items_list", delivery_id=delivery_id))

    def add_items(self, delivery_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "POST", self._endpoint("delivery_items_add", delivery_id=delivery_id), json_body=payload
        )

    # ------------------------------------------------------------------
    # Customer-scoped
    # ------------------------------------------------------------------

    def list_for_customer(
        self,
        customer_id: str,
        *,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("delivery_customer_list", customer_id=customer_id),
            params={"page": page, "page_size": page_size},
        )

    # ------------------------------------------------------------------
    # Delivery rules (pre-delivery compliance gates)
    # ------------------------------------------------------------------

    def list_rules(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("delivery_rules_list"))

    def create_rule(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", self._endpoint("delivery_rules_create"), json_body=payload)
