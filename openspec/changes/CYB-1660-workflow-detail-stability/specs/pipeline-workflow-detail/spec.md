## MODIFIED Requirements

### Requirement: Execution detail identity resolution

- **Before**: The execution detail page MAY resolve primarily by route workflow name, which can fail after Argo TTL cleanup or when the durable DataBrew run ID differs from the workflow name.
- **After**: The execution detail page SHALL prefer the durable DataBrew pipeline run ID when present, and SHALL fall back through workflow name, pipeline name, and route name candidates before showing a not-found state.
- **Reason**: Users need stable access to run history and diagnostics even when live Argo workflow names are stale, missing, or ambiguous.

**Priority**: P0 (Critical)

#### Scenario: Detail link carries stable run ID

- **Given** a pipeline execution row has a DataBrew `runId`
- **When** the user opens the execution detail page
- **Then** the page first loads the DataBrew run ledger by `runId`
- **And** workflow, events, asset-node, and cost lookups use the resolved run identity.

#### Scenario: Live workflow has a different resolved name

- **Given** an execution detail route uses a stale or display-oriented name
- **And** DataBrew resolves the run to a live Argo workflow name
- **When** the user opens node logs
- **Then** log requests use the resolved live workflow name.

### Requirement: Failed execution diagnosis visibility

- **Before**: Failed workflow details MAY show an empty DAG or require users to inspect nodes manually before seeing the likely failure cause.
- **After**: The detail page SHALL show a prominent failed-node summary when failed or error nodes are available, and SHALL expose a log action for failed asset-node rows even if a stored log reference is missing.
- **Reason**: Common failures such as image pull errors must be visible without hunting through secondary panels.

**Priority**: P1 (High)

#### Scenario: Failed node reason is visible

- **Given** a workflow contains failed nodes with messages
- **When** the user opens the execution detail page
- **Then** the header shows a failed-node alert with the first failure messages.

#### Scenario: Failed asset node still has log entry

- **Given** an asset-node row is failed
- **And** the row does not contain a `logRef`
- **When** the user views the asset-node table
- **Then** the row still exposes a log action that resolves through the DAG node.

### Requirement: Execution status drift warning

- **Before**: The detail page MAY show a single status even when DataBrew ledger status and live Argo workflow or node status disagree.
- **After**: The detail page SHALL warn when ledger status and live workflow status differ, or when live workflow status conflicts with node phases.
- **Reason**: Status sync delay can mislead users about whether a run is still active or already failed.

**Priority**: P1 (High)

#### Scenario: Ledger and workflow status differ

- **Given** DataBrew records a run as terminal
- **And** live Argo still reports the workflow as running
- **When** the user opens the execution detail page
- **Then** the page shows a status synchronization warning.
