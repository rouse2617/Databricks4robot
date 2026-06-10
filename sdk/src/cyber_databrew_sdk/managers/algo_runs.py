"""AlgoRunManager — first-class algorithm run lifecycle."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class AlgoRunManager(BaseManager):
    """Manage algorithm execution runs."""

    def get(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("algo_run_get", run_id=run_id))

    def list(
        self,
        *,
        asset_id: str | None = None,
        algo_key: str | None = None,
        status: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict[str, Any]:
        return self._request(
            "GET",
            self._endpoint("algo_run_list"),
            params={
                "asset_id": asset_id,
                "algo_key": algo_key,
                "status": status,
                "page": page,
                "page_size": page_size,
            },
        )

    def create(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", self._endpoint("algo_run_create"), json_body=payload)

    def start(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("algo_run_start", run_id=run_id))

    def finish(self, run_id: str, payload: dict[str, Any] | None = None) -> dict[str, Any]:
        return self._request("POST", self._endpoint("algo_run_finish", run_id=run_id), json_body=payload)

    def cancel(self, run_id: str) -> dict[str, Any]:
        return self._request("POST", self._endpoint("algo_run_cancel", run_id=run_id))

    def get_affected_assets(self, run_id: str) -> dict[str, Any]:
        return self._request("GET", self._endpoint("algo_run_affected_assets", run_id=run_id))
