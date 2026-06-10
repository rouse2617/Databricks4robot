"""Tests for ConfigManager — endpoint resolution, multi-source merge, remote fetch."""

from __future__ import annotations

import httpx
import respx

from cyber_databrew_sdk.config import ConfigManager
from cyber_databrew_sdk.config.endpoints import ENDPOINTS


class TestEndpointResolution:
    def test_resolve_simple(self):
        cfg = ConfigManager(base_url="http://test")
        assert cfg.resolve("asset_list") == "/api/v1/assets"

    def test_resolve_with_params(self):
        cfg = ConfigManager(base_url="http://test")
        assert cfg.resolve("asset_get", asset_id="abc") == "/api/v1/assets/abc"

    def test_resolve_with_multiple_params(self):
        cfg = ConfigManager(base_url="http://test")
        path = cfg.resolve("delivery_commit", delivery_id="d1")
        assert path == "/api/v1/deliveries/d1/commit"

    def test_unknown_endpoint_fallback(self):
        cfg = ConfigManager(base_url="http://test")
        path = cfg.resolve("unknown")
        assert path.startswith("/api/v1/")

    def test_build_url(self):
        cfg = ConfigManager(base_url="http://test")
        url = cfg.build_url("asset_get", asset_id="abc")
        assert url == "http://test/api/v1/assets/abc"


class TestConfigOverride:
    def test_override_single_endpoint(self):
        cfg = ConfigManager(
            base_url="http://test",
            endpoint_overrides={"asset_list": "/api/v2/assets"},
        )
        assert cfg.resolve("asset_list") == "/api/v2/assets"
        # Non-overridden endpoint still uses default
        assert cfg.resolve("asset_get", asset_id="abc") == "/api/v1/assets/abc"

    def test_override_via_load(self):
        """endpoint_overrides passed to load() are applied."""
        cfg = ConfigManager.load(
            base_url="http://test",
            timeout=60.0,
            endpoint_overrides={"asset_list": "/api/v2/assets"},
            http_client=httpx.Client(),
        )
        assert cfg.resolve("asset_list") == "/api/v2/assets"
        assert cfg.timeout == 60.0
        # Non-overridden preserved
        assert cfg.resolve("asset_get", asset_id="abc") == "/api/v1/assets/abc"

    def test_load_enriches_endpoints(self):
        """New endpoints returned from remote get merged into the map."""
        cfg = ConfigManager.load(
            base_url="http://test",
            http_client=httpx.Client(),
        )
        # All default endpoints are present
        assert cfg.resolve("asset_get", asset_id="x")
        assert cfg.resolve("delivery_list")


class TestDefaultEndpoints:
    def test_all_endpoints_available(self):
        """All ENDPOINTS from the registry are available."""
        cfg = ConfigManager(base_url="http://test")
        for name in ENDPOINTS:
            resolved = cfg.resolve(name)
            assert resolved.startswith("/api/v1/")


class TestRemoteFetch:
    @respx.mock
    def test_fetch_success(self):
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(
                200,
                json={
                    "endpoints": {"asset_list": "/api/v2/assets"},
                    "timeout": 60.0,
                },
            )
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={"X-Databrew-Token": "t"},
        )
        assert cfg.resolve("asset_list") == "/api/v2/assets"
        assert cfg.timeout == 60.0

    @respx.mock
    def test_fetch_failure_uses_defaults(self):
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(500)
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={"X-Databrew-Token": "t"},
        )
        assert cfg.resolve("asset_list") == "/api/v1/assets"
        assert cfg.timeout == 30.0

    @respx.mock
    def test_fetch_http_error(self):
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(404)
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={},
        )
        assert cfg.resolve("asset_list") == "/api/v1/assets"

    @respx.mock
    def test_fetch_empty_override_keeps_defaults(self):
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(200, json={})
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={},
        )
        assert cfg.resolve("asset_list") == "/api/v1/assets"

    @respx.mock
    def test_fetch_partial_override(self):
        """Only the endpoint returned by the backend is overridden."""
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(
                200,
                json={"endpoints": {"asset_list": "/api/v2/assets"}},
            )
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={},
        )
        # Overridden
        assert cfg.resolve("asset_list") == "/api/v2/assets"
        # Not overridden — still default
        assert cfg.resolve("asset_get", asset_id="x") == "/api/v1/assets/x"
        assert cfg.resolve("delivery_list") == "/api/v1/deliveries"

    @respx.mock
    def test_explicit_overrides_beat_remote(self):
        """Explicit endpoint_overrides in load() win over remote values."""
        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(
                200,
                json={"endpoints": {"asset_list": "/api/v2/assets"}},
            )
        )
        cfg = ConfigManager.load(
            base_url="http://test",
            auth_headers={},
            endpoint_overrides={"asset_list": "/api/v3/assets"},
        )
        assert cfg.resolve("asset_list") == "/api/v3/assets"


class TestRefresh:
    @respx.mock
    def test_refresh_updates_endpoints(self):
        cfg = ConfigManager(base_url="http://test")
        assert cfg.resolve("asset_list") == "/api/v1/assets"

        respx.get("http://test/api/v1/sdk-config").mock(
            return_value=httpx.Response(
                200, json={"endpoints": {"asset_list": "/api/v2/assets"}}
            )
        )
        cfg.refresh(base_url="http://test")
        assert cfg.resolve("asset_list") == "/api/v2/assets"
