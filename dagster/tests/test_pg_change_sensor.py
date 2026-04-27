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


class TestGetPgMaxUpdatedAt:
    """Test the _get_pg_max_updated_at helper."""

    def test_returns_none_on_connection_error(self):
        # Mock psycopg2 to simulate a connection error
        psycopg2_mock = MagicMock()
        psycopg2_mock.connect.side_effect = Exception("Connection refused")
        with patch.dict(sys.modules, {"psycopg2": psycopg2_mock}):
            from sensors.pg_change_sensor import _get_pg_max_updated_at
            result = _get_pg_max_updated_at()
            assert result is None

    def test_returns_iso_string_on_success(self):
        from datetime import datetime, timezone

        ts = datetime(2025, 7, 15, 12, 0, 0, tzinfo=timezone.utc)
        mock_cursor = MagicMock()
        mock_cursor.fetchone.return_value = (ts,)
        mock_cursor.__enter__ = MagicMock(return_value=mock_cursor)
        mock_cursor.__exit__ = MagicMock(return_value=False)

        mock_conn = MagicMock()
        mock_conn.cursor.return_value = mock_cursor

        psycopg2_mock = MagicMock()
        psycopg2_mock.connect.return_value = mock_conn

        with patch.dict(sys.modules, {"psycopg2": psycopg2_mock}):
            from sensors.pg_change_sensor import _get_pg_max_updated_at
            result = _get_pg_max_updated_at()
            assert result == ts.isoformat()
