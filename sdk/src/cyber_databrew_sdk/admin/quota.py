"""Quota management — usage tracking and limits."""

from __future__ import annotations

import logging
from dataclasses import dataclass

_logger = logging.getLogger(__name__)


@dataclass
class QuotaInfo:
    """Resource quota information."""
    resource: str
    used: int
    limit: int
    remaining: int


class QuotaManager:
    """Track and manage resource quotas.

    Resources include storage (GCS), API requests, concurrent runs, etc.
    """

    def __init__(self, requestor) -> None:
        self._requestor = requestor

    def get_usage(self, user_id: str, resource: str) -> QuotaInfo:
        """Get current usage for a user/resource."""
        raise NotImplementedError

    def list_quotas(self, user_id: str) -> list[QuotaInfo]:
        """List all quotas for a user."""
        raise NotImplementedError
