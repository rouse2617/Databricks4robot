## MODIFIED Requirements

### Requirement: Workflow node runtime empty states match runtime state
- **Before**: The system SHALL show generic "暂无监控数据" and "暂无计费数据" warnings whenever node monitoring or billing cards do not have snapshot values.
- **After**: The system SHALL distinguish waiting-for-runtime snapshots, completed-without-data, and unavailable runtime diagnostics in workflow node detail surfaces.
- **Reason**: Pending or Unschedulable nodes have not produced metrics, costs, or logs yet and should not look like monitoring or billing configuration failures.

**Priority**: P1 (High)
**Rationale**: Workflow detail is the operator's primary debugging surface. Misleading empty states slow down diagnosis and make healthy in-progress workflows look broken.

#### Scenario: Pending node waiting for monitoring metrics
- **Given** a workflow node is Pending or Running and no CPU, memory, GPU, network, or storage snapshot exists yet
- **When** the user opens the node runtime panel
- **Then** the monitoring section explains that runtime metrics are still waiting for a resource snapshot
- **And** it does not imply that monitoring collection is misconfigured

#### Scenario: Pending node waiting for billing snapshot
- **Given** a workflow node is Pending or Running and no node cost snapshot exists yet
- **When** the user opens the node runtime panel
- **Then** the billing section explains that the cost snapshot will appear after runtime resource data is available
- **And** it does not imply that pricing configuration is broken

#### Scenario: Completed node without monitoring or cost data
- **Given** a workflow node has completed and monitoring or billing snapshots are still absent
- **When** the user opens the node runtime panel
- **Then** the UI shows a quiet no-data state
- **And** the copy avoids implying that the node is still waiting to start

#### Scenario: Pending node log viewer has no historical logs yet
- **Given** a workflow node has not started producing logs because the Pod is pending or unschedulable
- **When** the user opens the log viewer
- **Then** the log empty state explains that logs will appear after the node starts running
- **And** any live-log pagination limitation is described in user-facing language

#### Scenario: Runtime diagnostics remain accessible
- **Given** a workflow node has no monitoring or billing snapshot
- **When** the user opens Pod diagnostics, logs, or terminal-related affordances
- **Then** those controls remain visible and usable
- **And** the runtime tab guidance does not duplicate the node-card terminal instruction
