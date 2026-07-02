# Runtime OS Spec Delta - CYB-3003

## ADDED Requirements

### Requirement: Run Kernel owns product lifecycle controls
The system SHALL route product Run lifecycle controls through the Run Kernel boundary before invoking runtime-specific APIs.

**Priority**: P0 (Critical)
**Rationale**: Run is the product execution entity, so user lifecycle actions must not be owned by Argo workflow names or debug endpoints.

#### Scenario: Product stop delegates through Run Kernel
- **Given** a stored Run has a runtime workflow reference
- **When** a client requests a product Run stop
- **Then** the Run Kernel resolves the runtime reference and delegates the runtime-specific stop call through the configured runtime boundary

#### Scenario: Missing runtime reference is rejected
- **Given** a stored Run has no runtime workflow reference
- **When** a client requests a product Run lifecycle control
- **Then** the system returns an invalid argument style error and does not call a runtime adapter

### Requirement: RuntimeAdapter handles Argo lifecycle operations
The system SHALL use the Argo runtime adapter for retry, stop, suspend, resume, and terminate operations when the adapter is configured.

**Priority**: P0 (Critical)
**Rationale**: Argo is one runtime implementation and must be isolated behind the adapter so future runtimes can implement the same Run lifecycle contract.

#### Scenario: Adapter uses stored namespace
- **Given** a Run runtime reference contains a workflow name and namespace
- **When** a lifecycle operation is delegated to the Argo adapter
- **Then** the adapter calls Argo with that namespace and workflow name

#### Scenario: Adapter operation fails
- **Given** the configured runtime adapter returns an error for a lifecycle operation
- **When** the product Run operation is requested
- **Then** the system returns the error and records a failed RunEvent on the source Run

### Requirement: Run lifecycle controls write ledger events
The system SHALL append RunEvent ledger entries for requested, succeeded, and failed product lifecycle controls.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS treats the event ledger as the durable explanation of product execution state changes.

#### Scenario: Runtime control succeeds
- **Given** a stored Run can be controlled by the runtime adapter
- **When** a lifecycle operation succeeds
- **Then** the event ledger contains requested and succeeded events for that Run

#### Scenario: Runtime control fails
- **Given** the runtime adapter rejects a lifecycle operation
- **When** the product operation returns the runtime error
- **Then** the event ledger contains requested and failed events with the failure reason

## MODIFIED Requirements

### Requirement: Runtime adapter boundary owns runtime system calls
- **Before**: Run product operations could still call Argo workflow-client methods directly from pipeline usecase code.
- **After**: Run product lifecycle operations SHALL prefer the RuntimeAdapter boundary while preserving a workflow-client fallback during migration.
- **Reason**: Runtime OS requires replaceable runtime implementations without breaking existing dev/test construction paths.

#### Scenario: Adapter missing fallback remains compatible
- **Given** a local or unit-test usecase has no configured RuntimeAdapter but still has the legacy workflow client
- **When** a Run lifecycle operation is requested
- **Then** the operation keeps the previous behavior through the legacy client path

### Requirement: Run operations distinguish retry from resubmit
- **Before**: `/runs/:id/retry` already represented runtime retry and `/pipeline-runs/:id/retry` represented full rerun, but implementation ownership was still mixed.
- **After**: The system SHALL preserve these product semantics while routing `/runs/:id/retry` through the Run Kernel runtime-control path.
- **Reason**: Operators need predictable same-Run retry behavior without losing legacy full-rerun compatibility.

#### Scenario: Run retry does not create a new Run
- **Given** a stored Run has a runtime workflow reference
- **When** a client posts to `/api/v1/runs/{id}/retry`
- **Then** the runtime retry is submitted for the same Run and no new product Run is created by that operation
