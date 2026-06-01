"""PipelineManager — pipeline run and execution target operations."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class PipelineManager(BaseManager):
    """Manage pipeline execution targets and template runs."""

    def list_execution_targets(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_execution_targets"))

    def deploy_template(
        self,
        template_id: str,
        *,
        asset_ids: list[str] | None = None,
        target_id: str | None = None,
        name: str | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {}
        if asset_ids is not None:
            payload["asset_ids"] = asset_ids
        if target_id is not None:
            payload["target_id"] = target_id
        if name is not None:
            payload["name"] = name
        return self._request(
            "POST",
            self._endpoint("pipeline_deploy_template", template_id=template_id),
            json_body=payload,
        )

    def list_deployments(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_deployment_list"))
