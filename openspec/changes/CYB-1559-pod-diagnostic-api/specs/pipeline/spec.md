## ADDED Requirements

### Requirement: Workflow node Pod diagnostics
The system SHALL expose Kubernetes Pod diagnostics for a workflow node through a backend-mediated DataBrew API.

**Priority**: P1 (High)
**Rationale**: Users need Pod-level debugging information for Argo steps, but the browser must not talk to Kubernetes directly.

#### Scenario: Inspect Pod diagnostics for a workflow node
- **Given** a workflow exists and the selected node resolves to a Kubernetes Pod
- **When** the user opens the node Pod diagnostics view
- **Then** the system returns Pod namespace, Pod name, Pod IP, service account, restart count, container statuses, Pod conditions, and Pod events

#### Scenario: Kubernetes diagnostics unavailable
- **Given** the workflow node exists but Kubernetes access is not configured, unreachable, or unauthorized
- **When** the user opens the node Pod diagnostics view
- **Then** the system returns a clear unavailable or forbidden error and the frontend keeps showing existing Argo node metadata

#### Scenario: Pod no longer exists
- **Given** the workflow node resolves to a Pod that has not been created or has already been cleaned up
- **When** the user opens the node Pod diagnostics view
- **Then** the system returns `POD_NOT_FOUND` without hiding workflow/node status and logs

### Requirement: Kubernetes diagnostics contract sync
The system SHALL document and test the workflow node Pod diagnostics endpoint as part of the same change that introduces it.

**Priority**: P1 (High)
**Rationale**: The frontend depends on a stable response shape, and API drift would make debugging data disappear silently.

#### Scenario: API contract describes Pod diagnostics
- **Given** the Pod diagnostics API is added
- **When** engineers inspect OpenAPI or the API guide
- **Then** they can see the path, path parameters, response fields, and at least one error behavior
