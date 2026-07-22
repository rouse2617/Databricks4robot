# CYB-3801 Design

## Message format

```json
{"asset_ids": ["a", "b", "c"], "topic": "optional-label"}   // → batch (>1)
{"asset_ids": ["a"], "topic": "optional-label"}             // → single run
```

Message struct (forward-extensible — unknown keys ignored):

```go
type assetMessage struct {
    AssetIDs []string `json:"asset_ids"`
    Topic    string   `json:"topic,omitempty"` // reserved (CYB-3801): future routing/labeling; parsed, not yet acted on
}
```

`extractAssetIDs(data) (ids []string, topic string, err error)`: unmarshal; trim + drop empty strings in `asset_ids`; error when none remain (malformed → acked/skipped as today). No legacy `asset_id` handling (dropped — user confirmed no back-compat needed). `topic` returned for logging/future use.

## Consumer (`pubsub.go`)

`PullResult.IDs []string` → `PullResult.Messages [][]string` (one inner slice per Pub/Sub message, preserving that message's asset set). Ack/Nack stay pull-level (all ackIDs together). Malformed messages still acked-to-skip.

## Dispatch (`subtask/usecase.go`)

New interface alongside `BatchCreator`:

```go
type RunCreator interface {
    CreateRun(ctx, templateID, name, assetID, targetID string, templateVersion int, owner string) (runID string, err error)
}
```

`executeTask` — for each `msgAssets` in `result.Messages`, for each binding:
- `len(msgAssets) == 1` → `RunCreator.CreateRun(...)` (single pipeline run)
- `len(msgAssets) > 1` → `BatchCreator.CreateBatch(...)` (batch)

Naming: run `"<task>-<ts>-p<i>"`, batch `"<task>-<ts>-p<i>"` (unchanged). All-success → Ack + RecordSuccess(collected ids); any error → Nack + failure notify. Empty pull → RunStatusEmpty (unchanged).

## Wiring (`core.go`)

`RunCreator` impl calls `puc.CreateRunByTemplateID(ctx, tmpl, name, []string{assetID}, DeployOptions{TargetID: target, Owner: owner, TemplateVersion: ver})`. Returns `run.ID`. Owner persists to `pipeline_runs.owner`.

## Runs reverse-lookup (`pipeline`)

- `models.PipelineRunListFilter` gains `CreatedBy string`.
- `ListRunSummaries` repo query: `AND COALESCE(owner,'') = $n` when `CreatedBy != ""`.
- `ListRuns` handler reads `c.Query("createdBy")` → filter. `GET /api/v1/runs?createdBy=subscription-task:<id>` returns that task's single runs. Response shape unchanged (`{items, total}`).

## Frontend (drawer merge)

`subscriptionTaskApi`:
- `listSubscriptionTaskRuns(taskId)` → `GET /runs?createdBy=subscription-task:<id>&excludeBatchParents=true` → `PipelineRun[]`.
- keep `listSubscriptionTaskBatches`.

Drawer: build one time-sorted list, each row normalized to `{ kind: 'run'|'batch', id, name, status, assetCount, createdAt, link }`. A 类型 tag column distinguishes them. Runs link to the run detail (`/pipeline/run/:id` or existing route), batches to `/pipeline/batch/:id`. Empty state unchanged.

## Out of scope

- Per-message ack (keep pull-level; at-least-once dup on retry, documented).
- Any migration (owner column exists).
- Changing how batches themselves execute.
