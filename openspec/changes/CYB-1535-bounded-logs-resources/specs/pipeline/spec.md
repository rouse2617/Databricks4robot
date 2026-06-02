# Pipeline Spec Delta — CYB-1535 Bounded Logs And Resources

## New Requirements

### Requirement: Bounded workflow log reads
The system SHALL bound workflow log reads by default.

#### Scenario: Default log request returns bounded tail
- **Given** a workflow node has pod logs
- **When** a client requests node logs without limit parameters
- **Then** the system returns a bounded tail response and does not read the full log stream into memory.

#### Scenario: Excessive log limits are rejected or clamped
- **Given** a client requests `tailLines` or `limitBytes` above the supported maximum
- **When** the system handles the request
- **Then** the system prevents an unbounded backend read and returns documented behavior.

### Requirement: Workflow log streaming
The system SHALL expose a registered SSE endpoint for workflow node logs.

#### Scenario: Stream logs for a node
- **Given** a workflow node resolves to a pod
- **When** a client opens the log stream endpoint
- **Then** the response uses `text/event-stream` and emits structured log events.

### Requirement: Resource usage source clarity
The system SHALL report the source and meaning of workflow resource values.

#### Scenario: Metrics API is unavailable
- **Given** live Kubernetes Metrics API is not configured
- **When** a client requests workflow resources
- **Then** the response marks live metrics as unavailable while still returning manifest requests/limits and Argo resource duration when available.
