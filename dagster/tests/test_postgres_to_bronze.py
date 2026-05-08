"""
Unit tests for dagster/assets/postgres_to_bronze.py

Tests the helper functions and configuration without requiring
a running Spark/Postgres/Iceberg/Dagster stack.

We mock the dagster dependency since it may not be installed in the
test environment, then import the module's pure-logic helpers.
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

from assets.postgres_to_bronze import (
    _pg_jdbc_url,
    _pg_jdbc_props,
    JSONB_COLUMNS,
    ALL_TABLES,
    INCREMENTAL_TABLES,
    CREATED_AT_TABLES,
    SOURCE_DB,
)


class TestConfiguration:
    """Test that table mappings and config are correct."""

    def test_all_tables_includes_incremental_and_created_at(self):
        assert set(ALL_TABLES.keys()) == (
            set(INCREMENTAL_TABLES.keys()) | set(CREATED_AT_TABLES.keys())
        )

    def test_five_source_tables(self):
        expected = {
            "assets", "mcap_files", "deliveries",
            "delivery_items", "asset_algo_events",
        }
        assert set(ALL_TABLES.keys()) == expected

    def test_five_bronze_tables(self):
        expected = {
            "bronze_assets", "bronze_mcap_files", "bronze_deliveries",
            "bronze_delivery_items", "bronze_asset_events",
        }
        assert set(ALL_TABLES.values()) == expected

    def test_jsonb_columns_include_cf_fields(self):
        for col in ("cf_meta", "cf_algo", "cf_tag", "cf_files"):
            assert col in JSONB_COLUMNS

    def test_source_db_is_data4cyber(self):
        assert SOURCE_DB == "data4cyber"


class TestJdbcHelpers:
    """Test JDBC URL and properties construction."""

    def test_pg_jdbc_url_format(self):
        url = _pg_jdbc_url()
        assert url.startswith("jdbc:postgresql://")
        assert "data4cyber" in url

    def test_pg_jdbc_props_has_driver(self):
        props = _pg_jdbc_props()
        assert props["driver"] == "org.postgresql.Driver"
        assert "user" in props
        assert "password" in props


class TestTableMappings:
    """Test incremental vs created_at table classification."""

    def test_assets_uses_updated_at(self):
        assert "assets" in INCREMENTAL_TABLES

    def test_mcap_files_uses_updated_at(self):
        assert "mcap_files" in INCREMENTAL_TABLES

    def test_deliveries_uses_updated_at(self):
        assert "deliveries" in INCREMENTAL_TABLES

    def test_delivery_items_uses_created_at(self):
        assert "delivery_items" in CREATED_AT_TABLES

    def test_asset_algo_events_uses_created_at(self):
        assert "asset_algo_events" in CREATED_AT_TABLES

    def test_bronze_table_naming_convention(self):
        for src, bronze in ALL_TABLES.items():
            assert bronze.startswith("bronze_"), f"{bronze} should start with bronze_"
