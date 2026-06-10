"""Tests for exception hierarchy and map_status_to_error."""

from __future__ import annotations

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
    map_status_to_error,
)


class TestExceptionHierarchy:
    def test_base_error_is_exception(self):
        assert issubclass(CyberDatabrewError, Exception)

    def test_all_subclasses(self):
        assert issubclass(BadRequestError, CyberDatabrewError)
        assert issubclass(AuthenticationError, CyberDatabrewError)
        assert issubclass(NotFoundError, CyberDatabrewError)
        assert issubclass(ConflictError, CyberDatabrewError)
        assert issubclass(ValidationError, CyberDatabrewError)
        assert issubclass(RateLimitError, CyberDatabrewError)
        assert issubclass(ServerError, CyberDatabrewError)
        assert issubclass(APIConnectionError, CyberDatabrewError)


class TestErrorMessage:
    def test_message_without_request_id(self):
        err = CyberDatabrewError("something broke")
        assert str(err) == "something broke"

    def test_message_with_request_id(self):
        err = NotFoundError("not found", request_id="req_abc")
        assert str(err) == "Request req_abc: not found"

    def test_message_with_all_fields(self):
        err = ConflictError(
            "duplicate",
            code="CONCURRENT_CONFLICT",
            request_id="req_xyz",
            http_status=409,
            details={"version": 2},
        )
        assert err.code == "CONCURRENT_CONFLICT"
        assert err.http_status == 409
        assert err.details == {"version": 2}
        assert "req_xyz" in str(err)

    def test_repr(self):
        err = BadRequestError("bad", code="INVALID", http_status=400, request_id="req_r")
        r = repr(err)
        assert "BadRequestError" in r
        assert "INVALID" in r
        assert "req_r" in r


class TestMapStatusToError:
    def test_400(self):
        err = map_status_to_error(400, "bad")
        assert isinstance(err, BadRequestError)
        assert err.http_status == 400

    def test_401(self):
        err = map_status_to_error(401, "unauth")
        assert isinstance(err, AuthenticationError)

    def test_404(self):
        err = map_status_to_error(404, "gone")
        assert isinstance(err, NotFoundError)

    def test_409(self):
        err = map_status_to_error(409, "conflict")
        assert isinstance(err, ConflictError)

    def test_422(self):
        err = map_status_to_error(422, "invalid")
        assert isinstance(err, ValidationError)

    def test_429(self):
        err = map_status_to_error(429, "rate")
        assert isinstance(err, RateLimitError)

    def test_500(self):
        err = map_status_to_error(500, "internal")
        assert isinstance(err, ServerError)

    def test_503(self):
        err = map_status_to_error(503, "unavailable")
        assert isinstance(err, ServerError)

    def test_unexpected_418(self):
        err = map_status_to_error(418, "teapot")
        assert isinstance(err, CyberDatabrewError)
        assert err.http_status == 418

    def test_code_and_details_passed_through(self):
        err = map_status_to_error(
            404, "gone", code="ASSET_NOT_FOUND", request_id="req_1", details={"asset_id": "x"}
        )
        assert err.code == "ASSET_NOT_FOUND"
        assert err.request_id == "req_1"
        assert err.details == {"asset_id": "x"}
