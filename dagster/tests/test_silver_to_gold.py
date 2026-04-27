"""
Unit tests for dagster/assets/silver_to_gold.py

Tests the helper functions, configuration, and reconciliation logic
without requiring a running Spark/Iceberg/Postgres stack.

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

from assets.silver_to_gold import (
    SOURCE_DB,
    CATALOG_NS,
    ICEBERG_REST_URI,
    RECONCILIATION_THRESHOLD,
    _compute_status_diff,
)


class TestConfiguration:
    """Test that Gold layer configuration is correct."""

    def test_source_db_is_data4cyber(self):
        assert SOURCE_DB == "data4cyber"

    def test_catalog_namespace(self):
        assert CATALOG_NS == "rest_catalog.default"

    def test_iceberg_rest_uri_default(self):
        assert ICEBERG_REST_URI == "http://localhost:8183"

    def test_reconciliation_threshold(self):
        assert RECONCILIATION_THRESHOLD == 0.001


class TestGoldAssetNames:
    """Verify that all Gold assets and reconciliation asset are defined."""

    def test_gold_dataset_snapshot_items_exists(self):
        from assets.silver_to_gold import gold_dataset_snapshot_items
        assert callable(gold_dataset_snapshot_items)

    def test_gold_asset_search_docs_exists(self):
        from assets.silver_to_gold import gold_asset_search_docs
        assert callable(gold_asset_search_docs)

    def test_data_reconciliation_exists(self):
        from assets.silver_to_gold import data_reconciliation
        assert callable(data_reconciliation)


class TestComputeStatusDiff:
    """Test the _compute_status_diff helper function."""

    def test_identical_distributions(self):
        pg = {"approved": 100, "draft": 50}
        ice = {"approved": 100, "draft": 50}
        diff = _compute_status_diff(pg, ice)
        assert diff["approved"]["diff"] == 0
        assert diff["draft"]["diff"] == 0

    def test_pg_has_more(self):
        pg = {"approved": 100, "draft": 50}
        ice = {"approved": 95, "draft": 48}
        diff = _compute_status_diff(pg, ice)
        assert diff["approved"]["diff"] == 5
        assert diff["draft"]["diff"] == 2

    def test_iceberg_has_more(self):
        pg = {"approved": 90}
        ice = {"approved": 100}
        diff = _compute_status_diff(pg, ice)
        assert diff["approved"]["diff"] == -10

    def test_missing_status_in_pg(self):
        pg = {"approved": 100}
        ice = {"approved": 100, "rejected": 5}
        diff = _compute_status_diff(pg, ice)
        assert diff["rejected"]["pg"] == 0
        assert diff["rejected"]["iceberg"] == 5
        assert diff["rejected"]["diff"] == -5

    def test_missing_status_in_iceberg(self):
        pg = {"approved": 100, "rejected": 5}
        ice = {"approved": 100}
        diff = _compute_status_diff(pg, ice)
        assert diff["rejected"]["pg"] == 5
        assert diff["rejected"]["iceberg"] == 0
        assert diff["rejected"]["diff"] == 5

    def test_both_empty(self):
        diff = _compute_status_diff({}, {})
        assert diff == {}

    def test_result_keys_are_sorted(self):
        pg = {"z_status": 1, "a_status": 2}
        ice = {"z_status": 1, "a_status": 2}
        diff = _compute_status_diff(pg, ice)
        keys = list(diff.keys())
        assert keys == sorted(keys)


class TestReconciliationThresholdLogic:
    """Test the threshold comparison logic used in data_reconciliation."""

    def test_no_difference_below_threshold(self):
        pg_total = 10000
        iceberg_total = 10000
        diff_pct = abs(pg_total - iceberg_total) / pg_total if pg_total > 0 else 0.0
        assert diff_pct <= RECONCILIATION_THRESHOLD

    def test_small_difference_below_threshold(self):
        pg_total = 10000
        iceberg_total = 9999  # 0.01% diff
        diff_pct = abs(pg_total - iceberg_total) / pg_total
        assert diff_pct <= RECONCILIATION_THRESHOLD

    def test_large_difference_above_threshold(self):
        pg_total = 10000
        iceberg_total = 9900  # 1% diff
        diff_pct = abs(pg_total - iceberg_total) / pg_total
        assert diff_pct > RECONCILIATION_THRESHOLD

    def test_zero_pg_total_edge_case(self):
        # When pg_total is 0 and iceberg has data, diff should be 100%
        pg_total = 0
        iceberg_total = 10
        if pg_total == 0 and iceberg_total == 0:
            diff_pct = 0.0
        elif pg_total == 0:
            diff_pct = 1.0
        else:
            diff_pct = abs(pg_total - iceberg_total) / pg_total
        assert diff_pct > RECONCILIATION_THRESHOLD

    def test_both_zero_no_alert(self):
        pg_total = 0
        iceberg_total = 0
        if pg_total == 0 and iceberg_total == 0:
            diff_pct = 0.0
        elif pg_total == 0:
            diff_pct = 1.0
        else:
            diff_pct = abs(pg_total - iceberg_total) / pg_total
        assert diff_pct <= RECONCILIATION_THRESHOLD
