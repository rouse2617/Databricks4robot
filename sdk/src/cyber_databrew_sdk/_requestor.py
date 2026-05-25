"""Core request engine — HTTP transport + error mapping + response parsing.

Adapted from Stripe's _APIRequestor. This is the single place
where HTTP requests actually happen. Every manager delegates to
this object.

Key design decisions from Stripe:
- request_id extracted from response headers (X-Request-ID)
- error response body parsed into typed exceptions
- single shared instance across all managers (like Stripe's
  self._requestor passed to every service)
"""

from __future__ import annotations

from typing import Any
from urllib.parse import urljoin

import httpx

from cyber_databrew_sdk.exceptions import (
    APIConnectionError,
    map_status_to_error,
)


class APIRequestor:
    """Single request engine shared across all managers.

    Analogous to Stripe's _APIRequestor — holds HTTP client
    config + executes requests + maps errors.
    """

    _client: httpx.Client
    _base_url: str
    _auth_headers: dict[str, str]

    def __init__(
        self,
        *,
        base_url: str,
        auth_headers: dict[str, str],
        timeout: float,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._auth_headers = auth_headers
        self._client = http_client or httpx.Client(timeout=timeout)

    def close(self) -> None:
        self._client.close()

    def _build_url(self, path: str) -> str:
        return urljoin(self._base_url, path)

    def _extract_request_id(self, headers: httpx.Headers) -> str | None:
        return headers.get("X-Request-ID") or headers.get("x-request-id")

    def request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Execute an HTTP request and return parsed JSON body.

        On HTTP error, raises the appropriate typed exception
        with request_id attached.
        """
        url = self._build_url(path)

        headers = {
            "Content-Type": "application/json",
            **self._auth_headers,
        }

        try:
            resp = self._client.request(
                method=method,
                url=url,
                params=_clean_params(params),
                json=json_body,
                headers=headers,
            )
        except httpx.TimeoutException as e:
            raise APIConnectionError(
                f"Request timed out: {method} {path}"
            ) from e
        except httpx.ConnectError as e:
            raise APIConnectionError(
                f"Connection failed: {method} {path} — {e}"
            ) from e

        return self._interpret_response(resp)

    def _interpret_response(self, resp: httpx.Response) -> dict[str, Any]:
        """Parse successful response or raise typed exception."""
        request_id = self._extract_request_id(resp.headers)

        if resp.is_success:
            if resp.status_code == 204:
                return {}
            return resp.json()

        # Parse backend ErrorBody — mirrors httpresp/response.go:9-15
        import json

        try:
            body = resp.json()
        except (json.JSONDecodeError, ValueError):
            body = {}

        message = body.get("message") or resp.text or f"HTTP {resp.status_code}"
        code = body.get("code")
        details = body.get("details")
        # Body request_id takes precedence over header (design decision)
        request_id = body.get("request_id") or request_id

        raise map_status_to_error(
            resp.status_code,
            message,
            code=code,
            request_id=request_id,
            details=details,
        )


def _clean_params(params: dict[str, Any] | None) -> dict[str, Any] | None:
    """Remove None values from query params (Stripe-style sanitization)."""
    if params is None:
        return None
    return {k: v for k, v in params.items() if v is not None}
