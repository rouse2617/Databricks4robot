# CYB-3798 Design

## API contract (define first)

### 1. `GET /api/v1/backfill?createdBy=<value>`

Existing endpoint; add one optional query param. Response shape **unchanged**.

- `createdBy` (optional string): exact-match filter on `backfill_jobs.created_by`. Omitted → all jobs (current behavior).
- For subscription tasks the frontend passes `createdBy=subscription-task:<taskId>`.

Response `200`: `{ "items": BackfillJob[] }` (ordered `created_at DESC`, as today).

### 2. `GET /api/v1/backfill/:id/items` (new)

Lists the per-asset items of one batch.

Response `200`: `{ "items": BackfillItem[] }` where each item mirrors the Go model:

```json
{
  "id": "…", "jobId": "batch_…", "assetId": "…",
  "status": "pending|running|completed|failed|cancelled",
  "pipelineRunId": "…?", "workflowName": "…?", "errorMessage": "…?",
  "attempts": 0, "startedAt": "…?", "finishedAt": "…?", "createdAt": "…"
}
```

- `:id` unknown → `200` with `{"items": []}` (consistent with a batch that has no items; not a 404 — mirrors ListNodeFailures leniency). Empty `id` → `400 INVALID_ARGUMENT`.
- Auth: same `X-Databrew-Token` group as the rest of `/backfill`.

## Backend layering

- **repo** `BackfillRepo.ListJobs(ctx, ListJobsFilter)` — `ListJobsFilter{ CreatedBy *string }`. When `CreatedBy != nil` append `WHERE created_by = $1`. `FindItemsByJobID(ctx, jobID)` already exists — reused as-is for endpoint 2.
- **usecase** `ListJobs(ctx, filter)` passthrough; new `ListItems(ctx, jobID) ([]BackfillItem, error)`.
- **handler** `ListJobs` reads `c.Query("createdBy")`; new `ListItems` handler bound at `GET /backfill/:id/items`.
- **mocks** every `BackfillRepository` / usecase interface implementer (incl. `*_test.go`) updated for the changed `ListJobs` signature in the same commit.

## Response-shape alignment (contract-first)

`Frontend/src/api/subscriptionTaskApi.ts` mirrors the two responses:
- `listBatches(taskId)` → `GET /backfill?createdBy=subscription-task:${taskId}` → `BackfillJob[]` (unwrap `.items`).
- `listBatchItems(batchId)` → `GET /backfill/${batchId}/items` → `BackfillItem[]` (unwrap `.items`).

TS types for `BackfillJob` / `BackfillItem` match the OpenAPI schemas exactly (`{items: [...]}` envelope, not bare array).

## Frontend UX

- `SubscriptionTasksPanel` row → click opens a detail `Drawer`.
- Drawer top: task summary (name, subscription, bindings, last run).
- "历史下发批次" `Table`: batch name, created time, template, status, asset count (`totalCount`) — rows from `listBatches`.
- Expand a batch row → nested asset `Table` from `listBatchItems`: `assetId`, `status` (tag), link to run/node-summary when `pipelineRunId` present.
- Empty state: 从未下发过（`lastBatchIds` 空且无历史批次）→ 明确空态文案。

## Out of scope (this PR)

- Backend pagination on the two list endpoints (history size is bounded per task; revisit if a task accumulates thousands of batches).
- Persisting a dedicated dispatch-history table on the subscription task (createdBy reverse-lookup is sufficient).
- SDK client methods (internal admin surface — see `decisions.md`).
