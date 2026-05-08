from dagster import sensor, RunRequest, SensorEvaluationContext


@sensor(job_name="ingest_mcap_job", minimum_interval_seconds=30)
def mcap_upload_sensor(context: SensorEvaluationContext):
    """
    Placeholder: polls Pub/Sub gcs.mcap.finalized.v1 for new MCAP uploads.
    Week 4: replace with real Pub/Sub pull + RunRequest per message.
    """
    context.log.info("mcap_upload_sensor: no new messages (placeholder)")
    return []
