"""WorkflowManager — Argo workflow monitoring and operations."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class WorkflowManager(BaseManager):
    """Manage Argo workflows."""

    def list(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("workflow_list"))

    def get(self, workflow_name: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("workflow_get", workflow_name=workflow_name))

    def logs(
        self,
        workflow_name: str,
        node_id: str,
        *,
        tail_lines: int | None = None,
        limit_bytes: int | None = None,
        cursor: str | None = None,
        container: str | None = None,
        since_seconds: int | None = None,
        since_time: str | None = None,
        previous: bool | None = None,
        timestamps: bool | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {"nodeId": node_id}
        optional = {
            "tailLines": tail_lines,
            "limitBytes": limit_bytes,
            "cursor": cursor,
            "container": container,
            "sinceSeconds": since_seconds,
            "sinceTime": since_time,
            "previous": previous,
            "timestamps": timestamps,
        }
        params.update({key: value for key, value in optional.items() if value is not None})
        return self._request(
            "GET",
            self._endpoint("workflow_logs", workflow_name=workflow_name),
            params=params,
        )

    def retry(self, workflow_name: str) -> dict[str, Any]:
        return self._operation("workflow_retry", workflow_name)

    def resubmit(self, workflow_name: str) -> dict[str, Any]:
        return self._operation("workflow_resubmit", workflow_name)

    def suspend(self, workflow_name: str) -> dict[str, Any]:
        return self._operation("workflow_suspend", workflow_name)

    def resume(self, workflow_name: str) -> dict[str, Any]:
        return self._operation("workflow_resume", workflow_name)

    def terminate(self, workflow_name: str) -> dict[str, Any]:
        return self._operation("workflow_terminate", workflow_name)

    def delete(self, workflow_name: str) -> dict[str, Any]:
        return self._request("DELETE", self._endpoint("workflow_delete", workflow_name=workflow_name))

    def _operation(self, endpoint: str, workflow_name: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint(endpoint, workflow_name=workflow_name))
