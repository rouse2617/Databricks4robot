"""Typed exception hierarchy — mirrors backend ErrorBody envelope.

Backend error response (httpresp/response.go:9-15):
    {"code": "ASSET_NOT_FOUND", "message": "...", "request_id": "...", "details": {...}}

Every SDK exception carries all four fields so users can branch on
error code (e.g. "CONCURRENT_CONFLICT" vs "DUPLICATE_ASSET_ID")
without parsing strings.

Pattern: Stripe's error.py (request_id in __str__)
         + OpenAI's _make_status_error (status → typed exception factory)
"""

from __future__ import annotations

from typing import Any


class CyberDatabrewError(Exception):
    """Base for all SDK exceptions — mirrors backend ErrorBody."""

    message: str
    code: str | None
    request_id: str | None
    http_status: int | None
    details: dict[str, Any] | None

    def __init__(
        self,
        message: str,
        *,
        code: str | None = None,
        request_id: str | None = None,
        http_status: int | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        self.message = message
        self.code = code
        self.request_id = request_id
        self.http_status = http_status
        self.details = details
        super().__init__(message)

    def __str__(self) -> str:
        if self.request_id is not None:
            return f"Request {self.request_id}: {self.message}"
        return self.message

    def __repr__(self) -> str:
        return (
            f"{self.__class__.__name__}("
            f"code={self.code!r}, "
            f"message={self.message!r}, "
            f"http_status={self.http_status!r}, "
            f"request_id={self.request_id!r})"
        )


class BadRequestError(CyberDatabrewError):
    """400 — invalid request body or query params."""


class AuthenticationError(CyberDatabrewError):
    """401, 403 — credentials missing, invalid, or insufficient."""


class NotFoundError(CyberDatabrewError):
    """404 — resource does not exist."""


class ConflictError(CyberDatabrewError):
    """409 — duplicate, concurrent modification, or idempotency conflict."""


class ValidationError(CyberDatabrewError):
    """422 — invalid state transition, business rule violation."""


class RateLimitError(CyberDatabrewError):
    """429 — too many requests."""


class ServerError(CyberDatabrewError):
    """5xx — backend internal error or service unavailable."""


class APIConnectionError(CyberDatabrewError):
    """Network-level failure (timeout, DNS, connection refused)."""


# ---------------------------------------------------------------------------
# Status code → exception class — mirrors backend response helpers
#
# Backend                                → SDK exception (status mapping)
# ──────────────────────────────────────────────────────────────────────
# httpresp.BadRequest(400)               → BadRequestError
# httpresp.Unauthorized(401)             → AuthenticationError
# ad-hoc httpresp.Error(403, ...)        → AuthenticationError
# httpresp.NotFound(404)                 → NotFoundError
# httpresp.Conflict(409)                 → ConflictError
# httpresp.Error(414, "URI_TOO_LONG")    → BadRequestError
# httpresp.Unprocessable(422)            → ValidationError
# httpresp.Error(429, "RATE_LIMITED")    → RateLimitError
# httpresp.Internal(500)                 → ServerError
# httpresp.Error(503, ...)               → ServerError
# ---------------------------------------------------------------------------
_STATUS_TO_ERROR: dict[int, type[CyberDatabrewError]] = {
    400: BadRequestError,
    401: AuthenticationError,
    403: AuthenticationError,
    404: NotFoundError,
    409: ConflictError,
    414: BadRequestError,
    422: ValidationError,
    429: RateLimitError,
}


def map_status_to_error(
    status_code: int,
    message: str,
    *,
    code: str | None = None,
    request_id: str | None = None,
    details: dict[str, Any] | None = None,
) -> CyberDatabrewError:
    """Map HTTP status + backend ErrorBody fields → typed exception.

    Mirrors the backend error envelope (httpresp.ErrorBody).
    The `code` field carries backend error codes like
    "ASSET_NOT_FOUND", "CONCURRENT_CONFLICT", "RATE_LIMITED".
    """
    exc_class = _STATUS_TO_ERROR.get(status_code)
    if exc_class is not None:
        return exc_class(
            message,
            code=code,
            request_id=request_id,
            http_status=status_code,
            details=details,
        )
    if 500 <= status_code < 600:
        return ServerError(
            message,
            code=code,
            request_id=request_id,
            http_status=status_code,
            details=details,
        )
    return CyberDatabrewError(
        message,
        code=code,
        request_id=request_id,
        http_status=status_code,
        details=details,
    )
