# Runtime OS Spec Delta — CYB-3072

## ADDED Requirements

### Requirement: DataBrew keeps video durations in sync from Grace

The system SHALL periodically fetch source-video durations from Grace and upsert them into the video duration store, so the batch subtask list's duration column stays populated automatically for batches created by any source.

**Priority**: P1 (High)
**Rationale**: `video_durations` is otherwise only populated by a one-off manual backfill; automating the pull (source-agnostic) removes manual steps and covers UI/API/grace-sync batches alike.

#### Scenario: Durations appear automatically after a batch is created
- **Given** a batch is created whose assets are Grace videos with recorded durations
- **When** a sync cycle runs (periodic, or the post-creation kick)
- **Then** those videos' durations are stored in the video duration store
- **Then** the batch subtask list shows the durations without any manual database write

#### Scenario: Re-sync is idempotent
- **Given** a video's duration is already stored
- **When** the sync runs again
- **Then** the stored duration is updated in place with no duplicate rows

#### Scenario: Videos without a Grace duration are left absent
- **Given** an asset that is not a Grace video, or a Grace video with no recorded duration
- **When** the sync runs
- **Then** no duration row is created for it and the subtask list shows it as empty

### Requirement: Grace sync is best-effort and non-blocking

The system SHALL treat Grace synchronization as best-effort: Grace being slow or unavailable MUST NOT block batch creation, the subtask list read path, or crash the server.

**Priority**: P0 (Critical)
**Rationale**: Grace is an external, Cloudflare-fronted dependency; it must never sit on a user-facing or write path.

#### Scenario: Grace unavailable does not break the product
- **Given** Grace is unreachable or returns errors
- **When** a sync cycle runs
- **Then** the error is logged and swallowed, the server keeps running, and the next cycle retries
- **Then** batch creation and the subtask list continue to work (durations simply stay as they were)

#### Scenario: Sync disabled when unconfigured
- **Given** Grace credentials / URL are not configured
- **When** the server starts
- **Then** the sync loop is not started and no Grace calls are made
