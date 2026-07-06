## MODIFIED Requirements

### Requirement: Pipeline run pods carry cost-tracking labels
- **Before**: The system SHALL attach labels identifying the batch job, template, and owner to every pod created for a submitted pipeline run, whenever the corresponding identifier is known at submission time — but the backfill batch execution path did not forward the job's owner to the deploy call, so its pods never carried the owner label even when the job's creator was known.
- **After**: The system SHALL attach the owner label for backfill batch items too, sourced from the owning job's creator, consistent with every other deploy path.
- **Reason**: `executeItem` had `job.CreatedBy` in scope but never read it into the deploy call's owner field — an omission, not an intentional gap.

#### Scenario: Backfill batch item deployed for a job with a known creator
- **Given** a backfill batch job was created by an authenticated user and that user's identity is stored as the job's creator
- **When** `executeItem` deploys one of that job's items
- **Then** the resulting pods carry an owner label equal to the job's creator, in addition to the batch-job and template labels
