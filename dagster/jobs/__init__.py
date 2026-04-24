from dagster import job, op


@op
def ingest_mcap_op(context):
    """Placeholder op: parse MCAP footer + update mcap_files.ingest_state."""
    context.log.info("ingest_mcap_op: placeholder")


@job(name="ingest_mcap_job")
def ingest_mcap_job():
    ingest_mcap_op()
