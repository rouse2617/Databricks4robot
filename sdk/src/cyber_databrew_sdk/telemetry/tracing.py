"""OpenTelemetry tracing — method-level decorators and tracer helpers.

Extends the low-level ``_tracing.py`` (HTTP request tracing) with
high-level decorators for business methods.
"""

from __future__ import annotations

import functools
import logging
from typing import Any, Callable, TypeVar

F = TypeVar("F", bound=Callable[..., Any])

_logger = logging.getLogger(__name__)

try:
    from opentelemetry import trace

    _tracer = trace.get_tracer(__name__)
    _otel_available = True
except ImportError:
    _otel_available = False
    _tracer = None  # type: ignore[assignment]


def get_tracer():
    """Get the module-level OpenTelemetry tracer."""
    return _tracer


def trace_method(span_name: str | None = None) -> Callable[[F], F]:
    """Decorator: trace a method with OpenTelemetry.

    Usage::

        @trace_method("storage.download")
        def download(self, uri, path):
            ...
    """
    def decorator(func: F) -> F:
        if not _otel_available:
            return func

        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            name = span_name or func.__qualname__
            with _tracer.start_as_current_span(name) as span:
                try:
                    result = func(*args, **kwargs)
                    span.set_attribute("success", True)
                    return result
                except Exception as e:
                    span.set_attribute("success", False)
                    span.record_exception(e)
                    raise
        return wrapper  # type: ignore[return-value]
    return decorator
