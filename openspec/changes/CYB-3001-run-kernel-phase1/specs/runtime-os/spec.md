# Runtime OS Spec Delta — CYB-3001

## ADDED Requirements

### Requirement: Run API is the product execution entry point
The system SHALL expose Run as the product execution API while preserving legacy execution APIs.

**Priority**: P0 (Critical)
**Rationale**: Product execution state must stop depending on Argo workflow names as the primary key.

#### Scenario: Listing runs through the product API
- **Given** durable DataBrew run records exist
- **When** a client requests the Run collection
- **Then** the response contains DataBrew runs and does not require live Argo workflows to exist

#### Scenario: Legacy APIs remain compatible
- **Given** an existing client still calls pipeline-run, workflow, or deployment endpoints
- **When** the client sends the request after Run API is introduced
- **Then** the legacy endpoint remains registered and keeps its compatibility behavior

### Requirement: Run detail exposes stable ledger subresources
The system SHALL expose Run events, nodes, asset nodes, cost, inputs, outputs, children, and runtime references from stable DataBrew data where available.

**Priority**: P0 (Critical)
**Rationale**: Run Inspector needs a single stable source for product data even when runtime debug data is absent.

#### Scenario: Run detail survives runtime object cleanup
- **Given** a Run exists in the DataBrew ledger and its runtime Workflow may be missing
- **When** a client requests Run detail and ledger subresources
- **Then** the API returns ledger-backed data or explicit empty states instead of failing because the runtime object is gone

#### Scenario: Unknown run returns not found
- **Given** no DataBrew Run exists for an id
- **When** a client requests Run detail or a Run subresource
- **Then** the API returns a not-found error

### Requirement: Workflow name links resolve to Run identity
The system SHALL resolve legacy workflow-name links to their owning Run when possible.

**Priority**: P1 (High)
**Rationale**: Existing shared links should keep working while new product links use runId.

#### Scenario: Workflow name has a matching Run
- **Given** a workflow name is stored on a Run
- **When** a client resolves the workflow name through the Run API
- **Then** the response contains the owning Run with its stable run id

#### Scenario: Workflow name has no matching Run
- **Given** a workflow name has no ledger-backed Run
- **When** a client resolves the workflow name through the Run API
- **Then** the API returns not found so the client can fall back to runtime debug

### Requirement: Execution Hub uses Runs as its product source
The system SHALL use the Run API as the primary source for product execution lists and SHALL treat live runtime workflow data only as optional debug enrichment.

**Priority**: P0 (Critical)
**Rationale**: Execution Hub is the main product list; mixing live Argo workflows with ledger runs reintroduces workflowName as a product key and makes TTL-cleaned workflows destabilize the UI.

#### Scenario: Execution Hub loads product rows from Runs
- **Given** the Execution Hub is opened outside a runtime-debug-only view
- **When** the page loads execution records
- **Then** it requests the Run API collection and renders rows keyed by stable run id

#### Scenario: Live workflow data does not create product-only rows
- **Given** a live Argo workflow exists but no DataBrew Run owns it
- **When** the Execution Hub refreshes with optional workflow enrichment enabled
- **Then** the workflow can affect debug labels or filtering but is not appended as a product execution row

#### Scenario: Batch-scoped execution lists use child Runs
- **Given** a batch detail embeds an execution list for a batch id
- **When** the embedded list loads
- **Then** it requests batch-scoped Runs and renders child Run rows rather than querying legacy pipeline-run paths

### Requirement: Run operations distinguish retry from resubmit
The system SHALL make runtime retry and new-run resubmit explicit operations on the Run API.

**Priority**: P0 (Critical)
**Rationale**: Operators need to know whether an action mutates the existing runtime execution or creates a new product execution record.

#### Scenario: Runtime retry keeps the same Run
- **Given** a Run has a runtime workflow reference
- **When** a client posts to `/api/v1/runs/{id}/retry`
- **Then** the backend requests a runtime retry for the existing workflow and returns the same Run identity

#### Scenario: Resubmit creates a new Run
- **Given** a Run has a stored pipeline spec
- **When** a client posts to `/api/v1/runs/{id}/resubmit`
- **Then** the backend creates a new Run from the stored spec and records the source Run relationship in the ledger

### Requirement: Runtime adapter boundary owns runtime system calls
The system SHALL route product Run operations through a runtime adapter boundary rather than binding product handlers directly to Argo-specific clients.

**Priority**: P1 (High)
**Rationale**: Run is the product execution model; Argo is one runtime implementation and should remain replaceable behind the adapter boundary.

#### Scenario: Run submit uses the configured runtime adapter
- **Given** a Run submit request resolves to a runtime manifest
- **When** the Run Kernel submits runtime work
- **Then** it calls the configured runtime adapter and records the returned runtime reference on the Run

#### Scenario: Runtime operations stay product-scoped
- **Given** a Run has a runtime reference
- **When** a client requests retry, stop, suspend, resume, terminate, logs, or resubmit through the Run API
- **Then** the Run Kernel delegates the runtime-specific call to the adapter while preserving Run identity and product error semantics

## MODIFIED Requirements

### Requirement: Runtime workflow API is debug-oriented
- **Before**: Product execution screens could treat workflowName as the main execution identity.
- **After**: Product execution screens SHALL prefer Run identity and use workflowName only as a runtime debug reference.
- **Reason**: Argo Workflow names are not stable product identifiers and may disappear after TTL cleanup.

#### Scenario: Runtime debug remains visible
- **Given** a Run has runtime workflow metadata
- **When** a user opens the Run inspector
- **Then** workflow name and namespace are visible as runtime debug metadata, not as the primary product id
