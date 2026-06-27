"""Telemetry — OpenTelemetry tracing, metrics, and logging.

Usage::

    from cyber_databrew_sdk.telemetry import trace_method

    class MyService:
        @trace_method("my_operation")
        def do_thing(self):
            ...
"""

from cyber_databrew_sdk.telemetry.tracing import trace_method, get_tracer
from cyber_databrew_sdk.telemetry.metrics import record_metric

__all__ = [
    "trace_method",
    "get_tracer",
    "record_metric",
]
