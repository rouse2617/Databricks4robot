## ADDED Requirements

### Requirement: Pipeline nodes can bind saved config-library entries
The system SHALL allow a pipeline node/component instance to reference one saved config-library entry and version as that node's runtime config.

**Priority**: P0 (Critical)
**Rationale**: Users configure different components with different parameters; one deploy-level config cannot represent node-specific task configuration.

#### Scenario: User binds a ready config to one node
- **Given** a ready config exists in the config library
- **And** the user is editing a pipeline node
- **When** the user selects that config in the node configuration panel and saves the node
- **Then** the node stores the selected config identity and mount destination
- **And** the canvas shows that the node has a config binding

#### Scenario: User clears a node config binding
- **Given** a pipeline node already has a config binding
- **When** the user clears the binding and saves the node
- **Then** the saved pipeline node no longer references that config
- **And** future deploys do not mount that config on the node

#### Scenario: No ready configs are available
- **Given** the user has no accessible ready configs
- **When** the user opens the node config section
- **Then** the UI explains that no ready config is available
- **And** it does not offer draft/deprecated configs as runnable choices

### Requirement: Pipeline template snapshots preserve node config binding
The system SHALL persist node-level config bindings in pipeline template snapshots and restore them when templates are reopened.

**Priority**: P0 (Critical)
**Rationale**: Node config is part of the pipeline's runtime contract and must be versioned with the pipeline template.

#### Scenario: Saved template preserves per-node configs
- **Given** a pipeline contains two nodes with different config bindings
- **When** the user saves the pipeline and later opens the saved template
- **Then** both nodes show their original config bindings
- **And** the bindings remain attached to the correct node IDs

#### Scenario: Old template without config still opens
- **Given** a saved pipeline template was created before node config binding existed
- **When** the user opens that template
- **Then** the pipeline loads successfully
- **And** every node is treated as having no config binding

### Requirement: Deploy resolves and mounts configs per node
The system SHALL resolve node-level saved config references during deploy and mount each resolved config only into the node that declared it.

**Priority**: P0 (Critical)
**Rationale**: Runtime isolation is the corrected behavior; a config selected for one component must not leak into unrelated components.

#### Scenario: Two nodes receive different configs
- **Given** a pipeline has node A bound to config A
- **And** node B is bound to config B
- **When** the user deploys the pipeline
- **Then** node A's generated runtime template mounts only config A
- **And** node B's generated runtime template mounts only config B
- **And** each node receives env vars describing its own config path and version

#### Scenario: Unconfigured node receives no config
- **Given** a pipeline has node A with a config binding
- **And** node B has no config binding
- **When** the user deploys the pipeline
- **Then** node B does not receive node A's config mount
- **And** node B does not receive node A's config env vars

#### Scenario: Invalid saved config reference fails before Argo submission
- **Given** a pipeline node references a missing, inaccessible, draft, or deprecated config
- **When** the user deploys the pipeline
- **Then** the backend rejects the deploy before creating the Argo workflow
- **And** the error message identifies the invalid node/config relationship

### Requirement: Deploy-selected assets are injected into every node Pod
The system SHALL treat selected asset IDs as run-level context and inject them into every pipeline node Pod through environment variables by default.

**Priority**: P0 (Critical)
**Rationale**: Asset selection answers what the whole pipeline run processes, while config binding answers how each node processes it. Users should not need to configure asset IDs repeatedly per node.

#### Scenario: Every node receives selected asset IDs
- **Given** the user deploys a pipeline with asset IDs `asset-a` and `asset-b`
- **When** the workflow is generated
- **Then** every node Pod receives env vars containing the selected asset set
- **And** the env vars include the aggregate asset list and per-asset indexed IDs

#### Scenario: Node config does not change asset context
- **Given** a pipeline has node A bound to a config
- **And** node B has no config binding
- **And** the user deploys the pipeline with selected asset IDs
- **When** the workflow is generated
- **Then** both node A and node B receive the same run-level asset env vars
- **And** only node A receives config-specific env vars

#### Scenario: No selected assets remains explicit
- **Given** the user deploys a no-asset debug run
- **When** the workflow is generated
- **Then** no fake asset ID is injected
- **And** node-level config binding still works independently if configured

## MODIFIED Requirements

### Requirement: Deploy-level config selection is compatibility behavior
- **Before**: The deploy panel treated a single config selection as the main runtime-config model for a pipeline run.
- **After**: The system SHALL treat node-level config bindings as the primary product model, while preserving deploy-level config selection only for compatibility or explicit advanced callers.
- **Reason**: The corrected requirement is per-node/component configuration from the config library, not one global deploy config.

#### Scenario: Node config takes precedence over deploy-level config
- **Given** a deploy request contains a deploy-level config selection
- **And** one pipeline node has its own node-level config binding
- **When** the pipeline is deployed
- **Then** the configured node uses its node-level config binding
- **And** the deploy-level config does not override that node's binding
