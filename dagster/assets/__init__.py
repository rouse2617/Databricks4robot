"""Dagster software-defined assets for cyber-databrew."""

from dagster import asset, AssetIn, Output
import dagster


@asset(group_name="mcap")
def mcap_summary(context: dagster.AssetExecutionContext) -> Output[dict]:
    """
    Placeholder: reads MCAP footer and extracts summary metadata.
    Week 2: replace with real go-mcap footer parse via GCS.
    """
    context.log.info("mcap_summary: placeholder — no-op in Phase 0 scaffold")
    return Output({"status": "placeholder"})


@asset(group_name="assets", ins={"mcap_summary": AssetIn()})
def auto_tag(context: dagster.AssetExecutionContext, mcap_summary: dict) -> Output[dict]:
    """
    Placeholder: calls the auto-tagging model on a segment.
    Week 4: wire real SAM2 mock via Ray.
    """
    context.log.info("auto_tag: placeholder — input=%s", mcap_summary)
    return Output({"tags": [], "status": "placeholder"})
