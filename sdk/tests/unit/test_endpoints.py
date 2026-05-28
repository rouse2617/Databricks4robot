"""Tests for the endpoint registry — completeness and consistency."""

from __future__ import annotations

from cyber_databrew_sdk._endpoints import ENDPOINTS


class TestEndpointRegistry:
    def test_all_endpoints_defined(self):
        """Verify we have the expected number of endpoint definitions."""
        assert len(ENDPOINTS) >= 75  # all ~78 endpoints

    def test_asset_endpoints(self):
        assert "asset_get" in ENDPOINTS
        assert "asset_list" in ENDPOINTS
        assert "asset_create" in ENDPOINTS
        assert "asset_update" in ENDPOINTS
        assert "asset_delete" in ENDPOINTS

    def test_delivery_endpoints(self):
        assert "delivery_get" in ENDPOINTS
        assert "delivery_list" in ENDPOINTS
        assert "delivery_draft" in ENDPOINTS
        assert "delivery_commit" in ENDPOINTS
        assert "delivery_rules_list" in ENDPOINTS

    def test_algo_run_endpoints(self):
        assert "algo_run_get" in ENDPOINTS
        assert "algo_run_start" in ENDPOINTS
        assert "algo_run_finish" in ENDPOINTS
        assert "algo_run_cancel" in ENDPOINTS
        assert "algo_run_affected_assets" in ENDPOINTS

    def test_config_endpoint(self):
        assert "sdk_config" in ENDPOINTS
        assert ENDPOINTS["sdk_config"] == "/api/v1/sdk-config"

    def test_workflow_endpoints(self):
        assert ENDPOINTS["workflow_get"] == "/api/v1/workflows/{workflow_name}"
        assert ENDPOINTS["workflow_retry"] == "/api/v1/workflows/{workflow_name}/retry"
        assert ENDPOINTS["workflow_delete"] == "/api/v1/workflows/{workflow_name}"

    def test_all_templates_use_format_syntax(self):
        """All path templates must be valid str.format templates."""
        for name, template in ENDPOINTS.items():
            # All templates should start with /
            assert template.startswith("/"), f"{name}: {template!r} does not start with /"
            # Verify they can be called with format() (syntax check only)
            from contextlib import suppress

            with suppress(KeyError):
                template.format()  # no args — just check syntax

    def test_parametrized_templates_format_correctly(self):
        """Spot-check that parameterized templates resolve."""
        assert ENDPOINTS["asset_get"].format(asset_id="abc") == "/api/v1/assets/abc"
        assert ENDPOINTS["delivery_commit"].format(delivery_id="d1") == "/api/v1/deliveries/d1/commit"
        assert ENDPOINTS["workflow_resume"].format(workflow_name="wf1") == "/api/v1/workflows/wf1/resume"
