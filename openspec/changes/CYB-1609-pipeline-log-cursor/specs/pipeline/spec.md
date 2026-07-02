# Pipeline Spec Delta — CYB-1609

## ADDED Requirements

### Requirement: Workflow log windows expose truthful pagination metadata
The system SHALL return bounded workflow log windows with metadata that tells clients whether more log data can be requested and which constraints were applied.

**Priority**: P0 (Critical)
**Rationale**: Users need to debug large logs without causing unbounded backend reads or assuming fake pagination exists.

#### Scenario: Default log request returns bounded window metadata
- **Given** a workflow node has pod logs larger than the default display size
- **When** a client requests node logs without explicit limit parameters
- **Then** the system returns a bounded window with truncation metadata and does not read or return the full log stream.

#### Scenario: Historical cursor is unavailable
- **Given** the active log source cannot provide stable historical cursors
- **When** a client requests node logs
- **Then** the response marks pagination as unavailable or returns no next cursor, and the UI explains that only bounded tail/follow is available.

#### Scenario: Excessive bounds stay controlled
- **Given** a client requests line or byte limits above the supported maximum
- **When** the system handles the request
- **Then** the system clamps or rejects the values according to the documented contract and reports the applied bounds.

### Requirement: Workflow log viewer manages realtime follow lifecycle
The system SHALL show a clear lifecycle state for realtime workflow log follow.

**Priority**: P1 (High)
**Rationale**: A user must know whether the log stream is connecting, live, stopped, completed, or failed.

#### Scenario: Follow connects successfully
- **Given** a user is viewing a workflow node with available pod logs
- **When** the user starts realtime follow
- **Then** the UI shows connecting and live states, appends incoming lines to the bounded display buffer, and allows the user to stop the stream.

#### Scenario: Follow ends or fails
- **Given** a realtime log stream closes because the pod completed or the connection fails
- **When** the frontend receives the end/error signal
- **Then** the UI shows an ended or failed state without leaving the user in an ambiguous active state.

### Requirement: Failed nodes provide direct log entry
The system SHALL let users open logs directly from a failed workflow node.

**Priority**: P1 (High)
**Rationale**: Failure diagnosis should not require manually finding the node in a separate list.

#### Scenario: Failed node opens selected logs
- **Given** a workflow DAG contains a failed node with a pod log reference
- **When** the user clicks the node log action
- **Then** the workflow detail page opens the log viewer with that node selected and starts from a bounded tail window.

#### Scenario: Node logs are unavailable
- **Given** the failed node has no resolvable pod or logs were cleaned up
- **When** the user opens logs from that node
- **Then** the UI shows a concise unavailable state with workflow/run context and no empty technical placeholder.

### Requirement: Users can download bounded log content
The system SHALL allow users to download the currently loaded or bounded log content with clear scope labeling.

**Priority**: P2 (Nice-to-have)
**Rationale**: Users often need to share logs for debugging, but the download must not imply a full archive when only a bounded window is loaded.

#### Scenario: Download loaded log window
- **Given** the log viewer has loaded one or more bounded log chunks
- **When** the user downloads logs
- **Then** the downloaded file contains the loaded content and metadata identifying workflow, node, container, and bounded scope.

#### Scenario: No logs loaded
- **Given** no log content is loaded
- **When** the user clicks download
- **Then** the UI disables the action or explains that there is nothing to download.

## MODIFIED Requirements

### Requirement: Bounded workflow log reads
- **Before**: The system SHALL bound workflow log reads by default.
- **After**: The system SHALL bound workflow log reads by default and SHALL expose the applied bounds, truncation, and pagination availability to the client.
- **Reason**: Bounded logs are necessary but insufficient unless users can understand what portion of the log they are viewing.

#### Scenario: User understands the visible log scope
- **Given** a log response was truncated by tail or byte bounds
- **When** the frontend renders the log viewer
- **Then** the viewer states the applied bounds and whether more history can be loaded.
