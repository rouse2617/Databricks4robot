"""Tests for APIRequestor — HTTP transport, error mapping, response parsing."""

from __future__ import annotations

import httpx
import pytest
import respx

from cyber_databrew_sdk._requestor import APIRequestor
from cyber_databrew_sdk.exceptions import (
    APIConnectionError,
    AuthenticationError,
    BadRequestError,
    ConflictError,
    CyberDatabrewError,
    NotFoundError,
    RateLimitError,
    ServerError,
    ValidationError,
)


@pytest.fixture
def requestor():
    return APIRequestor(
        base_url="http://test",
        auth_headers={"X-Grace-Token": "t"},
        timeout=30.0,
    )


class TestRequestorSuccess:
    @respx.mock
    def test_get_returns_json(self, requestor):
        route = respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(200, json={"key": "value"})
        )
        result = requestor.request("GET", "/api/v1/test")
        assert result == {"key": "value"}
        assert route.calls.last.request.headers["X-Grace-Token"] == "t"

    @respx.mock
    def test_post_with_body(self, requestor):
        route = respx.post("http://test/api/v1/test").mock(
            return_value=httpx.Response(201, json={"id": "1"})
        )
        result = requestor.request("POST", "/api/v1/test", json_body={"name": "x"})
        assert result == {"id": "1"}
        assert route.calls.last.request.content == b'{"name":"x"}'

    @respx.mock
    def test_query_params_sent(self, requestor):
        route = respx.get("http://test/api/v1/test?page=1&size=20").mock(
            return_value=httpx.Response(200, json={})
        )
        requestor.request("GET", "/api/v1/test", params={"page": 1, "size": 20})
        assert route.calls.last.request.url.params["page"] == "1"

    @respx.mock
    def test_none_params_filtered(self, requestor):
        route = respx.get("http://test/api/v1/test?only=keep").mock(
            return_value=httpx.Response(200, json={})
        )
        requestor.request(
            "GET", "/api/v1/test", params={"only": "keep", "skip": None}
        )
        assert "skip" not in route.calls.last.request.url.params

    @respx.mock
    def test_204_returns_empty_dict(self, requestor):
        respx.delete("http://test/api/v1/test/1").mock(
            return_value=httpx.Response(204)
        )
        result = requestor.request("DELETE", "/api/v1/test/1")
        assert result == {}


class TestRequestorErrors:
    @respx.mock
    def test_400_maps_to_bad_request(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(400, json={"message": "bad input", "code": "INVALID_ARGUMENT"})
        )
        with pytest.raises(BadRequestError) as exc:
            requestor.request("GET", "/api/v1/test")
        assert exc.value.code == "INVALID_ARGUMENT"

    @respx.mock
    def test_401_maps_to_authentication(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(401)
        )
        with pytest.raises(AuthenticationError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_403_maps_to_authentication(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(403)
        )
        with pytest.raises(AuthenticationError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_404_maps_to_not_found(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(404, json={"message": "not found"})
        )
        with pytest.raises(NotFoundError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_409_maps_to_conflict(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(409)
        )
        with pytest.raises(ConflictError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_422_maps_to_validation(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(422)
        )
        with pytest.raises(ValidationError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_429_maps_to_rate_limit(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(429)
        )
        with pytest.raises(RateLimitError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_500_maps_to_server_error(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(500)
        )
        with pytest.raises(ServerError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_503_maps_to_server_error(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(503)
        )
        with pytest.raises(ServerError):
            requestor.request("GET", "/api/v1/test")

    @respx.mock
    def test_request_id_in_headers(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(404, headers={"X-Request-ID": "req_abc"})
        )
        with pytest.raises(NotFoundError) as exc:
            requestor.request("GET", "/api/v1/test")
        assert exc.value.request_id == "req_abc"

    @respx.mock
    def test_request_id_from_error_body(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(
                404,
                json={"message": "gone", "request_id": "req_body"},
            )
        )
        with pytest.raises(NotFoundError) as exc:
            requestor.request("GET", "/api/v1/test")
        # request_id from headers since _requestor extracts it from response headers
        assert exc.value.request_id == "req_body"

    @respx.mock
    def test_unexpected_status_maps_to_base(self, requestor):
        respx.get("http://test/api/v1/test").mock(
            return_value=httpx.Response(418)
        )
        with pytest.raises(CyberDatabrewError):
            requestor.request("GET", "/api/v1/test")


class TestRequestorConnectionErrors:
    def test_timeout(self, requestor):
        transport = httpx.MockTransport(lambda _: (_ for _ in []).throw(httpx.TimeoutException("timeout")))
        client = httpx.Client(transport=transport)
        r = APIRequestor(
            base_url="http://test",
            auth_headers={},
            timeout=30.0,
            http_client=client,
        )
        with pytest.raises(APIConnectionError, match="timed out"):
            r.request("GET", "/api/v1/test")

    def test_connection_refused(self, requestor):
        transport = httpx.MockTransport(lambda _: (_ for _ in []).throw(httpx.ConnectError("refused")))
        client = httpx.Client(transport=transport)
        r = APIRequestor(
            base_url="http://test",
            auth_headers={},
            timeout=30.0,
            http_client=client,
        )
        with pytest.raises(APIConnectionError, match="Connection failed"):
            r.request("GET", "/api/v1/test")


class TestRequestorBuildUrl:
    def test_build_url_simple(self, requestor):
        url = requestor._build_url("/api/v1/test")
        assert url == "http://test/api/v1/test"

    def test_build_url_with_base_path(self):
        r = APIRequestor(
            base_url="http://test/api/v1",
            auth_headers={},
            timeout=30.0,
        )
        url = r._build_url("assets")
        assert url == "http://test/api/assets"
