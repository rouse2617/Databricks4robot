"""Token resolution — fetch, cache, and refresh credentials.

Supports multiple sources:
- Static token (explicit)
- Environment variables
- GCP ADC (Application Default Credentials)
- gcloud CLI
- Backend-issued token
"""

from __future__ import annotations

import logging
import os
import subprocess
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone

_logger = logging.getLogger(__name__)


@dataclass
class TokenResult:
    """Resolved credential with optional expiry."""
    token: str
    source: str         # "explicit", "env", "gcloud", "adc", "backend"
    expires_at: datetime | None = None


class TokenResolver:
    """Resolve GCS access tokens from multiple sources."""

    def __init__(self) -> None:
        self._cached: TokenResult | None = None

    def resolve(self, explicit_token: str | None = None) -> TokenResult:
        """Try sources in order until one works."""
        # 1. Explicit
        if explicit_token:
            return TokenResult(token=explicit_token, source="explicit")

        # 2. Environment
        for var in ("GCS_TOKEN", "GOOGLE_ACCESS_TOKEN"):
            if val := os.environ.get(var):
                return TokenResult(token=val, source="env")

        # 3. gcloud CLI
        try:
            out = subprocess.check_output(
                ["gcloud", "auth", "print-access-token"],
                stderr=subprocess.DEVNULL, timeout=15,
            ).decode().strip()
            if out:
                return TokenResult(token=out, source="gcloud")
        except Exception:
            pass

        # 4. ADC (handled by gcsfs automatically — return empty)
        return TokenResult(token="", source="adc")

    def clear_cache(self) -> None:
        self._cached = None
