## ADDED Requirements

### Requirement: Pipeline nodes bind platform secret resources
The system SHALL allow a pipeline node to reference a platform-defined secret mount resource without exposing secret plaintext or requiring the user to author Kubernetes SecretProviderClass details.

**Priority**: P0 (Critical)
**Rationale**: Many algorithm tasks cannot run without credentials, but credentials must remain platform-managed and hidden from ordinary configuration content.

#### Scenario: User binds a secret resource to one node
- **Given** a platform-defined secret resource is available for the current execution target
- **And** the user is editing a pipeline node
- **When** the user selects the secret resource and saves the node
- **Then** the saved pipeline node records the selected secret resource and mount destination
- **And** the workflow Pod for that node receives a read-only secret file or directory mount
- **And** other nodes without the binding do not receive that secret mount

#### Scenario: Unknown secret resource is rejected
- **Given** a saved pipeline references a secret resource that is not in the runtime mount catalog
- **When** the user deploys the pipeline
- **Then** DataBrew rejects the deployment before workflow creation
- **And** the error message identifies the missing or unsupported secret resource

### Requirement: Pipeline nodes bind platform storage resources
The system SHALL allow a pipeline node to reference a platform-defined storage mount resource backed by an existing shared storage implementation.

**Priority**: P0 (Critical)
**Rationale**: Algorithms need shared directories for models, caches, intermediate artifacts, and future RBD/CSI-backed workspaces without requiring users to understand Kubernetes storage primitives.

#### Scenario: User binds a PVC-backed storage resource
- **Given** a platform-defined storage resource backed by an existing PVC is available
- **And** the user is editing a pipeline node
- **When** the user selects that storage resource and saves the node
- **Then** the saved pipeline node records the selected storage resource and mount destination
- **And** the workflow Pod for that node receives the PVC mount with the allowed read/write mode

#### Scenario: Write access exceeds resource policy
- **Given** a storage resource is registered as read-only
- **When** a node binding requests write access to that resource
- **Then** DataBrew rejects the deployment before workflow creation
- **And** the error message tells the user that the selected storage resource is read-only

### Requirement: Runtime inputs remain separated in the node editor
The system SHALL present Configuration, Secrets, and Storage as separate node editor sections.

**Priority**: P1 (High)
**Rationale**: Configuration content, credentials, and shared directories have different ownership, security, and lifecycle rules. Combining them in one form causes unsafe user behavior.

#### Scenario: User edits node runtime inputs
- **Given** the user opens a pipeline node editor
- **When** the runtime input controls are displayed
- **Then** normal configuration files, secret mounts, and storage mounts appear in separate labeled sections
- **And** secret controls do not include plaintext fields
- **And** advanced Kubernetes details are not required for the main workflow

#### Scenario: Node badges summarize runtime inputs
- **Given** a node has selected configuration, secret, and storage bindings
- **When** the node is shown on the canvas
- **Then** the node shows compact summaries that identify the selected runtime inputs
- **And** the summaries do not expose secret plaintext or low-level Kubernetes fields

### Requirement: Runtime mount catalog is target-safe
The system SHALL expose only platform-approved runtime mount resources that are safe for the selected environment.

**Priority**: P1 (High)
**Rationale**: Users should not accidentally mount cluster internals, unsupported CSI resources, or storage that cannot run in the current execution target.

#### Scenario: User opens the mount selector
- **Given** platform-approved runtime mount resources exist
- **When** the user opens the node editor
- **Then** the UI lists selectable secret and storage resources with human-readable names, descriptions, mount type, and read/write policy

#### Scenario: Target does not support a resource
- **Given** a runtime mount resource is not supported by the selected execution target
- **When** the user deploys a pipeline referencing that resource
- **Then** DataBrew rejects the deployment before workflow creation
- **And** the error message identifies the target compatibility issue

### Requirement: Batch pipeline runs preserve execution target selection
The system SHALL preserve the execution target selected when creating a batch pipeline job and apply it to every materialized child pipeline run.

**Priority**: P0 (Critical)
**Rationale**: Batch jobs can be created long before individual items materialize. If the target is not persisted with the batch job, child runs silently fall back to the default namespace and can miss target-specific GPU scheduling, secret mounts, and quota policy.

#### Scenario: User creates a batch job for a non-default execution target
- **Given** the user selects a saved pipeline template
- **And** the user selects the `video-proc-dev` execution target
- **And** the user provides multiple asset or video IDs
- **When** the user creates the batch job
- **Then** DataBrew stores the selected execution target with the batch job
- **And** every child pipeline run created from that batch job uses the selected execution target instead of the default target
- **And** target-specific runtime mounts and scheduling defaults remain available during workflow creation
