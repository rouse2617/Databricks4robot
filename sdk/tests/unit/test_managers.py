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
        respx.get(f"{BASE_URL}/api/v1/mcap-files").mock(
            return_value=httpx.Response(200, json={"items": [], "total": 0})
        )
        assert client.storage.list_files()["total"] == 0

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
        respx.get(f"{BASE_URL}/api/v1/algo-runs").mock(
            return_value=httpx.Response(200, json={"items": []})
        )
        assert client.algo_runs.list(asset_id="a1") == {"items": []}

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
            return_value=httpx.Response(200, json={"name": "wf1", "nodes": []})
        )
        assert client.workflows.get("wf1")["name"] == "wf1"

    def test_logs(self, client):
        respx.get(f"{BASE_URL}/api/v1/workflows/wf1/logs").mock(
            return_value=httpx.Response(200, json={"logs": "hello"})
        )
        assert client.workflows.logs("wf1", "n1") == {"logs": "hello"}

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
