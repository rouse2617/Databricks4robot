"""Tests for auth providers: DatabrewTokenAuth, EmailAuth, CompositeAuth."""

from __future__ import annotations

from cyber_databrew_sdk.auth import CompositeAuth, EmailAuth, DatabrewTokenAuth


class TestDatabrewTokenAuth:
    def test_explicit_token(self):
        auth = DatabrewTokenAuth("my-token")
        assert auth.get_headers() == {"X-Databrew-Token": "my-token"}

    def test_empty_token(self):
        auth = DatabrewTokenAuth("")
        assert auth.get_headers()["X-Databrew-Token"] == ""

    def test_defaults_from_env(self, monkeypatch):
        monkeypatch.delenv("CYBER_DATABREW_TOKEN", raising=False)
        monkeypatch.delenv("DATABREW_TOKEN", raising=False)
        auth = DatabrewTokenAuth()
        assert auth.get_headers()["X-Databrew-Token"] == ""

    def test_env_cyber_databrew_token(self, monkeypatch):
        monkeypatch.setenv("CYBER_DATABREW_TOKEN", "env-token")
        monkeypatch.delenv("DATABREW_TOKEN", raising=False)
        auth = DatabrewTokenAuth()
        assert auth.get_headers()["X-Databrew-Token"] == "env-token"

    def test_env_databrew_token_fallback(self, monkeypatch):
        monkeypatch.delenv("CYBER_DATABREW_TOKEN", raising=False)
        monkeypatch.setenv("DATABREW_TOKEN", "databrew-fallback")
        auth = DatabrewTokenAuth()
        assert auth.get_headers()["X-Databrew-Token"] == "databrew-fallback"


class TestEmailAuth:
    def test_explicit_email(self):
        auth = EmailAuth("alice@example.com")
        assert auth.get_headers() == {"X-User-Email": "alice@example.com"}

    def test_empty_email(self):
        auth = EmailAuth("")
        assert auth.get_headers()["X-User-Email"] == ""

    def test_defaults_from_env(self, monkeypatch):
        monkeypatch.delenv("CYBER_DATABREW_EMAIL", raising=False)
        auth = EmailAuth()
        assert auth.get_headers()["X-User-Email"] == ""

    def test_env_email(self, monkeypatch):
        monkeypatch.setenv("CYBER_DATABREW_EMAIL", "env@example.com")
        auth = EmailAuth()
        assert auth.get_headers()["X-User-Email"] == "env@example.com"


class TestCompositeAuth:
    def test_single_provider(self):
        auth = CompositeAuth(DatabrewTokenAuth("t1"))
        assert auth.get_headers() == {"X-Databrew-Token": "t1"}

    def test_multiple_providers(self):
        auth = CompositeAuth(
            DatabrewTokenAuth("token-123"),
            EmailAuth("user@example.com"),
        )
        headers = auth.get_headers()
        assert headers["X-Databrew-Token"] == "token-123"
        assert headers["X-User-Email"] == "user@example.com"

    def test_later_overrides_earlier(self):
        class First:
            def get_headers(self):
                return {"X-Custom": "first"}

        class Second:
            def get_headers(self):
                return {"X-Custom": "second"}

        auth = CompositeAuth(First(), Second())
        assert auth.get_headers()["X-Custom"] == "second"

    def test_empty_values_filtered(self):
        auth = CompositeAuth(
            DatabrewTokenAuth(""),
            EmailAuth("real@example.com"),
        )
        headers = auth.get_headers()
        assert "X-Databrew-Token" not in headers
        assert headers["X-User-Email"] == "real@example.com"

    def test_empty_providers(self):
        auth = CompositeAuth()
        assert auth.get_headers() == {}

    def test_accepts_custom_protocol(self):
        class CustomAuth:
            def get_headers(self):
                return {"X-Custom": "value"}

        auth = CompositeAuth(CustomAuth())
        assert auth.get_headers() == {"X-Custom": "value"}
