# Spec delta — tags

## ADDED

### Requirement: `source` and `city` are baseline tags

`backend/config/tag_registry.yaml` MUST declare two additional entries under `tags:`:

- `source` — string, open-vocabulary. Records the platform that produced the underlying asset (`vibecap`, `grace`, `cybercap`, or free string for future platforms).
- `city` — string, `max_length: 32`. Records the city where the asset was collected (parsed from GPS address on ingest, or set by admin).

Both are visible via `GET /api/v1/tag-registry`. Both accept writes through `POST /api/v1/assets/{id}/tags` with any of the existing `source_type` values (`human`, `algo_sdk`, `rule_engine`, `system`, `compliance`), subject to the same source-name / source-version invariants that apply to all `type: string` baseline tags.

#### Scenario: tag-registry surfaces the new baseline entries

- **When** a client calls `GET /api/v1/tag-registry`
- **Then** the response `items` array includes an object with `key="source", type="string"` and an object with `key="city", type="string", max_length=32`.

#### Scenario: pangzi mcap gains task/source/city tags after backfill

- **Given** the backfill script `scripts/backfill_pangzi_metadata_tags_dev.sh` has run once on the current pangzi mcap set (`owner="pangzi-consumer"`, ~140 mcap)
- **When** a client queries `POST /api/v1/queries/run` with `where: {pred: {field: "tags.source", op: "eq", value: "vibecap"}}`
- **Then** the total is `>= 140` (all pangzi mcap have `source:vibecap` tag).

#### Scenario: task tag reaches values from VibeCap

- **Given** the same post-backfill state, and at least one pangzi mcap whose `metadata.vibecap_tasks` includes `"备餐操作"`
- **When** a client queries `POST /queries/run` with `where: {pred: {field: "tags.task", op: "eq", value: "备餐操作"}}`
- **Then** the total is `> 0` and the returned items all include a `task:备餐操作` entry in their `tags_detailed[]`.

#### Scenario: backfill is idempotent

- **Given** the backfill script has run once and populated the pangzi tags
- **When** the script is re-run against the same set (whether via cron, cleanup, or manual re-invocation)
- **Then** the script MUST exit success without duplicating rows. `POST /assets/:id/tags` is an Upsert; re-posting the same `(key, value, source_type, source_name)` tuple is a no-op at the DB level.
