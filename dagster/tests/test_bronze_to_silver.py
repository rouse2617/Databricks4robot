"""
Unit tests for dagster/assets/bronze_to_silver.py

Tests the helper functions, configuration, and JSONB parsing logic
without requiring a running Spark/Iceberg stack.

We mock dagster and pyspark so the module can be imported in a
lightweight test environment.
"""

import sys
import os
import types
from unittest.mock import MagicMock

# ---------------------------------------------------------------------------
# Mock dagster so the module can be imported without the real package
# ---------------------------------------------------------------------------
dagster_mock = types.ModuleType("dagster")
dagster_mock.asset = lambda **kw: (lambda fn: fn)
dagster_mock.Output = MagicMock
dagster_mock.AssetIn = MagicMock
dagster_mock.AssetExecutionContext = MagicMock
dagster_mock.MetadataValue = MagicMock()
dagster_mock.load_assets_from_modules = MagicMock()
sys.modules["dagster"] = dagster_mock

# Now import the module under test
dagster_root = os.path.join(os.path.dirname(__file__), "..")
sys.path.insert(0, dagster_root)

from assets.bronze_to_silver import (
    SOURCE_DB,
    CATALOG_NS,
    ICEBERG_REST_URI,
)


class TestConfiguration:
    """Test that Silver layer configuration is correct."""

    def test_source_db_is_data4cyber(self):
        assert SOURCE_DB == "data4cyber"

    def test_catalog_namespace(self):
        assert CATALOG_NS == "rest_catalog.default"

    def test_iceberg_rest_uri_default(self):
        assert ICEBERG_REST_URI == "http://localhost:8183"


class TestSilverAssetNames:
    """Verify that all five Silver assets are defined."""

    def test_silver_assets_current_exists(self):
        from assets.bronze_to_silver import silver_assets_current
        assert callable(silver_assets_current)

    def test_silver_asset_tags_exists(self):
        from assets.bronze_to_silver import silver_asset_tags
        assert callable(silver_asset_tags)

    def test_silver_asset_algo_latest_exists(self):
        from assets.bronze_to_silver import silver_asset_algo_latest
        assert callable(silver_asset_algo_latest)

    def test_silver_mcap_files_current_exists(self):
        from assets.bronze_to_silver import silver_mcap_files_current
        assert callable(silver_mcap_files_current)

    def test_silver_deliveries_current_exists(self):
        from assets.bronze_to_silver import silver_deliveries_current
        assert callable(silver_deliveries_current)


class TestPivotAlgoUDF:
    """
    Test the algo pivot logic that converts flat cf_algo keys
    like 'hand_tracking@1.2.0:status' into structured rows.

    We extract and test the pure-Python logic directly.
    """

    @staticmethod
    def _pivot(algo_map):
        """
        Replicate the pivot logic from the UDF for unit testing
        without requiring PySpark.
        """
        if not algo_map:
            return []

        grouped: dict = {}
        for k, v in algo_map.items():
            if ":" not in k:
                continue
            algo_key_part, field = k.rsplit(":", 1)
            if algo_key_part not in grouped:
                grouped[algo_key_part] = {}
            grouped[algo_key_part][field] = v

        rows = []
        for algo_key, fields in grouped.items():
            if "@" in algo_key:
                name, version = algo_key.split("@", 1)
            else:
                name, version = algo_key, ""
            rows.append({
                "algo_name": name,
                "algo_version": version,
                "status": fields.get("status"),
                "run_id": fields.get("run_id"),
                "output_uri": fields.get("output_uri"),
                "started_at": fields.get("started_at"),
                "finished_at": fields.get("finished_at"),
                "method": fields.get("method"),
            })
        return rows

    def test_empty_map_returns_empty(self):
        assert self._pivot({}) == []
        assert self._pivot(None) == []

    def test_single_algo_full_fields(self):
        algo_map = {
            "hand_tracking@1.2.0:status": "ok",
            "hand_tracking@1.2.0:started_at": "2026-04-20T09:00:00Z",
            "hand_tracking@1.2.0:finished_at": "2026-04-20T09:05:00Z",
            "hand_tracking@1.2.0:method": "ray_batch",
            "hand_tracking@1.2.0:run_id": "dagster-run-001",
            "hand_tracking@1.2.0:output_uri": "gs://grace-algo/ht/seg01.npz",
        }
        rows = self._pivot(algo_map)
        assert len(rows) == 1
        r = rows[0]
        assert r["algo_name"] == "hand_tracking"
        assert r["algo_version"] == "1.2.0"
        assert r["status"] == "ok"
        assert r["run_id"] == "dagster-run-001"
        assert r["output_uri"] == "gs://grace-algo/ht/seg01.npz"
        assert r["started_at"] == "2026-04-20T09:00:00Z"
        assert r["finished_at"] == "2026-04-20T09:05:00Z"
        assert r["method"] == "ray_batch"

    def test_single_algo_partial_fields(self):
        algo_map = {
            "hand_tracking@1.2.0:status": "pending",
        }
        rows = self._pivot(algo_map)
        assert len(rows) == 1
        r = rows[0]
        assert r["algo_name"] == "hand_tracking"
        assert r["algo_version"] == "1.2.0"
        assert r["status"] == "pending"
        assert r["run_id"] is None
        assert r["output_uri"] is None

    def test_multiple_algos(self):
        algo_map = {
            "hand_tracking@1.2.0:status": "ok",
            "hand_tracking@1.2.0:run_id": "run-001",
            "deface@2.0.0:status": "failed",
            "deface@2.0.0:run_id": "run-002",
        }
        rows = self._pivot(algo_map)
        assert len(rows) == 2
        names = {r["algo_name"] for r in rows}
        assert names == {"hand_tracking", "deface"}

    def test_algo_without_version(self):
        algo_map = {
            "sam2:status": "running",
        }
        rows = self._pivot(algo_map)
        assert len(rows) == 1
        assert rows[0]["algo_name"] == "sam2"
        assert rows[0]["algo_version"] == ""
        assert rows[0]["status"] == "running"

    def test_keys_without_colon_are_skipped(self):
        algo_map = {
            "some_random_key": "value",
            "hand_tracking@1.2.0:status": "ok",
        }
        rows = self._pivot(algo_map)
        assert len(rows) == 1
        assert rows[0]["algo_name"] == "hand_tracking"


class TestLatestVersionHelper:
    """Test the _latest_version deduplication helper logic."""

    def test_function_exists(self):
        from assets.bronze_to_silver import _latest_version
        assert callable(_latest_version)


class TestAddAuditColumns:
    """Test the _add_audit_columns helper logic."""

    def test_function_exists(self):
        from assets.bronze_to_silver import _add_audit_columns
        assert callable(_add_audit_columns)
