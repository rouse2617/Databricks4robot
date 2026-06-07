## MODIFIED Requirements

### Requirement: Designer generated name editing
- **Before**: The system SHALL show a generated default pipeline name, but
  browser automation and user edit flows can append a desired name to the
  generated value.
- **After**: The system SHALL replace an untouched generated pipeline name when
  the user begins editing it, while preserving names that the user has already
  edited or loaded from a saved template.
- **Reason**: Generated prefixes persisted into template names make saved
  pipeline lists noisy and make later search/version history harder to use.

**Priority**: P0 (Critical)
**Rationale**: Pipeline names are durable identifiers in the saved-template list
and execution context; accidental generated-name prefixes are costly to clean up.

#### Scenario: Generated name is replaced by first user edit
- **Given** the designer opened a new blank pipeline with a generated default name
- **When** the user focuses the name field and types `qa-designer-flow`
- **Then** the name field value becomes `qa-designer-flow`
- **And** saving the pipeline persists `qa-designer-flow` without the generated prefix

#### Scenario: User-edited name is preserved on later focus
- **Given** the user already changed the generated name to `qa-designer-flow`
- **When** the user focuses the name field again without clearing it
- **Then** the designer does not replace or select the value as a generated default
- **And** additional user editing follows normal input behavior

#### Scenario: Loaded template name is not treated as generated
- **Given** the designer loaded a saved pipeline template with name `existing-template`
- **When** the user focuses the name field
- **Then** the loaded template name remains editable as an ordinary user-owned value

### Requirement: Deterministic component add action
- **Before**: The system SHALL allow components to be dragged into the canvas and
  also wires a card click handler, but the visible affordance primarily says
  "drag into component".
- **After**: The system SHALL expose a visible add action on each component card
  that adds exactly that component to the canvas without relying on drag/drop.
- **Reason**: Drag/drop is useful but hard to automate reliably; users need a
  clear deterministic path for adding components and automated tests need a
  stable interaction seam.

**Priority**: P1 (High)
**Rationale**: The first authoring step in the designer is adding a component;
it must be obvious, deterministic, and testable.

#### Scenario: Add action inserts the selected component
- **Given** the component palette lists `Count Lines` and `Echo Message`
- **When** the user activates the visible add action on `Echo Message`
- **Then** the canvas contains a new `Echo Message` node
- **And** no `Count Lines` node is created by that action

#### Scenario: Add action enables designer controls
- **Given** the designer canvas is empty
- **When** the user activates a component add action
- **Then** the canvas is no longer empty
- **And** save/run controls become available according to existing validation rules

#### Scenario: Drag/drop remains available
- **Given** the component palette is visible
- **When** the user prefers dragging a component into the canvas
- **Then** the drag/drop path remains available as a secondary authoring option
