# Pipeline Spec Delta — CYB-3677

## MODIFIED Requirements

### Requirement: Batch dispatch survives process restarts
- **Before**: The system SHALL dispatch a batch's runs from a one-shot
  in-memory goroutine started by the create call, so a backend restart
  mid-batch strands every not-yet-submitted item (job stuck `processing`,
  items `pending` forever).
- **After**: The system SHALL persist a batch as a `running` job whose
  submission is owned by the durable backfill submitter (boot-eager +
  periodic re-list), so dispatch resumes automatically after any restart and
  racing instances converge idempotently (deterministic workflow name +
  AlreadyExists). A `BATCH_DISPATCH_MODE=legacy` flag SHALL restore the old
  path for one release as rollback.
- **Reason**: Field incidents: batches silently losing tasks on rollout/OOM.

#### Scenario: Restart mid-batch resumes dispatch
- **Given** a 10-item batch with 3 items submitted
- **When** the backend process is killed and restarts
- **Then** the remaining 7 items are submitted by the submitter with no
  duplicates and no manual action

#### Scenario: Cancel is DB-state driven
- **Given** a running batch in submitter mode
- **When** the user cancels it (job status `cancelled`)
- **Then** the submitter never claims its remaining items

### Requirement: A batch is pinned to one template version
- **Before**: The system SHALL record the pinned version only in
  `filter_json`, which the submitter does not read — submitter-driven
  dispatch would use the template's current active version, mixing versions
  within one batch if the template moves mid-dispatch.
- **After**: The system SHALL pin the resolved version in the
  `backfill_jobs.template_version` column at creation; the submitter SHALL
  use that value verbatim and SHALL NOT consult the active version when the
  column is set. Legacy NULL rows fall back to `filter_json`, then to the
  active version with a warning metric
  (`backend_dispatcher_template_fallback_total`).
- **Reason**: P0 — same-batch semantic consistency and audit traceability.

#### Scenario: Template upgraded mid-batch
- **Given** a batch created against template version X
- **When** the template's active version moves to Y while the batch is still
  dispatching
- **Then** every run in the batch uses version X

### Requirement: Submitter cycles are mutually exclusive across instances
- **Before**: The system SHALL rely solely on idempotent convergence when
  multiple backend instances run submitter cycles concurrently, wasting API
  calls and duplicating Argo reconcile events under Cloud Run autoscaling.
- **After**: The system SHALL serialize submitter cycles across instances via
  a Postgres session advisory lock (auto-released on connection loss); losing
  instances skip the cycle. Lock errors degrade to running unguarded —
  correctness never depends on the lock.
- **Reason**: Instance count follows HTTP traffic; dispatch economy must not.

#### Scenario: Two instances, one cycle
- **Given** two backend instances with pending batch items
- **When** both tick their submitters simultaneously
- **Then** exactly one runs the cycle and the other skips without error
