## MODIFIED Requirements

### Requirement: Workflow execution detail explains absent runtime snapshots
- **Before**: The workflow execution detail could show broad empty labels such as "暂无成本数据" or "暂无监控数据" for pending, completed, and unavailable states.
- **After**: The system SHALL explain absent monitoring and cost snapshots with runtime-state-specific copy so users can distinguish waiting resources, completed runs without generated snapshots, and temporarily unavailable data.
- **Reason**: Operators need to know whether they should wait, inspect logs/events, or treat missing data as a completed-run limitation.

**Priority**: P0 (Critical)
**Rationale**: Generic empty states slow down workflow troubleshooting and were repeatedly observed during dev browser regression.

#### Scenario: completed run has no cost snapshot
- **Given** a workflow execution has completed successfully
- **And** no cost snapshot is available for that run
- **When** the user opens the workflow execution detail
- **Then** the summary, alerts, and node list explain that the cost snapshot was not generated
- **And** the UI does not repeat a generic "暂无成本数据" label as the primary explanation

#### Scenario: pending run has no resource snapshot yet
- **Given** a workflow execution or node is still waiting for runtime resources
- **When** the user views cost or resource fields before any snapshot exists
- **Then** the UI labels the state as waiting for resources
- **And** the UI does not imply that a completed cost calculation returned empty data

#### Scenario: monitoring or billing data is unavailable
- **Given** monitoring or billing data cannot be fetched or is temporarily unavailable
- **When** the user opens workflow or node detail
- **Then** the empty state explains that the data is temporarily unavailable
- **And** the UI keeps logs, events, and node diagnostics accessible when those surfaces are available

### Requirement: Node troubleshooting copy is product-facing
- **Before**: Node log and troubleshooting surfaces could expose raw technical wording or generic empty states that did not explain pending nodes.
- **After**: The system SHALL use product-facing Chinese copy for node log, monitoring, and billing unavailable states, including pending nodes that have not started producing logs.
- **Reason**: Troubleshooting UI should describe the user's next diagnostic step instead of leaking implementation details.

**Priority**: P0 (Critical)
**Rationale**: Browser regression specifically called out raw English technical text and unclear pending-node behavior in the node detail experience.

#### Scenario: pending node has no logs
- **Given** a workflow node is pending and its pod has not started
- **When** the user opens the node log surface
- **Then** the UI explains in Chinese that logs will appear after the node starts
- **And** the UI points the user toward node events or scheduling diagnostics when available

#### Scenario: log data is unavailable
- **Given** log data cannot be loaded for a node
- **When** the user opens the log viewer or node log tab
- **Then** the UI uses Chinese product copy
- **And** the UI does not show raw English technical text from the log provider

### Requirement: Pipeline run entry points describe no-asset runs clearly
- **Before**: Some run buttons used "无资产运行" as the primary action label, making the command sound like a different workflow mode.
- **After**: The system SHALL use "运行" as the primary run command and SHALL explain no-asset runs as secondary context when no assets are selected.
- **Reason**: The main action is still running a pipeline; asset binding is contextual metadata.

**Priority**: P1 (High)
**Rationale**: Clear action labels reduce hesitation in the pipeline list and run dialog without changing execution behavior.

#### Scenario: user runs a pipeline without selected assets
- **Given** a pipeline can be run without selected assets
- **When** the user opens the run dialog or sees the run action
- **Then** the primary action is labeled "运行"
- **And** the UI explains that this run is not bound to assets

#### Scenario: user selects a pipeline version before running
- **Given** a pipeline has selectable saved versions
- **When** the user opens the run dialog
- **Then** the version selector copy makes the selected version understandable
- **And** changing the copy does not change which version is submitted
