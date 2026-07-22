# Subscription task — single-run vs batch dispatch (delta)

## MODIFIED: message format & per-message dispatch

### Requirement: a message carrying one asset dispatches a single run; more than one dispatches a batch

Each Pub/Sub message SHALL be dispatched independently. A message's asset set is `asset_ids` (array). For each pipeline binding: one asset → one pipeline run; more than one → one batch. The message MAY carry a reserved `topic` field and other future keys, which are parsed leniently and currently not acted on.

#### Scenario: single-asset message → single run
- **Given** a task bound to one template, enabled
- **When** a message `{"asset_ids":["a"]}` is consumed
- **Then** one pipeline run is created for asset `a`, owner `subscription-task:<taskId>`, and the message is acked

#### Scenario: reserved fields tolerated
- **When** a message `{"asset_ids":["a"], "topic":"x", "future":"y"}` is consumed
- **Then** it dispatches normally on `asset_ids`; unknown keys are ignored; `topic` is parsed (not acted on)

#### Scenario: multi-asset message → batch
- **Given** the same task
- **When** a message `{"asset_ids":["a","b"]}` is consumed
- **Then** one batch (backfill job) with 2 items is created, created_by `subscription-task:<taskId>`

#### Scenario: fan-out unchanged
- **Given** a task with two bindings
- **When** a single-asset message arrives
- **Then** two single runs are created (one per binding)

#### Scenario: malformed message
- **When** a message has no non-empty `asset_ids`
- **Then** it is acked and skipped (logged), not dispatched

## ADDED: filter pipeline runs by creator

### Requirement: `GET /api/v1/runs` accepts optional `createdBy`

The runs list SHALL accept an optional `createdBy` query param filtering on the run's `owner`.

#### Scenario: list a subscription task's single runs
- **Given** a task has dispatched two single runs
- **When** GET `/api/v1/runs?createdBy=subscription-task:<taskId>`
- **Then** those two runs are returned (response shape `{items, total}` unchanged)
