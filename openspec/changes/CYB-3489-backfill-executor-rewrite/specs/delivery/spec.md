## ADDED Requirements

### Requirement: Argo owns queueing/execution; backend only submits and projects
Backend SHALL NOT maintain its own execution work-queue for batch subtasks (no lease/visibility-timeout reaper, no long-lived claim-based worker pool that owns an item for the whole workflow lifetime). Argo owns queueing, parallelism, execution and transient retry; backend owns only ensuring submission and projecting Argo truth into the DB.

**Priority**: P0 (Critical)
**Rationale**: The self-built queue duplicated Argo and mis-modeled the unit of work (item `running` spanned the whole workflow while the lease was 120s), which is the common root of the oscillation / orphan / read-latency bugs. (Review doc §02 double-queue + §03 state-machine.)

#### Scenario: No reaper reclaim of live items
- **Given** a batch item whose Argo workflow is running normally
- **When** the reconcile cycle runs
- **Then** the item is NOT reclaimed or reset to `pending` on a time-based lease

### Requirement: Submission is idempotent, resumable, and self-healing
The submitter SHALL ensure every batch item is submitted to Argo exactly once, treating an item as submitted only once its run has a persisted Argo workflow UID, and MUST make progress without a process restart.

**Priority**: P0 (Critical)
**Rationale**: CYB-2774 (Cloud Run kills in-memory goroutines on redeploy) is the only durable need; workflows survive redeploy in Argo, so only submission progress must be durable. (Review doc §背景 rebuild-anchor.)

#### Scenario: Redeploy mid-dispatch self-heals
- **Given** a batch with un-submitted `pending` items after the dispatch process was killed by a redeploy
- **When** the next periodic submitter cycle runs
- **Then** the remaining `pending` items are submitted without any manual resume

#### Scenario: Already-exists is success
- **Given** an asset whose workflow already exists in Argo
- **When** the submitter re-attempts submission
- **Then** the AlreadyExists (gRPC 6 / HTTP 409) response is caught and treated as submitted (UID backfilled), not a failure
- **And** because the deterministic name is job-scoped (`backfill-{job_id}-{asset_id}-attempt-{N}`), the existing workflow is necessarily this job's own prior submission, never an unrelated job's

#### Scenario: Concurrent submitter cycles do not double-attempt
- **Given** multiple backend instances (or an overlapping periodic cycle and the initial materialize dispatch) selecting `pending` items concurrently
- **When** each picks its bounded batch via `FOR UPDATE SKIP LOCKED`
- **Then** no two workers attempt the same item concurrently, and AlreadyExists remains a last-resort backstop rather than the primary concurrency control

#### Scenario: Automatic re-submission does not bump attempts
- **Given** an item whose prior submission crashed before the UID was persisted
- **When** the next submitter cycle re-submits it
- **Then** it reuses the SAME deterministic name (attempt counter unchanged) so the retry is idempotent; `attempts` is only advanced by an operator's manual retry

#### Scenario: Deterministic name keyed on job + asset (no generateName)
- **Given** a batch item for a given (job_id, asset_id, attempt)
- **When** the submitter builds the Argo workflow name
- **Then** the name is deterministic and keyed on job_id (e.g. `backfill-{job_id}-{asset_id}-attempt-{N}`), never Argo `generateName`
- **And** two different jobs running the same pipeline on the same asset MUST NOT collide on the `pipeline_runs.workflow_name` UNIQUE constraint

#### Scenario: Concurrent same-asset across jobs is permitted (no cross-job guard)
- **Given** two different jobs each dispatching the same asset concurrently
- **When** both submit
- **Then** each gets its own independent item/run/workflow (idempotency is scoped per `(job_id, asset_id)`, NOT globally per asset)
- **And** the executor does NOT enforce an "at most one active run per asset" guard; the asset-keyed result (`algo_run_results`, keyed `(asset_id, algo_key, version)`) is last-writer-wins, which is an accepted, pre-existing behavior

### Requirement: Item status is forward-only
`backfill_items.status` SHALL progress `pending → submitted → completed|failed` with no `running → pending` regression. Automatic projection MUST NOT move status backward; the ONLY sanctioned backward transition is an explicit, operator-initiated, bounded manual retry (`failed → running`, see the retry requirement).

**Priority**: P0 (Critical)

#### Scenario: No oscillation
- **Given** a submitted item whose Argo workflow is queued (Pending phase)
- **When** the projector observes it
- **Then** the item stays `submitted` and is never flipped back to `pending`

### Requirement: Reads are pure projections
`GetJob`, batch node-summary, and batch item lists SHALL read the persisted ledger and aggregates only, making no Argo API calls on the request path.

**Priority**: P0 (Critical)
**Rationale**: Read latency was O(failed-item count) × Argo latency (measured 12.7s at failed=76). (Review doc §03 tangle B.)

#### Scenario: Read latency independent of failed count
- **Given** a settled batch with many failed items
- **When** a client opens the batch node-summary
- **Then** the response is served from the ledger in well under one second with zero Argo calls

#### Scenario: No Argo client on the read code path (structural)
- **Given** the `GetJob` / node-summary / list request handlers
- **When** the code is wired
- **Then** no Argo client is injected into those code paths, so sync/reconcile logic cannot be re-added to a read request without a structural change

### Requirement: A single background reconciler owns Argo→DB projection
State convergence SHALL be owned by one background path driven primarily by the Argo exit-hook webhook, backstopped by a periodic active-run poll and a slower drift sweep; overlapping read-path reconcilers MUST NOT remain. Continuous projection SHALL be workflow-level only (phase + progress `N/M` from Argo `status.progress`, one row per workflow); per-step node detail SHALL NOT be eagerly projected for the whole batch, but fetched lazily and cached when a single workflow is opened.

**Priority**: P1 (High)

#### Scenario: Webhook loss is backstopped
- **Given** an Argo workflow finished but its webhook was lost
- **When** the active-run poll or drift sweep runs
- **Then** the item is projected to its terminal status within the sweep interval

#### Scenario: Drift sweep is bounded and batched
- **Given** a large number of items in `submitted`/`running`
- **When** the drift sweep runs
- **Then** it queries only rows whose status is `submitted` or `running` (never a full-table scan)
- **And** it reconciles against Argo in bounded batches (≈100) with rate limiting, so it does not spike the Argo/K8s API or exhaust memory

#### Scenario: Workflow missing in Argo does not hang
- **Given** an item recorded as `submitted`/`running` in the DB
- **When** the reconciler finds no corresponding workflow in Argo (GC-collected or force-deleted) while its TTL window (30d) should still cover it
- **Then** the item is moved to an explicit anomaly/terminal state and an alert is raised, and it is NOT left hanging in `running` indefinitely

#### Scenario: Node detail is not eagerly projected at scale
- **Given** a batch of 1000 workflows × 100 steps (100k nodes)
- **When** the reconciler projects continuously
- **Then** it stores only workflow-level phase + progress per workflow (≈1000 rows), and does NOT continuously write per-node rows for all 100k nodes
- **And** when an operator opens one workflow, that workflow's ~100 node states are fetched on demand and cached

#### Scenario: Node detail is archived once at terminal
- **Given** a workflow that reaches a terminal phase
- **When** its exit-hook webhook is processed
- **Then** that workflow's final per-node snapshot is persisted exactly once (not on every poll), so step detail is later served from the ledger even after the Argo object is garbage-collected
- **And** a still-running workflow's node detail is instead fetched live from Argo on drill-in

### Requirement: Unsubmitted and failed are distinct
The projection SHALL distinguish an item never successfully submitted (no Argo workflow / empty UID) from one whose Argo workflow reached Failed/Error; a placeholder item MUST NOT be counted or shown as failed.

**Priority**: P0 (Critical)
**Rationale**: The historical bug counted placeholders (never submitted) as failed, inflating failure counts and hiding stuck work. (Review doc §09 key invariant.)

#### Scenario: Placeholder is re-submitted, not failed
- **Given** an item whose run has an empty Argo workflow UID
- **When** the executor reconciles it
- **Then** it is re-submitted, and it is NOT reported as a failure

#### Scenario: Genuine Argo failure surfaces
- **Given** an item whose Argo workflow reached Failed after retries
- **When** the projector observes it
- **Then** it is reported as failed and is NOT auto-resubmitted

### Requirement: Retry is layered by failure class
Infrastructure/transient failures SHALL be retried by Argo `retryStrategy` (retryPolicy OnError) with a per-step `activeDeadlineSeconds`; genuine application failures SHALL surface and MUST NOT be auto-resubmitted. Manual retry SHALL resume the existing workflow from the failed step via Argo `retryWorkflow` (reusing the same workflow name/UID, preserving already-succeeded steps), bounded by an attempt cap; it falls back to a fresh full re-submission only when the failed workflow no longer exists in Argo (GC'd past TTL).

**Priority**: P1 (High)

#### Scenario: Scheduling starvation is waiting, not failure
- **Given** a workflow whose pods cannot schedule (insufficient cluster capacity), for any duration
- **When** the projector observes it
- **Then** the run keeps its active phase (queued/waiting) and the item stays in-flight — it is NEVER auto-failed on a scheduling timeout
- **And** the read side surfaces the scheduler diagnostics as a blocking reason ("unschedulable"), so users see waiting-plus-why instead of a false failure
- **And** only the workload's own errors (or deterministic config errors such as a bad image) may reach `failed`

#### Scenario: Transient failure auto-retries in Argo
- **Given** a step killed by node loss or spot preemption
- **When** Argo evaluates the retry strategy
- **Then** the step is retried in-workflow without backend involvement

#### Scenario: Deterministic failure does not loop
- **Given** a step that exits non-zero on bad data after Argo retries are exhausted
- **When** the projector records the result
- **Then** the item is failed and is not automatically resubmitted

#### Scenario: Manual retry resumes from the failed step
- **Given** a failed workflow of 100 steps where 99 succeeded and 1 failed, still present in Argo
- **When** an operator retries the item (under the attempt cap)
- **Then** the same workflow is retried via `retryWorkflow`, re-running only the failed step and its descendants while keeping the 99 succeeded steps, and the item moves `failed → running`

#### Scenario: Retry after GC falls back to fresh submit
- **Given** a failed item whose workflow has been garbage-collected past its TTL
- **When** an operator retries it
- **Then** a fresh full workflow is submitted (whole re-run) rather than a step-level resume

### Requirement: Projection is attempt-versioned and monotonic
The item SHALL carry an `attempts` version that a manual retry advances, and all projection writes SHALL be monotonic on `(attempts, progress numerator)`, so a legitimate retry is not blocked by the terminal-no-regression guard and out-of-order/stale observations cannot regress state.

**Priority**: P0 (Critical)
**Rationale**: `retryWorkflow` reuses the same workflow UID; the existing terminal-no-regression guard (`persistRunObservation`) would otherwise discard the retry's `Running` observation and deadlock the item at `failed`. Multiple instances (max 5) also deliver observations out of order.

#### Scenario: Retry breaks the terminal lock
- **Given** a `failed` item whose retry has bumped `attempts` from N to N+1 in the same transaction that reset its status to `running`
- **When** the projector later observes the reused workflow as `Running`
- **Then** the observation is accepted (its attempt N+1 ≥ stored), NOT discarded by the terminal-no-regression guard

#### Scenario: Stale observation from the prior attempt is rejected
- **Given** an item at attempt N+1
- **When** a delayed observation carrying attempt N arrives
- **Then** it is rejected (lower version), so state does not regress

#### Scenario: Progress does not flicker
- **Given** two concurrent instances writing progress `35/100` and `38/100` out of order within the same attempt
- **When** both writes land
- **Then** the stored progress is monotonic (`38` wins; a later `35` write is rejected), and a new attempt resets the baseline rather than being pinned by a stale high value

#### Scenario: Double-retry is de-duplicated without a lock service
- **Given** two concurrent "retry" clicks on the same failed item
- **When** each attempts `UPDATE ... SET status='running', attempts=attempts+1 WHERE id=? AND status='failed'`
- **Then** exactly one update affects a row and calls `retryWorkflow`; the other affects zero rows and returns "already retrying" without calling Argo (no external lock service required)

#### Scenario: Argo not-retryable error is benign
- **Given** a `retryWorkflow` call against a workflow already in `Running`
- **When** Argo returns the "not in a failed state" error
- **Then** it is surfaced as a friendly "already retrying" message, NOT treated as an infrastructure failure

#### Scenario: Terminal archival survives a lost webhook
- **Given** a workflow that succeeded but whose exit-hook webhook was lost
- **When** the active poll or drift sweep observes the terminal phase and is about to write the item terminal
- **Then** if no node snapshot exists locally it first fetches and archives the node detail, then writes terminal — so drill-in still has step detail after Argo GC
