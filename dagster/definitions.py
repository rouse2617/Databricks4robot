"""Dagster Definitions — entry point for dagster dev / dagster-webserver."""

from dagster import Definitions, load_assets_from_modules

from dagster.jobs import ingest_mcap_job, lakehouse_sync_job
from dagster.jobs.iceberg_maintenance import (
    iceberg_maintenance_job,
    iceberg_maintenance_schedule,
)
from dagster.sensors import mcap_upload_sensor
from dagster.sensors.pg_change_sensor import pg_assets_change_sensor
from dagster import assets as dagster_assets
from dagster.assets import postgres_to_bronze
from dagster.assets import bronze_to_silver
from dagster.assets import silver_to_gold
from dagster.assets import gold_to_opensearch

defs = Definitions(
    assets=load_assets_from_modules([
        dagster_assets,
        postgres_to_bronze,
        bronze_to_silver,
        silver_to_gold,
        gold_to_opensearch,
    ]),
    jobs=[ingest_mcap_job, lakehouse_sync_job, iceberg_maintenance_job],
    sensors=[mcap_upload_sensor, pg_assets_change_sensor],
    schedules=[iceberg_maintenance_schedule],
)
