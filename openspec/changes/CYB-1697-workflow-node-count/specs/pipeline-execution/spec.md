## MODIFIED Requirements

### Requirement: Pipeline execution node counts reflect business steps
- **Before**: The system could display live workflow node counts that include Argo DAG/root/controller nodes in the pipeline execution list.
- **After**: The system SHALL display pipeline execution node counts using business step nodes only, excluding Argo DAG/root/controller nodes from summary counts.
- **Reason**: Users compare list rows with execution details. Counting internal Argo nodes makes a two-step pipeline appear to have three nodes and creates false confusion about the pipeline structure.

**Priority**: P1 (High)
**Rationale**: The execution list is the main operational overview. Its node-count column must match the business DAG users created and the node detail table they inspect.

#### Scenario: list count excludes root DAG node
- **Given** an Argo workflow status contains one root DAG node and two Pod step nodes for a pipeline run
- **When** the user opens the pipeline execution list
- **Then** the node-count column shows `2`

#### Scenario: detail still renders full workflow state
- **Given** the same workflow status contains a root DAG node and two Pod step nodes
- **When** the user opens the pipeline execution detail page
- **Then** the detail page can still render DAG, timeline, node diagnostics, and the business node summary

#### Scenario: stale or missing live workflow summary falls back to ledger count
- **Given** the live workflow summary is unavailable for a pipeline run
- **When** the user opens the pipeline execution list
- **Then** the list uses the persisted DataBrew run node count instead of displaying an inflated or empty live count
