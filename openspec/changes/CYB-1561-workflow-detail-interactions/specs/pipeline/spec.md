# Pipeline Spec Delta

## Modified Requirements

### Requirement: Workflow Execution Detail Node Debugging

The workflow execution detail page SHALL provide direct, low-friction node debugging actions from the DAG view.

#### Scenario: Open node logs from the DAG

- **Given** a workflow execution detail page with a displayable DAG node
- **When** the user clicks the node log action
- **Then** the page opens the log viewer for that node without requiring an intermediate drawer click
- **And** the selected node remains visually highlighted

#### Scenario: Open node runtime details from the DAG

- **Given** a workflow execution detail page with a displayable DAG node
- **When** the user clicks the node runtime action
- **Then** the page opens the node detail drawer on the runtime tab

#### Scenario: Inspect large rendered logs

- **Given** a node log response that is larger than the frontend render limit
- **When** the log viewer opens
- **Then** the page shows only the bounded visible log content
- **And** the page indicates that the content is truncated
- **And** the user can copy the currently visible log content

#### Scenario: Placeholder asset-node ledger is unavailable

- **Given** the backend asset-node ledger is not implemented
- **When** the workflow execution detail page renders
- **Then** the page reserves a compact placeholder for future asset-node status instead of a full empty table
