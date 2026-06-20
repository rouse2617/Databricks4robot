# Runtime OS Spec Delta — CYB-3002

## ADDED Requirements

### Requirement: Batch executions are represented as Run Trees
The system SHALL represent batch/backfill execution structure as a parent Run with child Runs where current storage has enough relationship data.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS requires Batch and Backfill to share the Run product model instead of remaining disconnected side-list concepts.

#### Scenario: Batch parent exposes child Runs
- **Given** a parent Run exists for a batch execution and child Runs are associated with that parent
- **When** a client requests the parent Run's children
- **Then** the response contains child Runs for that batch and relation metadata identifying them as batch children

#### Scenario: Parent Run has no child Runs yet
- **Given** a parent Run exists but no child Runs have been materialized
- **When** a client requests the parent Run's children
- **Then** the response contains an empty child list and an aggregate summary that does not require a runtime workflow

### Requirement: Run Tree status is aggregated deterministically
The system SHALL aggregate child Run statuses into a deterministic parent status summary using Run Kernel status semantics.

**Priority**: P0 (Critical)
**Rationale**: Operators need the parent Batch/Backfill view to explain whether the tree is still running, blocked, successful, or failed without reading every child row manually.

#### Scenario: All child Runs complete successfully
- **Given** a parent Run has child Runs and every child has a successful terminal status
- **When** the system aggregates the Run Tree status
- **Then** the summary reports all children successful and the aggregate status is successful

#### Scenario: At least one child Run fails
- **Given** a parent Run has child Runs and at least one child has a failed or error terminal status
- **When** the system aggregates the Run Tree status after no children are active
- **Then** the summary reports the failure count and the aggregate status is failed or error

#### Scenario: Some child Runs are still active
- **Given** a parent Run has a mix of pending, running, and terminal child Runs
- **When** the system aggregates the Run Tree status
- **Then** the summary reports detailed counts and the aggregate status remains active

### Requirement: Run Tree diagnostics explain blocked and failed children
The system SHALL expose normalized child Run failure or blocking reasons from existing Run ledger/runtime diagnostics without requiring clients to parse raw Kubernetes or Argo messages.

**Priority**: P0 (Critical)
**Rationale**: Operators need Batch and Run views to explain why work is stuck or failed before opening individual Pods or runtime debug panels.

#### Scenario: Child Runs fail because Kubernetes cannot schedule them
- **Given** child Runs contain scheduler diagnostic messages such as insufficient CPU, memory, ephemeral storage, or unschedulable
- **When** a client requests the parent Run's children
- **Then** the response includes normalized `unschedulable` diagnostics on child Runs and in the parent summary `topFailureReasons`

#### Scenario: Child Runs exist before runtime submission
- **Given** a child Run is pending and does not yet have a runtime workflow reference
- **When** the system projects Run diagnostics
- **Then** the child Run includes `blockingReason=runtime_not_submitted` and a user-readable `blockingMessage`

#### Scenario: Runtime image startup fails
- **Given** a Run or child Run contains image pull, invalid image name, or image parsing diagnostics
- **When** the system projects Run diagnostics
- **Then** the Run includes `failureReason=image_startup` and the original message remains available in `message`

#### Scenario: Run is rejected by target resource compatibility
- **Given** Run validation rejects a node because its requested CPU, memory, or GPU exceeds the execution target capability
- **When** the system projects Run diagnostics
- **Then** the Run includes `failureReason=resource_incompatible` and the original validation message remains available in `message`

### Requirement: RunRelation projection is available before relation-table migration
The system SHALL expose projected RunRelation rows from existing Run metadata until a dedicated relation table exists.

**Priority**: P1 (High)
**Rationale**: Frontend and API clients need stable Run Tree semantics before the storage migration is safe to perform.

#### Scenario: Existing batch child metadata projects a relation
- **Given** a child Run references a batch parent through existing metadata
- **When** a client requests the parent Run's children
- **Then** the response includes a relation projection connecting the parent and child

#### Scenario: Missing parent metadata does not invent a relation
- **Given** a Run has no parent metadata
- **When** the system builds RunRelation projections
- **Then** it does not create a child relation for that Run

## MODIFIED Requirements

### Requirement: Run detail exposes stable ledger subresources
- **Before**: The system returned a basic child Run list for `/runs/:id/children`.
- **After**: The system SHALL return child Runs, projected relations, and child status summary for `/runs/:id/children`.
- **Reason**: Run Inspector and Batch Inspector need a Run Tree contract, not just a filtered list.

#### Scenario: Child response remains backward compatible
- **Given** a client only reads the child Run items from the children response
- **When** the enriched children response is returned
- **Then** the existing item list remains available while new clients can read relation and summary fields

### Requirement: Execution Hub uses Runs as its product source
- **Before**: Batch detail could treat batch execution children as a product list separate from Run Tree semantics.
- **After**: Batch detail SHALL prefer Run API child rows and Run Tree summary for product execution child rows.
- **Reason**: Batch is a parent Run with child Runs in Runtime OS.

#### Scenario: Batch detail renders child Runs from Run API
- **Given** a batch has materialized child Runs
- **When** a user opens Batch Detail
- **Then** the page displays child Run links and aggregate summary from the Run API instead of a legacy-only execution list
