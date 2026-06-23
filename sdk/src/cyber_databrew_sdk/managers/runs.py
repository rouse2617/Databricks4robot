"""RunManager — product-facing DataBrew run operations."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class RunManager(BaseManager):
    """Manage DataBrew Runs.

    Runs are the product execution record. Runtime workflow names are treated as
    debug references and can still be resolved through ``get_by_workflow``.
    """

    def create(
        self,
        pipeline: dict[str, Any],
        *,
        asset_ids: list[str] | None = None,
        target_id: str | None = None,
        name: str | None = None,
        config_selection: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"pipeline": pipeline}
        if asset_ids is not None:
            payload["asset_ids"] = asset_ids
        if target_id is not None:
            payload["target_id"] = target_id
        if name is not None:
            payload["name"] = name
        if config_selection is not None:
            payload["configSelection"] = config_selection
        return self._request("POST", self._endpoint("run_create"), json_body=payload)

    def create_from_template(
        self,
        template_id: str,
        *,
        asset_ids: list[str] | None = None,
        target_id: str | None = None,
        name: str | None = None,
        version: int | None = None,
        config_selection: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {}
        if asset_ids is not None:
            payload["asset_ids"] = asset_ids
        if target_id is not None:
            payload["target_id"] = target_id
        if name is not None:
            payload["name"] = name
        if version is not None:
            payload["version"] = version
        if config_selection is not None:
            payload["configSelection"] = config_selection
        return self._request(
            "POST",
            self._endpoint("run_create_template", template_id=template_id),
            json_body=payload,
        )

    def list(
        self,
        *,
        view: str | None = None,
        exclude_batch: bool | None = None,
        batch_job_id: str | None = None,
        status: str | None = None,
        q: str | None = None,
        pipeline_node_id: str | None = None,
        node_status: str | None = None,
        page: int | None = None,
        page_size: int | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if view is not None:
            params["view"] = view
        if exclude_batch is not None:
            params["excludeBatch"] = "true" if exclude_batch else "false"
        if batch_job_id is not None:
            params["batchJobId"] = batch_job_id
        if status is not None:
            params["status"] = status
        if q is not None:
            params["q"] = q
        if pipeline_node_id is not None:
            params["pipelineNodeId"] = pipeline_node_id
        if node_status is not None:
            params["nodeStatus"] = node_status
        if page is not None:
            params["page"] = page
        if page_size is not None:
            params["pageSize"] = page_size
        return self._request("GET", self._endpoint("run_list"), params=params or None)

    def get(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_get", run_id=run_id))

    def get_by_workflow(self, workflow_name: str) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("run_by_workflow", workflow_name=workflow_name),
        )

    def watcher_status(self) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_watcher_status"))

    def events(
        self,
        run_id: str,
        *,
        limit: int | None = None,
        cursor: int | None = None,
        subject_type: str | None = None,
        event_type: str | None = None,
        status: str | None = None,
        q: str | None = None,
        from_: str | None = None,
        to: str | None = None,
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
        if from_ is not None:
            params["from"] = from_
        if to is not None:
            params["to"] = to
        return self._request(
            "GET",
            self._endpoint("run_events", run_id=run_id),
            params=params or None,
        )

    def nodes(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_nodes", run_id=run_id))

    def asset_nodes(
        self,
        run_id: str,
        *,
        limit: int | None = None,
        cursor: str | None = None,
        asset_id: str | None = None,
        node_id: str | None = None,
        status: str | None = None,
        order_by: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if limit is not None:
            params["limit"] = limit
        if cursor is not None:
            params["cursor"] = cursor
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
            self._endpoint("run_asset_nodes", run_id=run_id),
            params=params or None,
        )

    def cost_summary(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_cost_summary", run_id=run_id))

    def inputs(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_inputs", run_id=run_id))

    def outputs(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_outputs", run_id=run_id))

    def children(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_children", run_id=run_id))

    def runtime(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("run_runtime", run_id=run_id))

    def retry(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_retry", run_id=run_id))

    def resubmit(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_resubmit", run_id=run_id))

    def rerun(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_rerun", run_id=run_id))

    def stop(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_stop", run_id=run_id))

    def suspend(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_suspend", run_id=run_id))

    def resume(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_resume", run_id=run_id))

    def terminate(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("run_terminate", run_id=run_id))

    def delete(self, run_id: str) -> dict[str, Any]:
        return self._request("DELETE", self._endpoint("run_delete", run_id=run_id))
