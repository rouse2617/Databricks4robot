# Spec — algo-runs (delta)

## Requirement: algo run registration

The platform SHALL persist each algorithm execution as a row in `algo_runs` with a client-supplied 16-character alphanumeric `run_id`.

### Scenario: worker registers a batch run

**Given** a valid `run_id`, `algo_name`, `algo_version`, and `triggered_by`
**When** the client calls `POST /api/v1/algo-runs`
**Then** the response is 201 with `status=pending` and the same `run_id`

### Scenario: lifecycle completion

**Given** a pending run
**When** the client calls `start` then `finish` with `status=ok`
**Then** the run row has `status=ok`, `finished_at` set, and optional counters populated

## Requirement: run_id integrity on projections

**When** `POST /assets/{id}/algo/{key}/start` or `finish` includes `run_id`
**Then** the run MUST exist in `algo_runs` or the API returns 400

**When** finish succeeds with a registered `run_id`
**Then** an `algo_run_applied` event is appended for that asset in the same transaction
