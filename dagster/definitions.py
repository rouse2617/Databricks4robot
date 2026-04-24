"""Dagster Definitions — entry point for dagster dev / dagster-webserver."""

from dagster import Definitions, load_assets_from_modules

from dagster.jobs import ingest_mcap_job
from dagster.sensors import mcap_upload_sensor
from dagster import assets as dagster_assets

defs = Definitions(
    assets=load_assets_from_modules([dagster_assets]),
    jobs=[ingest_mcap_job],
    sensors=[mcap_upload_sensor],
)
