## ADDED Requirements

### Requirement: Registry hub separates reference data from configuration management
The system SHALL present `/registry` as a governance hub with clearly separated reference registries and configuration management surfaces.

**Priority**: P1 (High)
**Rationale**: Users need to understand which data are read-only vocabulary and which data are editable operational configuration.

#### Scenario: User opens the registry hub
- **Given** a user navigates to the registry page
- **When** the page loads successfully
- **Then** the user sees read-only reference registries and a separate configuration management area
- **And** the page does not present all registries as one undifferentiated table

#### Scenario: Reference registries partially fail
- **Given** configuration management is unavailable but reference registry data can still load
- **When** the user opens the registry page
- **Then** the reference registries remain visible
- **And** the page surfaces a partial failure state for the unavailable area

### Requirement: Configuration resources are standalone library records
The system SHALL treat configuration records as first-class user-owned resources with independent identity, ownership, tags, description, lifecycle state, and file payload metadata.

**Priority**: P0 (Critical)
**Rationale**: The configuration center must not be a child surface of components or releases.

#### Scenario: User searches for a config
- **Given** a config library contains multiple records
- **When** the user filters by name, description, tag, or owner
- **Then** the matching records are shown
- **And** unrelated component metadata is not required to use the search

#### Scenario: User registers a config file
- **Given** a user has a file they want to reuse during deploy
- **When** they register the file into the config library
- **Then** the system stores it as a standalone config record
- **And** the record is owned by that user unless explicitly shared

### Requirement: Configuration resources support immutable versions
The system SHALL manage configuration content through immutable versions and SHALL expose the current selectable version for each config.

**Priority**: P0 (Critical)
**Rationale**: Users need rollback and auditability for configuration changes, and pipeline snapshots must identify the exact version that was selected.

#### Scenario: User views config versions
- **Given** a config has multiple versions
- **When** the user expands or opens that config in the config library
- **Then** the user sees each version, status, author, update time, and change summary
- **And** the current selectable version is clearly identified

#### Scenario: User creates a new config version
- **Given** a user edits an existing config file
- **When** the user saves the change
- **Then** the system creates a new immutable version
- **And** previous versions remain available for audit and rollback decisions

#### Scenario: User edits an existing version as a new version
- **Given** a config version already exists
- **When** the user chooses to edit that version's file content
- **Then** the editor is prefilled from the chosen version
- **And** saving creates a new version instead of mutating the old version

### Requirement: Configuration resources expose CRUD-style management actions
The system SHALL allow users to create, inspect, update metadata for, deprecate, and create new versions for configuration records.

**Priority**: P0 (Critical)
**Rationale**: Users need a complete management loop before config records can be safely referenced during deploy.

#### Scenario: User opens config detail
- **Given** a config exists in the config library
- **When** the user selects its detail action
- **Then** the system shows identity, owner, lifecycle, tags, current version, and version history

#### Scenario: User updates config metadata
- **Given** a user owns a config
- **When** the user edits its name, description, tags, or lifecycle
- **Then** the config metadata is updated without rewriting historical versions

#### Scenario: User deprecates a config
- **Given** a config should no longer be selected for new deployments
- **When** the user deprecates it
- **Then** the config remains visible for audit
- **And** it is not treated as a hard delete that removes historical references

### Requirement: Deploy-time config selection is ownership-scoped
The system SHALL show only the current user's own or shared configs when the user attaches a config during deploy.

**Priority**: P0 (Critical)
**Rationale**: Users should only pick configs they are allowed to use, and the deploy experience should stay a one-click choice rather than a raw metadata editor.

#### Scenario: User opens deploy config picker
- **Given** the user is deploying a component
- **When** the config picker opens
- **Then** only configs owned by that user or shared to that user are listed
- **And** the user can attach one config with a single click

#### Scenario: Other users' configs are hidden
- **Given** another user owns a private config
- **When** the current user opens the deploy config picker
- **Then** that private config is not shown
- **And** the user receives no cross-account access to the file

#### Scenario: Config delete is blocked while still referenced
- **Given** a saved pipeline still references a config
- **When** the user attempts to remove that config
- **Then** the system blocks hard deletion or forces deprecation
- **And** the user sees a clear reason why the config cannot be removed immediately

### Requirement: Pipeline nodes reference configs by ID and retain a snapshot
The system SHALL persist `configId` on pipeline nodes and SHALL retain an immutable config snapshot in saved pipeline state.

**Priority**: P0 (Critical)
**Rationale**: Historical runs must remain reproducible even if the live config changes later.

#### Scenario: User saves a pipeline with a selected config
- **Given** the user has chosen a config for a pipeline node
- **When** the pipeline is saved
- **Then** the saved node records the chosen config identity
- **And** the saved pipeline state contains an immutable snapshot of that config

#### Scenario: Config changes after the pipeline is saved
- **Given** a saved pipeline already contains a config snapshot
- **When** the live config is edited later
- **Then** the previously saved pipeline continues to show the original snapshot
- **And** the historical run does not drift to the new live config state

### Requirement: Components define a config mount contract
The system SHALL allow a component to define the container directory where a selected config file is mounted, without making the component the owner of that config.

**Priority**: P1 (High)
**Rationale**: The deploy flow needs a deterministic place to put the selected config file inside the container while keeping config ownership independent from the component.

#### Scenario: Component declares a mount path
- **Given** a component is being edited for deploy/runtime behavior
- **When** the user specifies a config mount path
- **Then** the component stores the mount target as runtime contract metadata
- **And** no config record is bound to the component by identity

#### Scenario: Selected config is mounted into the declared directory
- **Given** a component has a declared mount path and the user selected a config at deploy time
- **When** the deployment is materialized
- **Then** the selected config file is projected into the declared container directory
- **And** the runtime uses that mounted file instead of live config lookup
