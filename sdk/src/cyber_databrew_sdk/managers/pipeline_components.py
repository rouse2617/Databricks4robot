"""PipelineComponentManager — pipeline component registry CRUD."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class PipelineComponentManager(BaseManager):
    """Manage reusable pipeline components."""

    def list(
        self,
        *,
        q: str | None = None,
        source: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if q:
            params["q"] = q
        if source:
            params["source"] = source
        return self._request("GET", self._endpoint("pipeline_component_list"), params=params)

    def get(self, component_id: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("pipeline_component_get", component_id=component_id),
        )

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pipeline_component_create"),
            json_body=payload,
        )

    def update(self, component_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "PUT",
            self._endpoint("pipeline_component_update", component_id=component_id),
            json_body=payload,
        )

    def delete(self, component_id: str) -> dict[str, Any]:
        return self._request(
            "DELETE",
            self._endpoint("pipeline_component_delete", component_id=component_id),
        )
