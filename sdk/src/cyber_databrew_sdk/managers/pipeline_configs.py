"""PipelineConfigManager — standalone pipeline config CRUD and versions."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class PipelineConfigManager(BaseManager):
    """Manage standalone user-owned pipeline config files."""

    def list(
        self,
        *,
        q: str | None = None,
        owner: str | None = None,
        scope: str | None = None,
        lifecycle: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if q:
            params["q"] = q
        if owner:
            params["owner"] = owner
        if scope:
            params["scope"] = scope
        if lifecycle:
            params["lifecycle"] = lifecycle
        return self._request("GET", self._endpoint("pipeline_config_list"), params=params)

    def get(self, config_id: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("pipeline_config_get", config_id=config_id),
        )

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pipeline_config_create"),
            json_body=payload,
        )

    def update(self, config_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "PUT",
            self._endpoint("pipeline_config_update", config_id=config_id),
            json_body=payload,
        )

    def create_version(self, config_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pipeline_config_version_create", config_id=config_id),
            json_body=payload,
        )

    def get_version(self, config_id: str, version: int) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint(
                "pipeline_config_version_get",
                config_id=config_id,
                version=str(version),
            ),
        )

    def deprecate(self, config_id: str) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pipeline_config_deprecate", config_id=config_id),
            json_body={},
        )
