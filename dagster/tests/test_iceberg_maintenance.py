"""
Unit tests for dagster/jobs/iceberg_maintenance.py

Tests configuration, table lists, op definitions, job graph, and schedule
without requiring a running Spark/Iceberg stack.

We mock dagster and pyspark so the module can be imported in a lightweight
test environment.
"""

import sys
import os
import types
from unittest.mock import MagicMock

# ---------------------------------------------------------------------------
# Mock dagster so the module can be imported without the real package
# ---------------------------------------------------------------------------
dagster_mock = types.ModuleType("dagster")


def _decorator_factory(**kw):
    """Handle both @decorator and @decorator(**kwargs) patterns."""
    def wrapper(fn):
        return fn
    return wrapper


def _flexible_decorator(fn=None, **kw):
    """Support both @op and @op(description=...) usage."""
    if fn is not None:
        return fn
    return _decorator_factory(**kw)


dagster_mock.job = _flexible_decorator
dagster_mock.op = _flexible_decorator
dagster_mock.schedule = _flexible_decorator
dagster_mock.OpExecutionContext = MagicMock
dagster_mock.define_asset_job = MagicMock
dagster_mock.AssetSelection = MagicMock()
dagster_mock.ScheduleDefinition = MagicMock

# Also need to mock the ScheduleDefinition as a class that stores args
class FakeScheduleDefinition:
    def __init__(self, **kwargs):
        for k, v in kwargs.items():
            setattr(self, k, v)

dagster_mock.ScheduleDefinition = FakeScheduleDefinition

sys.modules["dagster"] = dagster_mock

# Add dagster root to path
dagster_root = os.path.join(os.path.dirname(__file__), "..")
sys.path.insert(0, dagster_root)

from jobs.iceberg_maintenance import (
    CATALOG_NS,
    ICEBERG_REST_URI,
    S3_ENDPOINT,
    SNAPSHOT_RETENTION_DAYS,
    BRONZE_TABLES,
    SILVER_TABLES,
    GOLD_TABLES,
    ALL_TABLES,
    SILVER_GOLD_TABLES,
    expire_snapshots_op,
    rewrite_data_files_op,
    remove_orphan_files_op,
    iceberg_maintenance_job,
    iceberg_maintenance_schedule,
)


class TestConfiguration:
    """Test that maintenance configuration is correct."""

    def test_catalog_namespace(self):
        assert CATALOG_NS == "rest_catalog.default"

    def test_iceberg_rest_uri_default(self):
        assert ICEBERG_REST_URI == "http://localhost:8183"

    def test_s3_endpoint_default(self):
        assert S3_ENDPOINT == "http://localhost:9000"

    def test_snapshot_retention_days(self):
        assert SNAPSHOT_RETENTION_DAYS == 7


class TestTableLists:
    """Verify table lists are complete and correct."""

    def test_bronze_tables_count(self):
        assert len(BRONZE_TABLES) == 5

    def test_bronze_tables_content(self):
        expected = {
            "bronze_assets",
            "bronze_mcap_files",
            "bronze_deliveries",
            "bronze_delivery_items",
            "bronze_asset_events",
        }
        assert set(BRONZE_TABLES) == expected

    def test_silver_tables_count(self):
        assert len(SILVER_TABLES) == 5

    def test_silver_tables_content(self):
        expected = {
            "silver_assets_current",
            "silver_asset_tags",
            "silver_asset_algo_latest",
            "silver_mcap_files_current",
            "silver_deliveries_current",
        }
        assert set(SILVER_TABLES) == expected

    def test_gold_tables_count(self):
        assert len(GOLD_TABLES) == 2

    def test_gold_tables_content(self):
        expected = {
            "gold_dataset_snapshot_items",
            "gold_asset_search_docs",
        }
        assert set(GOLD_TABLES) == expected

    def test_all_tables_is_union(self):
        assert ALL_TABLES == BRONZE_TABLES + SILVER_TABLES + GOLD_TABLES

    def test_silver_gold_tables(self):
        assert SILVER_GOLD_TABLES == SILVER_TABLES + GOLD_TABLES

    def test_all_tables_total_count(self):
        assert len(ALL_TABLES) == 12


class TestOpsExist:
    """Verify that all three maintenance ops are defined and callable."""

    def test_expire_snapshots_op_callable(self):
        assert callable(expire_snapshots_op)

    def test_rewrite_data_files_op_callable(self):
        assert callable(rewrite_data_files_op)

    def test_remove_orphan_files_op_callable(self):
        assert callable(remove_orphan_files_op)


class TestJobExists:
    """Verify the maintenance job is defined."""

    def test_iceberg_maintenance_job_callable(self):
        assert callable(iceberg_maintenance_job)


class TestSchedule:
    """Verify the daily schedule is configured correctly."""

    def test_schedule_cron_is_daily(self):
        assert iceberg_maintenance_schedule.cron_schedule == "0 3 * * *"

    def test_schedule_name(self):
        assert iceberg_maintenance_schedule.name == "iceberg_maintenance_daily"

    def test_schedule_has_job(self):
        assert iceberg_maintenance_schedule.job is iceberg_maintenance_job
