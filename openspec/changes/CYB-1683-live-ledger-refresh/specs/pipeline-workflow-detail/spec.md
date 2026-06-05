## MODIFIED Requirements

### Requirement: Execution detail live ledger refresh

- **Before**: The execution detail page MAY keep the asset-node table, run events, and cost summary fixed after initial load while the DAG continues polling live workflow status.
- **After**: The execution detail page SHALL refresh run events, asset-node rows, and cost summary while an active workflow is polled, and SHALL perform catch-up refreshes after active workflows become terminal.
- **Reason**: Users need the DAG, node table, logs, and cost state to converge without a manual reload.

**Priority**: P1 (High)

#### Scenario: Active workflow refreshes ledger data

- **Given** a DataBrew workflow detail page is open for a running workflow
- **When** the live workflow polling interval fires
- **Then** the page refreshes the live workflow and the associated run events, asset-node rows, and cost summary.

#### Scenario: Terminal workflow gets ledger catch-up

- **Given** an active workflow transitions to a terminal phase
- **When** the detail page receives the terminal workflow state
- **Then** it performs quiet follow-up ledger refreshes so delayed DataBrew write-back can appear without reloading the page.

### Requirement: Asset-node status fallback from live DAG

- **Before**: The asset-node table MAY render stale pending rows even when the live Argo DAG node is already terminal.
- **After**: The asset-node table SHALL use live Argo node status as a display fallback for matching asset-node rows when the ledger status is still pending or running.
- **Reason**: The page must not show contradictory status and resource-snapshot copy for the same node.

**Priority**: P1 (High)

#### Scenario: Ledger row lags behind successful DAG node

- **Given** a live Argo node is `Succeeded`
- **And** the matching DataBrew asset-node row still says `Pending`
- **When** the user views the asset-node table
- **Then** the row displays the live terminal status
- **And** the cost copy says cost data is unavailable rather than waiting for an active resource snapshot.
