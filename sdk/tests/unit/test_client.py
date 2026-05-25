"""Tests for CyberDatabrewClient — constructor, lazy loading, lifecycle."""

from __future__ import annotations

import httpx
import pytest

from cyber_databrew_sdk import CyberDatabrewClient
from cyber_databrew_sdk.auth import DatabrewTokenAuth
from cyber_databrew_sdk.managers.assets import AssetManager
from cyber_databrew_sdk.managers.audit import AuditManager


# Shared mock client used to skip remote config fetch in unit tests
@pytest.fixture
def mock_http_client():
    return httpx.Client()


class TestClientConstructor:
    def test_default_auth(self, mock_http_client):
        client = CyberDatabrewClient(token="t", email="e@x.com", http_client=mock_http_client)
        headers = client._requestor._auth_headers
        assert headers["X-Databrew-Token"] == "t"
        assert headers["X-User-Email"] == "e@x.com"

    def test_custom_auth_overrides_default(self, mock_http_client):
        custom = DatabrewTokenAuth("custom")
        client = CyberDatabrewClient(token="default", auth=custom, http_client=mock_http_client)
        assert client._requestor._auth_headers["X-Databrew-Token"] == "custom"

    def test_custom_base_url(self, mock_http_client):
        client = CyberDatabrewClient(base_url="https://api.example.com", http_client=mock_http_client)
        assert client._requestor._base_url == "https://api.example.com"

    def test_default_base_url(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        assert client._requestor._base_url == "http://localhost:8080"

    def test_base_url_trailing_slash_stripped(self, mock_http_client):
        client = CyberDatabrewClient(base_url="http://localhost:8080/", http_client=mock_http_client)
        assert client._requestor._base_url == "http://localhost:8080"

    def test_custom_timeout(self):
        client = CyberDatabrewClient(timeout=60.0, base_url="http://localhost:1")
        assert client._config.timeout == 60.0
        assert client._requestor._client.timeout.connect == 60.0

    def test_inject_http_client(self):
        custom = httpx.Client()
        client = CyberDatabrewClient(http_client=custom)
        assert client._requestor._client is custom

    def test_default_timeout(self):
        client = CyberDatabrewClient(base_url="http://localhost:1")
        assert client._requestor._client.timeout.connect == 30.0

    def test_no_args_uses_env_fallbacks(self, monkeypatch, mock_http_client):
        monkeypatch.setenv("CYBER_DATABREW_TOKEN", "env-token")
        monkeypatch.setenv("CYBER_DATABREW_EMAIL", "env@x.com")
        monkeypatch.setenv("CYBER_DATABREW_BASE_URL", "http://env.url")
        client = CyberDatabrewClient(http_client=mock_http_client)
        h = client._requestor._auth_headers
        assert h["X-Databrew-Token"] == "env-token"
        assert h["X-User-Email"] == "env@x.com"
        assert client._requestor._base_url == "http://env.url"

    def test_config_default_endpoints_loaded(self, mock_http_client):
        """Verify that default endpoints are loaded in the config."""
        client = CyberDatabrewClient(http_client=mock_http_client)
        # Spot-check a few resolved paths
        assert client._config.resolve("asset_get", asset_id="abc") == "/api/v1/assets/abc"
        assert client._config.resolve("delivery_list") == "/api/v1/deliveries"
        assert client._config.resolve("audit_search") == "/api/v1/audit/search"

    def test_unknown_endpoint_falls_back(self, mock_http_client):
        """Unknown endpoint names fall back gracefully."""
        client = CyberDatabrewClient(http_client=mock_http_client)
        path = client._config.resolve("nonexistent")
        assert path == "/api/v1/nonexistent"


class TestClientConstructorNoMock:
    """Tests that create a client WITHOUT http_client to exercise the
    real init path (remote config fetch attempts localhost:8080,
    fails silently, falls back to defaults)."""

    def test_remote_config_failure_falls_back(self):
        """When backend is unreachable, defaults are used."""
        client = CyberDatabrewClient(token="t", base_url="http://localhost:1")
        # Should fall back to defaults without raising
        assert client._config.resolve("asset_get", asset_id="x") == "/api/v1/assets/x"
        assert client._requestor._base_url == "http://localhost:1"


class TestClientLazyLoading:
    def test_assets_returns_asset_manager(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        mgr = client.assets
        assert isinstance(mgr, AssetManager)

    def test_audit_returns_audit_manager(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        mgr = client.audit
        assert isinstance(mgr, AuditManager)

    def test_same_instance_cached(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        a1 = client.assets
        a2 = client.assets
        assert a1 is a2

    def test_unknown_attribute_raises(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        with pytest.raises(AttributeError, match="CyberDatabrewClient"):
            _ = client.unknown

    def test_all_managers_load(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        for name in ("assets", "storage", "delivery", "algo_runs", "search",
                     "queries", "customers", "lakehouse", "events", "registry",
                     "audit"):
            assert hasattr(client, name), f"missing {name}"


class TestClientLifecycle:
    def test_context_manager(self, mock_http_client):
        with CyberDatabrewClient(http_client=mock_http_client) as client:
            assert not client._closed
        assert client._closed

    def test_close(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        client.close()
        assert client._closed

    def test_close_idempotent(self, mock_http_client):
        client = CyberDatabrewClient(http_client=mock_http_client)
        client.close()
        client.close()  # should not raise
        assert client._closed
