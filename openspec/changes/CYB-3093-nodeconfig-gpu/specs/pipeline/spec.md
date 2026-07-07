## ADDED Requirements

### Requirement: 画布节点配置支持 GPU 与计算档位覆盖
The pipeline design canvas node-config override panel SHALL expose GPU (count) and compute-tier fields alongside CPU/memory/disk, so a step's GPU and compute tier can be overridden per node — consistent with the component-definition form. The panel SHALL prefill these from the node's stored values and persist edits back to the node.

**Priority**: P2 (Medium)
**Rationale**: The backend transpiler and the canvas→DSL serialization already carry per-node GPU/compute-tier; only the node-config UI lacked the inputs, leaving CPU/memory/disk overridable but GPU not.

#### Scenario: GPU and compute-tier fields are available in node config
- **Given** a node is selected on the design canvas and its config panel is opened
- **When** the user views the resource section
- **Then** GPU (count) and compute-tier inputs are shown next to CPU/memory/disk

#### Scenario: GPU / compute-tier overrides persist
- **Given** the user sets GPU and compute-tier on a node and saves
- **When** the node config panel is reopened
- **Then** the previously entered GPU and compute-tier values are shown
