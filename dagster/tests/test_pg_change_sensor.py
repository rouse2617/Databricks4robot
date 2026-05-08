"""
Unit tests for dagster/sensors/pg_change_sensor.py

Tests the sensor helper function and configuration without requiring
a running Postgres instance.
"""

import sys
import os
import types
from unittest.mock import MagicMock, patch

# ---------------------------------------------------------------------------
# Mock dagster so the module can be imported without the real package
# ---------------------------------------------------------------------------
dagster_mock = types.ModuleType("dagster")
dagster_mock.sensor = lambda **kw: (lambda fn: fn)
dagster_mock.RunRequest = MagicMock
dagster_mock.SensorEvaluationContext = MagicMock
dagster_mock.SkipReason = MagicMock
dagster_mock.DefaultSensorStatus = MagicMock()
dagster_mock.DefaultSensorStatus.STOPPED = "STOPPED"
sys.modules["dagster"] = dagster_mock

# Now import the module under test
dagster_root = os.path.join(os.path.dirname(__file__), "..")
sys.path.insert(0, dagster_root)

from sensors.pg_change_sensor import (
    PG_HOST,
    PG_PORT,
    PG_USER,
    PG_PASSWORD,
    PG_DATABASE,
    pg_assets_change_sensor,
)


class TestSensorConfiguration:
    """Test that sensor configuration defaults are correct."""

    def test_pg_host_default(self):
        assert PG_HOST == "localhost"

    def test_pg_port_default(self):
        assert PG_PORT == "5432"

    def test_pg_user_default(self):
        assert PG_USER == "postgres"

    def test_pg_database_default(self):
        assert PG_DATABASE == "data4cyber"


class TestSensorExists:
    """Verify the sensor function is defined and callable."""

    def test_sensor_is_callable(self):
        assert callable(pg_assets_change_sensor)


class FakeCursor:
    def __init__(self, existing_tables, table_watermarks):
        self.existing_tables = existing_tables
        self.table_watermarks = table_watermarks
        self._result = None

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def execute(self, query, params=None):
        if "to_regclass" in query:
            table = params[0]
            self._result = (table if table in self.existing_tables else None,)
            return

        for table, watermark in self.table_watermarks.items():
            if f"FROM {table}" in query:
                self._result = (watermark,)
                return

        self._result = (None,)

    def fetchone(self):
        return self._result


class TestGetPgChangeWatermark:
    """Test the _get_pg_change_watermark helper."""

    def test_returns_none_on_connection_error(self):
        # Mock psycopg2 to simulate a connection error
        psycopg2_mock = MagicMock()
        psycopg2_mock.connect.side_effect = Exception("Connection refused")
        with patch.dict(sys.modules, {"psycopg2": psycopg2_mock}):
            from sensors.pg_change_sensor import _get_pg_change_watermark
            result = _get_pg_change_watermark()
            assert result is None

    def test_prefers_latest_event_or_fallback_watermark(self):
        from datetime import datetime, timezone

        event_ts = datetime(2025, 7, 15, 12, 0, 0, tzinfo=timezone.utc)
        asset_ts = datetime(2025, 7, 15, 12, 5, 0, tzinfo=timezone.utc)
        cursor = FakeCursor(
            existing_tables={"asset_events", "assets"},
            table_watermarks={"asset_events": event_ts, "assets": asset_ts},
        )

        mock_conn = MagicMock()
        mock_conn.cursor.return_value = cursor

        psycopg2_mock = MagicMock()
        psycopg2_mock.connect.return_value = mock_conn

        with patch.dict(sys.modules, {"psycopg2": psycopg2_mock}):
            from sensors.pg_change_sensor import _get_pg_change_watermark
            result = _get_pg_change_watermark()
            assert result == f"assets:{asset_ts.isoformat()}"

    def test_returns_event_watermark_when_event_table_is_latest(self):
        from datetime import datetime, timezone

        event_ts = datetime(2025, 7, 15, 12, 10, 0, tzinfo=timezone.utc)
        asset_ts = datetime(2025, 7, 15, 12, 5, 0, tzinfo=timezone.utc)
        cursor = FakeCursor(
            existing_tables={"asset_events", "assets"},
            table_watermarks={"asset_events": event_ts, "assets": asset_ts},
        )

        mock_conn = MagicMock()
        mock_conn.cursor.return_value = cursor

        psycopg2_mock = MagicMock()
        psycopg2_mock.connect.return_value = mock_conn

        with patch.dict(sys.modules, {"psycopg2": psycopg2_mock}):
            from sensors.pg_change_sensor import _get_pg_change_watermark
            result = _get_pg_change_watermark()
            assert result == f"asset_events:{event_ts.isoformat()}"
