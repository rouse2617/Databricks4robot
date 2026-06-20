# Design - CYB-3006 Run Rerun

## API

Add:

```text
POST /api/v1/runs/{id}/rerun
```

Response: `201` with `PipelineRun`.

## Backend

Add `RerunRun(ctx, id)` to the Run Kernel service interface and pipeline usecase. It mirrors the existing legacy `RetryRun` full-rerun behavior but uses explicit product events:

- `run_rerun_requested`
- `run_rerun_failed`
- `run_rerun_created`

The new Run event payload includes:

```json
{
  "sourceRunId": "source",
  "relation": "rerun_of"
}
```

## Compatibility

The old `/pipeline-runs/{id}/retry` remains unchanged for compatibility. New clients should call `/runs/{id}/rerun` for full reruns.

## Frontend

Run Inspector includes a distinct `重新运行` operation. When a DataBrew Run id is available it calls `/runs/{id}/rerun`; external workflow debug fallback still uses Argo resubmit because no product Run exists.

## Rollback

The endpoint and wrapper methods are additive. Rolling back removes the new path without affecting existing retry/resubmit operations.
