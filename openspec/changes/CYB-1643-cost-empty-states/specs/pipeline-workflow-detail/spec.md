## MODIFIED Requirements

### Requirement: Workflow detail cost empty states match runtime state
- **Before**: The system SHALL show one generic unavailable-cost warning when cost rows exist but no estimated cost is available.
- **After**: The system SHALL distinguish cost snapshot pending, completed-without-cost, and cost unavailable states in workflow detail.
- **Reason**: A pending node has not produced resource duration yet and should not look like a backend pricing configuration failure.

**Priority**: P1 (High)
**Rationale**: Cost information is operational context; misleading warnings slow down debugging and make healthy running workflows look misconfigured.

#### Scenario: Pending node has no resource snapshot yet
- **Given** a workflow detail page contains a Pending or Running business node
- **When** the cost summary has no estimated cost for that node
- **Then** the UI explains that the cost snapshot is waiting for runtime resource data

#### Scenario: Completed node has no cost data
- **Given** a workflow detail page contains a completed business node
- **When** no estimated cost is available for that node
- **Then** the UI shows a quiet no-cost-data state without implying the workflow is misconfigured

#### Scenario: Cost configuration is unavailable
- **Given** the run cost summary is explicitly unavailable
- **When** no node is still waiting for runtime resource data
- **Then** the UI may show pricing/config unavailable copy

#### Scenario: Diagnostics remain available
- **Given** a workflow detail page has no estimated cost
- **When** the user opens logs, Pod diagnostics, or event timeline
- **Then** those controls remain visible and are not blocked by the cost empty state
