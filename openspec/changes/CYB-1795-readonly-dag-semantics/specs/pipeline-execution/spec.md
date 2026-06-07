## MODIFIED Requirements

### Requirement: Workflow execution DAG exposes read-only semantics
- **Before**: The execution DAG could visually behave as read-only while React Flow accessibility descriptions still announced edit-mode actions such as moving nodes or deleting edges.
- **After**: The execution DAG SHALL expose read-only semantics for nodes and edges, and SHALL NOT instruct users or assistive tooling to move nodes or delete graph elements.
- **Reason**: Execution details are an operational inspection surface. Misleading edit semantics confuse users and automation even when the visual graph cannot be edited.

**Priority**: P2 (Nice-to-have)
**Rationale**: This aligns accessibility semantics with actual product behavior and reduces QA noise without changing workflow execution.

#### Scenario: execution DAG does not announce graph editing
- **Given** a user opens a workflow execution detail DAG
- **When** the DAG nodes and edges are exposed through keyboard or accessibility tooling
- **Then** the graph does not announce node move instructions
- **And** the graph does not announce delete instructions for nodes or edges

#### Scenario: run diagnostics remain available
- **Given** a user opens a workflow execution detail DAG
- **When** they search for a node or select a node
- **Then** node selection and search still work
- **And** the node action controls for logs, runtime, IO, terminal, and details remain available when supported

#### Scenario: graph navigation remains available
- **Given** a user opens a workflow execution detail DAG
- **When** they pan, zoom, or use fit view controls
- **Then** the read-only graph remains navigable without exposing topology editing behavior
