from cyber_databrew_sdk._generated.models import RunChildSummary, RunRelation


def test_run_child_summary_accepts_health_status() -> None:
    summary = RunChildSummary.model_validate(
        {
            "total": 2,
            "aggregateStatus": "Running",
            "hasFailures": True,
            "hasBlocking": False,
            "healthStatus": "degraded",
        }
    )

    assert summary.healthStatus == "degraded"


def test_run_relation_accepts_snapshot() -> None:
    relation = RunRelation.model_validate(
        {
            "id": "rel-1",
            "parentRunId": "batch-1",
            "childRunId": "run-1",
            "relationType": "batch_child",
            "source": "run_kernel",
            "snapshot": {"assetId": "asset-1"},
        }
    )

    assert relation.snapshot == {"assetId": "asset-1"}
