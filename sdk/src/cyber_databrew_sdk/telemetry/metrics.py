"""Metrics — counters, histograms, and gauges for SDK operations.

Usage::

    from cyber_databrew_sdk.telemetry.metrics import record_metric

    record_metric("storage.download.bytes", 1048576, {"bucket": "my-bucket"})
"""

from __future__ import annotations

import logging
from typing import Any

_logger = logging.getLogger(__name__)

try:
    from opentelemetry import metrics
    from opentelemetry.metrics import Histogram, Counter

    _meter = metrics.get_meter(__name__)
    _otel_available = True
except ImportError:
    _otel_available = False
    _meter = None


def record_metric(name: str, value: Any, attributes: dict[str, str] | None = None) -> None:
    """Record a metric value.

    Falls back to logging when OpenTelemetry is not installed.
    """
    if _otel_available and _meter:
        instrument = _meter.create_histogram(
            name=f"sdk.{name}",
            description=f"SDK metric: {name}",
        )
        instrument.record(value, attributes=attributes or {})
    else:
        _logger.debug("metric %s = %s %s", name, value, attributes)
