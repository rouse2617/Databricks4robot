"""AdminSearchManager — search reindex operations (admin)."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class AdminSearchManager(BaseManager):
    """Admin operations for Elasticsearch reindexing."""

    def reindex(self, payload: dict[str, Any] | None = None) -> dict[str, Any]:
        """Trigger a full search reindex."""
        return self._request(
            "POST",
            self._endpoint("admin_search_reindex"),
            json_body=payload,
        )

    def create_reindex_job(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Create a search reindex job."""
        return self._request(
            "POST",
            self._endpoint("admin_search_reindex_job_create"),
            json_body=payload,
        )

    def list_reindex_jobs(self, **params: Any) -> dict[str, Any]:
        """List all search reindex jobs."""
        return self._request(
            "GET",
            self._endpoint("admin_search_reindex_job_list"),
            params=params,
        )

    def get_reindex_job(self, job_id: str) -> dict[str, Any]:
        """Get a single reindex job by ID."""
        return self._request(
            "GET",
            self._endpoint("admin_search_reindex_job_get", job_id=job_id),
        )

    def stop_reindex_job(self, job_id: str) -> dict[str, Any]:
        """Stop a running reindex job."""
        return self._request(
            "POST",
            self._endpoint("admin_search_reindex_job_stop", job_id=job_id),
        )

    def resume_reindex_job(self, job_id: str) -> dict[str, Any]:
        """Resume a paused reindex job."""
        return self._request(
            "POST",
            self._endpoint("admin_search_reindex_job_resume", job_id=job_id),
        )

    def abandon_reindex_job(self, job_id: str) -> dict[str, Any]:
        """Abandon a reindex job."""
        return self._request(
            "POST",
            self._endpoint("admin_search_reindex_job_abandon", job_id=job_id),
        )
