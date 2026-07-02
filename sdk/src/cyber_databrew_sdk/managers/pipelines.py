"""PipelineManager — pipeline run and execution target operations."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class PipelineManager(BaseManager):
    """Manage pipeline execution targets and template runs."""

    def list_execution_targets(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_execution_targets"))

    def list_runtime_mounts(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_runtime_mounts"))

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

    def create_run(
        self,
        pipeline: dict[str, Any],
        *,
        asset_ids: list[str] | None = None,
        target_id: str | None = None,
        name: str | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"pipeline": pipeline}
        if asset_ids is not None:
            payload["asset_ids"] = asset_ids
        if target_id is not None:
            payload["target_id"] = target_id
        if name is not None:
            payload["name"] = name
        return self._request(
            "POST",
            self._endpoint("pipeline_run_create"),
            json_body=payload,
        )

    def create_run_from_template(
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
            self._endpoint("pipeline_run_create_template", template_id=template_id),
            json_body=payload,
        )

    def list_runs(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_run_list"))

    def get_run(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_run_get", run_id=run_id))

    def get_run_watcher_status(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_run_watcher_status"))

    def list_run_events(
        self,
        run_id: str,
        *,
        limit: int | None = None,
        cursor: int | None = None,
        subject_type: str | None = None,
        event_type: str | None = None,
        status: str | None = None,
        q: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if limit is not None:
            params["limit"] = limit
        if cursor is not None:
            params["cursor"] = cursor
        if subject_type is not None:
            params["subjectType"] = subject_type
        if event_type is not None:
            params["eventType"] = event_type
        if status is not None:
            params["status"] = status
        if q is not None:
            params["q"] = q
        return self._request(
            "GET",
            self._endpoint("pipeline_run_events", run_id=run_id),
            params=params or None,
        )

    def list_run_asset_nodes(
        self,
        run_id: str,
        *,
        limit: int | None = None,
        asset_id: str | None = None,
        node_id: str | None = None,
        status: str | None = None,
        order_by: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if limit is not None:
            params["limit"] = limit
        if asset_id is not None:
            params["assetId"] = asset_id
        if node_id is not None:
            params["nodeId"] = node_id
        if status is not None:
            params["status"] = status
        if order_by is not None:
            params["orderBy"] = order_by
        return self._request(
            "GET",
            self._endpoint("pipeline_run_asset_nodes", run_id=run_id),
            params=params or None,
        )

    def get_run_cost_summary(self, run_id: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("pipeline_run_cost_summary", run_id=run_id),
        )

    def create_pod_terminal_session(
        self,
        workflow_name: str,
        node_id: str,
        *,
        command: str = "sh",
        container_name: str | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"command": command}
        if container_name is not None:
            payload["containerName"] = container_name
        return self._request(
            "POST",
            self._endpoint(
                "workflow_terminal_create",
                workflow_name=workflow_name,
                node_id=node_id,
            ),
            json_body=payload,
        )

    def get_pod_terminal_session(self, session_id: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("pod_terminal_session_get", session_id=session_id),
        )

    def terminate_pod_terminal_session(self, session_id: str) -> dict[str, Any]:
        return self._request(
            "POST",
            self._endpoint("pod_terminal_session_terminate", session_id=session_id),
        )

    def pod_terminal_attach_path(self, session_id: str, token: str) -> str:
        return (
            self._endpoint("pod_terminal_session_attach", session_id=session_id)
            + f"?token={token}"
        )

    def retry_run(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("pipeline_run_retry", run_id=run_id))

    def stop_run(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("pipeline_run_stop", run_id=run_id))

    def delete_run(self, run_id: str) -> dict[str, Any]:
        return self._request("DELETE", self._endpoint("pipeline_run_delete", run_id=run_id))

    def list_deployments(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("pipeline_deployment_list"))
