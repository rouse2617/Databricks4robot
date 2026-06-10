"""EvalMetricsManager — evaluation results and quality metrics."""

from __future__ import annotations

from typing import Any

from cyber_databrew_sdk._base_manager import BaseManager


class EvalMetricsManager(BaseManager):
    """Eval results and quality metrics for assets."""

    def report_eval_result(self, asset_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Report an evaluation result for an asset."""
        return self._request(
            "POST",
            self._endpoint("eval_result_report", asset_id=asset_id),
            json_body=payload,
        )

    def list_eval_results(self, asset_id: str, **params: Any) -> dict[str, Any]:
        """List evaluation results for an asset."""
        return self._request(
            "GET",
            self._endpoint("eval_result_list", asset_id=asset_id),
            params=params,
        )

    def list_metrics(self, asset_id: str, **params: Any) -> dict[str, Any]:
        """List quality metrics for an asset."""
        return self._request(
            "GET",
            self._endpoint("metric_list", asset_id=asset_id),
            params=params,
        )

    def get_registry(self) -> dict[str, Any]:
        """Get the metrics registry (available metric definitions)."""
        return self._request("GET", self._endpoint("metric_registry"))

    def search_by_metrics(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Search assets by metric criteria."""
        return self._request(
            "POST", self._endpoint("metric_search"), json_body=payload
        )
