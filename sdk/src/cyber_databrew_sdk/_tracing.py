"""OpenTelemetry instrumentation for SDK HTTP requests.

Optional — only activates when ``opentelemetry-api`` is installed::

    pip install cyber-databrew-sdk[otel]

Wires into httpx ``event_hooks`` to create spans for every API request
with HTTP method, path, status code, duration, and request_id attributes.
"""

from __future__ import annotations

import httpx

try:
    from opentelemetry import trace
    from opentelemetry.trace import Span, Status, StatusCode

    _otel_available = True
except ImportError:
    _otel_available = False

    # Stub for type hints when opentelemetry is not installed
    class Span:  # type: ignore[no-redef]
        pass

    class StatusCode:  # type: ignore[no-redef]
        OK = UNSET = ERROR = "Unset"

    class Status:  # type: ignore[no-redef]
        def __init__(self, status_code: StatusCode, description: str = "") -> None: ...


_TRACER_NAME = "cyber-databrew-sdk"


def is_enabled() -> bool:
    """Check whether OpenTelemetry instrumentation is available."""
    return _otel_available


def instrument_requestor(client: httpx.Client, tracer_name: str = _TRACER_NAME) -> None:
    """Attach OpenTelemetry hooks to an ``httpx.Client``.

    Adds ``event_hooks`` for request/response lifecycle:

    * **request** — creates a span with ``http.request.method`` and
      ``url.full`` attributes.
    * **response** — sets ``http.response.status_code``, ``http.request_id``
      and records duration.  Marks span as error for 4xx/5xx responses.

    Safe to call multiple times — hooks are appended, not replaced.
    When ``opentelemetry-api`` is not installed, this is a no-op.
    """
    if not _otel_available:
        return

    tracer = trace.get_tracer(tracer_name)

    def _on_request(request: httpx.Request) -> None:
        span = tracer.start_span(
            name=f"{request.method} {request.url.path}",
            kind=trace.SpanKind.CLIENT,
            attributes={
                "http.request.method": request.method,
                "url.full": str(request.url),
                "url.path": request.url.path,
            },
        )
        # Store span on request so _on_response can close it
        # Using httpx.Request.extensions dict (part of public API since httpx 0.27)
        request.extensions["_otel_span"] = span  # type: ignore[typeddict-unknown-key]

    def _on_response(response: httpx.Response) -> None:
        span: Span | None = response.request.extensions.get("_otel_span")  # type: ignore[typeddict-unknown-key]
        if span is None:
            return
        span.set_attribute("http.response.status_code", response.status_code)
        rid = response.headers.get("X-Request-ID")
        if rid:
            span.set_attribute("http.request_id", rid)

        if response.status_code >= 400:
            span.set_status(Status(StatusCode.ERROR, f"HTTP {response.status_code}"))
        else:
            span.set_status(Status(StatusCode.OK))

        span.end()

    def _on_exception(request: httpx.Request, exc: Exception) -> None:
        span: Span | None = request.extensions.get("_otel_span")  # type: ignore[typeddict-unknown-key]
        if span is None:
            return
        span.set_attribute("error.type", type(exc).__name__)
        span.set_attribute("error.message", str(exc))
        span.set_status(Status(StatusCode.ERROR, str(exc)))
        span.end()

    client.event_hooks["request"].append(_on_request)
    client.event_hooks["response"].append(_on_response)
    # httpx 0.28+ supports "exception" hook; skip if not available
    if "exception" in client.event_hooks:
        client.event_hooks["exception"].append(_on_exception)
