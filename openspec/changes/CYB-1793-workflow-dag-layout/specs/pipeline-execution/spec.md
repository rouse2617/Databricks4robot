## MODIFIED Requirements

### Requirement: Workflow execution DAG presents join dependencies clearly
- **Before**: The execution DAG could render fan-in joins as ordinary downstream nodes with incoming edges routed around the graph, making the convergence point visually ambiguous.
- **After**: The execution DAG SHALL position and route fan-in join nodes so the join reads as the convergence point for its visible upstream dependencies.
- **Reason**: Operators use the DAG view to understand workflow dependency and status. Fan-in joins must communicate "all upstream branches converge here" without implying hidden intermediate nodes.

**Priority**: P1 (High)
**Rationale**: The DAG is the primary visual diagnostic surface for workflow execution. Ambiguous join rendering slows debugging and makes complex workflows look unintentionally drawn.

#### Scenario: complex fan-in join is centered
- **Given** a workflow DAG has upstream nodes `qa-left-b`, `qa-right-b`, and `qa-mid-a`
- **And** all three connect to `qa-join-abc`
- **When** the user opens the workflow execution DAG view
- **Then** `qa-join-abc` is visually positioned as the convergence point of those three upstream nodes
- **And** the incoming edges are easy to follow

#### Scenario: simple linear DAG remains readable
- **Given** a workflow DAG has a simple linear dependency chain
- **When** the user opens the workflow execution DAG view
- **Then** the graph remains left-to-right, compact, and easy to scan

### Requirement: Workflow execution DAG avoids edit-mode affordances
- **Before**: Read-only workflow execution nodes displayed React Flow handles that looked like editable connection ports.
- **After**: The read-only execution DAG SHALL hide or de-emphasize edit handles while preserving node selection and action controls.
- **Reason**: Execution detail is an inspection surface, not the pipeline designer. Edit-mode affordances create visual noise and imply unsupported interactions.

**Priority**: P2 (Nice-to-have)
**Rationale**: Removing misleading affordances improves clarity without changing behavior.

#### Scenario: read-only nodes do not imply edge editing
- **Given** a user views a completed workflow execution DAG
- **When** they inspect nodes and edges
- **Then** nodes show execution status and actions without prominent editable connection handles

#### Scenario: node interactions still work
- **Given** handles are hidden or de-emphasized
- **When** the user clicks a node or node action
- **Then** selection, logs, runtime, IO, and terminal actions continue to work
