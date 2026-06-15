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

    def list_releases(
        self,
        *,
        q: str | None = None,
        component_id: str | None = None,
        task_name: str | None = None,
        status: str | None = None,
        channel: str | None = None,
        selectable: bool | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if q:
            params["q"] = q
        if component_id:
            params["componentId"] = component_id
        if task_name:
            params["taskName"] = task_name
        if status:
            params["status"] = status
        if channel:
            params["channel"] = channel
        if selectable is not None:
            params["selectable"] = str(selectable).lower()
        return self._request(
            "GET",
            self._endpoint("pipeline_component_release_list"),
            params=params,
        )

    def get_release(self, release_id: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("pipeline_component_release_get", release_id=release_id),
        )

    def sync_releases(self, items: list[dict[str, Any]]) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pipeline_component_release_sync"),
            json_body={"items": items},
        )
