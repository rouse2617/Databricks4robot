# Pipeline Spec Delta — CYB-3094 / CYB-3095

## ADDED Requirements

### Requirement: Component steps always declare at least one input and one output port

The component editor SHALL prevent a component from being defined with zero input
ports or zero output ports, so that a component's declared ports always match the
ports shown on a placed canvas node.

#### Scenario: The last input port cannot be removed

- **Given** the 新建组件 form shows exactly one input port
- **When** the user attempts to remove that port
- **Then** the remove control SHALL be disabled
- **And** a tooltip SHALL explain that at least one input port is required

#### Scenario: The last output port cannot be removed

- **Given** the 新建组件 form shows exactly one output port
- **When** the user attempts to remove that port
- **Then** the remove control SHALL be disabled

#### Scenario: Removal is allowed above the minimum

- **Given** the form shows more than one input port
- **When** the user views the input port rows
- **Then** every input port's remove control SHALL be enabled

### Requirement: Run detail DAG view excludes infrastructure hook nodes

The run detail DAG view, its step count, the top-level node count, and the timeline
SHALL exclude the workflow-level exit-notify hook node introduced by CYB-3058,
consistent with the 节点明细 table.

#### Scenario: onExit hook is not shown as a step

- **Given** a completed run whose workflow has one business step plus a
  `databrew-exit-notify` onExit hook node
- **When** the user opens the run's DAG view
- **Then** the DAG SHALL render only the business step
- **And** the step count SHALL be 1
- **And** the top-level node count SHALL be 1

#### Scenario: onExit hook is identified by name suffix when template name is absent

- **Given** a workflow node whose name ends with `.onExit`
- **When** the DAG view and node count are computed
- **Then** that node SHALL be excluded even if its template name is not
  `databrew-exit-notify`
