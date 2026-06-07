"""Tests for all 11 semantic managers using respx."""

from __future__ import annotations

import httpx
import pytest
import respx

from cyber_databrew_sdk import CyberDatabrewClient
from cyber_databrew_sdk.exceptions import NotFoundError

BASE_URL = "http://test"

pytestmark = pytest.mark.usefixtures("respx_mock")


@pytest.fixture
def client():
    """Create a client with remote config mocked out."""
    respx.get(f"{BASE_URL}/api/v1/sdk-config").mock(
        return_value=httpx.Response(200, json={"endpoints": {}})
    )
    return CyberDatabrewClient(token="t", email="e@x.com", base_url=BASE_URL)


# =========================================================================
# AssetManager
# =========================================================================

class TestAssetManager:
    def test_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/abc").mock(
            return_value=httpx.Response(200, json={"id": "abc", "status": "ready"})
        )
        result = client.assets.get("abc")
        assert result["id"] == "abc"

    def test_list_all(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets").mock(
            return_value=httpx.Response(200, json={"items": [], "total": 0})
        )
        result = client.assets.list_all(status="ready", tag="lidar")
        assert result["total"] == 0

    def test_create(self, client):
        respx.post(f"{BASE_URL}/api/v1/assets").mock(
            return_value=httpx.Response(201, json={"id": "new", "status": "draft"})
        )
        result = client.assets.create({"name": "test"})
        assert result["id"] == "new"

    def test_update(self, client):
        respx.patch(f"{BASE_URL}/api/v1/assets/abc").mock(
            return_value=httpx.Response(200, json={"id": "abc", "status": "ready"})
        )
        result = client.assets.update("abc", {"status": "ready"})
        assert result["status"] == "ready"

    def test_delete(self, client):
        respx.delete(f"{BASE_URL}/api/v1/assets/abc").mock(
            return_value=httpx.Response(204)
        )
        assert client.assets.delete("abc") == {}

    def test_batch_get(self, client):
        respx.post(f"{BASE_URL}/api/v1/assets:batch_get").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        result = client.assets.batch_get(["a", "b"])
        assert result["items"] == []

    def test_tags(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/abc/tags").mock(
            return_value=httpx.Response(200, json={"tags": []})
        )
        assert client.assets.get_tags("abc") == {"tags": []}

    def test_set_tags(self, client):
        respx.post(f"{BASE_URL}/api/v1/assets/abc/tags").mock(
            return_value=httpx.Response(200, json={"tags": [{"k": "v"}]})
        )
        result = client.assets.set_tags("abc", [{"key": "env", "value": "prod"}])
        assert result["tags"] == [{"k": "v"}]

    def test_lineage(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/abc/lineage").mock(
            return_value=httpx.Response(200, json={"nodes": []})
        )
        assert client.assets.get_lineage("abc") == {"nodes": []}

    def test_404(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/nonexistent").mock(
            return_value=httpx.Response(404, json={"message": "not found"})
        )
        with pytest.raises(NotFoundError):
            client.assets.get("nonexistent")


# =========================================================================
# StorageManager
# =========================================================================

class TestStorageManager:
    def test_list_files(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/mcap-files").mock(
            return_value=httpx.Response(200, json={"items": [], "total": 0})
        )
        assert client.storage.list_files(ingest_state="summarized", owner="qa")["total"] == 0
        assert dict(route.calls.last.request.url.params) == {
            "page": "1",
            "page_size": "20",
            "ingest_state": "summarized",
            "owner": "qa",
        }

    def test_get_file_info(self, client):
        respx.get(f"{BASE_URL}/api/v1/mcap-files/f1").mock(
            return_value=httpx.Response(200, json={"id": "f1", "size": 100})
        )
        assert client.storage.get_file_info("f1")["size"] == 100

    def test_download_mcap(self, client, tmp_path):
        respx.get(f"{BASE_URL}/api/v1/mcap-files/f1/bytes").mock(
            return_value=httpx.Response(200, content=b"mcap-data", headers={"content-type": "application/octet-stream"})
        )
        out = tmp_path / "test.mcap"
        result = client.storage.download_mcap("f1", out)
        assert result == out
        assert out.read_bytes() == b"mcap-data"

    def test_download_asset_mcap(self, client, tmp_path):
        respx.get(f"{BASE_URL}/api/v1/assets/a1/mcap-locator").mock(
            return_value=httpx.Response(200, json={"mcap_file_id": "f1"})
        )
        respx.get(f"{BASE_URL}/api/v1/mcap-files/f1/bytes").mock(
            return_value=httpx.Response(200, content=b"data", headers={"content-type": "application/octet-stream"})
        )
        out = tmp_path / "asset.mcap"
        result = client.storage.download_asset_mcap("a1", out)
        assert result == out
        assert out.read_bytes() == b"data"

    def test_finalize_upload(self, client):
        respx.post(f"{BASE_URL}/api/v1/mcap/upload/finalize").mock(
            return_value=httpx.Response(200, json={"status": "ok"})
        )
        assert client.storage.finalize_upload({"upload_id": "u1"})["status"] == "ok"

    def test_get_messages(self, client):
        respx.get(f"{BASE_URL}/api/v1/mcap/f1/messages").mock(
            return_value=httpx.Response(200, json={"messages": []})
        )
        assert client.storage.get_messages("f1") == {"messages": []}


# =========================================================================
# DeliveryManager
# =========================================================================

class TestDeliveryManager:
    def test_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/deliveries/d1").mock(
            return_value=httpx.Response(200, json={"id": "d1", "status": "pending"})
        )
        assert client.delivery.get("d1")["id"] == "d1"

    def test_list(self, client):
        respx.get(f"{BASE_URL}/api/v1/deliveries").mock(
            return_value=httpx.Response(200, json={"items": [], "total": 0})
        )
        assert client.delivery.list()["total"] == 0

    def test_create(self, client):
        respx.post(f"{BASE_URL}/api/v1/deliveries").mock(
            return_value=httpx.Response(201, json={"id": "d1"})
        )
        assert client.delivery.create({"customer_id": "c1"})["id"] == "d1"

    def test_draft_and_commit(self, client):
        respx.post(f"{BASE_URL}/api/v1/deliveries/draft").mock(
            return_value=httpx.Response(201, json={"id": "d1", "status": "draft"})
        )
        assert client.delivery.draft({})["status"] == "draft"

        respx.post(f"{BASE_URL}/api/v1/deliveries/d1/commit").mock(
            return_value=httpx.Response(200, json={"id": "d1", "status": "committed"})
        )
        assert client.delivery.commit("d1")["status"] == "committed"

    def test_cancel(self, client):
        respx.post(f"{BASE_URL}/api/v1/deliveries/d1/cancel").mock(
            return_value=httpx.Response(200, json={"status": "cancelled"})
        )
        assert client.delivery.cancel("d1")["status"] == "cancelled"

    def test_items(self, client):
        respx.get(f"{BASE_URL}/api/v1/deliveries/d1/items").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.delivery.get_items("d1") == {"items": []}

    def test_list_rules(self, client):
        respx.get(f"{BASE_URL}/api/v1/delivery-rules").mock(
            return_value=httpx.Response(200, json={"rules": []})
        )
        assert client.delivery.list_rules() == {"rules": []}


# =========================================================================
# AlgoRunManager
# =========================================================================

class TestAlgoRunManager:
    def test_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/algo-runs/r1").mock(
            return_value=httpx.Response(200, json={"id": "r1"})
        )
        assert client.algo_runs.get("r1")["id"] == "r1"

    def test_create(self, client):
        respx.post(f"{BASE_URL}/api/v1/algo-runs").mock(
            return_value=httpx.Response(201, json={"id": "r1"})
        )
        assert client.algo_runs.create({"algo_key": "lk"})["id"] == "r1"

    def test_start(self, client):
        respx.post(f"{BASE_URL}/api/v1/algo-runs/r1/start").mock(
            return_value=httpx.Response(200, json={"status": "running"})
        )
        assert client.algo_runs.start("r1")["status"] == "running"

    def test_finish(self, client):
        respx.post(f"{BASE_URL}/api/v1/algo-runs/r1/finish").mock(
            return_value=httpx.Response(200, json={"status": "completed"})
        )
        assert client.algo_runs.finish("r1")["status"] == "completed"

    def test_cancel(self, client):
        respx.post(f"{BASE_URL}/api/v1/algo-runs/r1/cancel").mock(
            return_value=httpx.Response(200, json={"status": "cancelled"})
        )
        assert client.algo_runs.cancel("r1")["status"] == "cancelled"

    def test_list_with_filters(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/algo-runs").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.algo_runs.list(
            algo_name="hand_track",
            status="failed",
            started_after="2026-05-22T00:00:00Z",
            started_before="2026-05-23T00:00:00Z",
            page=2,
            page_size=25,
        ) == {"items": []}
        assert dict(route.calls.last.request.url.params) == {
            "algo_name": "hand_track",
            "status": "failed",
            "started_after": "2026-05-22T00:00:00Z",
            "started_before": "2026-05-23T00:00:00Z",
            "page": "2",
            "page_size": "25",
        }

    def test_get_affected_assets(self, client):
        respx.get(f"{BASE_URL}/api/v1/algo-runs/r1/affected-assets").mock(
            return_value=httpx.Response(200, json={"assets": []})
        )
        assert client.algo_runs.get_affected_assets("r1") == {"assets": []}


# =========================================================================
# WorkflowManager
# =========================================================================

class TestWorkflowManager:
    def test_list_and_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/workflows").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.workflows.list() == {"items": []}

        respx.get(f"{BASE_URL}/api/v1/workflows/wf1").mock(
            return_value=httpx.Response(
                200,
                json={
                    "name": "wf1",
                    "nodes": [],
                    "edges": [
                        {
                            "id": "e-step-1-step-2",
                            "source": "step-1",
                            "target": "step-2",
                            "kind": "dag",
                        }
                    ],
                },
            )
        )
        workflow = client.workflows.get("wf1")
        assert workflow["name"] == "wf1"
        assert workflow["edges"][0]["kind"] == "dag"

    def test_logs(self, client):
        respx.get(f"{BASE_URL}/api/v1/workflows/wf1/logs").mock(
            return_value=httpx.Response(
                200,
                json={
                    "logs": "hello",
                    "pagination": {"available": False, "nextCursor": None},
                    "window": {"mode": "tail", "scope": "bounded-live-window"},
                },
            )
        )
        assert client.workflows.logs("wf1", "n1")["logs"] == "hello"

    def test_logs_with_window_params(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/workflows/wf1/logs").mock(
            return_value=httpx.Response(200, json={"logs": "hello"})
        )
        client.workflows.logs(
            "wf1",
            "n1",
            tail_lines=50,
            limit_bytes=65536,
            container="main",
            timestamps=True,
        )

        request = route.calls.last.request
        assert request.url.params["nodeId"] == "n1"
        assert request.url.params["tailLines"] == "50"
        assert request.url.params["limitBytes"] == "65536"
        assert request.url.params["container"] == "main"
        assert request.url.params["timestamps"] == "true"

    def test_operations(self, client):
        for operation in ("retry", "resubmit", "suspend", "resume", "terminate"):
            respx.post(f"{BASE_URL}/api/v1/workflows/wf1/{operation}").mock(
                return_value=httpx.Response(200, json={"message": "ok"})
            )
            assert getattr(client.workflows, operation)("wf1") == {"message": "ok"}

    def test_delete(self, client):
        respx.delete(f"{BASE_URL}/api/v1/workflows/wf1").mock(
            return_value=httpx.Response(200, json={"message": "ok"})
        )
        assert client.workflows.delete("wf1") == {"message": "ok"}


# =========================================================================
# PipelineManager
# =========================================================================

class TestPipelineManager:
    def test_list_execution_targets(self, client):
        respx.get(f"{BASE_URL}/api/v1/execution-targets").mock(
            return_value=httpx.Response(200, json={"items": [{"id": "default"}]})
        )
        result = client.pipelines.list_execution_targets()
        assert result["items"][0]["id"] == "default"

    def test_deploy_template_with_assets_and_target(self, client):
        route = respx.post(f"{BASE_URL}/api/v1/deploy/template/tmpl-1").mock(
            return_value=httpx.Response(201, json={"id": "dep-1"})
        )
        result = client.pipelines.deploy_template(
            "tmpl-1",
            asset_ids=["SDKT0202"],
            target_id="default",
            name="asset-run",
        )
        assert result["id"] == "dep-1"
        assert route.calls.last.request.read()
        assert route.calls.last.request.url.path == "/api/v1/deploy/template/tmpl-1"

    def test_create_run(self, client):
        route = respx.post(f"{BASE_URL}/api/v1/pipeline-runs").mock(
            return_value=httpx.Response(201, json={"id": "run-1"})
        )
        result = client.pipelines.create_run(
            {"name": "pipe", "nodes": [], "edges": []},
            asset_ids=["asset-1"],
            target_id="default",
            name="run-name",
        )
        assert result["id"] == "run-1"
        body = route.calls.last.request.read()
        assert b'"pipeline"' in body
        assert route.calls.last.request.url.path == "/api/v1/pipeline-runs"

    def test_create_run_from_template(self, client):
        route = respx.post(f"{BASE_URL}/api/v1/pipeline-runs/template/tmpl-1").mock(
            return_value=httpx.Response(201, json={"id": "run-1"})
        )
        result = client.pipelines.create_run_from_template("tmpl-1", asset_ids=["asset-1"])
        assert result["id"] == "run-1"
        assert route.calls.last.request.url.path == "/api/v1/pipeline-runs/template/tmpl-1"

    def test_template_version_and_batch_helpers(self, client):
        batch_route = respx.post(
            f"{BASE_URL}/api/v1/pipeline-runs/template/tmpl-1/batch"
        ).mock(return_value=httpx.Response(201, json={"batchId": "batch-1", "items": []}))
        active_route = respx.patch(
            f"{BASE_URL}/api/v1/pipelines/tmpl-1/active-version"
        ).mock(return_value=httpx.Response(200, json={"activeVersion": 2}))
        promote_route = respx.post(f"{BASE_URL}/api/v1/pipelines/tmpl-1/promote").mock(
            return_value=httpx.Response(200, json={"id": "tmpl-1"})
        )

        assert client.pipelines.create_batch_runs_from_template(
            "tmpl-1",
            ["asset-1"],
            target_id="default",
            version=2,
        )["batchId"] == "batch-1"
        assert client.pipelines.set_template_active_version("tmpl-1", 2)["activeVersion"] == 2
        assert client.pipelines.promote_template("tmpl-1")["id"] == "tmpl-1"
        assert batch_route.calls.last.request.url.path == "/api/v1/pipeline-runs/template/tmpl-1/batch"
        assert b'"asset_ids"' in batch_route.calls.last.request.read()
        assert active_route.calls.last.request.url.path == "/api/v1/pipelines/tmpl-1/active-version"
        assert b'"activeVersion"' in active_route.calls.last.request.read()
        assert promote_route.calls.last.request.url.path == "/api/v1/pipelines/tmpl-1/promote"

    def test_pipeline_run_crud_actions(self, client):
        respx.get(f"{BASE_URL}/api/v1/pipeline-runs").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        respx.get(f"{BASE_URL}/api/v1/pipeline-runs/run-1").mock(
            return_value=httpx.Response(200, json={"id": "run-1"})
        )
        respx.get(f"{BASE_URL}/api/v1/pipeline-runs/watcher/status").mock(
            return_value=httpx.Response(200, json={"id": "default", "healthy": True})
        )
        events_route = respx.get(f"{BASE_URL}/api/v1/pipeline-runs/run-1/events").mock(
            return_value=httpx.Response(200, json={"items": [{"id": "evt-1"}], "total": 1})
        )
        asset_nodes_route = respx.get(f"{BASE_URL}/api/v1/pipeline-runs/run-1/asset-nodes").mock(
            return_value=httpx.Response(200, json={"items": [{"id": "an-1"}], "total": 1})
        )
        respx.get(f"{BASE_URL}/api/v1/pipeline-runs/run-1/cost-summary").mock(
            return_value=httpx.Response(200, json={"runId": "run-1", "costSource": "not_available"})
        )
        respx.post(f"{BASE_URL}/api/v1/pipeline-runs/run-1/retry").mock(
            return_value=httpx.Response(201, json={"id": "run-2"})
        )
        respx.post(f"{BASE_URL}/api/v1/pipeline-runs/run-1/stop").mock(
            return_value=httpx.Response(200, json={"message": "pipeline run stopped"})
        )
        respx.delete(f"{BASE_URL}/api/v1/pipeline-runs/run-1").mock(
            return_value=httpx.Response(204)
        )
        assert client.pipelines.list_runs() == {"items": []}
        assert client.pipelines.get_run("run-1")["id"] == "run-1"
        assert client.pipelines.get_run_watcher_status()["healthy"] is True
        assert client.pipelines.list_run_events(
            "run-1",
            limit=50,
            subject_type="node",
            event_type="node_failed",
            status="Failed",
            q="image",
        )["items"][0]["id"] == "evt-1"
        assert dict(events_route.calls.last.request.url.params) == {
            "limit": "50",
            "subjectType": "node",
            "eventType": "node_failed",
            "status": "Failed",
            "q": "image",
        }
        assert client.pipelines.list_run_asset_nodes(
            "run-1",
            limit=20,
            asset_id="asset-1",
            order_by="cost",
        )["items"][0]["id"] == "an-1"
        assert dict(asset_nodes_route.calls.last.request.url.params) == {
            "limit": "20",
            "assetId": "asset-1",
            "orderBy": "cost",
        }
        assert client.pipelines.get_run_cost_summary("run-1")["runId"] == "run-1"
        assert client.pipelines.retry_run("run-1")["id"] == "run-2"
        assert client.pipelines.stop_run("run-1")["message"] == "pipeline run stopped"
        assert client.pipelines.delete_run("run-1") == {}

    def test_pod_terminal_session_helpers(self, client):
        create_route = respx.post(
            f"{BASE_URL}/api/v1/workflows/wf-1/nodes/node-1/terminal-sessions"
        ).mock(return_value=httpx.Response(201, json={"id": "sess-1", "status": "created"}))
        respx.get(f"{BASE_URL}/api/v1/pod-terminal/sessions/sess-1").mock(
            return_value=httpx.Response(200, json={"id": "sess-1", "status": "created"})
        )
        respx.post(f"{BASE_URL}/api/v1/pod-terminal/sessions/sess-1/terminate").mock(
            return_value=httpx.Response(200, json={"id": "sess-1", "status": "terminated"})
        )

        created = client.pipelines.create_pod_terminal_session(
            "wf-1",
            "node-1",
            command="pwd",
            container_name="main",
        )
        assert created["id"] == "sess-1"
        body = create_route.calls.last.request.read()
        assert b'"command":"pwd"' in body
        assert b'"containerName":"main"' in body
        assert client.pipelines.get_pod_terminal_session("sess-1")["status"] == "created"
        assert (
            client.pipelines.terminate_pod_terminal_session("sess-1")["status"]
            == "terminated"
        )
        assert (
            client.pipelines.pod_terminal_attach_path("sess-1", "tok")
            == "/api/v1/pod-terminal/sessions/sess-1/attach?token=tok"
        )

    def test_list_deployments(self, client):
        respx.get(f"{BASE_URL}/api/v1/deployments").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.pipelines.list_deployments() == {"items": []}


# =========================================================================
# PipelineComponentManager
# =========================================================================

class TestPipelineComponentManager:
    def test_list_and_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/pipeline-components").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.pipeline_components.list(q="processor", source="custom") == {"items": []}

        respx.get(f"{BASE_URL}/api/v1/pipeline-components/c1").mock(
            return_value=httpx.Response(200, json={"id": "c1", "name": "processor"})
        )
        assert client.pipeline_components.get("c1")["id"] == "c1"

    def test_create_update_delete(self, client):
        payload = {"name": "processor", "type": "container", "image": "busybox"}
        respx.post(f"{BASE_URL}/api/v1/pipeline-components").mock(
            return_value=httpx.Response(201, json={"id": "c1", **payload})
        )
        assert client.pipeline_components.create(payload)["id"] == "c1"

        respx.put(f"{BASE_URL}/api/v1/pipeline-components/c1").mock(
            return_value=httpx.Response(200, json={"id": "c1", "tag": "v2"})
        )
        updated = client.pipeline_components.update(
            "c1",
            {
                "name": "processor",
                "type": "container",
                "image": "busybox",
                "tag": "v2",
            },
        )
        assert updated["tag"] == "v2"

        respx.delete(f"{BASE_URL}/api/v1/pipeline-components/c1").mock(
            return_value=httpx.Response(204)
        )
        assert client.pipeline_components.delete("c1") == {}


# =========================================================================
# SearchManager
# =========================================================================

class TestSearchManager:
    def test_get_sync_status(self, client):
        respx.get(f"{BASE_URL}/api/v1/search/sync-status").mock(
            return_value=httpx.Response(200, json={"status": "ok"})
        )
        assert client.search.get_sync_status()["status"] == "ok"

    def test_assets_with_lineage_filter(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/search/assets").mock(
            return_value=httpx.Response(200, json={"items": [], "total": 0})
        )
        client.search.assets(
            lineage_with="asset-root",
            lineage_direction="downstream",
            lineage_depth=2,
            relation_types=["derived_from", "pipeline_output"],
        )
        params = dict(route.calls.last.request.url.params)
        assert params["lineage_with"] == "asset-root"
        assert params["lineage_direction"] == "downstream"
        assert params["lineage_depth"] == "2"
        assert params["relation_types"] == "derived_from,pipeline_output"

    def test_get_sync_progress(self, client):
        respx.get(f"{BASE_URL}/api/v1/search/sync-progress").mock(
            return_value=httpx.Response(200, json={"progress": 0.5})
        )
        assert client.search.get_sync_progress()["progress"] == 0.5


# =========================================================================
# QueryManager
# =========================================================================

class TestQueryManager:
    def test_validate(self, client):
        respx.post(f"{BASE_URL}/api/v1/queries/validate").mock(
            return_value=httpx.Response(200, json={"valid": True})
        )
        assert client.queries.validate({"sql": "SELECT 1"})["valid"] is True

    def test_run(self, client):
        respx.post(f"{BASE_URL}/api/v1/queries/run").mock(
            return_value=httpx.Response(200, json={"rows": []})
        )
        assert client.queries.run({"sql": "SELECT 1"})["rows"] == []

    def test_saved_crud(self, client):
        respx.get(f"{BASE_URL}/api/v1/saved-queries").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.queries.list_saved()["items"] == []

        respx.post(f"{BASE_URL}/api/v1/saved-queries").mock(
            return_value=httpx.Response(201, json={"id": "q1"})
        )
        assert client.queries.create_saved({"name": "q"})["id"] == "q1"

        respx.get(f"{BASE_URL}/api/v1/saved-queries/q1").mock(
            return_value=httpx.Response(200, json={"id": "q1"})
        )
        assert client.queries.get_saved("q1")["id"] == "q1"

        respx.delete(f"{BASE_URL}/api/v1/saved-queries/q1").mock(
            return_value=httpx.Response(204)
        )
        assert client.queries.delete_saved("q1") == {}


# =========================================================================
# CustomerManager
# =========================================================================

class TestCustomerManager:
    def test_list(self, client):
        respx.get(f"{BASE_URL}/api/v1/customers").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.customers.list()["items"] == []

    def test_get(self, client):
        respx.get(f"{BASE_URL}/api/v1/customers/c1").mock(
            return_value=httpx.Response(200, json={"id": "c1"})
        )
        assert client.customers.get("c1")["id"] == "c1"

    def test_create(self, client):
        respx.post(f"{BASE_URL}/api/v1/customers").mock(
            return_value=httpx.Response(201, json={"id": "c1"})
        )
        assert client.customers.create({"name": "Acme"})["id"] == "c1"

    def test_update(self, client):
        respx.patch(f"{BASE_URL}/api/v1/customers/c1").mock(
            return_value=httpx.Response(200, json={"id": "c1", "name": "Acme 2"})
        )
        assert client.customers.update("c1", {"name": "Acme 2"})["name"] == "Acme 2"


# =========================================================================
# LakehouseManager
# =========================================================================

class TestLakehouseManager:
    def test_report(self, client):
        respx.get(f"{BASE_URL}/api/v1/lakehouse/report").mock(
            return_value=httpx.Response(200, json={"total_assets": 100})
        )
        assert client.lakehouse.get_report()["total_assets"] == 100

    def test_status(self, client):
        respx.get(f"{BASE_URL}/api/v1/lakehouse/status").mock(
            return_value=httpx.Response(200, json={"healthy": True})
        )
        assert client.lakehouse.get_status()["healthy"] is True

    def test_tables(self, client):
        respx.get(f"{BASE_URL}/api/v1/lakehouse/tables").mock(
            return_value=httpx.Response(200, json={"tables": []})
        )
        assert client.lakehouse.get_tables()["tables"] == []

    def test_overview(self, client):
        respx.get(f"{BASE_URL}/api/v1/lakehouse/overview").mock(
            return_value=httpx.Response(200, json={"summary": {}})
        )
        assert client.lakehouse.get_overview()["summary"] == {}

    def test_sync_progress(self, client):
        respx.get(f"{BASE_URL}/api/v1/lakehouse/sync-progress").mock(
            return_value=httpx.Response(200, json={"progress": 0.8})
        )
        assert client.lakehouse.get_sync_progress()["progress"] == 0.8


# =========================================================================
# EventManager
# =========================================================================

class TestEventManager:
    def test_list_global(self, client):
        respx.get(f"{BASE_URL}/api/v1/events").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.events.list_global()["items"] == []

    def test_list_for_asset(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/a1/events").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.events.list_for_asset("a1")["items"] == []

    def test_stream_for_asset(self, client):
        respx.get(f"{BASE_URL}/api/v1/assets/a1/events/stream").mock(
            return_value=httpx.Response(200, json={"events": []})
        )
        assert client.events.stream_for_asset("a1")["events"] == []


# =========================================================================
# RegistryManager
# =========================================================================

class TestRegistryManager:
    def test_list_algos(self, client):
        respx.get(f"{BASE_URL}/api/v1/algo-registry").mock(
            return_value=httpx.Response(200, json={"algos": []})
        )
        assert client.registry.list_algos()["algos"] == []

    def test_list_tags(self, client):
        respx.get(f"{BASE_URL}/api/v1/tag-registry").mock(
            return_value=httpx.Response(200, json={"tags": []})
        )
        assert client.registry.list_tags()["tags"] == []

    def test_list_metrics(self, client):
        respx.get(f"{BASE_URL}/api/v1/metric-registry").mock(
            return_value=httpx.Response(200, json={"metrics": []})
        )
        assert client.registry.list_metrics()["metrics"] == []

    def test_list_action_labels(self, client):
        respx.get(f"{BASE_URL}/api/v1/action-label-registry").mock(
            return_value=httpx.Response(200, json={"labels": []})
        )
        assert client.registry.list_action_labels()["labels"] == []

    def test_list_lifecycle_states(self, client):
        respx.get(f"{BASE_URL}/api/v1/lifecycle-states").mock(
            return_value=httpx.Response(200, json={"states": []})
        )
        assert client.registry.list_lifecycle_states()["states"] == []


# =========================================================================
# AuditManager
# =========================================================================

class TestAuditManager:
    def test_search(self, client):
        respx.get(f"{BASE_URL}/api/v1/audit/search?limit=50").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.audit.search()["items"] == []

    def test_search_with_filters(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/audit/search").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.audit.search(actor="alice", event_type="asset.view", limit=10)
        req = route.calls.last.request
        assert req.url.params["actor"] == "alice"
        assert req.url.params["event_type"] == "asset.view"
        assert req.url.params["limit"] == "10"

    def test_lineage_search(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/audit/lineage-search").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.audit.lineage_search("a1")
        req = route.calls.last.request
        assert req.url.params["asset_id"] == "a1"


# =========================================================================
# ActionManager
# =========================================================================

class TestActionManager:
    def test_list(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/assets/a1/actions").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.actions.list("a1")
        assert route.calls.last.request.url.path == "/api/v1/assets/a1/actions"

    def test_list_with_params(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/assets/a1/actions").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.actions.list("a1", source_type="human", limit=10)
        req = route.calls.last.request
        assert req.url.params["source_type"] == "human"
        assert req.url.params["limit"] == "10"

    def test_create(self, client):
        respx.post(f"{BASE_URL}/api/v1/assets/a1/actions").mock(
            return_value=httpx.Response(201, json={"action_id": "act1"})
        )
        result = client.actions.create("a1", {"primary_label": "car"})
        assert result["action_id"] == "act1"

    def test_update(self, client):
        route = respx.patch(f"{BASE_URL}/api/v1/assets/a1/actions/act1").mock(
            return_value=httpx.Response(200, json={"action_id": "act1"})
        )
        client.actions.update("a1", "act1", {"primary_label": "truck"})
        assert route.calls.last.request.url.path == "/api/v1/assets/a1/actions/act1"

    def test_delete(self, client):
        respx.delete(f"{BASE_URL}/api/v1/assets/a1/actions/act1").mock(
            return_value=httpx.Response(204)
        )
        assert client.actions.delete("a1", "act1") == {}


# =========================================================================
# EvalMetricsManager
# =========================================================================

class TestEvalMetricsManager:
    def test_report_eval_result(self, client):
        respx.post(f"{BASE_URL}/api/v1/assets/a1/eval-results").mock(
            return_value=httpx.Response(201, json={"eval_id": "e1"})
        )
        result = client.eval_metrics.report_eval_result("a1", {"score": 0.95})
        assert result["eval_id"] == "e1"

    def test_list_eval_results(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/assets/a1/eval-results").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.eval_metrics.list_eval_results("a1")
        assert route.calls.last.request.url.path == "/api/v1/assets/a1/eval-results"

    def test_list_metrics(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/assets/a1/metrics").mock(
            return_value=httpx.Response(200, json={"metrics": {}})
        )
        client.eval_metrics.list_metrics("a1")
        assert route.calls.last.request.url.path == "/api/v1/assets/a1/metrics"

    def test_get_registry(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/metrics/registry").mock(
            return_value=httpx.Response(200, json={"metrics": []})
        )
        client.eval_metrics.get_registry()
        assert route.calls.last.request.url.path == "/api/v1/metrics/registry"

    def test_search_by_metrics(self, client):
        route = respx.post(f"{BASE_URL}/api/v1/metrics:search").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        client.eval_metrics.search_by_metrics({"quality_min": 0.9})
        assert route.calls.last.request.url.path == "/api/v1/metrics:search"


# =========================================================================
# AdminSearchManager
# =========================================================================

class TestAdminSearchManager:
    def test_reindex(self, client):
        respx.post(f"{BASE_URL}/api/v1/admin/search/reindex").mock(
            return_value=httpx.Response(202, json={"status": "started"})
        )
        result = client.admin_search.reindex()
        assert result["status"] == "started"

    def test_create_reindex_job(self, client):
        respx.post(f"{BASE_URL}/api/v1/admin/search/reindex-jobs").mock(
            return_value=httpx.Response(201, json={"job_id": "j1"})
        )
        result = client.admin_search.create_reindex_job({"type": "full"})
        assert result["job_id"] == "j1"

    def test_list_reindex_jobs(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/admin/search/reindex-jobs").mock(
            return_value=httpx.Response(200, json={"jobs": []})
        )
        client.admin_search.list_reindex_jobs()
        assert route.calls.last.request.url.path == "/api/v1/admin/search/reindex-jobs"

    def test_get_reindex_job(self, client):
        route = respx.get(f"{BASE_URL}/api/v1/admin/search/reindex-jobs/j1").mock(
            return_value=httpx.Response(200, json={"job_id": "j1"})
        )
        client.admin_search.get_reindex_job("j1")
        assert route.calls.last.request.url.path == "/api/v1/admin/search/reindex-jobs/j1"

    def test_stop_reindex_job(self, client):
        respx.post(f"{BASE_URL}/api/v1/admin/search/reindex-jobs/j1/stop").mock(
            return_value=httpx.Response(200, json={"status": "stopping"})
        )
        result = client.admin_search.stop_reindex_job("j1")
        assert result["status"] == "stopping"

    def test_resume_reindex_job(self, client):
        respx.post(f"{BASE_URL}/api/v1/admin/search/reindex-jobs/j1/resume").mock(
            return_value=httpx.Response(200, json={"status": "resumed"})
        )
        result = client.admin_search.resume_reindex_job("j1")
        assert result["status"] == "resumed"

    def test_abandon_reindex_job(self, client):
        respx.post(f"{BASE_URL}/api/v1/admin/search/reindex-jobs/j1/abandon").mock(
            return_value=httpx.Response(200, json={"status": "abandoned"})
        )
        result = client.admin_search.abandon_reindex_job("j1")
        assert result["status"] == "abandoned"
