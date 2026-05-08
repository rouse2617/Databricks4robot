from dagster import job, op, define_asset_job, AssetSelection


@op
def ingest_mcap_op(context):
    """Placeholder op: parse MCAP footer + update mcap_files.ingest_state."""
    context.log.info("ingest_mcap_op: placeholder")


@job(name="ingest_mcap_job")
def ingest_mcap_job():
    ingest_mcap_op()


# ---------------------------------------------------------------------------
# Lakehouse sync job — triggered by pg_assets_change_sensor
# Materializes the full pipeline: bronze → silver → gold → reconciliation
# ---------------------------------------------------------------------------

lakehouse_sync_job = define_asset_job(
    name="lakehouse_sync_job",
    selection=AssetSelection.groups("bronze", "silver", "gold"),
    description=(
        "Full lakehouse sync pipeline: Postgres → Bronze → Silver → Gold + reconciliation. "
        "Triggered by pg_assets_change_sensor when Postgres data changes."
    ),
)
