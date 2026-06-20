# Runtime OS Spec Delta - CYB-3004

## ADDED Requirements

### Requirement: Run submission uses RuntimeAdapter when configured
The system SHALL submit non-dry-run product Runs through the configured RuntimeAdapter before falling back to legacy runtime clients.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS requires the Run Kernel to own product execution while runtime-specific systems remain replaceable behind an adapter boundary.

#### Scenario: Adapter-backed submit succeeds
- **Given** a Run creation request resolves a pipeline manifest and a RuntimeAdapter is configured
- **When** the system submits the Run
- **Then** it calls RuntimeAdapter.Submit with the Run identity, target namespace, and runtime manifest
- **Then** it persists the resulting runtime reference data where available

#### Scenario: Adapter submit fails
- **Given** a RuntimeAdapter is configured but rejects submission
- **When** the system creates a non-dry-run Run
- **Then** the Run is not persisted as successfully submitted
- **Then** the client receives a product error consistent with runtime submit failure

### Requirement: Submit fallback remains compatible
The system SHALL keep the legacy workflow-client submit path when no RuntimeAdapter is configured.

**Priority**: P1 (High)
**Rationale**: Existing tests, local constructors, and compatibility paths still instantiate pipeline usecase directly with a workflow client.

#### Scenario: Adapter missing with workflow client configured
- **Given** no RuntimeAdapter is configured and the legacy workflow client is available
- **When** the system submits a non-dry-run Run
- **Then** it uses the previous workflow-client submit behavior

#### Scenario: No runtime backend configured
- **Given** neither RuntimeAdapter nor legacy workflow client is configured
- **When** the system submits a non-dry-run Run
- **Then** it returns the existing runtime-unavailable error semantics

### Requirement: Dry-run previews do not submit runtime jobs
The system SHALL keep dry-run Run previews side-effect free.

**Priority**: P0 (Critical)
**Rationale**: Dry-run is used to inspect generated runtime manifests and must not create real runtime work.

#### Scenario: Dry-run with adapter configured
- **Given** a RuntimeAdapter is configured
- **When** a Run creation request is marked dry-run
- **Then** the system returns a preview deployment and does not call RuntimeAdapter.Submit

## MODIFIED Requirements

### Requirement: Runtime adapter boundary owns runtime system calls
- **Before**: Product Run lifecycle controls preferred RuntimeAdapter, but Run submission still directly invoked Argo workflow-client creation.
- **After**: Product Run submission SHALL also prefer RuntimeAdapter while preserving legacy workflow-client fallback during migration.
- **Reason**: Runtime OS needs runtime submission and lifecycle operations to share the same adapter boundary.

#### Scenario: Submit and lifecycle share runtime boundary
- **Given** a configured ArgoRuntimeAdapter
- **When** a Run is created and later controlled
- **Then** both submit and lifecycle operations go through the adapter boundary rather than treating Argo workflow name as the product owner
