## MODIFIED Requirements

### Requirement: Desktop pipeline designer layout
- **Before**: The system SHALL render the pipeline designer in a desktop layout where side panels may be cramped and the canvas may dominate without enough guidance.
- **After**: The system SHALL render the pipeline designer in a desktop layout where the component palette, canvas, and current node configuration have separate responsibilities, while saved-pipeline management is handled outside the designer side panel.
- **Reason**: Users need a focused graph editing surface; saved-pipeline CRUD should be a first-class management workflow rather than a cramped side panel.

**Priority**: P1 (High)
**Rationale**: The designer is a core workflow surface and layout instability makes routine operations error-prone.

#### Scenario: desktop designer stays focused
- **Given** a desktop user opens the pipeline design tab
- **When** the canvas has no selected node
- **Then** the design tab shows the component palette, canvas, and a current-node configuration area without a saved-pipeline list competing for the same space

#### Scenario: node configuration owns the right panel
- **Given** a desktop user has no node selected
- **When** they inspect the right side of the designer
- **Then** the panel explains that selecting a node will show configuration instead of showing unrelated saved-pipeline CRUD

### Requirement: Empty canvas guidance
- **Before**: The system SHALL show a minimal empty canvas message while retaining canvas controls that can distract from the first action.
- **After**: The system SHALL show clear empty-canvas guidance and reduce nonessential canvas chrome until the user adds nodes.
- **Reason**: First-time users need an obvious starting point before they understand the graph editor controls.

**Priority**: P1 (High)
**Rationale**: Empty-state clarity directly affects whether users can start building a pipeline.

#### Scenario: empty canvas guides first action
- **Given** the design canvas has no nodes
- **When** the user opens the pipeline designer
- **Then** the empty state explains that components can be dragged from the palette and connected before saving

#### Scenario: empty canvas avoids unnecessary controls
- **Given** the design canvas has no nodes
- **When** the canvas is rendered
- **Then** nonessential graph controls such as the mini map do not compete with the empty-state instruction

### Requirement: Pipeline toolbar action feedback
- **Before**: The system SHALL render save, deploy, import, export, and clear actions with limited hierarchy and may disable deploy without explanation.
- **After**: The system SHALL render primary, secondary, and destructive toolbar actions with clear hierarchy and SHALL explain disabled deploy conditions.
- **Reason**: Users need to understand which action advances the workflow and why deployment is unavailable.

**Priority**: P1 (High)
**Rationale**: Ambiguous action states cause failed attempts and reduce confidence in the designer.

#### Scenario: deploy explains why it is disabled
- **Given** the current pipeline cannot be deployed
- **When** the user hovers or focuses the disabled deploy action
- **Then** the UI explains what must be done before deploying

#### Scenario: save provides feedback
- **Given** the user saves a pipeline
- **When** the save operation completes
- **Then** the UI clearly reports success or failure and the saved management area makes the updated pipeline easy to locate

### Requirement: Saved pipeline management readability
- **Before**: The system SHALL show saved pipelines and run history in compact lists that can make long names, metadata, and actions hard to scan.
- **After**: The system SHALL show saved pipelines and run history in a dedicated management tab with readable item structure and focused grouping.
- **Reason**: Users repeatedly create, open, deploy, edit, and delete saved pipelines; those operations require a full management surface rather than a sidebar.

**Priority**: P1 (High)
**Rationale**: Saved pipeline management is the bridge between design and execution.

#### Scenario: saved pipeline rows are scannable
- **Given** saved pipeline names include long or similar names
- **When** the user scans the saved list
- **Then** names, node counts, timestamps, and actions are visually separated and readable

#### Scenario: expanded management view separates concerns
- **Given** the user opens the pipeline management tab
- **When** saved pipelines and run history are both present
- **Then** the UI presents them in a focused organization that avoids mixing both lists into one dense wall of controls

### Requirement: Component and pipeline resource separation
- **Before**: The system SHALL expose component management and saved pipeline management alongside the designer in ways that can blur drag-to-use controls with CRUD controls.
- **After**: The system SHALL expose components and saved pipelines as separate manageable resources: component CRUD in the component management tab, saved-pipeline CRUD in the pipeline management tab, and drag-to-use components in the designer palette.
- **Reason**: Components and pipelines have different lifecycle operations; separating them reduces accidental destructive actions while editing a graph.

**Priority**: P1 (High)
**Rationale**: Clear resource boundaries are needed before expanding pipeline authoring workflows.

#### Scenario: components are managed separately from the palette
- **Given** a user wants to create, edit, delete, or inspect reusable pipeline components
- **When** they open the components tab
- **Then** they can use component CRUD controls there, while the design palette remains focused on dragging components onto the canvas

#### Scenario: saved pipelines are managed separately from the canvas
- **Given** a user wants to create, open, run, delete, or inspect saved pipelines
- **When** they open the pipeline management tab
- **Then** they can use saved-pipeline actions there without interacting with the design canvas side panel
