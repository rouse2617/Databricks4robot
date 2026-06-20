# Runtime OS Spec Delta - CYB-3005

## ADDED Requirements

### Requirement: Config inputs appear in RunInput projection
The system SHALL represent deploy-level and node-level pipeline configs as product Run inputs.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS treats Config as an input dependency of a Run, while runtime ConfigMaps are only materialization details.

#### Scenario: Deploy-level config selection becomes RunInput
- **Given** a Run is created with deploy-level config selection
- **When** the client reads `/api/v1/runs/{id}/inputs`
- **Then** the response includes a `config` input for the selection
- **Then** the input includes mode, config id/version when saved, file name, mount path, target filename, and content hash where available
- **Then** raw config content is not returned

#### Scenario: Node-level config bindings become RunInputs
- **Given** a Run is created with node-level runtime config bindings
- **When** the client reads `/api/v1/runs/{id}/inputs`
- **Then** the response includes a `config` input for each bound node
- **Then** each input identifies the node id and immutable config metadata

#### Scenario: Historical Runs remain readable
- **Given** a historical Run lacks materialized config input metadata
- **When** the client reads `/api/v1/runs/{id}/inputs`
- **Then** the system continues to derive node config inputs from existing pipeline JSON where possible

## MODIFIED Requirements

### Requirement: Run inputs include config dependencies
- **Before**: Run inputs included assets, runtime target, basic node config fallback, and parameters.
- **After**: Run inputs SHALL include materialized config input metadata for both deploy-level and node-level config bindings, including a content hash and no raw content.
- **Reason**: Config must be visible as a first-class Run dependency for Run Inspector and audit flows.
