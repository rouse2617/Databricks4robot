## MODIFIED Requirements

### Requirement: Execution list source of truth

- **Before**: The system SHALL render Pipeline execution records primarily from live Argo workflow list data.
- **After**: The system SHALL render Pipeline execution records primarily from durable DataBrew run ledger data, and SHALL use live Argo workflow data only as optional enrichment.
- **Reason**: Argo workflows may be removed by TTL, while users still need historical run status, cost, event, and asset-node information.

**Priority**: P0 (Critical)
**Rationale**: Users cannot trust cost or historical execution visibility if the list disappears when Argo removes runtime resources.

#### Scenario: Historical run keeps cost after Argo TTL cleanup

- **Given** a DataBrew pipeline run has `totalEstimatedCost` in the run ledger
- **And** the corresponding Argo workflow is no longer returned by the live workflow list
- **When** the user opens the Pipeline execution list
- **Then** the run is shown in the list
- **And** the total estimated cost is displayed from the DataBrew run ledger

#### Scenario: Live workflow without run ledger still appears

- **Given** Argo returns a live workflow that does not yet have a matching DataBrew run record
- **When** the user opens the Pipeline execution list
- **Then** the workflow is still shown as a live execution record
- **And** unavailable cost is shown as pending or unavailable based on workflow status

#### Scenario: Matching live workflow enriches ledger record

- **Given** a DataBrew run and a live Argo workflow share the same workflow name
- **When** the user opens the Pipeline execution list
- **Then** the list row uses DataBrew ledger identity and cost fields
- **And** live Argo labels, status, and timing can enrich the row without replacing ledger cost

### Requirement: Runtime workflow retention for diagnostics

- **Before**: Newly submitted Argo workflows used a 1-hour completion TTL.
- **After**: Newly submitted Argo workflows SHALL use a 30-day completion TTL.
- **Reason**: DataBrew owns durable execution history, but operators still need the live Argo object for recent DAG, pod, and log diagnostics.

**Priority**: P1 (High)
**Rationale**: Short TTL cleanup makes recent debugging harder and creates confusing 404 states immediately after successful test runs.

#### Scenario: New run keeps Argo workflow object for 30 days

- **Given** a user deploys a Pipeline run
- **When** DataBrew generates the Argo Workflow manifest
- **Then** the manifest sets `ttlStrategy.secondsAfterCompletion` to `2592000`
