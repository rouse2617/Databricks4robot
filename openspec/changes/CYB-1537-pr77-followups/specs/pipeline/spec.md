# Pipeline Spec Delta — CYB-1537 PR #77 Review Follow-up

## New Requirements

### Requirement: Malformed template body is rejected
The system SHALL reject malformed JSON request bodies on `POST /api/v1/pipeline-runs/template/{id}` with a 400 response.

#### Scenario: Empty body is allowed
- **Given** the template endpoint accepts an optional body
- **When** a client posts with no body or an empty body
- **Then** the system invokes the usecase with the zero-value body and behaves identically to the previous implementation.

#### Scenario: Malformed JSON is rejected
- **Given** a client posts a body that is not valid JSON
- **When** the handler binds the request
- **Then** the system returns 400 with a clear error message and does not invoke the usecase.

### Requirement: Resource usage uses the first-class run repo
The system SHALL look up workflow resource usage via the first-class `pipeline_runs` table when available.

#### Scenario: Run row exists
- **Given** a workflow with a row in `pipeline_runs`
- **When** a client calls `GET /api/v1/workflows/{name}/resources`
- **Then** the system resolves the run via `pipelineRepo.FindByWorkflowName` and does not scan the legacy `pipeline_deployments` table.

#### Scenario: Run row missing, legacy deployment exists
- **Given** a workflow with a legacy `pipeline_deployments` row but no `pipeline_runs` row
- **When** a client calls `GET /api/v1/workflows/{name}/resources`
- **Then** the system falls back to scanning `pipeline_deployments` and returns the same response shape as before.

### Requirement: SSE log streaming avoids unnecessary byte conversions
The system SHALL stream pod logs without converting each line to a `[]byte` for length computation or slicing.

#### Scenario: Line byte accounting uses string length
- **Given** the SSE log streamer enforces a `limitBytes` budget
- **When** it processes each scanned line
- **Then** it uses `len(line)` (string length) to track emitted bytes and `line[:remaining]` to truncate, with no intermediate `[]byte` allocation per line.
